// cmd/seed-demo — popula o SQLite com dados fake realistas pra demo.
//
// Gera:
//   - ~80 envios STA (accepted/rejected/pending/error) nos últimos 60 dias
//   - ~400 rule_failures distribuídas em 14 dias (heatmap + top-failing)
//   - 12 radar_alerts (mix de critical/warn/info em vários CADOCs)
//   - 60+ audit_events com SHA-256 chain válida via auditlog.Logger oficial
//   - 1 acknowledgement em acknowledged_recommendations
//
// Roda: go run ./cmd/seed-demo -db=radiant.db
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/fortvna/radiant-norma/backend/internal/db"
)

type envio struct {
	ID, IFID, Cadoc, DataBase, Period, Status string
	ProtocolSTA, ErrorCode, ErrorMessage      string
	RulesPassed, RulesFailed, DurationMs      int
	SentAt, ConfirmedAt                       interface{}
	CreatedAt                                 time.Time
	XMLContent, XMLHash, ZIPHash              string
}

func main() {
	dbPath := flag.String("db", "radiant.db", "path to SQLite")
	flag.Parse()

	db, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1) // serializa — chain SHA-256 do audit

	ctx := context.Background()

	for _, t := range []string{"rule_failures", "acknowledged_recommendations", "audit_events", "audit_log", "radar_alerts", "envios"} {
		if _, err := db.ExecContext(ctx, "DELETE FROM "+t); err != nil {
			log.Fatalf("delete %s: %v", t, err)
		}
	}
	fmt.Println("✓ tabelas operacionais limpas")

	ifs, err := loadIFs(ctx, db)
	if err != nil {
		log.Fatalf("load ifs: %v", err)
	}
	if len(ifs) == 0 {
		log.Fatal("nenhuma IF encontrada — rode cmd/seed primeiro")
	}
	fmt.Printf("✓ %d IFs carregadas: %v\n", len(ifs), ifs)

	cadocs := []string{"3040", "3050", "3060", "3070", "4020", "4030"}
	rng := rand.New(rand.NewSource(42))
	now := time.Now().UTC()

	statusDist := []struct {
		Status   string
		Weight   int
		Pass     [2]int
		Fail     [2]int
		Duration [2]int
	}{
		{"accepted", 70, [2]int{45, 60}, [2]int{0, 5}, [2]int{800, 2400}},
		{"rejected", 15, [2]int{30, 50}, [2]int{5, 20}, [2]int{1200, 3800}},
		{"pending", 10, [2]int{0, 0}, [2]int{0, 0}, [2]int{0, 0}},
		{"error", 5, [2]int{20, 40}, [2]int{10, 30}, [2]int{2400, 5000}},
	}

	totalWeight := 0
	for _, s := range statusDist {
		totalWeight += s.Weight
	}

	var envios []envio
	for i := 0; i < 80; i++ {
		dayOffset := rng.Intn(60)
		envTime := now.AddDate(0, 0, -dayOffset).Add(time.Duration(rng.Intn(24)) * time.Hour)
		cadoc := cadocs[rng.Intn(len(cadocs))]
		ifID := ifs[rng.Intn(len(ifs))]

		r := rng.Intn(totalWeight)
		acc := 0
		var pick struct {
			Status   string
			Pass     [2]int
			Fail     [2]int
			Duration [2]int
		}
		for _, s := range statusDist {
			acc += s.Weight
			if r < acc {
				pick.Status = s.Status
				pick.Pass = s.Pass
				pick.Fail = s.Fail
				pick.Duration = s.Duration
				break
			}
		}

		pass := rng.Intn(pick.Pass[1]-pick.Pass[0]+1) + pick.Pass[0]
		fail := 0
		if pick.Fail[1] > 0 {
			fail = rng.Intn(pick.Fail[1]-pick.Fail[0]+1) + pick.Fail[0]
		}
		dur := 0
		if pick.Duration[1] > 0 {
			dur = rng.Intn(pick.Duration[1]-pick.Duration[0]+1) + pick.Duration[0]
		}

		// FIX BUG: data_base em YYYY-MM-DD, period em MM/YYYY (separados!)
		dataBase := envTime.Format("2006-01-02")
		period := envTime.Format("02/2006")
		envioID := "ENV-" + randString(rng, 12)

		var protocol, errorCode, errorMsg string
		var sentAt, confirmedAt interface{} = nil, nil
		switch pick.Status {
		case "accepted":
			protocol = "PSTA300" + randString(rng, 14)
			sentAt = envTime.Format("2006-01-02 15:04:05")
			confirmedAt = envTime.Add(time.Duration(dur) * time.Millisecond).Format("2006-01-02 15:04:05")
		case "rejected":
			protocol = "PSTA300" + randString(rng, 14)
			errorCode = pickRandError(rng)
			errorMsg = rejectionMsg(errorCode)
			sentAt = envTime.Format("2006-01-02 15:04:05")
			confirmedAt = envTime.Add(time.Duration(dur) * time.Millisecond).Format("2006-01-02 15:04:05")
		case "error":
			errorMsg = "Falha de comunicação com STA — timeout após " + fmt.Sprint(dur) + "ms"
			sentAt = envTime.Format("2006-01-02 15:04:05")
		case "pending":
			// tudo nil
		}

		xmlHash := hashStr(envioID + "xml")
		zipHash := hashStr(envioID + "zip")

		envios = append(envios, envio{
			ID: envioID, IFID: ifID, Cadoc: cadoc, DataBase: dataBase, Period: period,
			Status: pick.Status, ProtocolSTA: protocol, RulesPassed: pass, RulesFailed: fail,
			DurationMs: dur, SentAt: sentAt, ConfirmedAt: confirmedAt,
			ErrorCode: errorCode, ErrorMessage: errorMsg,
			XMLContent: "<xml/>", XMLHash: xmlHash, ZIPHash: zipHash,
			CreatedAt: envTime,
		})
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("begin tx: %v", err)
	}
	for _, e := range envios {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO envios (id, if_id, cadoc_code, data_base, remessa, xml_hash, zip_hash,
			                    status, protocol_sta, error_code, error_message,
			                    rules_passed, rules_failed, period, duration_ms,
			                    approver, sent_at, confirmed_at, xml_content, created_at)
			VALUES (?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'sistema', ?, ?, ?, ?)`,
			e.ID, e.IFID, e.Cadoc, e.DataBase, e.XMLHash, e.ZIPHash,
			e.Status, nullIfEmpty(e.ProtocolSTA), nullIfEmpty(e.ErrorCode), nullIfEmpty(e.ErrorMessage),
			e.RulesPassed, e.RulesFailed, e.Period, e.DurationMs,
			e.SentAt, e.ConfirmedAt, e.XMLContent, e.CreatedAt)
		if err != nil {
			log.Fatalf("insert envio %s: %v", e.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatalf("commit envios: %v", err)
	}
	fmt.Printf("✓ %d envios importados\n", len(envios))

	ruleDist := []struct {
		Code, Sev, Cadoc string
		Weight           int
	}{
		{"F23", "E", "3040", 28},
		{"B12", "E", "3040", 18},
		{"S05", "E", "3040", 12},
		{"C04", "A", "3040", 10},
		{"F08", "E", "3050", 8},
		{"B07", "A", "3040", 7},
		{"S11", "E", "3070", 6},
		{"F15", "A", "4020", 5},
		{"C09", "A", "3060", 4},
		{"S03", "E", "3040", 2},
	}
	totalW := 0
	for _, r := range ruleDist {
		totalW += r.Weight
	}

	failureCount := 0
	for i := 0; i < 400; i++ {
		dayOffset := rng.Intn(14)
		failedAt := now.AddDate(0, 0, -dayOffset).Add(-time.Duration(rng.Intn(24)) * time.Hour)

		r := rng.Intn(totalW)
		acc := 0
		var pick struct {
			Code, Sev, Cadoc string
		}
		for _, rd := range ruleDist {
			acc += rd.Weight
			if r < acc {
				pick.Code = rd.Code
				pick.Sev = rd.Sev
				pick.Cadoc = rd.Cadoc
				break
			}
		}

		var envioID string
		err := db.QueryRowContext(ctx,
			"SELECT id FROM envios WHERE cadoc_code=? ORDER BY RANDOM() LIMIT 1",
			pick.Cadoc).Scan(&envioID)
		if err != nil {
			continue
		}

		_, err = db.ExecContext(ctx,
			"INSERT INTO rule_failures (envio_id, if_id, cadoc_code, rule_code, rule_severity, failed_at) VALUES (?, ?, ?, ?, ?, ?)",
			envioID, ifs[rng.Intn(len(ifs))], pick.Cadoc, pick.Code, pick.Sev,
			failedAt.Format("2006-01-02 15:04:05"))
		if err == nil {
			failureCount++
		}
	}
	fmt.Printf("✓ %d rule_failures importados\n", failureCount)

	type auditEv struct {
		IFID, Actor, Action, Target, Description string
		PayloadJSON                              string
		At                                       time.Time
	}

	var evs []auditEv
	evs = append(evs,
		auditEv{"9999901", "auth@radiantnorma", "auth.login", "demo-admin",
			"Login Demo Admin via dev-token", `{"method":"dev","ttl_seconds":604800}`, now.Add(-2 * time.Hour)},
		auditEv{"9999901", "system@radar", "schema.synced", "3040",
			"Schema 3040 v3.4.1 sincronizado via BACEN", `{"cadoc":"3040","version":"3.4.1","source":"bcb.gov.br"}`, now.Add(-18 * time.Hour)},
		auditEv{"9999901", "system@radar", "rule.enabled", "B12",
			"Regra B12 reabilitada após fix", `{"rule_code":"B12","reason":"false_positive_resolved"}`, now.Add(-2 * 24 * time.Hour)},
		auditEv{"9999901", "system@radar", "rule.enabled", "F08",
			"Regra F08 reabilitada após fix", `{"rule_code":"F08"}`, now.Add(-5 * 24 * time.Hour)},
	)

	for _, e := range envios {
		switch e.Status {
		case "accepted":
			evs = append(evs, auditEv{
				e.IFID, "system", "envio.approved", e.ID,
				fmt.Sprintf("CADOC %s base %s aprovado · %d regras passaram", e.Cadoc, e.Period, e.RulesPassed),
				fmt.Sprintf(`{"cadoc":"%s","period":"%s","rules_passed":%d,"rules_failed":%d,"duration_ms":%d}`,
					e.Cadoc, e.Period, e.RulesPassed, e.RulesFailed, e.DurationMs),
				e.CreatedAt,
			})
		case "rejected":
			evs = append(evs, auditEv{
				e.IFID, "system", "envio.rejected", e.ID,
				fmt.Sprintf("CADOC %s base %s rejeitado · %d regras falharam (%s)",
					e.Cadoc, e.Period, e.RulesFailed, e.ErrorCode),
				fmt.Sprintf(`{"cadoc":"%s","period":"%s","rules_passed":%d,"rules_failed":%d,"error_code":"%s"}`,
					e.Cadoc, e.Period, e.RulesPassed, e.RulesFailed, e.ErrorCode),
				e.CreatedAt,
			})
		}
	}

	for _, e := range envios {
		if e.Status == "pending" {
			continue
		}
		evs = append(evs, auditEv{
			e.IFID, "sta-submit", "sta.submit", e.ID,
			fmt.Sprintf("STA submit iniciado · CADOC %s · %s", e.Cadoc, e.Period),
			fmt.Sprintf(`{"cadoc":"%s","period":"%s","xml_size":1247}`, e.Cadoc, e.Period),
			e.CreatedAt.Add(-time.Minute),
		})
	}

	// Sort por timestamp
	for i := 0; i < len(evs); i++ {
		for j := i + 1; j < len(evs); j++ {
			if evs[j].At.Before(evs[i].At) {
				evs[i], evs[j] = evs[j], evs[i]
			}
		}
	}

	// INSERT manual com timestamp RFC3339Nano (compatível com Verify do logger).
	// Usa BEGIN IMMEDIATE implícito via _txlock=immediate no DSN.
	prevHash := strings.Repeat("0", 64)
	inserted := 0
	for _, ev := range evs {
		payloadHash := sha256.Sum256([]byte(ev.PayloadJSON))
		payloadHashHex := hex.EncodeToString(payloadHash[:])
		timestamp := ev.At.UTC().Format(time.RFC3339Nano)
		metaJSON := fmt.Sprintf(`{"description":%q,"if_id":%q}`, ev.Description, ev.IFID)
		concat := prevHash + payloadHashHex + metaJSON + ev.Actor + ev.Action + ev.Target + ev.IFID + timestamp
		entrySum := sha256.Sum256([]byte(concat))
		entryHash := hex.EncodeToString(entrySum[:])

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			log.Fatalf("audit tx: %v", err)
		}
		res, err := tx.ExecContext(ctx,
			"INSERT INTO audit_log (if_id, actor, action, target, payload_hash, prev_hash, entry_hash, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			ev.IFID, ev.Actor, ev.Action, ev.Target,
			payloadHashHex, prevHash, entryHash, metaJSON, timestamp)
		if err != nil {
			tx.Rollback()
			log.Fatalf("audit_log insert: %v", err)
		}
		auditLogID, err := res.LastInsertId()
		if err != nil {
			tx.Rollback()
			log.Fatalf("last_insert_id: %v", err)
		}
		_, err = tx.ExecContext(ctx,
			"INSERT INTO audit_events (audit_log_id, if_id, actor, action, target, description, payload, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			auditLogID, ev.IFID, ev.Actor, ev.Action, ev.Target, ev.Description, ev.PayloadJSON, timestamp)
		if err != nil {
			tx.Rollback()
			log.Fatalf("audit_events insert: %v", err)
		}
		if err := tx.Commit(); err != nil {
			log.Fatalf("audit commit: %v", err)
		}
		prevHash = entryHash
		inserted++

		if errors.Is(err, context.DeadlineExceeded) {
			log.Fatalf("timeout — encurtar ou ajustar busy_timeout")
		}
	}
	fmt.Printf("✓ %d audit_events importados (SHA-256 chain válida)\n", inserted)

	radarAlerts := []struct {
		Cadoc, Severity, AlertType, Title, Description, SourceURL string
		DaysAgo                                                   int
	}{
		{"3040", "critical", "leiaute_mudou",
			"CADOC 3040: nova versão do leiaute BACEN publicada",
			"Hash anterior: 7a3f9c... | Hash novo: b2e841...\nAlterações detectadas em 14 campos estruturais. Reanálise automática em curso.",
			"https://www.bcb.gov.br/estabilidadefinanceira/cadoc3040", 0},
		{"3040", "critical", "prazo_encerrando",
			"3040: prazo de envio fecha em 2 dias úteis",
			"Data-base 06/2026 deve ser enviada até 08/07/2026 às 23:59 BRT.",
			"https://www.bcb.gov.br/estabilidadefinanceira/calendario-cadoc", 0},
		{"3050", "warn", "documentacao_atualizada",
			"3050 v11: novas instruções normativas publicadas",
			"IN BCB 412/2024 altera procedimentos de classificação. Impacto esperado em ~12 regras da categoria C.",
			"https://www.bcb.gov.br/estabilidadefinanceira/in412", 2},
		{"3040", "warn", "schema_sync",
			"3040: sincronização automática concluída",
			"Schema 3040 v3.4.1 baixado e validado contra baseline. Nenhuma mudança requer ação imediata.",
			"https://www.bcb.gov.br/estabilidadefinanceira/cadoc3040", 6},
		{"3070", "warn", "novo_cadoc",
			"3070: nova categoria de crítica publicada",
			"BACEN incluiu 3 novas críticas estruturais para CADOC 3070 (DLO). Cobertura atual: 87%.",
			"https://www.bcb.gov.br/estabilidadefinanceira/cadoc3070", 12},
		{"3060", "warn", "prazo_alterado",
			"3060: cronograma de envios atualizado",
			"Prazos trimestrais alterados para DLP. Próximo envio: 15/07/2026.",
			"https://www.bcb.gov.br/estabilidadefinanceira/cadoc3060", 9},
		{"4020", "info", "novo_leiaute",
			"4020: leiaute revisado publicado",
			"Revisão anual do leiaute DRL — sem impacto em regras ativas.",
			"https://www.bcb.gov.br/estabilidadefinanceira/cadoc4020", 18},
		{"4030", "info", "documentacao_atualizada",
			"4030: FAQ atualizado",
			"FAQ BACEN recebeu 2 novas perguntas relacionadas a DRP. Documentação sincronizada.",
			"https://www.bcb.gov.br/faq-cadoc4030", 24},
		{"3040", "info", "schema_sync",
			"3040: varredura periódica concluída",
			"Nenhuma mudança estrutural detectada nas últimas 24h.",
			"https://www.bcb.gov.br/estabilidadefinanceira/cadoc3040", 30},
		{"3050", "info", "novo_leiaute",
			"3050: atualização menor de documentação",
			"Documentação de campos auxiliares atualizada — sem impacto em validações.",
			"https://www.bcb.gov.br/estabilidadefinanceira/cadoc3050", 36},
		{"3070", "info", "schema_sync",
			"3070: varredura periódica concluída",
			"Baseline estável nas últimas 72h.",
			"https://www.bcb.gov.br/estabilidadefinanceira/cadoc3070", 48},
		{"3040", "warn", "documentacao_atualizada",
			"3040: manuais operacionais revisados",
			"Manuais BACEN atualizados para refletir mudanças no procedimento de retificação.",
			"https://www.bcb.gov.br/estabilidadefinanceira/manuais", 14},
	}

	for _, a := range radarAlerts {
		detectedAt := now.AddDate(0, 0, -a.DaysAgo).Format("2006-01-02 15:04:05")
		_, err := db.ExecContext(ctx,
			"INSERT INTO radar_alerts (cadoc_code, alert_type, severity, title, description, source_url, detected_at, resolved_at) VALUES (?, ?, ?, ?, ?, ?, ?, NULL)",
			a.Cadoc, a.AlertType, a.Severity, a.Title, a.Description, a.SourceURL, detectedAt)
		if err != nil {
			log.Fatalf("radar insert: %v", err)
		}
	}
	fmt.Printf("✓ %d radar_alerts importados\n", len(radarAlerts))

	_, err = db.ExecContext(ctx,
		"INSERT INTO acknowledged_recommendations (if_id, rec_id, acknowledged_at, acknowledged_by) VALUES (?, ?, ?, ?)",
		"demo", "concentration", now.Add(-3*24*time.Hour).Format("2006-01-02 15:04:05"),
		"admin@demo")
	if err != nil {
		fmt.Printf("  (ack insert falhou: %v — ignorado)\n", err)
	}

	fmt.Println("\n✅ Seed demo completo!")
	fmt.Println("   IFs:        ", ifs)
	fmt.Println("   Envios:     ", len(envios))
	fmt.Println("   Failures:   ", failureCount)
	fmt.Println("   Audit evts: ", inserted)
	fmt.Println("   Radar:      ", len(radarAlerts))
}

func loadIFs(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, "SELECT id FROM ifs ORDER BY id LIMIT 10")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func hashStr(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func randString(rng *rand.Rand, n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}
	return string(b)
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func pickRandError(rng *rand.Rand) string {
	codes := []string{"F23", "B12", "S05", "C04", "F08", "B07", "S11"}
	return codes[rng.Intn(len(codes))]
}

func rejectionMsg(code string) string {
	msgs := map[string]string{
		"F23": "CNPJ inválido (formato)",
		"B12": "Cabeçalho fora do layout esperado",
		"S05": "Soma de subtotais não confere",
		"C04": "Campo obrigatório ausente",
		"F08": "Formato de data inválido",
		"B07": "Encoding não suportado (esperado UTF-8)",
		"S11": "Coerência cruzada entre blocos falhou",
	}
	return msgs[code]
}

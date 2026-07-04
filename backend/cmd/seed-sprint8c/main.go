// cmd/seed-sprint8c: popula o banco com dados realistas de envios + audit
// + rule_failures, para destravar o frontend v3.0.0 que estava em empty
// states (e tinha dados fake hardcoded em alguns lugares — validação 29).
//
// Uso:
//
//	go run ./cmd/seed-sprint8c
//
// Idempotente: limpa envios/audit_events/rule_failures antes de popular.
// Gera ~50 envios, 200+ audit events, 300+ rule failures — números que
// fazem o dashboard mostrar dados realistas sem inflar.
package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/db"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	dbPath := os.Getenv("DATABASE_URL")
	if dbPath == "" {
		dbPath = "radiant.db"
	}
	d, err := db.Open(dbPath)
	if err != nil {
		logger.Error("open db", "err", err)
		os.Exit(1)
	}
	defer d.Close()

	if err := db.Migrate(d); err != nil {
		logger.Error("migrate", "err", err)
		os.Exit(1)
	}

	// Garante IFs existentes
	if _, err := d.Exec(`
		INSERT OR IGNORE INTO ifs (id, cnpj, nome, tipo, segmento, plano)
		VALUES
			('demo', '12345678', 'IF Demonstração SCD', 'SCD', 'S5', 'pro'),
			('demo-banco', '00000000', 'Banco Demo', 'BC', 'S1', 'scale'),
			('9999901', '99999010', 'IF Demo Admin', 'BC', 'S1', 'scale')
	`); err != nil {
		logger.Error("seed ifs", "err", err)
		os.Exit(1)
	}

	auditLog := auditlog.New(d)

	// Limpa dados anteriores (idempotente)
	for _, table := range []string{"rule_failures", "audit_events", "envios"} {
		if _, err := d.Exec("DELETE FROM " + table); err != nil {
			logger.Error("clean "+table, "err", err)
			os.Exit(1)
		}
	}
	logger.Info("tablas limpas")

	// Seed envios
	if err := seedEnvios(d, auditLog, logger); err != nil {
		logger.Error("seed envios", "err", err)
		os.Exit(1)
	}

	// Seed rule_failures
	if err := seedRuleFailures(d, logger); err != nil {
		logger.Error("seed rule failures", "err", err)
		os.Exit(1)
	}

	logger.Info("✓ seed-sprint8c completo")
}

func seedEnvios(d *sql.DB, auditLog *auditlog.Logger, logger *slog.Logger) error {
	cadocs := []string{"3040", "3050", "3060", "3070", "4020", "4030"}
	statuses := []struct {
		Status    string
		Weight    int
		PassRange [2]int // [min, max] rules passing
		FailRange [2]int
	}{
		{"accepted", 70, [2]int{45, 60}, [2]int{0, 5}},
		{"rejected", 15, [2]int{30, 50}, [2]int{5, 20}},
		{"pending", 10, [2]int{0, 0}, [2]int{0, 0}},
		{"error", 5, [2]int{20, 40}, [2]int{10, 30}},
	}
	ifs := []string{"demo", "demo-banco", "9999901"}

	rng := rand.New(rand.NewSource(42)) // deterministic seed

	totalInserted := 0
	now := time.Now()

	// Gera ~50 envios ao longo dos últimos 30 dias
	for dayOffset := 0; dayOffset < 30; dayOffset++ {
		// Distribuição: ~2 envios/dia com random
		enviosPerDay := 1 + rng.Intn(3)
		for i := 0; i < enviosPerDay; i++ {
			// Escolhe status ponderado
			r := rng.Intn(100)
			cum := 0
			var chosen struct {
				Status    string
				PassRange [2]int
				FailRange [2]int
			}
			for _, s := range statuses {
				cum += s.Weight
				if r < cum {
					chosen.Status = s.Status
					chosen.PassRange = s.PassRange
					chosen.FailRange = s.FailRange
					break
				}
			}

			cadoc := cadocs[rng.Intn(len(cadocs))]
			ifID := ifs[rng.Intn(len(ifs))]

			// Period: MM/YYYY baseado no envio
			envTime := now.AddDate(0, 0, -dayOffset).Add(time.Duration(rng.Intn(24)) * time.Hour)
			period := fmt.Sprintf("%02d/%04d", (envTime.Month()-1)%12+1, envTime.Year())

			envioID := "ENV-" + strconv.FormatInt(envTime.UnixNano(), 36)
			pass := 0
			fail := 0
			if chosen.Status != "pending" {
				pass = chosen.PassRange[0] + rng.Intn(chosen.PassRange[1]-chosen.PassRange[0]+1)
				fail = chosen.FailRange[0] + rng.Intn(chosen.FailRange[1]-chosen.FailRange[0]+1)
			}
			durationMs := 800 + rng.Intn(3000)
			xmlHash := sha256.Sum256([]byte(envioID + cadoc))
			zipHash := sha256.Sum256([]byte(envioID + "zip"))

			protocolSTA := ""
			errorCode := ""
			errorMsg := ""
			sentAt := "NULL"
			confirmedAt := "NULL"

			switch chosen.Status {
			case "accepted":
				protocolSTA = fmt.Sprintf("%018d", rng.Int63n(999999999999999999))
				sentAt = "'" + envTime.Format("2006-01-02 15:04:05") + "'"
				confirmedAt = "'" + envTime.Add(time.Duration(durationMs)*time.Millisecond).Format("2006-01-02 15:04:05") + "'"
			case "rejected":
				protocolSTA = fmt.Sprintf("%018d", rng.Int63n(999999999999999999))
				errorCode = "F23"
				errorMsg = "CNPJ inválido (formato)"
				sentAt = "'" + envTime.Format("2006-01-02 15:04:05") + "'"
				confirmedAt = "'" + envTime.Add(time.Duration(durationMs)*time.Millisecond).Format("2006-01-02 15:04:05") + "'"
			case "error":
				errorMsg = "Falha de comunicação com STA"
				sentAt = "'" + envTime.Format("2006-01-02 15:04:05") + "'"
				confirmedAt = "NULL"
			}

			_, err := d.Exec(`
				INSERT INTO envios (id, if_id, cadoc_code, data_base, remessa,
				                    xml_hash, zip_hash, status, protocol_sta,
				                    error_code, error_message, rules_passed, rules_failed,
				                    period, duration_ms, approver, sent_at, confirmed_at,
				                    xml_content, created_at)
				VALUES (?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'sistema',
				        `+sentAt+`, `+confirmedAt+`, ?, ?)
			`,
				envioID, ifID, cadoc, period,
				hex.EncodeToString(xmlHash[:]), hex.EncodeToString(zipHash[:]),
				chosen.Status, protocolSTA,
				errorCode, errorMsg, pass, fail,
				period, durationMs,
				"<xml/>", envTime,
			)
			if err != nil {
				logger.Warn("insert envio falhou", "id", envioID, "err", err)
				continue
			}
			totalInserted++

			// Audit log: emite 1 evento "sta.submit" + opcional "sta.approved/rejected"
			_, _ = auditLog.Log(ifID, "system", "sta.submit", cadoc, []byte(envioID), map[string]any{
				"envio_id": envioID,
				"period":   period,
				"rules_passed": pass,
				"rules_failed": fail,
				"duration_ms":  durationMs,
			})

			if chosen.Status == "accepted" {
				emitAuditEvent(d, envTime.Add(time.Minute), ifID, "system", "envio.approved", envioID,
					fmt.Sprintf("CADOC %s base %s aprovado · %d regras passaram", cadoc, period, pass),
					map[string]any{"rules_passed": pass, "rules_failed": fail})
			} else if chosen.Status == "rejected" {
				emitAuditEvent(d, envTime.Add(time.Minute), ifID, "system", "envio.rejected", envioID,
					fmt.Sprintf("CADOC %s base %s rejeitado · %d regras falharam", cadoc, period, fail),
					map[string]any{"rules_passed": pass, "rules_failed": fail, "error_code": errorCode})
			}
		}
	}

	logger.Info("✓ envios importados", "total", totalInserted)

	// Também emite eventos de login/regra/schema pra activity feed ficar rica
	emitSystemEvents(d, now, logger)

	return nil
}

func emitAuditEvent(d *sql.DB, at time.Time, ifID, actor, action, target, description string, payload map[string]any) {
	payloadJSON, _ := json.Marshal(payload)
	// Escreve em audit_log (chain) E audit_events (denormalizado)
	_, _ = d.Exec(`
		INSERT INTO audit_log (if_id, actor, action, target, payload_hash,
		                        prev_hash, entry_hash, metadata, created_at)
		VALUES (?, ?, ?, ?, 'placeholder-hash',
		        'placeholder-prev', 'placeholder-entry', ?, ?)
	`, ifID, actor, action, target, string(payloadJSON), at)
	logID, _ := d.Exec(`SELECT last_insert_rowid()`)
	id, _ := logID.LastInsertId()
	_, _ = d.Exec(`
		INSERT INTO audit_events (audit_log_id, if_id, actor, action, target,
		                          description, payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, ifID, actor, action, target, description, string(payloadJSON), at)
}

func emitSystemEvents(d *sql.DB, now time.Time, logger *slog.Logger) {
	// Login recente
	emitAuditEvent(d, now.Add(-2*time.Hour), "9999901", "9999901", "auth.login", "",
		"Sessão iniciada", map[string]any{"ip": "10.0.x.x"})

	// Schema synced (ontem)
	emitAuditEvent(d, now.Add(-18*time.Hour), "9999901", "system", "schema.synced", "3040",
		"Schema 3040 v8.2.1 sincronizado", map[string]any{"from": "8.2.0", "to": "8.2.1"})

	// Radar detected (alguns dias atrás)
	emitAuditEvent(d, now.Add(-3*24*time.Hour), "9999901", "system", "radar.detected", "3040",
		"URL BACEN 3040 alterada", map[string]any{"old_url": "https://...", "new_url": "https://..."})

	// Rule enabled
	emitAuditEvent(d, now.Add(-2*24*time.Hour), "9999901", "9999901", "rule.enabled", "B12",
		"Regra B12 habilitada — campos obrigatórios", map[string]any{"rule": "B12", "previous": "disabled"})

	logger.Info("✓ eventos de sistema importados")
}

func seedRuleFailures(d *sql.DB, logger *slog.Logger) error {
	// Distribuição realista: F23 é o top (CNPJ), depois B12, S05, etc.
	rules := []struct {
		Code     string
		Severity string
		Weight   int // % do total de failures
		Cadoc    string
	}{
		{"F23", "E", 28, "3040"},
		{"B12", "E", 18, "3040"},
		{"S05", "A", 12, "3040"},
		{"C04", "E", 10, "3040"},
		{"F08", "A", 8, "3050"},
		{"B07", "E", 7, "3040"},
		{"S11", "A", 6, "3070"},
		{"F15", "I", 5, "4020"},
		{"C09", "E", 4, "3060"},
		{"S03", "A", 2, "3040"},
	}

	rng := rand.New(rand.NewSource(43))
	totalRules := 320

	// Distribui failures no tempo (últimos 14 dias, com picos aleatórios)
	now := time.Now()

	// Pega envio IDs existentes pra associar failures
	rows, err := d.Query(`SELECT id, if_id, cadoc_code FROM envios`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type envioRow struct {
		ID, IFID, Cadoc string
	}
	var envios []envioRow
	for rows.Next() {
		var e envioRow
		_ = rows.Scan(&e.ID, &e.IFID, &e.Cadoc)
		envios = append(envios, e)
	}
	if len(envios) == 0 {
		logger.Warn("sem envios — pulando rule_failures")
		return nil
	}

	tx, err := d.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`
		INSERT INTO rule_failures (envio_id, if_id, cadoc_code, rule_code, rule_severity, failed_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	totalInserted := 0
	for i := 0; i < totalRules; i++ {
		// Sorteia rule baseada em weight
		r := rng.Intn(100)
		cum := 0
		var chosen struct {
			Code, Severity, Cadoc string
		}
		for _, rule := range rules {
			cum += rule.Weight
			if r < cum {
				chosen.Code = rule.Code
				chosen.Severity = rule.Severity
				chosen.Cadoc = rule.Cadoc
				break
			}
		}

		// Associa a um envio do mesmo CADOC (preferencialmente)
		var candidatos []envioRow
		for _, e := range envios {
			if e.Cadoc == chosen.Cadoc {
				candidatos = append(candidatos, e)
			}
		}
		if len(candidatos) == 0 {
			candidatos = envios
		}
		envio := candidatos[rng.Intn(len(candidatos))]

		// Time: últimos 14 dias, distribuição não-uniforme (mais falhas nos últimos dias)
		daysAgo := rng.Intn(14)
		hoursAgo := rng.Intn(24)
		failedAt := now.AddDate(0, 0, -daysAgo).Add(-time.Duration(hoursAgo) * time.Hour)
		// Formato compatível com strftime do SQLite (sem timezone offset).
		// Caso contrário, strftime('%Y-%m-%d', ...) retorna NULL.
		failedAtSQLite := failedAt.UTC().Format("2006-01-02 15:04:05")

		_, err := stmt.Exec(envio.ID, envio.IFID, envio.Cadoc, chosen.Code, chosen.Severity, failedAtSQLite)
		if err != nil {
			logger.Warn("insert failure falhou", "err", err)
			continue
		}
		totalInserted++
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	logger.Info("✓ rule_failures importadas", "total", totalInserted)
	return nil
}
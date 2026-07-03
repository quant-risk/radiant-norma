// cmd/seed: popula o banco com críticas e leiautes a partir dos JSONs extraídos.
//
// Uso:
//
//	go run ./cmd/seed -json ../_catalogos/criticas.json -leiautes ../_catalogos/leiautes.json -xsd ../_catalogos/3040_generated.xsd
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/db"
)

type CriticaJSON struct {
	Cadoc           string `json:"cadoc"`
	Sheet           string `json:"sheet"`
	Codigo          string `json:"codigo"`
	Regra           string `json:"regra"`
	Descricao       string `json:"descrição"`
	DescricaoCritica string `json:"descrição da crítica"` // 2061 DLO
	DescricaoRegra   string `json:"descrição da regra"`     // 3050
	Observacoes    string `json:"observações"`
	Gravidade      string `json:"gravidade"`
	TipoIndicio    string `json:"tipo"`
	DataBaseInicio string `json:"data-base inicio"`
	DataBaseInicioRaw any `json:"data-base_inicio"`
	MensagemErro   string `json:"mensagem de erro"`
	Mensagem       string `json:"mensagem"`
	Habilitado     string `json:"habilitado?"`
	Enabled        *bool  `json:"enabled"`
	Source         string `json:"fonte"`
	BaseConfrontada string `json:"base_confrontada"`
	Tipo            string `json:"tipo_"` // "tipo" já usado por TipoIndicio
}

type CriticasFile struct {
	Metadata map[string]any                     `json:"_metadata"`
	Criticas map[string][]map[string]any        `json:"criticas"`
}

type LeiautesFile struct {
	Metadata map[string]any                     `json:"_metadata"`
	Leiautes map[string]json.RawMessage         `json:"leiautes"`
}

func main() {
	jsonPath := flag.String("json", "../_catalogos/criticas.json", "caminho do criticas.json")
	leiautesPath := flag.String("leiautes", "../_catalogos/leiautes.json", "caminho do leiautes.json")
	xsdPath := flag.String("xsd", "../_catalogos/3040_generated.xsd", "caminho do 3040_generated.xsd")
	dbPath := flag.String("db", "radiant.db", "caminho do banco SQLite")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	// DB + migrations
	d, err := db.Open(*dbPath)
	if err != nil {
		logger.Error("open db", "err", err)
		os.Exit(1)
	}
	defer d.Close()

	if err := db.Migrate(d); err != nil {
		logger.Error("migrate", "err", err)
		os.Exit(1)
	}

	// Seed IFs (multi-tenant demo)
	if err := seedIFs(d, logger); err != nil {
		logger.Error("seed ifs", "err", err)
		os.Exit(1)
	}

	// Seed críticas
	if err := seedCriticas(d, *jsonPath, logger); err != nil {
		logger.Error("seed criticas", "err", err)
		os.Exit(1)
	}

	// Seed schema_registry
	if err := seedSchemaRegistry(d, *leiautesPath, *xsdPath, logger); err != nil {
		logger.Error("seed schemas", "err", err)
		os.Exit(1)
	}

	logger.Info("✓ seed completo")
}

// seedIFs popula 1 IF de exemplo (demo) usada em testes via header X-IF-ID: demo.
// A FK constraint de envios.if_id exige ifs pré-cadastradas.
func seedIFs(d *sql.DB, logger *slog.Logger) error {
	ifs := []struct {
		ID       string
		CNPJ     string
		Nome     string
		Tipo     string
		Segmento string
		Plano    string
	}{
		{"demo", "12345678", "IF Demonstração SCD", "SCD", "S5", "pro"},
		{"demo-banco", "00000000", "Banco Demo", "BC", "S1", "scale"},
	}

	stmt, err := d.Prepare(`
		INSERT OR REPLACE INTO ifs (id, cnpj, nome, tipo, segmento, plano)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, i := range ifs {
		if _, err := stmt.Exec(i.ID, i.CNPJ, i.Nome, i.Tipo, i.Segmento, i.Plano); err != nil {
			return fmt.Errorf("insert if %s: %w", i.ID, err)
		}
	}
	logger.Info("✓ ifs importadas", "total", len(ifs))
	return nil
}

func seedCriticas(d *sql.DB, path string, logger *slog.Logger) error {
	logger.Info("lendo criticas.json", "path", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cf CriticasFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return err
	}

	totalInserted := 0
	for cadoc, lista := range cf.Criticas {
		logger.Info("importando", "cadoc", cadoc, "count", len(lista))

		// Limpa críticas antigas desse CADOC (re-seed limpo)
		if _, err := d.Exec("DELETE FROM criticas WHERE cadoc_code = ?", cadoc); err != nil {
			return err
		}

		stmt, err := d.Prepare(`
			INSERT INTO criticas (cadoc_code, sheet, codigo, regra, descricao, gravidade,
			                       tipo_indicio, data_base_inicio, mensagem_erro, enabled, source)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}

		for _, raw := range lista {
			c := mapToCritica(cadoc, raw)
			enabled := 1
			if c.Enabled != nil && !*c.Enabled {
				enabled = 0
			}
			var dataInicio any
			if c.DataBaseInicio != "" {
				t, err := time.Parse("2006-01-02", c.DataBaseInicio)
				if err == nil {
					dataInicio = t.Format("2006-01-02")
				} else {
					dataInicio = nil
				}
			}
			_, err := stmt.Exec(
				cadoc, c.Sheet, c.Codigo, c.Regra, c.Descricao, c.Gravidade,
				c.TipoIndicio, dataInicio, c.MensagemErro, enabled, c.Source,
			)
			if err != nil {
				logger.Warn("insert falhou", "codigo", c.Codigo, "err", err)
				continue
			}
			totalInserted++
		}
		stmt.Close()
	}
	logger.Info("✓ criticas importadas", "total", totalInserted)
	return nil
}

func mapToCritica(cadoc string, raw map[string]any) CriticaJSON {
	c := CriticaJSON{
		Cadoc:     cadoc,
		Sheet:     strOr(raw, "sheet", ""),
		Codigo:    strOr(raw, "codigo", ""),
		Regra:     firstNonEmpty(raw, "regra", "crítica", "critica"),
		Descricao: firstNonEmpty(raw, "descrição", "descricao", "descrição da crítica", "descricao da critica", "descrição da regra"),
	}
	c.Gravidade = strOr(raw, "gravidade", "")
	c.TipoIndicio = strOr(raw, "tipo_indicio", "")
	c.MensagemErro = firstNonEmpty(raw, "mensagem de erro", "mensagem_erro", "mensagem")
	c.Source = strOr(raw, "fonte", "")
	c.Tipo = strOr(raw, "tipo", "")
	c.BaseConfrontada = strOr(raw, "base_confrontada", "")
	c.DataBaseInicio = firstNonEmpty(raw, "data-base inicio", "data-base_inicio")

	// Habilitado
	if h, ok := raw["habilitado?"].(string); ok && h == "n" {
		c.Enabled = boolPtr(false)
	} else if e, ok := raw["enabled"].(*bool); ok && e != nil {
		c.Enabled = e
	} else {
		c.Enabled = boolPtr(true)
	}

	return c
}

func seedSchemaRegistry(d *sql.DB, jsonPath, xsdPath string, logger *slog.Logger) error {
	logger.Info("lendo leiautes.json", "path", jsonPath)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}
	var lf LeiautesFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return err
	}

	// XSD 3040 (lê do arquivo separado)
	xsd3040 := ""
	if f, err := os.ReadFile(xsdPath); err == nil {
		xsd3040 = string(f)
	}

	totalInserted := 0
	for cadoc, rawLei := range lf.Leiautes {
		logger.Info("importando schema", "cadoc", cadoc)
		if _, err := d.Exec("DELETE FROM schema_versions WHERE cadoc_code = ?", cadoc); err != nil {
			return err
		}

		// Decode leiaute
		var lei struct {
			Source    string                   `json:"source"`
			TotalRows int                      `json:"total_rows"`
			Rows      []map[string]any         `json:"rows"`
		}
		if err := json.Unmarshal(rawLei, &lei); err != nil {
			logger.Warn("decode leiaute falhou", "cadoc", cadoc, "err", err)
			continue
		}

		// Mapeia rows para fields (formato simplificado)
		fields := extractFields(lei.Rows)
		fieldsJSON, _ := json.Marshal(fields)

		// Pega effective_from (data atual ou 2000-01-01 se desconhecida)
		effective := "2000-01-01"

		// XSD só pra 3040
		xsd := ""
		if cadoc == "3040" {
			xsd = xsd3040
		}

		_, err := d.Exec(`
			INSERT INTO schema_versions (cadoc_code, effective_from, source_uri, fields_json, xsd, changelog)
			VALUES (?, ?, ?, ?, ?, ?)
		`,
			cadoc, effective, lei.Source, string(fieldsJSON), nullable(xsd),
			fmt.Sprintf("Sprint 3 seed — %d rows extraídos do %s", lei.TotalRows, filepath.Base(jsonPath)),
		)
		if err != nil {
			logger.Warn("insert schema falhou", "cadoc", cadoc, "err", err)
			continue
		}
		totalInserted++
	}
	logger.Info("✓ schemas importados", "total", totalInserted)
	return nil
}

func extractFields(rows []map[string]any) []map[string]any {
	var fields []map[string]any
	for _, r := range rows {
		vals, _ := r["values"].([]any)
		if len(vals) < 1 {
			continue
		}
		first, _ := vals[0].(string)
		if first == "" {
			continue
		}
		// Linha de header: ["Campo", "Formato", ...]
		if first == "Campo" {
			continue
		}
		fields = append(fields, map[string]any{
			"value": first,
			"all":   vals,
		})
		if len(fields) >= 500 { // limita
			break
		}
	}
	return fields
}

func strOr(m map[string]any, key, def string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return def
}

func firstNonEmpty(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func boolPtr(b bool) *bool { return &b }

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
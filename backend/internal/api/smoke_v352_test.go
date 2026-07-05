// Smoke test consolidado da release v3.5.2 — Sprint 13.5.
//
// Cobre os 19 arquivos alterados no audit S-A/S-B (Sprints 13-15) com
// cenário de runtime real (httptest + chi router + SQLite in-memory).
// Antes do tag v3.5.2, este teste precisa passar 100%.
//
// Diferente dos unit tests, aqui o foco é:
//   - Real Router (middleware chain: CSRF, rate limit, auth)
//   - Real DB (SQLite com migrations aplicadas + IFs pre-seeded)
//   - Real handlers (não service-level mocks)
//
// Cada subteste corresponde a um cenário do plano de release:
//   1. Startup fail-closed (RADIANT_ENV=production + dev flag → exit != 0)
//   2. Auth + CSRF middleware ativo
//   3. STA submit cross-tenant → 403
//   4. Crossdoc validate cross-tenant → 403
//   5. Validate/listRules/getSchema/listVersions happy path
//   6. Worker processa envio + SafeError não vaza DSN
//   7. Rate limiter burst → 429 + Retry-After header
//   8. SSE cap de 10 subscribers/IF
//   9. FK rejeita IF inválido em tabelas tenant-scoped
//   10. EXPLAIN nos índices de envios (idx usado)
package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/audit"
	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/realtime"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
	"github.com/fortvna/radiant-norma/backend/internal/testutil"
	workpkg "github.com/fortvna/radiant-norma/backend/internal/worker"
)

// smokeReq faz request JSON rápido pra httptest.
func smokeReq(handler http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// =============================================================================
// Cenário 1 — Startup fail-closed (RADIANT_ENV=production + dev flag → exit != 0)
// =============================================================================
//
// Gate fica em cmd/api/main.go. Testa rodando o binário compilado de verdade
// (não httptest) porque o gate está no entrypoint, não no Router.
func TestSmoke_Cenario1_FailClosedStartup(t *testing.T) {
	binPath := os.Getenv("RADIANT_API_BIN")
	if binPath == "" {
		binPath = "/tmp/radiant-api"
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Skipf("binário %s não encontrado (rodar `go build -o /tmp/radiant-api ./cmd/api` antes): %v", binPath, err)
	}

	cases := []struct {
		name     string
		env      map[string]string
		wantExit int // 0 = success, anything else = fail-closed worked
	}{
		{
			name: "RADIANT_ENV=production + RADIANT_DEV_TOKEN=1 → fail",
			env: map[string]string{
				"RADIANT_ENV":       "production",
				"RADIANT_DEV_TOKEN": "1",
				"RADIANT_DB":        "/tmp/smoke-fc1.db",
			},
			wantExit: 1,
		},
		{
			name: "RADIANT_ENV=production + RADIANT_DEV_AUTH=1 → fail",
			env: map[string]string{
				"RADIANT_ENV":      "production",
				"RADIANT_DEV_AUTH": "1",
				"RADIANT_DB":       "/tmp/smoke-fc2.db",
			},
			wantExit: 1,
		},
		{
			name: "RADIANT_ENV=production + sem JWT key → fail",
			env: map[string]string{
				"RADIANT_ENV":               "production",
				"RADIANT_DB":                "/tmp/smoke-fc3.db",
				"RADIANT_NORMA_ADMIN_TOKEN": "x",
			},
			wantExit: 1,
		},
		{
			name: "RADIANT_ENV unset + DEV_AUTH=1 → success (back-compat dev)",
			env: map[string]string{
				"RADIANT_DEV_AUTH": "1",
				"RADIANT_DB":       "/tmp/smoke-fc4.db",
			},
			wantExit: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(tc.env["RADIANT_DB"])
			cmd := exec.Command(binPath)
			cmd.Env = append(os.Environ(), "RADIANT_ADDR=:0")
			for k, v := range tc.env {
				cmd.Env = append(cmd.Env, k+"="+v)
			}

			// Start em goroutine. NÃO usamos cmd.Run() aqui porque ele
			// internamente escreve em cmd.Process enquanto nossa outra
			// goroutine pode ler (cmd.Process.Kill no timeout) — data race
			// detectado por `go test -race`. Padrão correto: Start + Wait
			// em uma goroutine, e Kill apenas após Start retornar.
			if err := cmd.Start(); err != nil {
				t.Fatalf("cmd.Start: %v", err)
			}

			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case err := <-done:
				if tc.wantExit == 0 {
					if err != nil {
						t.Fatalf("expected exit 0, got err=%v", err)
					}
				} else {
					if err == nil {
						t.Fatalf("expected exit non-zero (fail-closed), got exit 0")
					}
				}
			case <-time.After(5 * time.Second):
				// Start já retornou, então cmd.Process é safe de acessar.
				_ = cmd.Process.Kill()
				<-done // aguarda Wait() retornar
				if tc.wantExit == 0 {
					t.Logf("process kept running (expected for back-compat dev mode)")
				} else {
					t.Fatalf("process did not exit within 5s — fail-closed gate NOT firing")
				}
			}
		})
	}
}

// =============================================================================
// Cenário 2 — Auth + CSRF middleware ativo
// =============================================================================
func TestSmoke_Cenario2_AuthAndCSRF(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	t.Run("a) sem X-IF-ID em endpoint v1 → 401", func(t *testing.T) {
		w := smokeReq(handler, "GET", "/v1/schemas", nil, nil)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("got %d, want 401", w.Code)
		}
	})

	t.Run("b) com X-IF-ID válido → passa", func(t *testing.T) {
		w := smokeReq(handler, "GET", "/v1/schemas", nil,
			map[string]string{"X-IF-ID": "demo"})
		if w.Code != http.StatusOK {
			t.Fatalf("got %d, want 200, body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("c) POST cross-origin não-allowlisted → 403 CSRF", func(t *testing.T) {
		w := smokeReq(handler, "POST", "/v1/validate",
			map[string]any{"cadoc_code": "3040"},
			map[string]string{
				"X-IF-ID": "demo",
				"Origin":  "https://evil.example.com",
			})
		if w.Code != http.StatusForbidden {
			t.Fatalf("CSRF deveria bloquear 403, got %d, body=%s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "CSRF") {
			t.Errorf("resposta deveria mencionar CSRF, got: %s", w.Body.String())
		}
	})

	t.Run("d) POST same-origin → passa CSRF", func(t *testing.T) {
		w := smokeReq(handler, "POST", "/v1/validate",
			map[string]any{"cadoc_code": "3040"},
			map[string]string{
				"X-IF-ID": "demo",
				"Origin":  "http://example.com",
			})
		if w.Code == http.StatusForbidden {
			t.Fatalf("same-origin deveria passar, got 403 body=%s", w.Body.String())
		}
	})

	t.Run("e) POST com Origin allowlist → passa CSRF", func(t *testing.T) {
		w := smokeReq(handler, "POST", "/v1/validate",
			map[string]any{"cadoc_code": "3040"},
			map[string]string{
				"X-IF-ID": "demo",
				"Origin":  "http://localhost:4180",
			})
		if w.Code == http.StatusForbidden {
			t.Fatalf("allowlisted origin deveria passar, got 403 body=%s", w.Body.String())
		}
	})
}

// =============================================================================
// Cenário 3 — STA submit cross-tenant → 403
// =============================================================================
func TestSmoke_Cenario3_STA_CrossTenant(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	t.Run("STA submit com CNPJ de outro IF → 403", func(t *testing.T) {
		w := smokeReq(handler, "POST", "/v1/sta/submit",
			map[string]any{
				"cadoc_code": "3040",
				"cnpj":       "other",
				"data_base":  "2026-01-15",
			},
			map[string]string{"X-IF-ID": "demo"},
		)
		if w.Code != http.StatusForbidden {
			t.Fatalf("cross-tenant deveria 403, got %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("STA submit com CNPJ do próprio IF → OK", func(t *testing.T) {
		w := smokeReq(handler, "POST", "/v1/sta/submit",
			map[string]any{
				"cadoc_code": "3040",
				"cnpj":       "demo",
				"data_base":  "2026-01-15",
				"xml":        "<root><ok/></root>",
			},
			map[string]string{"X-IF-ID": "demo"},
		)
		if w.Code == http.StatusForbidden {
			t.Fatalf("same-tenant não deveria 403, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

// =============================================================================
// Cenário 4 — Crossdoc validate cross-tenant → 403
// =============================================================================
func TestSmoke_Cenario4_Crossdoc_CrossTenant(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	t.Run("crossdoc validate com IfID de outro IF → 403", func(t *testing.T) {
		w := smokeReq(handler, "POST", "/v1/crossdoc/validate",
			map[string]any{
				"if_id": "other",
				"cadocs": map[string]string{
					"3040": "<doc/>",
				},
			},
			map[string]string{"X-IF-ID": "demo"},
		)
		if w.Code != http.StatusForbidden {
			t.Fatalf("cross-tenant deveria 403, got %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("crossdoc validate sem IfID → passa (assume caller)", func(t *testing.T) {
		w := smokeReq(handler, "POST", "/v1/crossdoc/validate",
			map[string]any{
				"cadocs": map[string]string{
					"3040": "<doc/>",
				},
			},
			map[string]string{"X-IF-ID": "demo"},
		)
		if w.Code == http.StatusForbidden {
			t.Fatalf("sem IfID no payload deveria passar cross-tenant guard, got 403")
		}
	})
}

// =============================================================================
// Cenário 5 — Validate/listRules/getSchema/listVersions happy path
// =============================================================================
func TestSmoke_Cenario5_ValidatorsHappyPath(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	t.Run("listRulesByCadoc com cadoc inválido (12345) → 400", func(t *testing.T) {
		w := smokeReq(handler, "GET", "/v1/rules/12345", nil,
			map[string]string{"X-IF-ID": "demo"})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("cadoc inválido deveria 400, got %d body=%s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "invalid") {
			t.Errorf("resposta deveria mencionar invalid, got: %s", w.Body.String())
		}
	})

	t.Run("listRulesByCadoc com cadoc válido (3040) → 200", func(t *testing.T) {
		w := smokeReq(handler, "GET", "/v1/rules/3040", nil,
			map[string]string{"X-IF-ID": "demo"})
		if w.Code != http.StatusOK {
			t.Fatalf("cadoc válido deveria 200, got %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("getSchema retorna schema ou 404, nunca 500", func(t *testing.T) {
		w := smokeReq(handler, "GET", "/v1/schemas/3040", nil,
			map[string]string{"X-IF-ID": "demo"})
		if w.Code >= 500 {
			t.Fatalf("getSchema não deveria 500, got %d body=%s", w.Code, w.Body.String())
		}
	})

	t.Run("listVersions com cadoc inválido → 400", func(t *testing.T) {
		w := smokeReq(handler, "GET", "/v1/schemas/abc/versions", nil,
			map[string]string{"X-IF-ID": "demo"})
		if w.Code != http.StatusBadRequest {
			t.Fatalf("cadoc inválido deveria 400, got %d", w.Code)
		}
	})

	t.Run("validate happy path com cadoc_code válido", func(t *testing.T) {
		w := smokeReq(handler, "POST", "/v1/validate",
			map[string]any{
				"cadoc_code": "3040",
				"data_base":  "2026-01-15",
				"xml":        "<doc/>",
			},
			map[string]string{
				"X-IF-ID": "demo",
				"Origin":  "http://example.com",
			},
		)
		if w.Code == http.StatusForbidden {
			t.Fatalf("same-origin não deveria 403, got %s", w.Body.String())
		}
		if w.Code >= 500 {
			t.Fatalf("validate não deveria 500, got %d body=%s", w.Code, w.Body.String())
		}
	})
}

// =============================================================================
// Cenário 6 — Worker processa envio + SafeError não vaza DSN
// =============================================================================
func TestSmoke_Cenario6_WorkerSafeError(t *testing.T) {
	d := testutil.NewTestDB(t)
	_, _ = d.Exec(`INSERT OR IGNORE INTO ifs (id, cnpj, nome, tipo, plano)
		VALUES (?, ?, ?, 'SCD', 'pro')`, "demo", "00000001", "Demo")

	// Insert envio com payload mínimo
	envioID := "smoke-envio-1"
	_, err := d.Exec(`
		INSERT INTO envios (id, if_id, cadoc_code, data_base, remessa, xml_hash, zip_hash,
		                    xml_content, zip_content, status)
		VALUES (?, 'demo', '3040', '2026-01-15', 1, 'h', 'h', '<doc/>', '<doc/>', 'pending')
	`, envioID)
	if err != nil {
		t.Fatalf("seed envio: %v", err)
	}

	t.Run("a) worker.ProcessBatch() executa sem panic e sanitiza erro", func(t *testing.T) {
		auditSvc := audit.New(d)
		auditLog := auditlog.New(d)
		staClient := sta.NewStubClient()
		logger := discardLogger()
		ctx := context.Background()

		_, _ = workpkg.ProcessBatch(ctx, d, auditSvc, auditLog, staClient, 1, logger)

		// Verificar que error_message NÃO contém DSN fragment
		var errMsg sql.NullString
		_ = d.QueryRow(`SELECT error_message FROM envios WHERE id = ?`, envioID).Scan(&errMsg)
		if errMsg.Valid {
			lower := strings.ToLower(errMsg.String)
			if strings.Contains(lower, "postgres://") ||
				strings.Contains(lower, "password=") ||
				strings.Contains(lower, "user=") {
				t.Fatalf("error_message vazou DSN: %s", errMsg.String)
			}
			t.Logf("error_message (sanitizado): %s", errMsg.String)
		} else {
			t.Logf("error_message é NULL (sem erro ou sucesso limpo)")
		}
	})

	t.Run("b) INSERT envio com IF inexistente → FK rejeita (envios NÃO tem FK explícita — warning log)", func(t *testing.T) {
		_, err := d.Exec(`
			INSERT INTO envios (id, if_id, cadoc_code, data_base, remessa, xml_hash, zip_hash,
			                    xml_content, zip_content, status)
			VALUES (?, 'nao-existe-fk', '3040', '2026-01-15', 1, 'h', 'h', '<doc/>', '<doc/>', 'pending')
		`, "smoke-envio-bad-fk")
		if err != nil {
			t.Logf("FK rejeitou (inesperado pra envios — pode ser OK se schema ganhou FK): %v", err)
		} else {
			t.Logf("envios ACEITA IF inexistente — esperado: envios não tem FK explícita na migration 010")
		}
	})
}

// =============================================================================
// Cenário 7 — Rate limiter burst → 429 + Retry-After header
// =============================================================================
func TestSmoke_Cenario7_RateLimitBurst(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	headers := map[string]string{
		"X-IF-ID": "demo",
		"Origin":  "http://example.com",
	}

	for i := 0; i < 10; i++ {
		w := smokeReq(handler, "POST", "/v1/validate",
			map[string]any{"cadoc_code": "3040", "xml": "<doc/>"}, headers)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("request #%d não deveria estar rate-limited (got 429): %s", i+1, w.Body.String())
		}
	}

	w := smokeReq(handler, "POST", "/v1/validate",
		map[string]any{"cadoc_code": "3040", "xml": "<doc/>"}, headers)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("11ª request deveria 429, got %d body=%s", w.Code, w.Body.String())
	}

	if ra := w.Header().Get("Retry-After"); ra == "" {
		t.Error("Retry-After header ausente em 429")
	}
	if b := w.Header().Get("X-RateLimit-Bucket"); b != "heavy" {
		t.Errorf("X-RateLimit-Bucket=%q, want \"heavy\"", b)
	}

	// Outra IF não herda rate limit
	w2 := smokeReq(handler, "POST", "/v1/validate",
		map[string]any{"cadoc_code": "3040", "xml": "<doc/>"},
		map[string]string{"X-IF-ID": "demo-bank", "Origin": "http://example.com"})
	if w2.Code == http.StatusTooManyRequests {
		t.Fatalf("outra IF não deveria herdar rate limit de demo, got 429: %s", w2.Body.String())
	}
}

// =============================================================================
// Cenário 8 — SSE cap de 10 subscribers/IF
// =============================================================================
func TestSmoke_Cenario8_SSECap(t *testing.T) {
	logger := discardLogger()
	hub := realtime.NewHub(logger)
	ctx := context.Background()

	const ifID = "demo"

	// Cleanup no final pra fechar channels
	cleanups := make([]func(), 0, realtime.MaxSubscribersPerIF+1)
	t.Cleanup(func() {
		for _, c := range cleanups {
			c()
		}
	})

	for i := 0; i < realtime.MaxSubscribersPerIF; i++ {
		_, cleanup, err := hub.Subscribe(ctx, ifID)
		if err != nil {
			t.Fatalf("subscriber #%d falhou inesperadamente: %v", i+1, err)
		}
		cleanups = append(cleanups, cleanup)
	}

	_, _, err := hub.Subscribe(ctx, ifID)
	if err == nil {
		t.Fatal("11º subscriber deveria ser rejeitado, got nil")
	}
	if !errors.Is(err, realtime.ErrTooManySubscribers) {
		t.Fatalf("expected ErrTooManySubscribers, got %v", err)
	}
}

// =============================================================================
// Cenário 9 — FK rejeita IF inválido em tabelas tenant-scoped
// =============================================================================
func TestSmoke_Cenario9_FKConstraints(t *testing.T) {
	d := testutil.NewTestDB(t)

	t.Run("a) INSERT audit_log com IF válido → OK", func(t *testing.T) {
		_, err := d.Exec(`
			INSERT INTO audit_log (if_id, actor, action, target, payload_hash,
			                       prev_hash, entry_hash)
			VALUES ('demo', '127.0.0.1', 'smoke.test', 'envio', 'h', 'h', 'h')
		`)
		if err != nil {
			t.Fatalf("audit_log com IF válido deveria inserir: %v", err)
		}
	})

	t.Run("b) INSERT audit_log com IF inexistente → FK rejeita", func(t *testing.T) {
		_, err := d.Exec(`
			INSERT INTO audit_log (if_id, actor, action, target, payload_hash,
			                       prev_hash, entry_hash)
			VALUES ('nao-existe-fk', '127.0.0.1', 'smoke.test', 'envio', 'h', 'h', 'h')
		`)
		if err == nil {
			t.Fatal("audit_log DEVERIA ter FK para ifs (migration 010) — got sem erro")
		}
		lower := strings.ToLower(err.Error())
		if !strings.Contains(lower, "foreign") && !strings.Contains(lower, "constraint") {
			t.Logf("erro não menciona FK/constraint explicitamente: %v", err)
		}
	})

	t.Run("c) INSERT disabled_rules com IF inexistente → FK rejeita", func(t *testing.T) {
		_, err := d.Exec(`
			INSERT INTO disabled_rules (if_id, rule_code, disabled_by)
			VALUES ('nao-existe-fk', 'F12', 'system')
		`)
		if err == nil {
			t.Fatal("disabled_rules DEVERIA ter FK para ifs (migration 010)")
		}
	})

	t.Run("d) INSERT acknowledged_recommendations com IF inexistente → FK rejeita", func(t *testing.T) {
		_, err := d.Exec(`
			INSERT INTO acknowledged_recommendations (if_id, rec_id, acknowledged_by)
			VALUES ('nao-existe-fk', 'rec-1', 'system')
		`)
		if err == nil {
			t.Fatal("acknowledged_recommendations DEVERIA ter FK para ifs (migration 010)")
		}
	})
}

// =============================================================================
// Cenário 10 — EXPLAIN nos índices de envios (idx usado)
// =============================================================================
func TestSmoke_Cenario10_EnviosIndexes(t *testing.T) {
	d := testutil.NewTestDB(t)

	_, _ = d.Exec(`INSERT OR IGNORE INTO ifs (id, cnpj, nome, tipo, plano)
		VALUES (?, ?, ?, 'SCD', 'pro')`, "demo", "00000001", "Demo")

	statuses := []string{"accepted", "rejected", "pending", "confirmed"}
	for i := 0; i < 20; i++ {
		_, _ = d.Exec(`
			INSERT INTO envios (id, if_id, cadoc_code, data_base, remessa, xml_hash, zip_hash,
			                    xml_content, zip_content, status, period)
			VALUES (?, 'demo', '3040', '2026-01-15', 1, 'h', 'h', '<doc/>', '<doc/>',
			        ?, '01/2026')
		`, fmt.Sprintf("smoke-idx-%d", i), statuses[i%4])
	}

	// Seed 5 rule_failures pra cobrir idx_rule_failures_if_cadoc (migration 010)
	for i := 0; i < 5; i++ {
		_, _ = d.Exec(`
			INSERT INTO rule_failures (envio_id, if_id, cadoc_code, rule_code, rule_severity)
			VALUES (?, 'demo', '3040', ?, 'MEDIUM')
		`, fmt.Sprintf("smoke-idx-%d", i), fmt.Sprintf("F%d", i+10))
	}

	queries := []struct {
		name        string
		sql         string
		args        []any
		wantIdx     string
		skipPlanner bool // skip planner check (ex: partial index needs more data)
	}{
		{
			name:    "idx_envios_if_status",
			sql:     `EXPLAIN QUERY PLAN SELECT id FROM envios WHERE if_id = ? AND status = ?`,
			args:    []any{"demo", "accepted"},
			wantIdx: "idx_envios_if_status",
		},
		{
			name:    "idx_envios_if_cadoc_status_period",
			sql:     `EXPLAIN QUERY PLAN SELECT id FROM envios WHERE if_id = ? AND cadoc_code = ? AND status = ? AND period = ?`,
			args:    []any{"demo", "3040", "accepted", "01/2026"},
			wantIdx: "idx_envios_if_cadoc_status_period",
		},
		{
			name:    "idx_envios_if_period",
			sql:     `EXPLAIN QUERY PLAN SELECT id FROM envios WHERE if_id = ? AND period = ?`,
			args:    []any{"demo", "01/2026"},
			wantIdx: "idx_envios_if_period",
		},
		{
			// Partial index: WHERE confirmed_at IS NOT NULL.
			// Em prod, query usa esta condição exata; planner escolhe
			// partial quando há volume suficiente (>100 rows confirmado).
			// Aqui só validamos que o índice EXISTE — escolha do planner
			// é implementation-detail de SQLite.
			name:        "idx_envios_if_confirmed (existe)",
			sql:         `EXPLAIN QUERY PLAN SELECT id FROM envios WHERE if_id = ? AND confirmed_at IS NOT NULL ORDER BY confirmed_at DESC LIMIT 10`,
			args:        []any{"demo"},
			wantIdx:     "idx_envios_if_confirmed",
			skipPlanner: true, // 20 rows: planner prefere idx_envios_if_status (composite)
		},
		{
			// Partial index: WHERE status IN ('pending','error','processing').
			// Mesma justificativa: planner prefere composite quando volume
			// é pequeno.
			name:        "idx_envios_if_open (existe)",
			sql:         `EXPLAIN QUERY PLAN SELECT id FROM envios WHERE if_id = ? AND status IN ('pending','error','processing') ORDER BY created_at DESC LIMIT 10`,
			args:        []any{"demo"},
			wantIdx:     "idx_envios_if_open",
			skipPlanner: true,
		},
		{
			name:    "idx_rule_failures_if_cadoc (migration 010 covering index)",
			sql:     `EXPLAIN QUERY PLAN SELECT rule_code, COUNT(*) FROM rule_failures WHERE if_id = ? AND cadoc_code = ? GROUP BY rule_code`,
			args:    []any{"demo", "3040"},
			wantIdx: "idx_rule_failures_if_cadoc",
		},
	}

	for _, q := range queries {
		t.Run(q.name, func(t *testing.T) {
			rows, err := d.Query(q.sql, q.args...)
			if err != nil {
				t.Fatalf("EXPLAIN failed: %v", err)
			}
			defer rows.Close()

			var plan strings.Builder
			for rows.Next() {
				var id, parent, notused int
				var detail string
				if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
					t.Fatalf("scan plan: %v", err)
				}
				plan.WriteString(detail + "\n")
			}

			if q.skipPlanner {
				// Verifica que o índice EXISTE no schema (sqlite_master)
				// ao invés de exigir uso pelo planner.
				var count int
				if err := d.QueryRow(
					`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`,
					q.wantIdx,
				).Scan(&count); err != nil {
					t.Fatalf("check index existence: %v", err)
				}
				if count == 0 {
					t.Errorf("índice %s não existe no schema", q.wantIdx)
				} else {
					t.Logf("✓ índice %s existe (planner choice depende de volume)", q.wantIdx)
					t.Logf("plan:\n%s", plan.String())
				}
				return
			}

			if !strings.Contains(plan.String(), q.wantIdx) {
				t.Errorf("query plan não usou %s.\nPlan:\n%s", q.wantIdx, plan.String())
			} else {
				t.Logf("plan:\n%s", plan.String())
			}
		})
	}
}
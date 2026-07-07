// Package version expõe a versão da build para todos os packages.
//
// Validação 18 (GAP-7.4 / F10.10 follow-up): single source of truth.
// Antes da v1.5.0 v17:
//   - `api.Version = "1.5.0"` (constante no api package)
//   - "Radiant-Norma-Radar/1.5.0" hardcoded no User-Agent de radar
//
// Cada bump de versão precisava atualizar 2 lugares sincronizados —
// vetor de version drift. Cross-package constante não propaga por
// mágica (radiant F10.9 — radar não pode importar api porque api
// importa radar → dependência unilateral).
//
// Solução: pacote `internal/version` é folha (sem deps internas) e
// pode ser importado por qualquer package (api, radar, worker, cmd/).
//
// Build-time override via ldflags:
//
//	go build -ldflags "-X 'github.com/fortvna/radiant-norma/backend/internal/version.Version=v1.5.0+commit123'"
//
// Default em dev: "dev".
package version

// Version é a versão da build. Override via:
//   - ldflags "-X ...version.Version=v2.0.0" (build)
//   - constante editada abaixo (dev)
//
// Para bumpar: alterar este string + api.Version re-export + tag git.
//
// Validação 27 (F27.2): v1.5.0 foi deixado para trás após Sprint 7c/v2.0.0.
// Constante deveria ter sido bumped ao fechar o release v2.0.0 — sem isso
// `/healthz` continua reportando "1.5.0" enquanto CHANGELOG diz v2.0.0.
//
// Histórico de bumps:
//
//	v1.5.0 (Sprint 6)  → v2.0.0 (Sprint 7c, validação 27 fix)
//	v2.0.0             → v2.1.0 (Sprint 8a, JWT bridge)
//	v2.1.0             → v3.0.0 (Sprint 9, frontend redesign)
//	v3.0.0             → v3.1.0 (Sprint 8c, backend intelligence)
//	v3.1.0             → v3.2.0 (Sprint 8d, URL filters + CSV/JSON export)
//	v3.2.0             → v3.3.0 (Sprint 10, real-time SSE)
//	v3.3.0             → v3.4.0 (Sprint 11, drill-down server actions)
//	v3.4.0             → v3.5.0 (Sprint 12, production hardening + engine integration)
const Version = "3.34.32"

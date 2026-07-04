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
//   - ldflags "-X ...version.Version=v1.5.0" (build)
//   - constante editada abaixo (dev)
//
// Para bumpar: alterar este string + api.Version re-export + tag git.
const Version = "1.5.0"

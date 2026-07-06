# ADR-0005: STA Client — interface segregation (Client / ReadClient / ChunkedClient)

> **Status:** Aceito (existente desde Sprint 25, ratificado em 2026-07-05)
> **Data original:** 2026-06-15 (Sprint 25 v3.16.0)
> **Ratificação:** 2026-07-05
> **Decisor(es):** Henrique Costa · Mavis

## Contexto

STA WS (Sistema de Transferência de Arquivos do BACEN) tem 3 subsets distintos de operações:

1. **Write side** (Submit + Upload de ZIP)
2. **Read side** (Download + StatusUpload + ListDisponiveis + AlterarSituacao)
3. **Chunked side** (SubmitRange + DownloadRange)

Em dev/test, usamos `StubClient` que **só faz sentido pra write** (não há BACEN real pra listar/baixar). Forçar StubClient a implementar read/chunked com zero-values seria **hollow stub** — caller acharia que funcionou mas nada foi chamado.

## Decisão

Interface segregation: 3 interfaces separadas, 1 struct concreta implementa todas, 1 struct stub implementa só a base.

```go
// Base — write operations
type Client interface {
    Submit(ctx context.Context, sub *Submission) (*Result, error)
}

// Read operations
type ReadClient interface {
    ListDisponiveis(ctx context.Context, opts ListDisponiveisOpts) (*ListDisponiveisResult, error)
    AlterarSituacao(ctx context.Context, req AlterarSituacaoReq) error
}

// Chunked operations
type ChunkedClient interface {
    SubmitRange(ctx context.Context, protocolo string, inicio, fim, total int64, chunk []byte) error
    DownloadRange(ctx context.Context, protocolo string, inicio, fim int64, expectedTotalHash, ifMatch, ifUnmodifiedSince string) (*DownloadResult, error)
}

// Compile-time guarantees (production source, não test files)
var (
    _ Client        = (*WSClient)(nil)
    _ ReadClient    = (*WSClient)(nil)
    _ ChunkedClient = (*WSClient)(nil)
    _ Client        = (*StubClient)(nil)
    // ReadClient / ChunkedClient: NÃO implementados por StubClient
)
```

**Caller pattern:**

```go
// Handler
func (s *Server) staListHandler(w http.ResponseWriter, r *http.Request) {
    rc, ok := s.STAClient.(sta.ReadClient)
    if !ok {
        http.Error(w, "read side não disponível neste backend (configure RADIANT_STA_BACKEND=ws)", http.StatusServiceUnavailable)
        return
    }
    result, err := rc.ListDisponiveis(ctx, opts)
    // ...
}
```

## Consequências

**Positivas:**
- ✅ **Hollow stub evitado** — StubClient não mente que tem read side.
- ✅ Compile-time assert garante implementação.
- ✅ Capability check explícito em runtime via type assertion.
- ✅ Test injection pattern: tests sobrescrevem `staNewClientFromEnv` var.

**Negativas:**
- ❌ Type assertion em runtime (não tem method em Client que retorne capability).
- ❌ Caller precisa saber quais interfaces existem. Mitigável: documentação clara.

## Alternativas consideradas

| Alternativa | Por que não |
|---|---|
| **Single Client interface com todas as ops** | StubClient implementa com zero-values — hollow stub. |
| **Capability method (HasRead, HasChunked)** | Funciona, mas type assertion é mais idiomático Go. |
| **Embedded interfaces em Client** | Acoplamento desnecessário. |

## Notas de implementação

- Cada nova interface deve ter compile-time assert em production source (não test).
- Caller checa capability antes de chamar método de interface opcional.
- Doc comment em cada interface explica "por que segregation".
- Validação profunda (Sprint 44) ratificou o pattern — manter como standard.
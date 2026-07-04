# VALIDATION v1.5.0 DEEPEST8 — 22ª validação profunda (singleflight + observability)

> **Status:** ACCEPTED
> **Data:** 2026-07-03
> **Trigger:** Henrique pediu mais uma validação. Foco: configuração,
> observability, cache stampede.
> **Versão:** v1.5.0 inalterada (sem bump).

## 🎯 Resumo executivo

12ª validação consecutiva com findings. v22 mudou de foco para
**resiliência** (cache stampede, config consistency).

**2 findings, 1 médio**

1. **F22.2** 🟡 — `CadocListCache.GetOrFetch` tem vetor de cache stampede:
   N goroutines simultâneas em cache expirado disparariam N
   queries ao DB (mesma query). Singleflight adicionado.

2. **F22.5** 🟢 — Cross-doc engine sem context.WithTimeout interno,
   mas WriteTimeout 60s do http.Server já protege. Não aplicado.

**Stats:**
- 246 → 247 tests passing (+1 stampede stress test)
- vet-clean, race-clean, build-clean
- 0 cache stampede risk
- WriteTimeout 60s já é defesa existente contra DOS cross-doc

---

## 🟡 MÉDIOS (P1)

### F22.2 — CadocListCache stampede (singleflight protection)

**Severidade:** 🟡 MÉDIO (DOS-via-DB em produção)

**Discovery via code reading:**

```go
// schema/registry.go:209 (estado anterior)
func (c *CadocListCache) GetOrFetch(fetch func() ([]string, error)) ([]string, error) {
    c.mu.Lock()
    if len(c.cadocs) > 0 && time.Since(c.cachedAt) < c.ttl {
        // cache hit
    }
    c.mu.Unlock()

    // Cache miss — fetch fora do lock
    cadocs, err := fetch()  // ← pode rodar N vezes em paralelo
    ...
}
```

**Vetor:**

Quando TTL expira e N goroutines chamam `GetOrFetch`
simultaneamente:
- Goroutine 1: vê cache expirado → busca fetch() (DB query)
- Goroutine 2: vê cache expirado → busca fetch() (DB query)
- ... até N goroutines
- DB recebe N queries idênticas no mesmo instante

**Cenário de produção:**

```
1000 dashboards carregando simultaneamente após TTL expirar (5min)
    ↓
1000 queries `SELECT DISTINCT cadoc_code` paralelas
    ↓
SQLite com max 8 connections: fila de 992
Postgres com max 25 connections: fila de 975
    ↓
Server slow / timeout / connection exhausted
```

**Fix aplicado — singleflight:**

```go
type CadocListCache struct {
    ...
    sf singleflight.Group // F22.2: cache stampede protection
}

func (c *CadocListCache) GetOrFetch(fetch func() ([]string, error)) ([]string, error) {
    // Cache fast path
    c.mu.Lock()
    if len(c.cadocs) > 0 && time.Since(c.cachedAt) < c.ttl {
        ...cache hit
    }
    c.mu.Unlock()

    // Cache miss — singleflight garante fetch() executa 1 vez
    v, err, _ := c.sf.Do("cadocs", func() (any, error) {
        // Re-check dentro do singleflight (race protection)
        c.mu.Lock()
        if len(c.cadocs) > 0 && time.Since(c.cachedAt) < c.ttl {
            ...return out
        }
        c.mu.Unlock()

        cadocs, fetchErr := fetch()
        ...
    })
    ...
}
```

**Por que singleflight:**

1. **Idempotente**: `singleflight.Do(key, fn)` garante que apenas 1
   `fn` roda, mesmo com N goroutines chamando simultaneamente.
2. **Barato**: ~1KB de overhead por key. Sem alocações em cache hit.
3. **Biblioteca stdlib-ish**: `golang.org/x/sync/singleflight` é
   canônica (mantida por time oficial do Go).

**Test de regressão criado:**

```go
func TestCadocListCache_NoStampede_Concurrent(t *testing.T) {
    c := schema.NewCadocListCache(100 * time.Millisecond)
    var fetchCount int64

    fetch := func() ([]string, error) {
        atomic.AddInt64(&fetchCount, 1)
        time.Sleep(50 * time.Millisecond)
        return []string{"3040", "3050", "4111"}, nil
    }

    time.Sleep(150 * time.Millisecond) // cache expira

    const N = 200
    var wg sync.WaitGroup
    wg.Add(N)
    for i := 0; i < N; i++ {
        go func(idx int) {
            defer wg.Done()
            c.GetOrFetch(fetch)
        }(i)
    }
    wg.Wait()

    count := atomic.LoadInt64(&fetchCount)
    if count != 1 {
        t.Errorf("fetch chamado %d vezes com singleflight (esperado 1)", count)
    }
}
```

**Resultado:** 200 goroutines simultâneas → fetch() chamado exatamente 1 vez.

**Aplicabilidade cross-project:**

Qualquer cache in-memory com TTL tem vetor de cache stampede.
Sempre adicione singleflight:

- Redis client-side wrappers
- LRU caches (tollbooth, hashicorp/golang-lru)
- Database query caches
- API response caches

Pattern:
```go
type Cache struct {
    data atomic.Value
    ttl  time.Duration
    sf   singleflight.Group  // ← proteção stampede
}

func (c *Cache) Get(key string, fetch func() (any, error)) (any, error) {
    // Try cache fast path
    if v, ok := c.data.Load().(cached); ok { return v, nil }

    // singleflight via key
    v, err, _ := c.sf.Do(key, func() (any, error) {
        // re-check + fetch
        ...
    })
    return v, err
}
```

**Anti-pattern:** Cache TTL sem singleflight. Cada cache-miss
simultâneo = N queries ao backend. Load test antes de produção
sempre.

---

## 🟢 BAIXO

### F22.5 — Cross-doc engine sem context.WithTimeout interno

**Severidade:** 🟢 BAIXO (já protegido por WriteTimeout 60s)

`engine.Validate` recebe `ctx context.Context` mas não cria
context.WithTimeout/WithDeadline próprio. Workers paralelos (1 por
regra) podem demorar.

Mas o http.Server tem:
```go
WriteTimeout: 60 * time.Second
```

Que mata requests > 60s. Vetor de DOS-via-slow-analysis é mitigado
por defesa existente.

**Não aplicado v22** — defesa já cobre. Re-avaliar se validação 23
focus em production-grade per-request.

---

## 📊 Achados consolidados (validação 22)

| Categoria | Críticos | Médios | Baixos |
|-----------|----------|--------|--------|
| Cache stampede | 0 | 1 (F22.2) | 0 |
| Context timeout | 0 | 0 | 1 (F22.5) |
| **TOTAL** | **0** | **1** | **1** |

**Achados caindo:** v18=8, v19=7, v20=7, v21=5, **v22=2**. Provável
saturação — codebase está estabilizando.

---

## 📊 Acumulado 22 validações

| Validação | Findings | Críticos |
|-----------|----------|----------|
| 11 | 9  | 0 |
| 12 | 9  | 4 |
| 13 | 4  | 1 |
| 14 | 5  | 1 |
| 15 | 4  | 1 |
| 16 | 4  | 1 |
| 17 | 3  | 0 |
| 18 | 8  | 3 |
| 19 | 7  | 4 |
| 20 | 7  | 2 |
| 21 | 5  | 1 |
| 22 | 2  | 0 |
| **TOTAL** | **67** | **18** |

**22 validações. 12 com findings consecutivos. v22 com 2 achados e 0 críticos.**

---

## 🎯 Estado final pós-22

```
247 tests passing (246 → 247, +1 stampede stress)
vet-clean, race-clean, build-clean

13 categorias fechadas:
  + CadocListCache stampede (F22.2) [NOVO]

Padrão: 18 críticos em cascata pós-release v1.5.0.
v22 ainda achou 1 médio + 1 baixo, mas 0 críticos.
Codebase visivelmente convergindo.
```

---

## ✅ Acceptance da validação 22

- ✅ F22.2 — singleflight adicionado (1 fetch() para N goroutines)
- ✅ Stress test confirma: 200 goroutines → 1 fetch
- ✅ 247 tests passing
- ✅ vet/race/build clean
- ⏳ F22.5 — defesa existente (WriteTimeout) suficiente

---

## 📌 Próximo passo (Sprint 7)

**Estado consolidado:**
- ✅ 18 críticos fechados em 12 validações
- ✅ 67 findings totais
- ✅ 247 tests passing
- ⏳ 32 commits ahead of origin

**Recomendação Sprint 7 (decrescente de prioridade):**

1. **PUSH operacional** (32 commits é janela)
2. **Encerrar Fase 1** quando v23 der 0-1 findings (codebase
   está claramente saturada)
3. **Feature nova** ou começar próxima fase
4. Cleanup F19.7/F19.15 + F21.8/F21.9 (cosmetic)

**Heurística atualizada:** taxa de achados caiu de 8→5→2 nas últimas
3 validações. Quando taxa chegar a 0-1, codebase está estável
pra release final ou feature grande. Cobertura estrutural
possível: secrets handling em cmd/*, error chain cross-goroutine,
Postgres integration tests (production-flavor).

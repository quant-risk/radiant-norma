// Package schema implementa o Schema Registry (versionamento por data-base).
//
// Padrão: cada release do BACEN cria nova linha em schema_versions,
// identificada por (cadoc_code, effective_from). IFs consultam sempre a
// versão efetiva na data-base do envio.
package schema

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Version representa uma versão de leiaute de um CADOC.
type Version struct {
	ID            int64     `json:"id"`
	CadocCode     string    `json:"cadoc_code"`
	EffectiveFrom time.Time `json:"effective_from"`
	SourceURI     string    `json:"source_uri"`
	Fields        []Field   `json:"fields"`
	XSD           string    `json:"xsd,omitempty"`
	Changelog     string    `json:"changelog,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Field representa um campo do leiaute.
type Field struct {
	Tag      string `json:"tag,omitempty"`
	Attr     string `json:"attr,omitempty"`
	Type     string `json:"type"` // BACEN: A8, N19,2, A1, etc
	Required bool   `json:"required"`
	Desc     string `json:"desc,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Group    string `json:"group,omitempty"` // sheet de origem
}

// Registry é o serviço de Schema Registry.
type Registry struct {
	db *sql.DB
}

// New cria um novo Registry.
func New(db *sql.DB) *Registry {
	return &Registry{db: db}
}

// GetEffective retorna a versão efetiva de um CADOC na data-base informada.
// Se dataBase for zero, retorna a versão mais recente.
func (r *Registry) GetEffective(cadocCode string, dataBase time.Time) (*Version, error) {
	q := `
		SELECT id, cadoc_code, effective_from, source_uri, fields_json, xsd, changelog, created_at
		FROM schema_versions
		WHERE cadoc_code = ?
	`
	args := []any{cadocCode}
	if !dataBase.IsZero() {
		q += " AND effective_from <= ?"
		args = append(args, dataBase.Format("2006-01-02"))
	}
	q += " ORDER BY effective_from DESC LIMIT 1"

	var v Version
	var fieldsJSON string
	var xsd, changelog sql.NullString
	err := r.db.QueryRow(q, args...).Scan(
		&v.ID, &v.CadocCode, &v.EffectiveFrom, &v.SourceURI, &fieldsJSON, &xsd, &changelog, &v.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("nenhuma versão encontrada para CADOC %s em %s", cadocCode, dataBase)
	}
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	if err := json.Unmarshal([]byte(fieldsJSON), &v.Fields); err != nil {
		return nil, fmt.Errorf("parse fields: %w", err)
	}
	if xsd.Valid {
		v.XSD = xsd.String
	}
	if changelog.Valid {
		v.Changelog = changelog.String
	}
	return &v, nil
}

// Insert adiciona uma nova versão de schema.
func (r *Registry) Insert(v *Version) error {
	fieldsJSON, err := json.Marshal(v.Fields)
	if err != nil {
		return fmt.Errorf("marshal fields: %w", err)
	}
	_, err = r.db.Exec(`
		INSERT INTO schema_versions (cadoc_code, effective_from, source_uri, fields_json, xsd, changelog)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		v.CadocCode, v.EffectiveFrom.Format("2006-01-02"), v.SourceURI, string(fieldsJSON),
		nullableString(v.XSD), nullableString(v.Changelog),
	)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

// List retorna todas as versões de um CADOC, ordenadas da mais recente para mais antiga.
func (r *Registry) List(cadocCode string) ([]Version, error) {
	rows, err := r.db.Query(`
		SELECT id, cadoc_code, effective_from, source_uri, fields_json, xsd, changelog, created_at
		FROM schema_versions
		WHERE cadoc_code = ?
		ORDER BY effective_from DESC
	`, cadocCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []Version
	for rows.Next() {
		var v Version
		var fieldsJSON string
		var xsd, changelog sql.NullString
		if err := rows.Scan(&v.ID, &v.CadocCode, &v.EffectiveFrom, &v.SourceURI, &fieldsJSON, &xsd, &changelog, &v.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(fieldsJSON), &v.Fields); err != nil {
			return nil, err
		}
		if xsd.Valid {
			v.XSD = xsd.String
		}
		if changelog.Valid {
			v.Changelog = changelog.String
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ListCadocs retorna a união de CADOCs que têm schema versionado OU
// críticas cadastradas. Ordenado alfabeticamente.
//
// Sprint 6 v1.5.0 (W4): substituiu lista hardcoded em api/server.go.
// Mantém `internal/api/server.go::listSchemas/listRules` dinâmico
// conforme regras são adicionadas ao DB.
//
// Caso DB vazio (sem schema_versions/criticas), retorna slice vazio
// (NÃO faz fallback pra lista hardcoded — comportamento é orientado
// pelo estado real do banco).
func (r *Registry) ListCadocs(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT cadoc_code FROM (
			SELECT cadoc_code FROM schema_versions
			UNION
			SELECT cadoc_code FROM criticas
		)
		ORDER BY cadoc_code ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list cadocs: %w", err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CadocListCache é um cache in-memory (5min TTL) para ListCadocs.
//
// Justificativa (W4): endpoint /v1/schemas era chamado em todo dashboard
// load. Cada call = SELECT DISTINCT cadoc_code (UNION) sobre schema_versions
// + criticas. Com cache, 99% das chamadas retornam memória.
//
// Em produção: trocar para Redis (in-memory não escala horizontalmente).
//
// Validação 22 (F22.2): singleflight adicionado para evitar cache
// stampede (thundering herd) — sem isso, N goroutines em cache-miss
// simultâneo executam fetch() em paralelo, sobrecarregando DB.
type CadocListCache struct {
	mu       sync.Mutex
	cadocs   []string
	cachedAt time.Time
	ttl      time.Duration
	sf       singleflight.Group
}

func NewCadocListCache(ttl time.Duration) *CadocListCache {
	return &CadocListCache{ttl: ttl}
}

// GetOrFetch retorna cache se válido, senão chama fetch() e cacheia.
//
// fetch() é um func que retorna ([]string, error) — abstração para que
// possa ser usado tanto em produção (Registry.ListCadocs) quanto em
// testes (fake).
//
// Validação 22 (F22.2): usa singleflight para evitar cache stampede.
// Se N goroutines chamarem GetOrFetch simultaneamente com cache
// expirado, apenas 1 chama fetch(); N-1 esperam o resultado e
// compartilham. Re-check dentro do singleflight evita race entre
// primeira checagem e Do().
func (c *CadocListCache) GetOrFetch(fetch func() ([]string, error)) ([]string, error) {
	// Cache fast path.
	c.mu.Lock()
	if len(c.cadocs) > 0 && time.Since(c.cachedAt) < c.ttl {
		out := make([]string, len(c.cadocs))
		copy(out, c.cadocs)
		c.mu.Unlock()
		return out, nil
	}
	c.mu.Unlock()

	// Cache miss / expirado — singleflight protege DB.
	v, err, _ := c.sf.Do("cadocs", func() (any, error) {
		// Re-check dentro do singleflight — outra goroutine pode ter
		// acabado de popular o cache entre a primeira checagem e o Do.
		c.mu.Lock()
		if len(c.cadocs) > 0 && time.Since(c.cachedAt) < c.ttl {
			out := make([]string, len(c.cadocs))
			copy(out, c.cadocs)
			c.mu.Unlock()
			return out, nil
		}
		c.mu.Unlock()

		cadocs, fetchErr := fetch()
		if fetchErr != nil {
			return nil, fetchErr
		}
		c.mu.Lock()
		c.cadocs = cadocs
		c.cachedAt = time.Now()
		c.mu.Unlock()
		return cadocs, nil
	})
	if err != nil {
		return nil, err
	}
	cadocs, ok := v.([]string)
	if !ok {
		return nil, fmt.Errorf("internal: singleflight returned non-[]string (%T)", v)
	}
	// Caller recebe slice novo (não referenciar interno).
	out := make([]string, len(cadocs))
	copy(out, cadocs)
	return out, nil
}

// Invalidate limpa cache (usado em testes ou após mudanças manuais).
func (c *CadocListCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cadocs = nil
	c.cachedAt = time.Time{}
}

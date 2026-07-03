// Package schema implementa o Schema Registry (versionamento por data-base).
//
// Padrão: cada release do BACEN cria nova linha em schema_versions,
// identificada por (cadoc_code, effective_from). IFs consultam sempre a
// versão efetiva na data-base do envio.
package schema

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
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

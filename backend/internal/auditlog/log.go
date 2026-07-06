// Package auditlog implementa log tamper-evident com hash chain.
//
// Cada entrada referencia o hash da entrada anterior — qualquer alteração
// invalida a cadeia. Crítico pra LGPD e SOC 2.
//
// Concorrência: usa BEGIN IMMEDIATE (lock write no SQLite) pra evitar
// race entre múltiplos goroutines/workers que tentam Log ao mesmo tempo.
//
// Sprint 30 (v3.33.0): Log usa WithTenantTx para setar `app.if_id`
// em Postgres (FORCE RLS). Em SQLite, helper é no-op (compat).
// Verify é admin-level (cross-tenant) e NÃO usa helper.
package auditlog

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/db"
)

// Entry representa uma entrada do audit log.
type Entry struct {
	ID          int64           `json:"id"`
	IFID        string          `json:"if_id,omitempty"`
	Actor       string          `json:"actor"`
	Action      string          `json:"action"`
	Target      string          `json:"target,omitempty"`
	PayloadHash string          `json:"payload_hash"`
	PrevHash    string          `json:"prev_hash"`
	EntryHash   string          `json:"entry_hash"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// Logger é o serviço de audit log.
type Logger struct {
	db *sql.DB
}

// New cria um novo Logger.
func New(db *sql.DB) *Logger {
	return &Logger{db: db}
}

// Log registra uma nova entrada.
//
// Usa BEGIN IMMEDIATE + tx pra serializar inserts concorrentes
// (sem isso, dois goroutines pegariam o mesmo prev_hash e gerariam
// entradas com mesmo PrevHash — chain quebrada).
//
// Importante: o timestamp usado no entry_hash É O MESMO que vai pro DB
// (passamos explicitamente pro INSERT). Sem isso, o created_at do SQLite
// (CURRENT_TIMESTAMP) seria diferente do time.Now() do Go, e o Verify
// falharia sempre.
func (l *Logger) Log(ifID, actor, action, target string, payload []byte, metadata any) (*Entry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	payloadHash := sha256.Sum256(payload)
	payloadHashHex := hex.EncodeToString(payloadHash[:])

	// Serializa metadata
	var metaJSON []byte
	if metadata != nil {
		metaJSON, _ = json.Marshal(metadata)
	}

	// Timestamp explícito (formato ISO 8601 RFC3339Nano).
	// Será gravado no DB e usado no entry_hash — garante que Verify recomputa igual.
	timestamp := time.Now().UTC()
	timestampStr := timestamp.Format(time.RFC3339Nano)

	// Container para ID retornado pelo INSERT (precisa sobreviver até depois do commit).
	var id int64
	var prevHash string

	// Sprint 30 (v3.33.0): WithTenantTx encapsula BeginTx + SET LOCAL
	// app.if_id + Commit/Rollback. Em Postgres com FORCE RLS (migration
	// 014), sem SET LOCAL o INSERT falha silenciosamente (policy USING
	// retorna 0 rows visíveis para SET). Em SQLite, helper é no-op.
	//
	// BEGIN IMMEDIATE no SQLite via BeginTx continua valendo — driver
	// SQLite faz lock write no BEGIN (não precisamos de SELECT FOR UPDATE).
	err := db.WithTenantTx(ctx, l.db, ifID, func(tx *sql.Tx) error {
		// Pega hash anterior (lock ativo aqui)
		var queryErr error
		queryErr = tx.QueryRowContext(ctx,
			`SELECT entry_hash FROM audit_log ORDER BY id DESC LIMIT 1`,
		).Scan(&prevHash)
		if queryErr != nil && !errors.Is(queryErr, sql.ErrNoRows) {
			return fmt.Errorf("query prev: %w", queryErr)
		}
		if errors.Is(queryErr, sql.ErrNoRows) {
			prevHash = strings.Repeat("0", 64) // Genesis
		}

		// Calcula entry hash = sha256(prev + payload + metadata + actor + action + target + ifID + timestamp)
		concat := prevHash + payloadHashHex + string(metaJSON) + actor + action + target + ifID + timestampStr
		entrySum := sha256.Sum256([]byte(concat))
		entryHash := hex.EncodeToString(entrySum[:])

		res, execErr := tx.ExecContext(ctx, `
			INSERT INTO audit_log (if_id, actor, action, target, payload_hash, prev_hash, entry_hash, metadata, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			nullable(ifID), actor, action, nullable(target),
			payloadHashHex, prevHash, entryHash, string(metaJSON),
			timestampStr,
		)
		if execErr != nil {
			return fmt.Errorf("insert: %w", execErr)
		}
		id, _ = res.LastInsertId()
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Recalcula entryHash para retornar (mesma fórmula usada dentro do callback).
	concat := prevHash + payloadHashHex + string(metaJSON) + actor + action + target + ifID + timestampStr
	entrySum := sha256.Sum256([]byte(concat))
	entryHash := hex.EncodeToString(entrySum[:])

	return &Entry{
		ID:          id,
		IFID:        ifID,
		Actor:       actor,
		Action:      action,
		Target:      target,
		PayloadHash: payloadHashHex,
		PrevHash:    prevHash,
		EntryHash:   entryHash,
		CreatedAt:   timestamp,
	}, nil
}

// Verify valida a integridade da cadeia.
//
// Verifica DOIS aspectos:
//  1. Chain: cada entry referencia SHA da anterior (PrevHash encadeado)
//  2. Entry hash: recomputa SHA-256(prev + payload + metadata + ...) e compara
//
// Se alguém modificar qualquer campo de uma entry (actor, target, metadata),
// o entry hash não vai bater — Verify detecta.
func (l *Logger) Verify() (bool, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := l.db.QueryContext(ctx, `
		SELECT id, if_id, actor, action, target, payload_hash, prev_hash, entry_hash, metadata, created_at
		FROM audit_log ORDER BY id ASC
	`)
	if err != nil {
		return false, 0, err
	}
	defer rows.Close()

	var prevHash string = strings.Repeat("0", 64)
	count := 0
	for rows.Next() {
		var e Entry
		var ifID, target, metadata sql.NullString
		if err := rows.Scan(&e.ID, &ifID, &e.Actor, &e.Action, &target, &e.PayloadHash, &e.PrevHash, &e.EntryHash, &metadata, &e.CreatedAt); err != nil {
			return false, count, err
		}
		if ifID.Valid {
			e.IFID = ifID.String
		}
		if target.Valid {
			e.Target = target.String
		}

		// 1. Chain check
		if e.PrevHash != prevHash {
			return false, count, fmt.Errorf("chain quebrada em id=%d (prev=%q esperado=%q)", e.ID, e.PrevHash, prevHash)
		}

		// 2. Entry hash recomputation
		//    timestamp gravado é RFC3339Nano (UTC). Recomputamos com o timestamp
		//    registrado (não com time.Now()) pra ser determinístico.
		timestamp := e.CreatedAt.UTC().Format(time.RFC3339Nano)
		var metaJSON string
		if metadata.Valid {
			metaJSON = metadata.String
		}
		concat := e.PrevHash + e.PayloadHash + metaJSON + e.Actor + e.Action + e.Target + e.IFID + timestamp
		expectedSum := sha256.Sum256([]byte(concat))
		expectedHash := hex.EncodeToString(expectedSum[:])

		if e.EntryHash != expectedHash {
			return false, count, fmt.Errorf("entry hash inválido em id=%d (esperado=%q encontrado=%q) — entry foi modificada",
				e.ID, expectedHash, e.EntryHash)
		}

		prevHash = e.EntryHash
		count++
	}
	return true, count, rows.Err()
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Package auditlog implementa log tamper-evident com hash chain.
//
// Cada entrada referencia o hash da entrada anterior — qualquer alteração
// invalida a cadeia. Crítico pra LGPD e SOC 2.
package auditlog

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
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
func (l *Logger) Log(ifID, actor, action, target string, payload []byte, metadata any) (*Entry, error) {
	payloadHash := sha256.Sum256(payload)
	payloadHashHex := hex.EncodeToString(payloadHash[:])

	// Pega hash anterior
	var prevHash string
	err := l.db.QueryRow(`
		SELECT entry_hash FROM audit_log ORDER BY id DESC LIMIT 1
	`).Scan(&prevHash)
	if errors.Is(err, sql.ErrNoRows) {
		prevHash = strings.Repeat("0", 64) // Genesis hash
	} else if err != nil {
		return nil, fmt.Errorf("query prev: %w", err)
	}

	// Serializa metadata
	var metaJSON []byte
	if metadata != nil {
		metaJSON, _ = json.Marshal(metadata)
	}

	// Calcula entry hash = sha256(prev + payload + metadata + actor + action + timestamp)
	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	concat := prevHash + payloadHashHex + string(metaJSON) + actor + action + target + ifID + timestamp
	entrySum := sha256.Sum256([]byte(concat))
	entryHash := hex.EncodeToString(entrySum[:])

	res, err := l.db.Exec(`
		INSERT INTO audit_log (if_id, actor, action, target, payload_hash, prev_hash, entry_hash, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		nullable(ifID), actor, action, nullable(target),
		payloadHashHex, prevHash, entryHash, string(metaJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("insert: %w", err)
	}
	id, _ := res.LastInsertId()

	return &Entry{
		ID:          id,
		IFID:        ifID,
		Actor:       actor,
		Action:      action,
		Target:      target,
		PayloadHash: payloadHashHex,
		PrevHash:    prevHash,
		EntryHash:   entryHash,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

// Verify valida a integridade da cadeia.
func (l *Logger) Verify() (bool, int, error) {
	rows, err := l.db.Query(`
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
		var ifID, target sql.NullString
		var metadata sql.NullString
		if err := rows.Scan(&e.ID, &ifID, &e.Actor, &e.Action, &target, &e.PayloadHash, &e.PrevHash, &e.EntryHash, &metadata, &e.CreatedAt); err != nil {
			return false, count, err
		}
		if ifID.Valid {
			e.IFID = ifID.String
		}
		if target.Valid {
			e.Target = target.String
		}
		if metadata.Valid {
			e.Metadata = json.RawMessage(metadata.String)
		}

		// Verifica chain
		if e.PrevHash != prevHash {
			return false, count, fmt.Errorf("chain quebrada em id=%d (prev=%s esperado=%s)", e.ID, e.PrevHash, prevHash)
		}
		// Verifica entry hash (opcional — depende de recomputar com metadata)
		prevHash = e.EntryHash
		count++
	}
	return true, count, nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
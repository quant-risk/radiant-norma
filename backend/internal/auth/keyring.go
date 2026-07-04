// Package auth — Keyring com rotação grace.
//
// Rota policy:
//   - Active: usada para emissão (sign).
//   - Retired: tokens existentes (até expiry) ainda verificam.
//   - Periodically rotate: nova active key, antiga fica retired por
//     grace de tokens antigos (recomendado: max(token TTL × 2)).
package auth

import (
	"crypto/rsa"
	"sort"
	"sync"
	"time"
)

// Key é uma chave pública RSA com metadata de rotação.
type Key struct {
	// Kid: Key ID. Único dentro do Keyring. JWT header inclui kid
	// para verifier selecionar chave certa.
	Kid string
	// PublicKey: chave RSA pública.
	PublicKey *rsa.PublicKey
	// Active: ainda usada para emissão. False = retired (apenas
	// verificação de tokens antigos até expiry).
	Active bool
	// CreatedAt: timestamp para auditoria.
	CreatedAt time.Time
}

// Keyring mantém conjunto de keys por kid com mutex.
type Keyring struct {
	mu   sync.RWMutex
	keys map[string]*Key
}

// NewKeyring cria keyring vazio.
func NewKeyring() *Keyring {
	return &Keyring{keys: make(map[string]*Key)}
}

// Add adiciona key. Substitui se kid já existe.
func (k *Keyring) Add(key *Key) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.keys[key.Kid] = key
}

// Get retorna key por kid.
func (k *Keyring) Get(kid string) (*Key, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	key, ok := k.keys[kid]
	return key, ok
}

// ActiveKey retorna 1 chave ativa (assume single-active por emissão).
// Erro se não houver — caller deve panic-bootstrap.
func (k *Keyring) ActiveKey() (*Key, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	for _, key := range k.keys {
		if key.Active {
			return key, nil
		}
	}
	return nil, ErrNoActiveKey
}

// RetiredKeys retorna lista de keys retired (read-only access).
func (k *Keyring) RetiredKeys() []*Key {
	k.mu.RLock()
	defer k.mu.RUnlock()
	var out []*Key
	for _, key := range k.keys {
		if !key.Active {
			out = append(out, key)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

// Rotate substitui active key por nova. Antiga fica retired.
//
// Após Rotate, tokens emitidos com kid antigo AINDA verificam (grace
// period até expiry). Novas emissões usam kid novo.
func (k *Keyring) Rotate(newActive *Key) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, key := range k.keys {
		if key.Active {
			key.Active = false
		}
	}
	if _, exists := k.keys[newActive.Kid]; exists {
		return ErrDuplicateKid
	}
	k.keys[newActive.Kid] = newActive
	return nil
}

// ErrNoActiveKey é retornado quando Keyring tem 0 active keys.
var ErrNoActiveKey = errNoActiveKey("auth: keyring sem active key — bootstrap pendente")

type errNoActiveKey string

func (e errNoActiveKey) Error() string { return string(e) }

// ErrDuplicateKid é retornado em Rotate quando kid novo já existe.
var ErrDuplicateKid = errDuplicateKid("auth: kid já existe no keyring")

type errDuplicateKid string

func (e errDuplicateKid) Error() string { return string(e) }

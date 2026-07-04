// Package auth implementa JWT bearer authentication para substituir o
// placeholder X-IF-ID do Sprint 6.
//
// Sprint 7a (v1.6.0): JWT RS256 com claims { sub, if_id, role, exp, iat, iss }.
//
// Por design:
//   - Issuer pinning: tokens NÃO nossos são rejeitados (defesa contra
//     alg-tech token forgery).
//   - Constant-time compare: defesa contra timing attack em claims.
//   - Key rotation: keyring com active + previous (grace period para
//     tokens antigos após rotação).
//   - Dev flag: X-IF-ID ainda aceito se RADIANT_DEV_AUTH=1 (migration
//     helper).
//
// Backward compatibility: até v1.7.0, X-IF-ID coexistia com JWT para
// permitir migration clients. Default em produção é JWT obrigatório.
package auth

import (
	"errors"
	"fmt"
	"time"
)

// Role identifica perfil do portador do token. Usado para autorização
// em endpoints admin (ex: /v1/radar/scan).
type Role string

const (
	// RoleIF: Instituição Financeira. Cliente normal — lê schemas,
	// submete envios STA.
	RoleIF Role = "if"
	// RoleAdmin: operador Radiant Norma. Admin actions (radar/scan,
	// audit log queries).
	RoleAdmin Role = "admin"
	// RoleReadOnly: leitura sem mutação. Útil para integrações
	// de relatórios.
	RoleReadOnly Role = "readonly"
)

// Claims é o payload JWT. Mantemos em struct separada do jwt.RegisteredClaims
// para type-safety e validação extra (chaves customizadas).
type Claims struct {
	// sub: subject (id único do usuário/IF). Para IFs, é o CNPJ raiz.
	Sub string `json:"sub" validate:"required"`
	// if_id: tenant identifier (multi-IF multi-tenant). Substitui o
	// placeholder X-IF-ID em runtime.
	IFID string `json:"if_id" validate:"required,max=64"`
	// role: autorização coarse-grained. Validado contra roles ativas.
	Role Role `json:"role" validate:"required,oneof=if admin readonly"`
	// iss: issuer pinning. Deve bater com Issuer configurado no
	// verifier. Defesa contra tokens cross-tenant ou cross-org.
	Iss string `json:"iss" validate:"required"`
	// exp: expiry. Validado pelo golang-jwt RegisteredClaims.
	Exp time.Time `json:"exp"`
	// iat: issued-at. Para auditoria (token recent vs velho).
	Iat time.Time `json:"iat"`
	// jti: JWT id. Útil para revocação (se implementarmos jti store).
	Jti string `json:"jti,omitempty"`
}

// Validate checa invariants da claim em adição ao JWT-level validation.
func (c *Claims) Validate() error {
	if c.Sub == "" {
		return errors.New("claims: sub (subject) é obrigatório")
	}
	if c.IFID == "" {
		return errors.New("claims: if_id (tenant) é obrigatório")
	}
	if len(c.IFID) > 64 {
		return errors.New("claims: if_id > 64 chars (X-IF-ID-compatible)")
	}
	switch c.Role {
	case RoleIF, RoleAdmin, RoleReadOnly:
		// ok
	case "":
		return errors.New("claims: role é obrigatório")
	default:
		return fmt.Errorf("claims: role inválido %q (esperado if/admin/readonly)", c.Role)
	}
	if c.Iss == "" {
		return errors.New("claims: iss (issuer) é obrigatório — defesa contra cross-tenant")
	}
	if c.Exp.IsZero() {
		return errors.New("claims: exp (expiry) é obrigatório")
	}
	if c.Exp.Before(time.Now()) {
		return errors.New("claims: token expired")
	}
	return nil
}

// HasRole retorna true se a role tem autorização requisitada.
//
// Hierarquia: admin > if > readonly. Admin tem todos os acessos; if
// tem apenas seu tenant; readonly não tem acesso a mutações.
func (c *Claims) HasRole(required Role) bool {
	if c.Role == RoleAdmin {
		return true // admin bypassa checagem
	}
	return c.Role == required
}

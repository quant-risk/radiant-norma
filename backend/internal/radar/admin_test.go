// Tests para Radar hardening (R1 — DOS-via-API prevention, Sprint 6 v1.5.0):
//   - ScanLimiter: rate limit por chave (1/min)
//   - ScanCache: cache de último resultado com TTL
//   - AdminAuth: FAIL CLOSED sem token, X-Admin header, Bearer token
//
// Cobertura complementa radar_test.go (que foca em scan/baseline/resolve).
package radar_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/radar"
)

// ===========================
// ScanLimiter
// ===========================

func TestScanLimiter_AllowsFirstCall(t *testing.T) {
	l := radar.NewScanLimiter(1 * time.Minute)

	ok, _ := l.Allow("if-1")
	if !ok {
		t.Errorf("Primeira chamada deveria ser permitida")
	}
}

func TestScanLimiter_BlocksSecondCallWithinCooldown(t *testing.T) {
	l := radar.NewScanLimiter(1 * time.Minute)

	l.Allow("if-1")
	ok, retryAfter := l.Allow("if-1")
	if ok {
		t.Errorf("Segunda chamada em < 1min deveria ser bloqueada")
	}
	if retryAfter <= 0 || retryAfter > time.Minute {
		t.Errorf("retryAfter = %v, want (0, 1min]", retryAfter)
	}
}

func TestScanLimiter_PerKeyIsolation(t *testing.T) {
	l := radar.NewScanLimiter(1 * time.Minute)

	l.Allow("if-1")
	// if-2 é independente — deve permitir
	ok, _ := l.Allow("if-2")
	if !ok {
		t.Errorf("if-2 deveria ser independente de if-1")
	}
}

func TestScanLimiter_AllowsAfterCooldown(t *testing.T) {
	l := radar.NewScanLimiter(50 * time.Millisecond)

	l.Allow("if-1")
	time.Sleep(100 * time.Millisecond) // passa cooldown

	ok, _ := l.Allow("if-1")
	if !ok {
		t.Errorf("Após cooldown deveria permitir de novo")
	}
}

// ===========================
// ScanCache
// ===========================

func TestScanCache_MissInitially(t *testing.T) {
	c := radar.NewScanCache(5 * time.Minute)

	_, ok := c.Get()
	if ok {
		t.Errorf("Cache deveria estar vazio inicialmente")
	}
}

func TestScanCache_HitAfterPut(t *testing.T) {
	c := radar.NewScanCache(5 * time.Minute)

	expected := []radar.Alert{{ID: 1, CadocCode: "3040", Title: "test"}}
	c.Put(expected)

	got, ok := c.Get()
	if !ok {
		t.Fatalf("Cache deveria ter hit")
	}
	if len(got) != 1 {
		t.Errorf("Esperado 1 alerta, got %d", len(got))
	}
	if got[0].CadocCode != "3040" {
		t.Errorf("CadocCode = %q, want 3040", got[0].CadocCode)
	}
}

func TestScanCache_TTLExpiration(t *testing.T) {
	c := radar.NewScanCache(50 * time.Millisecond)
	c.Put([]radar.Alert{{ID: 1}})

	// Imediatamente após Put: hit
	_, ok := c.Get()
	if !ok {
		t.Errorf("Imediatamente após Put deveria ter hit")
	}

	// Após TTL: miss
	time.Sleep(100 * time.Millisecond)
	_, ok = c.Get()
	if ok {
		t.Errorf("Após TTL deveria ter miss")
	}
}

func TestScanCache_Invalidate(t *testing.T) {
	c := radar.NewScanCache(5 * time.Minute)
	c.Put([]radar.Alert{{ID: 1}})

	c.Invalidate()

	_, ok := c.Get()
	if ok {
		t.Errorf("Após Invalidate deveria ter miss")
	}
}

func TestScanCache_ReturnedCopyIsIndependent(t *testing.T) {
	c := radar.NewScanCache(5 * time.Minute)
	original := []radar.Alert{{ID: 1, Title: "original"}}
	c.Put(original)

	got, _ := c.Get()
	got[0].Title = "mutated"

	// Próximo Get deve retornar título original (cache não foi mutado)
	got2, _ := c.Get()
	if got2[0].Title != "original" {
		t.Errorf("Cache deveria ser imutável ao consumidor, got %q", got2[0].Title)
	}
}

// ===========================
// AdminAuth
// ===========================

func TestAdminAuth_FailClosedWithoutToken(t *testing.T) {
	a := &radar.AdminAuth{Token: ""}

	req := httptest.NewRequest("POST", "/v1/radar/scan", nil)
	req.Header.Set("X-Admin", "anything")
	if a.IsAdmin(req) {
		t.Errorf("Auth deveria FECHAR quando Token vazio")
	}
}

func TestAdminAuth_XAdminHeaderValid(t *testing.T) {
	a := &radar.AdminAuth{Token: "secret123"}

	req := httptest.NewRequest("POST", "/v1/radar/scan", nil)
	req.Header.Set("X-Admin", "secret123")
	if !a.IsAdmin(req) {
		t.Errorf("X-Admin com token correto deveria passar")
	}
}

func TestAdminAuth_XAdminHeaderInvalid(t *testing.T) {
	a := &radar.AdminAuth{Token: "secret123"}

	req := httptest.NewRequest("POST", "/v1/radar/scan", nil)
	req.Header.Set("X-Admin", "wrong-token")
	if a.IsAdmin(req) {
		t.Errorf("X-Admin com token errado deveria falhar")
	}
}

func TestAdminAuth_BearerTokenValid(t *testing.T) {
	a := &radar.AdminAuth{Token: "secret123"}

	req := httptest.NewRequest("POST", "/v1/radar/scan", nil)
	req.Header.Set("Authorization", "Bearer secret123")
	if !a.IsAdmin(req) {
		t.Errorf("Bearer com token correto deveria passar")
	}
}

func TestAdminAuth_BearerTokenInvalid(t *testing.T) {
	a := &radar.AdminAuth{Token: "secret123"}

	req := httptest.NewRequest("POST", "/v1/radar/scan", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	if a.IsAdmin(req) {
		t.Errorf("Bearer com token errado deveria falhar")
	}
}

func TestAdminAuth_RequiresHeader(t *testing.T) {
	a := &radar.AdminAuth{Token: "secret123"}

	req := httptest.NewRequest("POST", "/v1/radar/scan", nil)
	if a.IsAdmin(req) {
		t.Errorf("Sem header deveria falhar")
	}
}

func TestAdminAuth_TimingSafe(t *testing.T) {
	// Teste-fumaça: garante que constantTimeEqual existe e funciona.
	// Não é teste de timing real (complexo de medir em unit test),
	// mas exercita o código.
	a := &radar.AdminAuth{Token: "secret123"}

	// Equal-length strings com mesmo conteúdo
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("X-Admin", "secret123")
	if !a.IsAdmin(req) {
		t.Errorf("equal-length match deveria passar")
	}

	// Equal-length strings com conteúdo diferente
	req = httptest.NewRequest("POST", "/", nil)
	req.Header.Set("X-Admin", "secret124")
	if a.IsAdmin(req) {
		t.Errorf("equal-length mismatch deveria falhar")
	}

	// Different-length strings
	req = httptest.NewRequest("POST", "/", nil)
	req.Header.Set("X-Admin", "secret1234")
	if a.IsAdmin(req) {
		t.Errorf("different-length mismatch deveria falhar")
	}
}

// Tests para o pacote version — basic sanity.
package version_test

import (
	"regexp"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/version"
)

func TestVersion_NotEmpty(t *testing.T) {
	if version.Version == "" {
		t.Fatal("version.Version está vazio — atualizar este pacote")
	}
}

func TestVersion_FormatoSemver(t *testing.T) {
	// Aceita "1.5.0", "v1.5.0", "1.5.0-dev", "1.5.0+commit123".
	// Não aceita "v1", "1.5", string vazia.
	re := regexp.MustCompile(`^(v?\d+\.\d+\.\d+(?:[-+][\w.-]+)?|dev)$`)
	if !re.MatchString(version.Version) {
		t.Errorf("version.Version não segue semver/ldflags: %q", version.Version)
	}
}

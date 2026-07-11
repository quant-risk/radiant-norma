// Package api — helpers compartilhados por tests no package api.
//
// Estes helpers são usados por todos os arquivos *_test.go neste package.
package api

import (
	"net/http"
)

// authRequest attaches X-IF-ID header for dev auth (RADIANT_DEV_AUTH=1).
func authRequest(r *http.Request, ifID string) {
	r.Header.Set("X-IF-ID", ifID)
}

// Package realtime — decorator que publica audit events no Hub.
//
// Sprint 10: cada chamada a auditlog.Logger.Log() também faz Publish
// no Hub SSE (além de gravar no DB com chain). Isso mantém LGPD/SOC2
// (audit_log chain) + push real-time (SSE).
//
// Em produção: também pode ligar a Kafka/Redis pub/sub — mas pra
// Sprint 10 in-process Hub é suficiente.

package realtime

import (
	"context"

	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/observability"
)

// HubAwareLogger é wrapper que adiciona Publish após Log.
//
// Uso no main.go:
//
//	auditLog := auditlog.New(d)
//	hub := realtime.NewHub(logger)
//	auditLog = realtime.WrapAuditLog(auditLog, hub)
type HubAwareLogger struct {
	*auditlog.Logger
	hub *Hub
}

// WrapAuditLog retorna HubAwareLogger que delega ao original + publica
// evento no Hub após gravar.
func WrapAuditLog(base *auditlog.Logger, hub *Hub) *HubAwareLogger {
	return &HubAwareLogger{Logger: base, hub: hub}
}

// Log sobrescreve o método Log original (mesmo signature) pra chamar
// o do base + publicar evento no Hub.
//
// Sprint 36 — v3.36.0: após gravar no DB, adiciona Sentry breadcrumb
// para que mutations apareçam no trace de erros. Usa context.Background()
// como fallback se nenhum contexto estiver em scope (auditlog não Propaga
// ctx, mas o wrapper pode usar o span atual via GetHubFromContext nil-safe).
func (h *HubAwareLogger) Log(
	ifID, actor, action, target string,
	payload []byte,
	metadata any,
) (*auditlog.Entry, error) {
	// Chama o base (sem ctx — auditlog.Logger.Log não aceita ctx).
	entry, err := h.Logger.Log(ifID, actor, action, target, payload, metadata)
	if err != nil {
		return entry, err
	}

	// Sprint 36: Sentry breadcrumb para mutations (audit trail in error traces).
	// Usa-se context.Background() aqui porque AddBreadcrumb só requer o hub,
	// não o span — breadcrumbs são attached ao hub global, não ao span.
	observability.AddBreadcrumb(context.Background(), action, target, map[string]any{
		"actor":    actor,
		"if_id":    ifID,
		"entry_id": entry.EntryHash[:16],
	})

	// Publica no hub (best-effort — não bloqueia audit_log se hub não estiver saudável)
	if h.hub != nil {
		h.hub.Publish(Event{
			Kind:      action,
			IFID:      ifID,
			Payload:   entryToPayload(entry, metadata),
			Timestamp: entry.CreatedAt,
		})
	}

	return entry, nil
}

func entryToPayload(e *auditlog.Entry, metadata any) map[string]any {
	out := map[string]any{
		"id":       e.ID,
		"actor":    e.Actor,
		"action":   e.Action,
		"target":   e.Target,
		"entry_id": e.EntryHash[:16],
		"created":  e.CreatedAt,
	}
	if metadata != nil {
		out["metadata"] = metadata
	}
	return out
}

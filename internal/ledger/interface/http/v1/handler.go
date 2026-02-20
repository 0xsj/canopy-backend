package v1

import (
	"net/http"
	"time"

	"github.com/0xsj/canopy-backend/internal/ledger/domain"
	"github.com/0xsj/canopy-backend/internal/ledger/service"
	"github.com/0xsj/canopy-backend/pkg/httpserver"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

type Handler struct {
	svc *service.Service
	log logger.Logger
}

func NewHandler(svc *service.Service, log logger.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/ledger/system", h.handleQuerySystem)
	mux.HandleFunc("GET /api/v1/admin/ledger/domain", h.handleQueryDomain)
	mux.HandleFunc("GET /api/v1/workspaces/{wsId}/activity", h.handleWorkspaceActivity)
}

func (h *Handler) handleQuerySystem(w http.ResponseWriter, r *http.Request) {
	filter := domain.SystemFilter{
		Limit: httpserver.QueryInt(r, "limit", 50),
	}

	if v := httpserver.QueryString(r, "source_context", ""); v != "" {
		filter.SourceContext = &v
	}
	if v := httpserver.QueryString(r, "event_subject", ""); v != "" {
		filter.EventSubject = &v
	}
	if v := httpserver.QueryString(r, "after", ""); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			types.WriteError(w, http.StatusBadRequest, "invalid_param", "after must be RFC3339")
			return
		}
		filter.After = &t
	}
	if v := httpserver.QueryString(r, "before", ""); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			types.WriteError(w, http.StatusBadRequest, "invalid_param", "before must be RFC3339")
			return
		}
		filter.Before = &t
	}

	entries, err := h.svc.QuerySystem(r.Context(), filter)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, SystemEntriesFromDomain(entries))
}

func (h *Handler) handleQueryDomain(w http.ResponseWriter, r *http.Request) {
	filter := domain.DomainFilter{
		Limit: httpserver.QueryInt(r, "limit", 50),
	}

	if v := httpserver.QueryString(r, "actor_id", ""); v != "" {
		filter.ActorID = &v
	}
	if v := httpserver.QueryString(r, "action", ""); v != "" {
		a := domain.Action(v)
		filter.Action = &a
	}
	if v := httpserver.QueryString(r, "resource_type", ""); v != "" {
		filter.ResourceType = &v
	}
	if v := httpserver.QueryString(r, "resource_id", ""); v != "" {
		filter.ResourceID = &v
	}
	if v := httpserver.QueryString(r, "org_id", ""); v != "" {
		filter.OrgID = &v
	}
	if v := httpserver.QueryString(r, "workspace_id", ""); v != "" {
		filter.WorkspaceID = &v
	}
	if v := httpserver.QueryString(r, "after", ""); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			types.WriteError(w, http.StatusBadRequest, "invalid_param", "after must be RFC3339")
			return
		}
		filter.After = &t
	}
	if v := httpserver.QueryString(r, "before", ""); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			types.WriteError(w, http.StatusBadRequest, "invalid_param", "before must be RFC3339")
			return
		}
		filter.Before = &t
	}

	entries, err := h.svc.QueryDomain(r.Context(), filter)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, DomainEntriesFromDomain(entries))
}

func (h *Handler) handleWorkspaceActivity(w http.ResponseWriter, r *http.Request) {
	wsID := types.WorkspaceIDFrom(r.PathValue("wsId"))

	limit := httpserver.QueryInt(r, "limit", 50)
	if limit > 200 {
		limit = 200
	}

	entries, err := h.svc.QueryWorkspaceActivity(r.Context(), wsID, limit)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, DomainEntriesFromDomain(entries))
}

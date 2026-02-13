package v1

import (
	"net/http"

	"github.com/0xsj/canopy-backend/internal/deliverable/domain"
	"github.com/0xsj/canopy-backend/internal/deliverable/service"
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
	mux.HandleFunc("POST /api/v1/workspaces/{wsId}/deliverables", h.handleCreateDraft)
	mux.HandleFunc("PATCH /api/v1/deliverables/{deliverableId}", h.handleUpdate)
	mux.HandleFunc("POST /api/v1/deliverables/{deliverableId}/finalize", h.handleFinalize)
	mux.HandleFunc("GET /api/v1/deliverables/{deliverableId}", h.handleGetDeliverable)
	mux.HandleFunc("GET /api/v1/workspaces/{wsId}/deliverables", h.handleListByWorkspace)
}

func (h *Handler) handleCreateDraft(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req CreateDraftRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	leafIDs := make([]types.LeafID, len(req.SourceLeafIDs))
	for i, raw := range req.SourceLeafIDs {
		id, err := types.ParseLeafID(raw)
		if err != nil {
			types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
			return
		}
		leafIDs[i] = id
	}

	deliverable, err := h.svc.CreateDraft(r.Context(), wsID, domain.Format(req.Format), req.Content, leafIDs)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteCreated(w, DeliverableFromDomain(deliverable))
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	deliverableID, err := types.ParseDeliverableID(httpserver.PathParam(r, "deliverableId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req UpdateDeliverableRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	deliverable, err := h.svc.Update(r.Context(), deliverableID, req.Content)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, DeliverableFromDomain(deliverable))
}

func (h *Handler) handleFinalize(w http.ResponseWriter, r *http.Request) {
	deliverableID, err := types.ParseDeliverableID(httpserver.PathParam(r, "deliverableId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	deliverable, err := h.svc.Finalize(r.Context(), deliverableID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, DeliverableFromDomain(deliverable))
}

func (h *Handler) handleGetDeliverable(w http.ResponseWriter, r *http.Request) {
	deliverableID, err := types.ParseDeliverableID(httpserver.PathParam(r, "deliverableId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	deliverable, err := h.svc.FindByID(r.Context(), deliverableID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, DeliverableFromDomain(deliverable))
}

func (h *Handler) handleListByWorkspace(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	deliverables, err := h.svc.FindByWorkspace(r.Context(), wsID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, DeliverablesFromDomain(deliverables))
}

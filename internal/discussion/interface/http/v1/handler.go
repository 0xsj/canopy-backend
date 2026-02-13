package v1

import (
	"net/http"

	"github.com/0xsj/canopy-backend/internal/discussion/service"
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
	mux.HandleFunc("POST /api/v1/workspaces/{wsId}/leaves/{leafId}/comments", h.handleAddComment)
	mux.HandleFunc("GET /api/v1/leaves/{leafId}/thread", h.handleGetThread)
}

func (h *Handler) handleAddComment(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	leafID, err := types.ParseLeafID(httpserver.PathParam(r, "leafId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req AddCommentRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	if err := h.svc.AddComment(r.Context(), wsID, leafID, req.Content); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteCreated(w, struct{}{})
}

func (h *Handler) handleGetThread(w http.ResponseWriter, r *http.Request) {
	leafID, err := types.ParseLeafID(httpserver.PathParam(r, "leafId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	thread, err := h.svc.FindThread(r.Context(), leafID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, ThreadFromDomain(thread))
}

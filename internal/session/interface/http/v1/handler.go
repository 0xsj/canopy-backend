package v1

import (
	"net/http"

	"github.com/0xsj/canopy-backend/internal/session/domain"
	"github.com/0xsj/canopy-backend/internal/session/service"
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
	mux.HandleFunc("POST /api/v1/workspaces/{wsId}/sessions", h.handleStartSession)
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/messages", h.handleAddMessage)
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/checkpoint", h.handleCheckpoint)
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/complete", h.handleComplete)
}

func (h *Handler) handleStartSession(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req StartSessionRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	seedID, err := types.ParseSeedID(req.SeedID)
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var parentLeafID types.LeafID
	if req.ParentLeafID != "" {
		parentLeafID, err = types.ParseLeafID(req.ParentLeafID)
		if err != nil {
			types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
			return
		}
	}

	session, err := h.svc.StartSession(r.Context(), wsID, seedID, parentLeafID, domain.SessionType(req.SessionType))
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteCreated(w, SessionFromDomain(session))
}

func (h *Handler) handleAddMessage(w http.ResponseWriter, r *http.Request) {
	sessionID, err := domain.ParseSessionID(httpserver.PathParam(r, "sessionId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req AddMessageRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	session, err := h.svc.AddMessage(r.Context(), sessionID, req.Message)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, SessionFromDomain(session))
}

func (h *Handler) handleCheckpoint(w http.ResponseWriter, r *http.Request) {
	sessionID, err := domain.ParseSessionID(httpserver.PathParam(r, "sessionId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	session, err := h.svc.Checkpoint(r.Context(), sessionID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, SessionFromDomain(session))
}

func (h *Handler) handleComplete(w http.ResponseWriter, r *http.Request) {
	sessionID, err := domain.ParseSessionID(httpserver.PathParam(r, "sessionId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req CompleteSessionRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	leafID, err := types.ParseLeafID(req.LeafID)
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	session, err := h.svc.CompleteSession(r.Context(), sessionID, leafID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, SessionFromDomain(session))
}

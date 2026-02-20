package v1

import (
	"net/http"

	"github.com/0xsj/canopy-backend/internal/synthesis/domain"
	"github.com/0xsj/canopy-backend/internal/synthesis/service"
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
	mux.HandleFunc("POST /api/v1/workspaces/{wsId}/syntheses", h.handleStartSynthesis)
	mux.HandleFunc("POST /api/v1/syntheses/{synthesisId}/complete", h.handleComplete)
	mux.HandleFunc("POST /api/v1/syntheses/{synthesisId}/fail", h.handleFail)
	mux.HandleFunc("GET /api/v1/workspaces/{wsId}/syntheses", h.handleListByWorkspace)
}

func (h *Handler) handleStartSynthesis(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req StartSynthesisRequest
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

	sw, err := h.svc.StartSynthesis(r.Context(), wsID, leafIDs)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteAccepted(w, SynthesisFromDomain(sw))
}

func (h *Handler) handleComplete(w http.ResponseWriter, r *http.Request) {
	synthesisID, err := domain.ParseSynthesisID(httpserver.PathParam(r, "synthesisId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req CompleteSynthesisRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	resultLeafID, err := types.ParseLeafID(req.ResultLeafID)
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	sw, err := h.svc.CompleteSynthesis(r.Context(), synthesisID, resultLeafID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, SynthesisFromDomain(sw))
}

func (h *Handler) handleFail(w http.ResponseWriter, r *http.Request) {
	synthesisID, err := domain.ParseSynthesisID(httpserver.PathParam(r, "synthesisId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req FailSynthesisRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	if err := h.svc.FailSynthesis(r.Context(), synthesisID, req.Reason); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleListByWorkspace(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	workflows, err := h.svc.FindByWorkspace(r.Context(), wsID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, SynthesesFromDomain(workflows))
}

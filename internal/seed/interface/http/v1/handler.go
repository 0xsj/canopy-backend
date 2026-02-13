package v1

import (
	"net/http"

	"github.com/0xsj/canopy-backend/internal/seed/service"
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
	mux.HandleFunc("POST /api/v1/workspaces/{wsId}/seeds", h.handlePlant)
	mux.HandleFunc("GET /api/v1/seeds/{seedId}", h.handleGetSeed)
	mux.HandleFunc("GET /api/v1/workspaces/{wsId}/seeds", h.handleListByWorkspace)
	mux.HandleFunc("PATCH /api/v1/seeds/{seedId}/constraints", h.handleUpdateConstraints)
}

func (h *Handler) handlePlant(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req PlantSeedRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	seed, err := h.svc.Plant(r.Context(), wsID, req.Title, req.Description)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteCreated(w, SeedFromDomain(seed))
}

func (h *Handler) handleGetSeed(w http.ResponseWriter, r *http.Request) {
	seedID, err := types.ParseSeedID(httpserver.PathParam(r, "seedId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	seed, err := h.svc.FindByID(r.Context(), seedID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, SeedFromDomain(seed))
}

func (h *Handler) handleListByWorkspace(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	seeds, err := h.svc.FindByWorkspace(r.Context(), wsID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, SeedsFromDomain(seeds))
}

func (h *Handler) handleUpdateConstraints(w http.ResponseWriter, r *http.Request) {
	seedID, err := types.ParseSeedID(httpserver.PathParam(r, "seedId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req UpdateConstraintsRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	seed, err := h.svc.UpdateConstraints(r.Context(), seedID, req.Constraints)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, SeedFromDomain(seed))
}

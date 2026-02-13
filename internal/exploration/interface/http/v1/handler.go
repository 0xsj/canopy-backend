package v1

import (
	"net/http"
	"strings"

	"github.com/0xsj/canopy-backend/internal/exploration/domain"
	"github.com/0xsj/canopy-backend/internal/exploration/service"
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
	mux.HandleFunc("POST /api/v1/workspaces/{wsId}/leaves", h.handleCreateLeaf)
	mux.HandleFunc("GET /api/v1/leaves/{leafId}", h.handleGetLeaf)
	mux.HandleFunc("GET /api/v1/workspaces/{wsId}/leaves", h.handleListLeaves)
	mux.HandleFunc("POST /api/v1/workspaces/{wsId}/branches", h.handleStartBranch)
	mux.HandleFunc("GET /api/v1/branches/{branchId}", h.handleGetBranch)
	mux.HandleFunc("POST /api/v1/workspaces/{wsId}/connections", h.handleCreateConnection)
	mux.HandleFunc("GET /api/v1/leaves/{leafId}/connections", h.handleListConnections)
	mux.HandleFunc("POST /api/v1/leaves/{leafId}/promote", h.handlePromoteLeaf)
}

func (h *Handler) handleCreateLeaf(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req CreateLeafRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	seedID, err := types.ParseSeedID(req.SeedID)
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	branchID, err := types.ParseBranchID(req.BranchID)
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	parentLeafID, err := types.ParseLeafID(req.ParentLeafID)
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	leaf, err := h.svc.CreateLeaf(r.Context(), wsID, seedID, branchID, parentLeafID, req.Title, req.Summary, req.KeyPoints, req.OpenQuestions, req.Tags)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteCreated(w, LeafFromDomain(leaf))
}

func (h *Handler) handleGetLeaf(w http.ResponseWriter, r *http.Request) {
	leafID, err := types.ParseLeafID(httpserver.PathParam(r, "leafId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	leaf, err := h.svc.FindLeafByID(r.Context(), leafID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, LeafFromDomain(leaf))
}

func (h *Handler) handleListLeaves(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	filter := domain.LeafFilter{}

	if raw := httpserver.QueryString(r, "seed_id", ""); raw != "" {
		id, err := types.ParseSeedID(raw)
		if err != nil {
			types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
			return
		}
		filter.SeedID = &id
	}

	if raw := httpserver.QueryString(r, "author_id", ""); raw != "" {
		id, err := types.ParseUserID(raw)
		if err != nil {
			types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
			return
		}
		filter.AuthorID = &id
	}

	if raw := httpserver.QueryString(r, "layer", ""); raw != "" {
		layer := domain.Layer(raw)
		filter.Layer = &layer
	}

	if raw := httpserver.QueryString(r, "tags", ""); raw != "" {
		filter.Tags = strings.Split(raw, ",")
	}

	leaves, err := h.svc.FindLeavesByWorkspace(r.Context(), wsID, filter)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, LeavesFromDomain(leaves))
}

func (h *Handler) handleStartBranch(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req StartBranchRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	seedID, err := types.ParseSeedID(req.SeedID)
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	branch, leaf, err := h.svc.StartBranch(r.Context(), wsID, seedID, req.Title, req.Summary, req.KeyPoints, req.OpenQuestions, req.Tags)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteCreated(w, StartBranchResponse{
		Branch: BranchFromDomain(branch),
		Leaf:   LeafFromDomain(leaf),
	})
}

func (h *Handler) handleGetBranch(w http.ResponseWriter, r *http.Request) {
	branchID, err := types.ParseBranchID(httpserver.PathParam(r, "branchId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	branch, err := h.svc.FindBranchByID(r.Context(), branchID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, BranchFromDomain(branch))
}

func (h *Handler) handleCreateConnection(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req CreateConnectionRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	leafIDs := make([]types.LeafID, len(req.LeafIDs))
	for i, raw := range req.LeafIDs {
		id, err := types.ParseLeafID(raw)
		if err != nil {
			types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
			return
		}
		leafIDs[i] = id
	}

	conn, err := h.svc.CreateConnection(r.Context(), wsID, leafIDs)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteCreated(w, ConnectionFromDomain(conn))
}

func (h *Handler) handleListConnections(w http.ResponseWriter, r *http.Request) {
	leafID, err := types.ParseLeafID(httpserver.PathParam(r, "leafId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	conns, err := h.svc.FindConnectionsByLeaf(r.Context(), leafID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, ConnectionsFromDomain(conns))
}

func (h *Handler) handlePromoteLeaf(w http.ResponseWriter, r *http.Request) {
	leafID, err := types.ParseLeafID(httpserver.PathParam(r, "leafId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	if err := h.svc.PromoteLeaf(r.Context(), leafID); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

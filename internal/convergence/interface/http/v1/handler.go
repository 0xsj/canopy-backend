package v1

import (
	"net/http"

	"github.com/0xsj/canopy-backend/internal/convergence/domain"
	"github.com/0xsj/canopy-backend/internal/convergence/service"
	"github.com/0xsj/canopy-backend/pkg/auth"
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
	mux.HandleFunc("POST /api/v1/workspaces/{wsId}/leaves/{leafId}/signals", h.handleRecordSignal)
	mux.HandleFunc("DELETE /api/v1/signals/{signalId}", h.handleRemoveSignal)
	mux.HandleFunc("GET /api/v1/leaves/{leafId}/signals/counts", h.handleGetSignalCounts)
	mux.HandleFunc("GET /api/v1/workspaces/{wsId}/signals/mine", h.handleFindUserSignals)
	mux.HandleFunc("POST /api/v1/workspaces/{wsId}/checkpoints", h.handleCreateCheckpoint)
	mux.HandleFunc("GET /api/v1/workspaces/{wsId}/checkpoints", h.handleListCheckpoints)
	mux.HandleFunc("GET /api/v1/checkpoints/{checkpointId}", h.handleGetCheckpoint)
	mux.HandleFunc("POST /api/v1/checkpoints/{checkpointId}/signals", h.handleRecordConsensusPosition)
	mux.HandleFunc("POST /api/v1/checkpoints/{checkpointId}/resolve", h.handleResolveCheckpoint)
}

func (h *Handler) handleRecordSignal(w http.ResponseWriter, r *http.Request) {
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

	var req RecordSignalRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	signal, err := h.svc.RecordSignal(r.Context(), wsID, leafID, domain.SignalType(req.SignalType), req.Annotation)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteCreated(w, SignalFromDomain(signal))
}

func (h *Handler) handleRemoveSignal(w http.ResponseWriter, r *http.Request) {
	signalID, err := domain.ParseSignalID(httpserver.PathParam(r, "signalId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	if err := h.svc.RemoveSignal(r.Context(), signalID); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleGetSignalCounts(w http.ResponseWriter, r *http.Request) {
	leafID, err := types.ParseLeafID(httpserver.PathParam(r, "leafId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	counts, err := h.svc.GetSignalCounts(r.Context(), leafID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	out := make(map[string]int, len(counts))
	for k, v := range counts {
		out[string(k)] = v
	}

	types.WriteOK(w, SignalCountsResponse{Counts: out})
}

func (h *Handler) handleFindUserSignals(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	claims := auth.MustFromClaims(r.Context())
	userID := types.UserIDFrom(claims.Subject)

	signals, err := h.svc.FindUserSignals(r.Context(), wsID, userID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, SignalsFromDomain(signals))
}

func (h *Handler) handleCreateCheckpoint(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req CreateCheckpointRequest
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

	checkpoint, err := h.svc.CreateCheckpoint(r.Context(), wsID, leafIDs)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteCreated(w, CheckpointFromDomain(checkpoint))
}

func (h *Handler) handleGetCheckpoint(w http.ResponseWriter, r *http.Request) {
	checkpointID, err := types.ParseCheckpointID(httpserver.PathParam(r, "checkpointId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	checkpoint, err := h.svc.GetCheckpoint(r.Context(), checkpointID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, CheckpointFromDomain(checkpoint))
}

func (h *Handler) handleListCheckpoints(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	openOnly := r.URL.Query().Get("status") == "open"

	checkpoints, err := h.svc.FindCheckpointsByWorkspace(r.Context(), wsID, openOnly)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	out := make([]CheckpointResponse, len(checkpoints))
	for i, c := range checkpoints {
		out[i] = CheckpointFromDomain(c)
	}

	types.WriteOK(w, out)
}

func (h *Handler) handleRecordConsensusPosition(w http.ResponseWriter, r *http.Request) {
	checkpointID, err := types.ParseCheckpointID(httpserver.PathParam(r, "checkpointId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req RecordConsensusPositionRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	checkpoint, err := h.svc.RecordConsensusPosition(r.Context(), checkpointID, domain.Position(req.Position), req.Explanation)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, CheckpointFromDomain(checkpoint))
}

func (h *Handler) handleResolveCheckpoint(w http.ResponseWriter, r *http.Request) {
	checkpointID, err := types.ParseCheckpointID(httpserver.PathParam(r, "checkpointId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	if err := h.svc.ResolveCheckpoint(r.Context(), checkpointID); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

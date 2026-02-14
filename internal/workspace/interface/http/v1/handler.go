package v1

import (
	"net/http"

	"github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/internal/workspace/service"
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
	mux.HandleFunc("POST /api/v1/orgs/{orgId}/workspaces", h.handleCreateWorkspace)
	mux.HandleFunc("GET /api/v1/workspaces/{wsId}", h.handleGetWorkspace)
	mux.HandleFunc("GET /api/v1/orgs/{orgId}/workspaces", h.handleListByOrg)
	mux.HandleFunc("POST /api/v1/workspaces/{wsId}/join", h.handleJoin)
	mux.HandleFunc("POST /api/v1/workspaces/{wsId}/leave", h.handleLeave)
	mux.HandleFunc("GET /api/v1/workspaces/{wsId}/members", h.handleListMembers)
	mux.HandleFunc("POST /api/v1/workspaces/{wsId}/members", h.handleAddMember)
	mux.HandleFunc("DELETE /api/v1/workspaces/{wsId}/members/{userId}", h.handleRemoveMember)
	mux.HandleFunc("PATCH /api/v1/workspaces/{wsId}/members/{userId}/role", h.handleUpdateRole)
	mux.HandleFunc("PATCH /api/v1/workspaces/{wsId}/details", h.handleUpdateDetails)
	mux.HandleFunc("PATCH /api/v1/workspaces/{wsId}/configuration", h.handleUpdateConfiguration)
	mux.HandleFunc("PATCH /api/v1/workspaces/{wsId}/config", h.handleUpdateConfig)
	mux.HandleFunc("POST /api/v1/workspaces/{wsId}/phase", h.handleTransitionPhase)
	mux.HandleFunc("PUT /api/v1/workspaces/{wsId}/llm-config", h.handleSetLLMConfig)
	mux.HandleFunc("GET /api/v1/workspaces/{wsId}/llm-config", h.handleGetLLMConfig)
	mux.HandleFunc("DELETE /api/v1/workspaces/{wsId}/llm-config", h.handleDeleteLLMConfig)
}

func (h *Handler) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	orgID, err := types.ParseOrgID(httpserver.PathParam(r, "orgId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req CreateWorkspaceRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	ws, err := h.svc.CreateWorkspace(r.Context(), orgID, req.Name, req.Description)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteCreated(w, WorkspaceFromDomain(ws))
}

func (h *Handler) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	ws, err := h.svc.FindByID(r.Context(), wsID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, WorkspaceFromDomain(ws))
}

func (h *Handler) handleListByOrg(w http.ResponseWriter, r *http.Request) {
	orgID, err := types.ParseOrgID(httpserver.PathParam(r, "orgId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	workspaces, err := h.svc.FindByOrg(r.Context(), orgID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, WorkspacesFromDomain(workspaces))
}

func (h *Handler) handleJoin(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	if err := h.svc.JoinWorkspace(r.Context(), wsID); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleLeave(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	if err := h.svc.LeaveWorkspace(r.Context(), wsID); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleUpdateRole(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	userID, err := types.ParseUserID(httpserver.PathParam(r, "userId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req UpdateRoleRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	if err := h.svc.UpdateRole(r.Context(), wsID, userID, domain.WorkspaceRole(req.Role)); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleListMembers(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	members, err := h.svc.ListMembers(r.Context(), wsID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, MembersFromDomain(members))
}

func (h *Handler) handleAddMember(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req AddMemberRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	userID, err := types.ParseUserID(req.UserID)
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	if err := h.svc.AddMember(r.Context(), wsID, userID); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	userID, err := types.ParseUserID(httpserver.PathParam(r, "userId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	if err := h.svc.RemoveMember(r.Context(), wsID, userID); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleUpdateDetails(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req UpdateDetailsRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	ws, err := h.svc.UpdateDetails(r.Context(), wsID, req.Name, req.Description)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, WorkspaceFromDomain(ws))
}

func (h *Handler) handleUpdateConfiguration(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req UpdateConfigurationRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	ws, err := h.svc.UpdateConfiguration(r.Context(), wsID, req.Configuration)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, WorkspaceFromDomain(ws))
}

func (h *Handler) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req UpdateConfigRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	loreKeeperID, err := types.ParseUserID(req.LoreKeeperID)
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	if err := h.svc.UpdateConfig(r.Context(), wsID, domain.LoreKeeperMode(req.LoreKeeperMode), loreKeeperID); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleTransitionPhase(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req TransitionPhaseRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	if err := h.svc.TransitionPhase(r.Context(), wsID, domain.Phase(req.Target)); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleSetLLMConfig(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req SetLLMConfigRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	if err := h.svc.SetLLMConfig(r.Context(), wsID, req.Provider, req.Model, req.APIKey); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleGetLLMConfig(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	result, err := h.svc.GetLLMConfig(r.Context(), wsID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, LLMConfigFromResult(result))
}

func (h *Handler) handleDeleteLLMConfig(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	if err := h.svc.DeleteLLMConfig(r.Context(), wsID); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

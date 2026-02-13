package v1

import (
	"net/http"

	"github.com/0xsj/canopy-backend/internal/organization/domain"
	"github.com/0xsj/canopy-backend/internal/organization/service"
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
	mux.HandleFunc("POST /api/v1/orgs", h.handleCreateOrg)
	mux.HandleFunc("GET /api/v1/orgs", h.handleLookupOrg)
	mux.HandleFunc("GET /api/v1/orgs/{orgId}", h.handleGetOrg)
	mux.HandleFunc("POST /api/v1/orgs/{orgId}/members", h.handleAddMember)
	mux.HandleFunc("DELETE /api/v1/orgs/{orgId}/members/{userId}", h.handleRemoveMember)
	mux.HandleFunc("PATCH /api/v1/orgs/{orgId}/members/{userId}/role", h.handleUpdateMemberRole)
	mux.HandleFunc("GET /api/v1/orgs/{orgId}/members", h.handleListMembers)
	mux.HandleFunc("POST /api/v1/orgs/{orgId}/teams", h.handleCreateTeam)
	mux.HandleFunc("GET /api/v1/orgs/{orgId}/teams", h.handleListTeams)
	mux.HandleFunc("POST /api/v1/orgs/{orgId}/teams/{teamId}/members", h.handleAddTeamMember)
	mux.HandleFunc("DELETE /api/v1/orgs/{orgId}/teams/{teamId}/members/{userId}", h.handleRemoveTeamMember)
	mux.HandleFunc("GET /api/v1/orgs/{orgId}/teams/{teamId}/members", h.handleListTeamMembers)
}

func (h *Handler) handleCreateOrg(w http.ResponseWriter, r *http.Request) {
	var req CreateOrgRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	org, err := h.svc.CreateOrg(r.Context(), req.Name, req.Slug)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteCreated(w, OrgFromDomain(org))
}

func (h *Handler) handleGetOrg(w http.ResponseWriter, r *http.Request) {
	orgID, err := types.ParseOrgID(httpserver.PathParam(r, "orgId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	org, err := h.svc.FindOrgByID(r.Context(), orgID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, OrgFromDomain(org))
}

func (h *Handler) handleLookupOrg(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("slug")
	if slug == "" {
		types.WriteError(w, http.StatusBadRequest, "missing_param", "slug query parameter is required")
		return
	}

	org, err := h.svc.FindOrgBySlug(r.Context(), slug)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, OrgFromDomain(org))
}

func (h *Handler) handleAddMember(w http.ResponseWriter, r *http.Request) {
	orgID, err := types.ParseOrgID(httpserver.PathParam(r, "orgId"))
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

	if err := h.svc.AddMember(r.Context(), orgID, userID, domain.Role(req.Role)); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	orgID, err := types.ParseOrgID(httpserver.PathParam(r, "orgId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	userID, err := types.ParseUserID(httpserver.PathParam(r, "userId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	if err := h.svc.RemoveMember(r.Context(), orgID, userID); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	orgID, err := types.ParseOrgID(httpserver.PathParam(r, "orgId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	userID, err := types.ParseUserID(httpserver.PathParam(r, "userId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req UpdateMemberRoleRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	if err := h.svc.UpdateMemberRole(r.Context(), orgID, userID, domain.Role(req.Role)); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleListMembers(w http.ResponseWriter, r *http.Request) {
	orgID, err := types.ParseOrgID(httpserver.PathParam(r, "orgId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	members, err := h.svc.ListMembers(r.Context(), orgID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, OrgMembersFromDomain(members))
}

func (h *Handler) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	orgID, err := types.ParseOrgID(httpserver.PathParam(r, "orgId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req CreateTeamRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	team, err := h.svc.CreateTeam(r.Context(), orgID, req.Name)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteCreated(w, TeamFromDomain(team))
}

func (h *Handler) handleListTeams(w http.ResponseWriter, r *http.Request) {
	orgID, err := types.ParseOrgID(httpserver.PathParam(r, "orgId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	teams, err := h.svc.ListTeams(r.Context(), orgID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, TeamsFromDomain(teams))
}

func (h *Handler) handleAddTeamMember(w http.ResponseWriter, r *http.Request) {
	orgID, err := types.ParseOrgID(httpserver.PathParam(r, "orgId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	teamID, err := types.ParseTeamID(httpserver.PathParam(r, "teamId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req AddTeamMemberRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	userID, err := types.ParseUserID(req.UserID)
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	if err := h.svc.AddTeamMember(r.Context(), orgID, teamID, userID); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleRemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	orgID, err := types.ParseOrgID(httpserver.PathParam(r, "orgId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	teamID, err := types.ParseTeamID(httpserver.PathParam(r, "teamId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	userID, err := types.ParseUserID(httpserver.PathParam(r, "userId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	if err := h.svc.RemoveTeamMember(r.Context(), orgID, teamID, userID); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleListTeamMembers(w http.ResponseWriter, r *http.Request) {
	orgID, err := types.ParseOrgID(httpserver.PathParam(r, "orgId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	teamID, err := types.ParseTeamID(httpserver.PathParam(r, "teamId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	members, err := h.svc.ListTeamMembers(r.Context(), orgID, teamID)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, TeamMembersFromDomain(members))
}

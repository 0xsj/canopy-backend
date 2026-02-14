package v1

import (
	"net/http"

	"github.com/0xsj/canopy-backend/internal/identity/service"
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
	mux.HandleFunc("POST /api/v1/auth/register", h.handleRegister)
	mux.HandleFunc("GET /api/v1/users/me", h.handleGetMe)
	mux.HandleFunc("GET /api/v1/users/{userId}", h.handleGetUser)
	mux.HandleFunc("PATCH /api/v1/users/me", h.handleUpdateProfile)
	mux.HandleFunc("PUT /api/v1/users/me/llm-config", h.handleSetLLMConfig)
	mux.HandleFunc("GET /api/v1/users/me/llm-config", h.handleGetLLMConfig)
	mux.HandleFunc("DELETE /api/v1/users/me/llm-config", h.handleDeleteLLMConfig)
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	claims := auth.MustFromClaims(r.Context())

	user, err := h.svc.Register(r.Context(), claims.Subject, req.Email, req.DisplayName)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteCreated(w, UserFromDomain(user))
}

func (h *Handler) handleGetMe(w http.ResponseWriter, r *http.Request) {
	claims := auth.MustFromClaims(r.Context())

	user, err := h.svc.FindByExternalID(r.Context(), claims.Subject)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, UserFromDomain(user))
}

func (h *Handler) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := types.ParseUserID(httpserver.PathParam(r, "userId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	user, err := h.svc.FindByID(r.Context(), id)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, UserFromDomain(user))
}

func (h *Handler) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req UpdateProfileRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	displayName := ""
	if req.DisplayName != nil {
		displayName = *req.DisplayName
	}
	email := ""
	if req.Email != nil {
		email = *req.Email
	}
	avatarURL := ""
	if req.AvatarURL != nil {
		avatarURL = *req.AvatarURL
	}

	user, err := h.svc.UpdateProfile(r.Context(), displayName, email, avatarURL)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, UserFromDomain(user))
}

func (h *Handler) handleSetLLMConfig(w http.ResponseWriter, r *http.Request) {
	var req SetUserLLMConfigRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	if err := h.svc.SetLLMConfig(r.Context(), req.Provider, req.Model, req.APIKey); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleGetLLMConfig(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.GetLLMConfig(r.Context())
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, UserLLMConfigFromResult(result))
}

func (h *Handler) handleDeleteLLMConfig(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteLLMConfig(r.Context()); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

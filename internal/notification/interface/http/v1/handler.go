package v1

import (
	"net/http"

	"github.com/0xsj/canopy-backend/internal/notification/domain"
	"github.com/0xsj/canopy-backend/internal/notification/service"
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
	mux.HandleFunc("GET /api/v1/notifications", h.handleListNotifications)
	mux.HandleFunc("GET /api/v1/notifications/unread", h.handleListUnread)
	mux.HandleFunc("POST /api/v1/notifications/{notificationId}/read", h.handleMarkRead)
	mux.HandleFunc("POST /api/v1/notifications/read-all", h.handleMarkAllRead)
	mux.HandleFunc("PUT /api/v1/workspaces/{wsId}/notifications/subscription", h.handleUpdateSubscription)
}

func (h *Handler) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	limit := httpserver.QueryInt(r, "limit", 20)

	notifications, err := h.svc.FindByUser(r.Context(), limit)
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, NotificationsFromDomain(notifications))
}

func (h *Handler) handleListUnread(w http.ResponseWriter, r *http.Request) {
	notifications, err := h.svc.FindUnread(r.Context())
	if err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, NotificationsFromDomain(notifications))
}

func (h *Handler) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	notificationID, err := domain.ParseNotificationID(httpserver.PathParam(r, "notificationId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	if err := h.svc.MarkRead(r.Context(), notificationID); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleMarkAllRead(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MarkAllRead(r.Context()); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

func (h *Handler) handleUpdateSubscription(w http.ResponseWriter, r *http.Request) {
	wsID, err := types.ParseWorkspaceID(httpserver.PathParam(r, "wsId"))
	if err != nil {
		types.WriteError(w, http.StatusBadRequest, "invalid_id", err.Error())
		return
	}

	var req UpdateSubscriptionRequest
	if !httpserver.DecodeBody(w, r, &req) {
		return
	}

	channels := make([]domain.Channel, len(req.Channels))
	for i, ch := range req.Channels {
		channels[i] = domain.Channel(ch)
	}

	if err := h.svc.UpdateSubscription(r.Context(), wsID, channels, domain.DigestFrequency(req.DigestFrequency)); err != nil {
		httpserver.WriteServiceError(w, h.log, err)
		return
	}

	types.WriteOK(w, struct{}{})
}

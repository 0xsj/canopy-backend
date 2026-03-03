package websocket

import (
	"net/http"

	"github.com/0xsj/canopy-backend/pkg/auth"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Handler returns an http.Handler that upgrades connections and serves them.
// Extracts workspace_id and token from the query string.
// The token is validated before the upgrade — 401 is returned on failure.
//
// For custom routing (e.g., path parameters), write your own handler
// using hub.ServeConn directly:
//
//	mux.HandleFunc("GET /ws/{workspace_id}", func(w http.ResponseWriter, r *http.Request) {
//	    wsID := r.PathValue("workspace_id")
//	    conn, err := upgrader.Upgrade(w, r)
//	    if err != nil { return }
//	    hub.ServeConn(r.Context(), conn, wsID, nil)
//	})
func Handler(hub *Hub, upgrader Upgrader, validator auth.TokenValidator, log logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate auth token before upgrade (standard HTTP response still possible).
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "token query parameter is required", http.StatusUnauthorized)
			return
		}

		claims, err := validator.Validate(r.Context(), token)
		if err != nil {
			log.Warn("websocket: token validation failed", logger.Err(err))
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		workspaceID := r.URL.Query().Get("workspace_id")
		if workspaceID == "" {
			http.Error(w, "workspace_id query parameter is required", http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			log.Error("websocket upgrade failed", logger.Err(err))
			return
		}

		ctx := auth.WithClaims(r.Context(), claims)
		hub.ServeConn(ctx, conn, workspaceID, nil)
	})
}

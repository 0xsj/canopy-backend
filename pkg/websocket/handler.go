package websocket

import (
	"net/http"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Handler returns an http.Handler that upgrades connections and serves them.
// Extracts workspace_id from the query string.
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
func Handler(hub *Hub, upgrader Upgrader, log logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		hub.ServeConn(r.Context(), conn, workspaceID, nil)
	})
}

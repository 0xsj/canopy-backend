package websocket

import (
	"context"
	"net/http"
)

// Conn is the port for a WebSocket connection.
// Implementations wrap a specific WebSocket library (nhooyr, gorilla, etc.).
type Conn interface {
	ReadMessage(ctx context.Context) ([]byte, error)
	WriteMessage(ctx context.Context, data []byte) error
	Close(code int, reason string) error
}

// Upgrader is the port for upgrading HTTP connections to WebSocket.
type Upgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request) (Conn, error)
}

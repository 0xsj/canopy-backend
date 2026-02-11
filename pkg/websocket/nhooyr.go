package websocket

import (
	"context"
	"fmt"
	"net/http"

	ws "github.com/coder/websocket"
)

// nhooyrConn adapts nhooyr.io/websocket to the Conn interface.
type nhooyrConn struct {
	conn *ws.Conn
}

func (c *nhooyrConn) ReadMessage(ctx context.Context) ([]byte, error) {
	_, data, err := c.conn.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("websocket: read: %w", err)
	}
	return data, nil
}

func (c *nhooyrConn) WriteMessage(ctx context.Context, data []byte) error {
	if err := c.conn.Write(ctx, ws.MessageText, data); err != nil {
		return fmt.Errorf("websocket: write: %w", err)
	}
	return nil
}

func (c *nhooyrConn) Close(code int, reason string) error {
	return c.conn.Close(ws.StatusCode(code), reason)
}

// NhooyrUpgrader adapts nhooyr.io/websocket to the Upgrader interface.
type NhooyrUpgrader struct {
	opts *ws.AcceptOptions
}

// NewNhooyrUpgrader creates an Upgrader using nhooyr.io/websocket.
func NewNhooyrUpgrader() Upgrader {
	return &NhooyrUpgrader{
		opts: &ws.AcceptOptions{},
	}
}

func (u *NhooyrUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (Conn, error) {
	c, err := ws.Accept(w, r, u.opts)
	if err != nil {
		return nil, fmt.Errorf("websocket: upgrade: %w", err)
	}
	return &nhooyrConn{conn: c}, nil
}

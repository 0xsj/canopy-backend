package websocket

import (
	"context"
	"sync"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// client is an internal handle for a registered WebSocket connection.
type client struct {
	conn        Conn
	workspaceID string
	send        chan []byte
}

// Hub manages WebSocket connections organized by workspace rooms.
// It handles registration, broadcasting, and connection lifecycle.
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*client]struct{}
	log   logger.Logger
	cfg   Config
}

// NewHub creates an empty hub ready to accept connections.
func NewHub(cfg Config, log logger.Logger) *Hub {
	return &Hub{
		rooms: make(map[string]map[*client]struct{}),
		log:   log,
		cfg:   cfg,
	}
}

// register adds a connection to a workspace room.
func (h *Hub) register(conn Conn, workspaceID string) *client {
	c := &client{
		conn:        conn,
		workspaceID: workspaceID,
		send:        make(chan []byte, h.cfg.SendBufferSize),
	}

	h.mu.Lock()
	if h.rooms[workspaceID] == nil {
		h.rooms[workspaceID] = make(map[*client]struct{})
	}
	h.rooms[workspaceID][c] = struct{}{}
	h.mu.Unlock()

	h.log.Debug("client registered",
		logger.String("workspace", workspaceID),
		logger.Int("room_size", h.RoomSize(workspaceID)),
	)

	return c
}

// unregister removes a connection from its workspace room.
func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	if room, ok := h.rooms[c.workspaceID]; ok {
		delete(room, c)
		if len(room) == 0 {
			delete(h.rooms, c.workspaceID)
		}
	}
	close(c.send)
	h.mu.Unlock()

	h.log.Debug("client unregistered",
		logger.String("workspace", c.workspaceID),
	)
}

// Broadcast sends raw data to all clients in a workspace room.
func (h *Hub) Broadcast(workspaceID string, data []byte) {
	h.mu.RLock()
	room := h.rooms[workspaceID]
	h.mu.RUnlock()

	for c := range room {
		select {
		case c.send <- data:
		default:
			h.log.Warn("client send buffer full, dropping message",
				logger.String("workspace", workspaceID),
			)
		}
	}
}

// BroadcastMessage encodes a Message and broadcasts it to a workspace room.
func (h *Hub) BroadcastMessage(workspaceID string, msg Message) error {
	data, err := msg.Encode()
	if err != nil {
		return err
	}
	h.Broadcast(workspaceID, data)
	return nil
}

// ServeConn registers a connection, runs the read/write pumps,
// and cleans up when done. Blocks until the connection closes.
//
// The onMessage callback is invoked for each valid message received
// from the client. Pass nil to ignore incoming messages.
func (h *Hub) ServeConn(ctx context.Context, conn Conn, workspaceID string, onMessage func(Message)) {
	c := h.register(conn, workspaceID)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Write pump — sends queued messages to the client.
	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		for {
			select {
			case data, ok := <-c.send:
				if !ok {
					return
				}
				if err := c.conn.WriteMessage(ctx, data); err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Read pump — reads messages from the client.
	for {
		data, err := c.conn.ReadMessage(ctx)
		if err != nil {
			break
		}

		msg, err := DecodeMessage(data)
		if err != nil {
			h.log.Warn("invalid message from client", logger.Err(err))
			continue
		}

		if onMessage != nil {
			onMessage(msg)
		}
	}

	cancel()         // signal write pump to stop
	<-writeDone      // wait for write pump
	h.unregister(c)  // remove from room, close send channel
	conn.Close(1000, "")
}

// RoomSize returns the number of connections in a workspace room.
func (h *Hub) RoomSize(workspaceID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[workspaceID])
}

// TotalConnections returns the total number of active connections.
func (h *Hub) TotalConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for _, room := range h.rooms {
		total += len(room)
	}
	return total
}

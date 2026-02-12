package websocket

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// mockConn implements Conn for testing.
type mockConn struct {
	mu      sync.Mutex
	inbox   chan []byte // messages to be read
	outbox  [][]byte    // messages written
	closed  bool
	closeCh chan struct{}
}

func newMockConn() *mockConn {
	return &mockConn{
		inbox:   make(chan []byte, 16),
		closeCh: make(chan struct{}),
	}
}

func (c *mockConn) ReadMessage(ctx context.Context) ([]byte, error) {
	select {
	case data := <-c.inbox:
		return data, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.closeCh:
		return nil, context.Canceled
	}
}

func (c *mockConn) WriteMessage(ctx context.Context, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.outbox = append(c.outbox, data)
	return nil
}

func (c *mockConn) Close(code int, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.closeCh)
	}
	return nil
}

func (c *mockConn) written() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([][]byte, len(c.outbox))
	copy(out, c.outbox)
	return out
}

func testHub(t *testing.T) *Hub {
	t.Helper()
	return NewHub(Config{SendBufferSize: 16}, logger.NewNoop())
}

func TestHub_ServeConn_RegistersAndUnregisters(t *testing.T) {
	hub := testHub(t)
	conn := newMockConn()

	done := make(chan struct{})
	go func() {
		hub.ServeConn(context.Background(), conn, "ws_1", nil)
		close(done)
	}()

	// Wait for registration.
	time.Sleep(50 * time.Millisecond)
	if hub.RoomSize("ws_1") != 1 {
		t.Errorf("RoomSize = %d, want 1", hub.RoomSize("ws_1"))
	}
	if hub.TotalConnections() != 1 {
		t.Errorf("TotalConnections = %d, want 1", hub.TotalConnections())
	}

	// Close triggers unregister.
	conn.Close(1000, "")
	<-done

	if hub.RoomSize("ws_1") != 0 {
		t.Errorf("RoomSize after close = %d, want 0", hub.RoomSize("ws_1"))
	}
}

func TestHub_Broadcast_DeliversToRoom(t *testing.T) {
	hub := testHub(t)
	conn1 := newMockConn()
	conn2 := newMockConn()
	connOther := newMockConn()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Two clients in ws_1.
	for _, c := range []*mockConn{conn1, conn2} {
		wg.Add(1)
		go func(c *mockConn) {
			defer wg.Done()
			hub.ServeConn(ctx, c, "ws_1", nil)
		}(c)
	}

	// One client in ws_2.
	wg.Add(1)
	go func() {
		defer wg.Done()
		hub.ServeConn(ctx, connOther, "ws_2", nil)
	}()

	time.Sleep(50 * time.Millisecond)

	// Broadcast to ws_1 only.
	hub.Broadcast("ws_1", []byte(`{"type":"test"}`))
	time.Sleep(50 * time.Millisecond)

	cancel()
	wg.Wait()

	if len(conn1.written()) != 1 {
		t.Errorf("conn1 received %d messages, want 1", len(conn1.written()))
	}
	if len(conn2.written()) != 1 {
		t.Errorf("conn2 received %d messages, want 1", len(conn2.written()))
	}
	if len(connOther.written()) != 0 {
		t.Errorf("connOther received %d messages, want 0 (different room)", len(connOther.written()))
	}
}

func TestHub_BroadcastMessage(t *testing.T) {
	hub := testHub(t)
	conn := newMockConn()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		hub.ServeConn(ctx, conn, "ws_1", nil)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	msg, _ := NewMessage("leaf_created", "ws_1", map[string]string{"title": "test"})
	if err := hub.BroadcastMessage("ws_1", msg); err != nil {
		t.Fatalf("BroadcastMessage() error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	written := conn.written()
	if len(written) != 1 {
		t.Fatalf("received %d messages, want 1", len(written))
	}

	decoded, err := DecodeMessage(written[0])
	if err != nil {
		t.Fatalf("DecodeMessage() error: %v", err)
	}
	if decoded.Type != "leaf_created" {
		t.Errorf("Type = %q, want leaf_created", decoded.Type)
	}
}

func TestHub_ServeConn_OnMessage(t *testing.T) {
	hub := testHub(t)
	conn := newMockConn()

	var received []Message
	var mu sync.Mutex

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		hub.ServeConn(ctx, conn, "ws_1", func(msg Message) {
			mu.Lock()
			received = append(received, msg)
			mu.Unlock()
		})
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	// Simulate client sending a message.
	msg, _ := NewMessage("ping", "ws_1", nil)
	data, _ := msg.Encode()
	conn.inbox <- data

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("received %d messages, want 1", len(received))
	}
	if received[0].Type != "ping" {
		t.Errorf("Type = %q, want ping", received[0].Type)
	}
}

func TestHub_MultipleRooms(t *testing.T) {
	hub := testHub(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	for _, wsID := range []string{"ws_a", "ws_b", "ws_c"} {
		conn := newMockConn()
		wg.Add(1)
		go func(c *mockConn, id string) {
			defer wg.Done()
			hub.ServeConn(ctx, c, id, nil)
		}(conn, wsID)
	}

	time.Sleep(50 * time.Millisecond)

	if hub.TotalConnections() != 3 {
		t.Errorf("TotalConnections = %d, want 3", hub.TotalConnections())
	}

	cancel()
	wg.Wait()

	if hub.TotalConnections() != 0 {
		t.Errorf("TotalConnections after shutdown = %d, want 0", hub.TotalConnections())
	}
}

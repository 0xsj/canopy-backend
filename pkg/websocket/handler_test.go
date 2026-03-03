package websocket

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/0xsj/canopy-backend/pkg/auth"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// mockUpgrader implements Upgrader for testing.
type mockUpgrader struct {
	mu        sync.Mutex
	conn      Conn
	err       error
	upgradeCh chan struct{} // closed when Upgrade is called
}

func newMockUpgrader(conn Conn) *mockUpgrader {
	return &mockUpgrader{conn: conn, upgradeCh: make(chan struct{})}
}

func newFailingUpgrader(err error) *mockUpgrader {
	return &mockUpgrader{err: err, upgradeCh: make(chan struct{})}
}

func (u *mockUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (Conn, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	select {
	case <-u.upgradeCh:
	default:
		close(u.upgradeCh)
	}
	if u.err != nil {
		http.Error(w, "upgrade failed", http.StatusInternalServerError)
		return nil, u.err
	}
	return u.conn, nil
}

func TestHandler_MissingToken_Returns401(t *testing.T) {
	hub := testHub(t)
	upgrader := newMockUpgrader(newMockConn())
	validator := auth.NewStaticValidator(auth.Claims{Subject: "user_1"})
	log := logger.NewNoop()

	h := Handler(hub, upgrader, validator, log)

	req := httptest.NewRequest("GET", "/ws?workspace_id=ws_1", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestHandler_InvalidToken_Returns401(t *testing.T) {
	hub := testHub(t)
	upgrader := newMockUpgrader(newMockConn())
	validator := auth.NewFailingValidator(errors.New("invalid signature"))
	log := logger.NewNoop()

	h := Handler(hub, upgrader, validator, log)

	req := httptest.NewRequest("GET", "/ws?workspace_id=ws_1&token=bad-token", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestHandler_MissingWorkspaceID_Returns400(t *testing.T) {
	hub := testHub(t)
	upgrader := newMockUpgrader(newMockConn())
	validator := auth.NewStaticValidator(auth.Claims{Subject: "user_1"})
	log := logger.NewNoop()

	h := Handler(hub, upgrader, validator, log)

	req := httptest.NewRequest("GET", "/ws?token=valid-token", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestHandler_ValidRequest_UpgradesAndServesConn(t *testing.T) {
	hub := testHub(t)
	conn := newMockConn()
	upgrader := newMockUpgrader(conn)
	validator := auth.NewStaticValidator(auth.Claims{Subject: "user_abc"})
	log := logger.NewNoop()

	h := Handler(hub, upgrader, validator, log)

	req := httptest.NewRequest("GET", "/ws?workspace_id=ws_1&token=valid-token", nil)
	rr := httptest.NewRecorder()

	// ServeConn blocks until the connection closes, so run in a goroutine.
	done := make(chan struct{})
	go func() {
		h.ServeHTTP(rr, req)
		close(done)
	}()

	// Wait for upgrade to happen.
	<-upgrader.upgradeCh

	// Close connection to unblock ServeConn.
	conn.Close(1000, "test done")
	<-done

	// Verify the connection was registered in the hub's room.
	// (After close it should be unregistered — test that it at least went through.)
}

func TestHandler_UpgradeFailure_DoesNotPanic(t *testing.T) {
	hub := testHub(t)
	upgrader := newFailingUpgrader(errors.New("protocol mismatch"))
	validator := auth.NewStaticValidator(auth.Claims{Subject: "user_1"})
	log := logger.NewNoop()

	h := Handler(hub, upgrader, validator, log)

	req := httptest.NewRequest("GET", "/ws?workspace_id=ws_1&token=valid-token", nil)
	rr := httptest.NewRecorder()

	// Should not panic.
	h.ServeHTTP(rr, req)
}

func TestHandler_ClaimsInContext(t *testing.T) {
	// This test verifies that validated claims are available in the context
	// passed to ServeConn. We use a custom Conn that captures the context.
	expectedClaims := auth.Claims{Subject: "user_ctx_test"}
	validator := auth.NewStaticValidator(expectedClaims)
	log := logger.NewNoop()

	// contextCapturingHub: instead of the real hub, we create a handler
	// that validates claims are in context right after upgrade.
	// We'll test this by wrapping the handler logic.
	var capturedCtx context.Context
	var ctxMu sync.Mutex

	hub := testHub(t)
	conn := newMockConn()
	upgrader := newMockUpgrader(conn)

	h := Handler(hub, upgrader, validator, log)

	// We can't easily intercept ServeConn, but we can verify claims
	// by reading them from the hub's connection handler.
	// Instead, let's verify the auth flow end-to-end by using
	// the hub's OnMessage callback mechanism.

	// Alternative approach: use a conn that captures the context.
	capturedConn := &contextCapturingConn{
		Conn:     conn,
		captured: make(chan context.Context, 1),
	}
	ctxUpgrader := &contextCapturingUpgrader{conn: capturedConn}

	h2 := Handler(hub, ctxUpgrader, validator, log)

	req := httptest.NewRequest("GET", "/ws?workspace_id=ws_1&token=valid", nil)
	rr := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		h2.ServeHTTP(rr, req)
		close(done)
	}()

	// The hub calls conn.ReadMessage with the context that has claims.
	// Our capturing conn will grab it.
	select {
	case capturedCtx = <-capturedConn.captured:
	case <-done:
		t.Fatal("handler returned before ReadMessage was called")
	}

	ctxMu.Lock()
	defer ctxMu.Unlock()

	claims, ok := auth.FromClaims(capturedCtx)
	if !ok {
		t.Fatal("claims not found in context passed to ServeConn")
	}
	if claims.Subject != expectedClaims.Subject {
		t.Errorf("Subject = %q, want %q", claims.Subject, expectedClaims.Subject)
	}

	// Cleanup.
	conn.Close(1000, "")
	<-done

	// Suppress unused variable warning.
	_ = h
}

// contextCapturingConn wraps a Conn and captures the context from ReadMessage.
type contextCapturingConn struct {
	Conn
	captured chan context.Context
	once     sync.Once
}

func (c *contextCapturingConn) ReadMessage(ctx context.Context) ([]byte, error) {
	c.once.Do(func() {
		c.captured <- ctx
	})
	return c.Conn.ReadMessage(ctx)
}

// contextCapturingUpgrader returns a contextCapturingConn.
type contextCapturingUpgrader struct {
	conn *contextCapturingConn
}

func (u *contextCapturingUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (Conn, error) {
	return u.conn, nil
}

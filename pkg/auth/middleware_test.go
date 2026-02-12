package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

func TestMiddleware_ValidToken(t *testing.T) {
	claims := Claims{
		Subject:   "user_abc123",
		Email:     "test@canopy.dev",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	v := NewStaticValidator(claims)
	log := logger.NewNoop()

	var gotClaims Claims
	handler := Middleware(v, log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := FromClaims(r.Context())
		if !ok {
			t.Error("no claims in context")
		}
		gotClaims = c
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if gotClaims.Subject != "user_abc123" {
		t.Errorf("Subject = %q, want user_abc123", gotClaims.Subject)
	}
}

func TestMiddleware_MissingAuthHeader(t *testing.T) {
	v := NewStaticValidator(Claims{Subject: "user_1"})
	log := logger.NewNoop()

	handler := Middleware(v, log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without auth header")
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/protected", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestMiddleware_EmptyBearer(t *testing.T) {
	v := NewStaticValidator(Claims{Subject: "user_1"})
	log := logger.NewNoop()

	handler := Middleware(v, log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with empty bearer")
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestMiddleware_WrongScheme(t *testing.T) {
	v := NewStaticValidator(Claims{Subject: "user_1"})
	log := logger.NewNoop()

	handler := Middleware(v, log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with wrong scheme")
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestMiddleware_InvalidToken(t *testing.T) {
	v := NewFailingValidator(errors.New("invalid signature"))
	log := logger.NewNoop()

	handler := Middleware(v, log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid token")
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}

	// Should return JSON error body.
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "invalid token" {
		t.Errorf("error = %q, want %q", body["error"], "invalid token")
	}
}

func TestMiddleware_DoesNotLeakErrorDetails(t *testing.T) {
	v := NewFailingValidator(errors.New("RSA key mismatch at offset 42"))
	log := logger.NewNoop()

	handler := Middleware(v, log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Response should say "invalid token", not the internal error.
	var body map[string]string
	json.NewDecoder(rr.Body).Decode(&body)
	if body["error"] != "invalid token" {
		t.Errorf("error = %q, should not contain internal details", body["error"])
	}
}

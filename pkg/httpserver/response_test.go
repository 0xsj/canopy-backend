package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON_WritesResponse(t *testing.T) {
	rr := httptest.NewRecorder()

	data := map[string]string{"message": "hello"}
	JSON(rr, http.StatusOK, data)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var result map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result["message"] != "hello" {
		t.Errorf("message = %q, want hello", result["message"])
	}
}

func TestJSON_CustomStatus(t *testing.T) {
	rr := httptest.NewRecorder()
	JSON(rr, http.StatusCreated, map[string]int{"id": 42})

	if rr.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rr.Code)
	}
}

func TestError_WritesErrorResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	Error(rr, http.StatusNotFound, "item not found")

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}

	var result map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result["error"] != "item not found" {
		t.Errorf("error = %q, want 'item not found'", result["error"])
	}
}

package types

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestOK_SetsData(t *testing.T) {
	resp := OK("hello")
	if resp.Data == nil {
		t.Fatal("OK response Data should not be nil")
	}
	if *resp.Data != "hello" {
		t.Errorf("Data = %q, want %q", *resp.Data, "hello")
	}
	if resp.Error != nil {
		t.Error("OK response Error should be nil")
	}
}

func TestFail_SetsError(t *testing.T) {
	resp := Fail[string]("not_found", "leaf not found")
	if resp.Error == nil {
		t.Fatal("Fail response Error should not be nil")
	}
	if resp.Error.Code != "not_found" {
		t.Errorf("Code = %q, want %q", resp.Error.Code, "not_found")
	}
	if resp.Error.Message != "leaf not found" {
		t.Errorf("Message = %q, want %q", resp.Error.Message, "leaf not found")
	}
	if resp.Data != nil {
		t.Error("Fail response Data should be nil")
	}
}

func TestFailWithDetails_SetsFieldErrors(t *testing.T) {
	details := []FieldError{
		{Field: "title", Message: "is required"},
		{Field: "summary", Message: "is required"},
	}

	resp := FailWithDetails[string]("validation_failed", "invalid", details)

	if resp.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if len(resp.Error.Details) != 2 {
		t.Fatalf("Details count = %d, want 2", len(resp.Error.Details))
	}
	if resp.Error.Details[0].Field != "title" {
		t.Errorf("Details[0].Field = %q, want %q", resp.Error.Details[0].Field, "title")
	}
}

func TestIsError(t *testing.T) {
	ok := OK("data")
	if ok.IsError() {
		t.Error("OK response should not be error")
	}

	fail := Fail[string]("err", "msg")
	if !fail.IsError() {
		t.Error("Fail response should be error")
	}
}

func TestValidate_OK(t *testing.T) {
	resp := OK("data")
	if err := resp.Validate(); err != nil {
		t.Errorf("valid OK response: Validate() = %v", err)
	}
}

func TestValidate_Fail(t *testing.T) {
	resp := Fail[string]("err", "msg")
	if err := resp.Validate(); err != nil {
		t.Errorf("valid Fail response: Validate() = %v", err)
	}
}

func TestValidate_BothSet(t *testing.T) {
	data := "hello"
	resp := Response[string]{
		Data:  &data,
		Error: &APIError{Code: "err", Message: "msg"},
	}
	if err := resp.Validate(); err == nil {
		t.Error("both set: Validate() should return error")
	}
}

func TestValidate_NeitherSet(t *testing.T) {
	resp := Response[string]{}
	if err := resp.Validate(); err == nil {
		t.Error("neither set: Validate() should return error")
	}
}

func TestOK_JSON(t *testing.T) {
	resp := OK(map[string]int{"count": 42})
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var raw map[string]any
	json.Unmarshal(data, &raw)

	if _, ok := raw["data"]; !ok {
		t.Error("JSON should contain 'data' key")
	}
	if _, ok := raw["error"]; ok {
		t.Error("JSON should not contain 'error' key for OK response")
	}
}

func TestFail_JSON(t *testing.T) {
	resp := Fail[string]("not_found", "leaf not found")
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var raw map[string]any
	json.Unmarshal(data, &raw)

	if _, ok := raw["error"]; !ok {
		t.Error("JSON should contain 'error' key")
	}
	if _, ok := raw["data"]; ok {
		t.Error("JSON should not contain 'data' key for Fail response")
	}
}

func TestFailWithDetails_JSON(t *testing.T) {
	details := []FieldError{
		{Field: "title", Message: "is required"},
	}
	resp := FailWithDetails[string]("validation_failed", "invalid", details)

	data, _ := json.Marshal(resp)
	var raw map[string]any
	json.Unmarshal(data, &raw)

	errObj := raw["error"].(map[string]any)
	if errObj["details"] == nil {
		t.Error("validation error JSON should contain 'details'")
	}
}

func TestWriteOK(t *testing.T) {
	w := httptest.NewRecorder()
	WriteOK(w, map[string]string{"status": "ok"})

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var resp Response[map[string]string]
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data == nil {
		t.Error("response data should not be nil")
	}
}

func TestWriteCreated(t *testing.T) {
	w := httptest.NewRecorder()
	WriteCreated(w, "created")

	if w.Code != 201 {
		t.Errorf("status = %d, want 201", w.Code)
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteError(w, 404, "not_found", "leaf not found")

	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}

	var resp Response[struct{}]
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error == nil {
		t.Error("response error should not be nil")
	}
	if resp.Error.Code != "not_found" {
		t.Errorf("error code = %q, want not_found", resp.Error.Code)
	}
}

func TestWriteValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	WriteValidationError(w, []FieldError{
		{Field: "title", Message: "is required"},
	})

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}

	var resp Response[struct{}]
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error == nil {
		t.Fatal("response error should not be nil")
	}
	if len(resp.Error.Details) != 1 {
		t.Errorf("details count = %d, want 1", len(resp.Error.Details))
	}
}

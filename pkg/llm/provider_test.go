package llm

import (
	"context"
	"testing"
)

// mockProvider is a test double for the Provider interface.
type mockProvider struct {
	response  ChatResponse
	err       error
	lastReq   ChatRequest
	callCount int
}

func (m *mockProvider) ChatCompletion(_ context.Context, req ChatRequest) (ChatResponse, error) {
	m.callCount++
	m.lastReq = req
	return m.response, m.err
}

func TestMockProvider_ImplementsInterface(t *testing.T) {
	var _ Provider = (*mockProvider)(nil)
}

func TestMockProvider_ReturnsCannedResponse(t *testing.T) {
	mock := &mockProvider{
		response: ChatResponse{
			Content: "Hello from the mock",
			Model:   "test-model",
			Usage:   Usage{InputTokens: 10, OutputTokens: 5},
		},
	}

	resp, err := mock.ChatCompletion(context.Background(), ChatRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "Hi"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}
	if resp.Content != "Hello from the mock" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello from the mock")
	}
	if resp.Model != "test-model" {
		t.Errorf("Model = %q, want %q", resp.Model, "test-model")
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("InputTokens = %d, want 10", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 5 {
		t.Errorf("OutputTokens = %d, want 5", resp.Usage.OutputTokens)
	}
	if mock.callCount != 1 {
		t.Errorf("callCount = %d, want 1", mock.callCount)
	}
}

func TestMockProvider_CapturesRequest(t *testing.T) {
	mock := &mockProvider{
		response: ChatResponse{Content: "ok"},
	}

	req := ChatRequest{
		Model: "claude-sonnet-4-5-20250929",
		Messages: []Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "What is Go?"},
		},
		Options: Options{
			Temperature: 0.5,
			MaxTokens:   1024,
			TopP:        0.9,
			Stop:        []string{"\n\n"},
		},
	}

	_, err := mock.ChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}

	if mock.lastReq.Model != "claude-sonnet-4-5-20250929" {
		t.Errorf("captured Model = %q, want claude-sonnet-4-5-20250929", mock.lastReq.Model)
	}
	if len(mock.lastReq.Messages) != 2 {
		t.Fatalf("captured Messages len = %d, want 2", len(mock.lastReq.Messages))
	}
	if mock.lastReq.Messages[0].Role != "system" {
		t.Errorf("Messages[0].Role = %q, want system", mock.lastReq.Messages[0].Role)
	}
	if mock.lastReq.Messages[1].Role != "user" {
		t.Errorf("Messages[1].Role = %q, want user", mock.lastReq.Messages[1].Role)
	}
	if mock.lastReq.Options.Temperature != 0.5 {
		t.Errorf("Options.Temperature = %v, want 0.5", mock.lastReq.Options.Temperature)
	}
	if mock.lastReq.Options.MaxTokens != 1024 {
		t.Errorf("Options.MaxTokens = %d, want 1024", mock.lastReq.Options.MaxTokens)
	}
}

func TestMockProvider_ReturnsError(t *testing.T) {
	mock := &mockProvider{
		err: context.DeadlineExceeded,
	}

	_, err := mock.ChatCompletion(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "Hi"}},
	})
	if err != context.DeadlineExceeded {
		t.Errorf("error = %v, want context.DeadlineExceeded", err)
	}
}

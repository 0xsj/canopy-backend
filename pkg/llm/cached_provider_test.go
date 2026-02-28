package llm

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeProvider counts calls and returns a fixed response.
type fakeProvider struct {
	calls atomic.Int64
	resp  ChatResponse
}

func (f *fakeProvider) ChatCompletion(_ context.Context, _ ChatRequest) (ChatResponse, error) {
	f.calls.Add(1)
	// Simulate LLM latency.
	time.Sleep(10 * time.Millisecond)
	return f.resp, nil
}

func deterministicReq() ChatRequest {
	return ChatRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "ping"}},
		Options:  Options{Temperature: 0, MaxTokens: 100},
	}
}

func nonDeterministicReq() ChatRequest {
	return ChatRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "ping"}},
		Options:  Options{Temperature: 0.7, MaxTokens: 100},
	}
}

func TestCachedProvider_CachesHit(t *testing.T) {
	inner := &fakeProvider{resp: ChatResponse{Content: "pong", Model: "m"}}
	cache := NewMemoryCache(100)
	p := NewCachedProvider(inner, cache, time.Minute)

	req := deterministicReq()

	// First call: miss, should go to inner.
	resp1, err := p.ChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp1.Content != "pong" {
		t.Fatalf("got %q, want pong", resp1.Content)
	}
	if inner.calls.Load() != 1 {
		t.Fatalf("expected 1 inner call, got %d", inner.calls.Load())
	}

	// Second call: cache hit, inner should NOT be called again.
	resp2, err := p.ChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.Content != "pong" {
		t.Fatalf("got %q, want pong", resp2.Content)
	}
	if inner.calls.Load() != 1 {
		t.Fatalf("expected still 1 inner call, got %d", inner.calls.Load())
	}
}

func TestCachedProvider_BypassesForNonDeterministic(t *testing.T) {
	inner := &fakeProvider{resp: ChatResponse{Content: "random"}}
	cache := NewMemoryCache(100)
	p := NewCachedProvider(inner, cache, time.Minute)

	req := nonDeterministicReq()

	// Both calls should hit inner since temperature > 0.
	p.ChatCompletion(context.Background(), req)
	p.ChatCompletion(context.Background(), req)

	if inner.calls.Load() != 2 {
		t.Fatalf("expected 2 inner calls for non-deterministic, got %d", inner.calls.Load())
	}
}

func TestCachedProvider_Singleflight(t *testing.T) {
	inner := &fakeProvider{resp: ChatResponse{Content: "deduped"}}
	cache := NewMemoryCache(100)
	p := NewCachedProvider(inner, cache, time.Minute)

	req := deterministicReq()
	n := 10

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			resp, err := p.ChatCompletion(context.Background(), req)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if resp.Content != "deduped" {
				t.Errorf("got %q, want deduped", resp.Content)
			}
		}()
	}
	wg.Wait()

	// Singleflight: only 1 call should have been made to inner.
	calls := inner.calls.Load()
	if calls != 1 {
		t.Fatalf("singleflight failed: expected 1 inner call, got %d", calls)
	}
}

func TestCachedProvider_DifferentRequestsNotDeduplicated(t *testing.T) {
	inner := &fakeProvider{resp: ChatResponse{Content: "ok"}}
	cache := NewMemoryCache(100)
	p := NewCachedProvider(inner, cache, time.Minute)

	reqA := ChatRequest{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "hello"}},
		Options:  Options{Temperature: 0},
	}
	reqB := ChatRequest{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "world"}},
		Options:  Options{Temperature: 0},
	}

	p.ChatCompletion(context.Background(), reqA)
	p.ChatCompletion(context.Background(), reqB)

	if inner.calls.Load() != 2 {
		t.Fatalf("expected 2 inner calls for different requests, got %d", inner.calls.Load())
	}
}

func TestCachedProvider_ImplementsProvider(t *testing.T) {
	var _ Provider = (*CachedProvider)(nil)
}

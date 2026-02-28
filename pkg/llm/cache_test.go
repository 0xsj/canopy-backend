package llm

import (
	"testing"
	"time"
)

func TestCacheKey_DeterministicForSameRequest(t *testing.T) {
	req := ChatRequest{
		Model: "claude-sonnet-4-20250514",
		Messages: []Message{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hello"},
		},
		Options: Options{Temperature: 0, MaxTokens: 1024},
	}

	k1 := CacheKey(req)
	k2 := CacheKey(req)
	if k1 != k2 {
		t.Fatalf("same request produced different keys: %s vs %s", k1, k2)
	}
}

func TestCacheKey_DiffersOnModel(t *testing.T) {
	base := ChatRequest{
		Model:    "model-a",
		Messages: []Message{{Role: "user", Content: "hi"}},
	}
	other := base
	other.Model = "model-b"

	if CacheKey(base) == CacheKey(other) {
		t.Fatal("different models should produce different keys")
	}
}

func TestCacheKey_DiffersOnMessages(t *testing.T) {
	a := ChatRequest{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "hello"}},
	}
	b := ChatRequest{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "world"}},
	}
	if CacheKey(a) == CacheKey(b) {
		t.Fatal("different messages should produce different keys")
	}
}

func TestCacheKey_DiffersOnOptions(t *testing.T) {
	base := ChatRequest{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "hi"}},
		Options:  Options{MaxTokens: 100},
	}
	other := base
	other.Options.MaxTokens = 200

	if CacheKey(base) == CacheKey(other) {
		t.Fatal("different options should produce different keys")
	}
}

func TestMemoryCache_GetMiss(t *testing.T) {
	c := NewMemoryCache(100)
	_, ok := c.Get("nonexistent")
	if ok {
		t.Fatal("expected miss for nonexistent key")
	}
}

func TestMemoryCache_SetAndGet(t *testing.T) {
	c := NewMemoryCache(100)
	resp := ChatResponse{Content: "cached", Model: "m"}
	c.Set("k1", resp, time.Minute)

	got, ok := c.Get("k1")
	if !ok {
		t.Fatal("expected hit")
	}
	if got.Content != "cached" {
		t.Fatalf("got content %q, want %q", got.Content, "cached")
	}
}

func TestMemoryCache_TTLExpiry(t *testing.T) {
	c := NewMemoryCache(100)
	c.Set("k1", ChatResponse{Content: "old"}, time.Millisecond)

	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("k1")
	if ok {
		t.Fatal("expected miss after TTL expiry")
	}
}

func TestMemoryCache_EvictsOldestAtCapacity(t *testing.T) {
	c := NewMemoryCache(2)
	c.Set("k1", ChatResponse{Content: "first"}, time.Hour)

	// Nudge k1 expiry to be earliest.
	c.mu.Lock()
	e := c.entries["k1"]
	e.expiresAt = time.Now().Add(time.Minute)
	c.entries["k1"] = e
	c.mu.Unlock()

	c.Set("k2", ChatResponse{Content: "second"}, time.Hour)
	c.Set("k3", ChatResponse{Content: "third"}, time.Hour)

	// k1 should have been evicted (earliest expiry).
	if _, ok := c.Get("k1"); ok {
		t.Fatal("k1 should have been evicted")
	}
	if _, ok := c.Get("k3"); !ok {
		t.Fatal("k3 should still be present")
	}
}

func TestMemoryCache_DefaultMaxSize(t *testing.T) {
	c := NewMemoryCache(0)
	if c.maxSize != 1000 {
		t.Fatalf("expected default maxSize 1000, got %d", c.maxSize)
	}
}

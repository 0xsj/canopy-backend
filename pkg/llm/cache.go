package llm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ResponseCache is a port for caching LLM responses. Implementations may use
// in-memory storage, Redis, or any other backend.
type ResponseCache interface {
	Get(key string) (ChatResponse, bool)
	Set(key string, resp ChatResponse, ttl time.Duration)
}

// CacheKey builds a deterministic cache key from a ChatRequest.
// The key is a SHA-256 hash of model + options + message contents so that
// identical prompts produce the same key regardless of caller.
func CacheKey(req ChatRequest) string {
	h := sha256.New()
	fmt.Fprintf(h, "model:%s\n", req.Model)
	fmt.Fprintf(h, "temp:%.4f\n", req.Options.Temperature)
	fmt.Fprintf(h, "top_p:%.4f\n", req.Options.TopP)
	fmt.Fprintf(h, "max_tokens:%d\n", req.Options.MaxTokens)

	if len(req.Options.Stop) > 0 {
		sorted := make([]string, len(req.Options.Stop))
		copy(sorted, req.Options.Stop)
		sort.Strings(sorted)
		fmt.Fprintf(h, "stop:%s\n", strings.Join(sorted, ","))
	}

	for _, msg := range req.Messages {
		fmt.Fprintf(h, "%s:%s\n", msg.Role, msg.Content)
	}

	return hex.EncodeToString(h.Sum(nil))
}

// MemoryCache is a simple in-memory ResponseCache with TTL-based expiry.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	maxSize int
}

type cacheEntry struct {
	resp      ChatResponse
	expiresAt time.Time
}

// NewMemoryCache creates an in-memory cache. maxSize limits the number of
// entries; when exceeded the oldest entries are evicted on the next Set.
func NewMemoryCache(maxSize int) *MemoryCache {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &MemoryCache{
		entries: make(map[string]cacheEntry),
		maxSize: maxSize,
	}
}

func (c *MemoryCache) Get(key string) (ChatResponse, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return ChatResponse{}, false
	}
	if time.Now().After(entry.expiresAt) {
		// Expired — remove lazily.
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return ChatResponse{}, false
	}
	return entry.resp, true
}

func (c *MemoryCache) Set(key string, resp ChatResponse, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict expired entries if at capacity.
	if len(c.entries) >= c.maxSize {
		now := time.Now()
		for k, e := range c.entries {
			if now.After(e.expiresAt) {
				delete(c.entries, k)
			}
		}
	}

	// If still at capacity after expired eviction, drop oldest entry.
	if len(c.entries) >= c.maxSize {
		var oldestKey string
		var oldestTime time.Time
		for k, e := range c.entries {
			if oldestKey == "" || e.expiresAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = e.expiresAt
			}
		}
		if oldestKey != "" {
			delete(c.entries, oldestKey)
		}
	}

	c.entries[key] = cacheEntry{
		resp:      resp,
		expiresAt: time.Now().Add(ttl),
	}
}

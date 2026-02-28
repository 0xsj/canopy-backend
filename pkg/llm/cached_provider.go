package llm

import (
	"context"
	"sync"
	"time"
)

// CachedProvider wraps a Provider with response caching. Identical requests
// (same model, messages, options) return a cached response when available.
// Concurrent identical requests are deduplicated via singleflight-style
// coordination so only one LLM call is made.
type CachedProvider struct {
	inner Provider
	cache ResponseCache
	ttl   time.Duration

	// Singleflight: deduplicate concurrent identical requests.
	mu       sync.Mutex
	inflight map[string]*call
}

type call struct {
	wg   sync.WaitGroup
	resp ChatResponse
	err  error
}

// NewCachedProvider wraps inner with response caching. ttl controls how long
// responses remain cached.
func NewCachedProvider(inner Provider, cache ResponseCache, ttl time.Duration) *CachedProvider {
	return &CachedProvider{
		inner:    inner,
		cache:    cache,
		ttl:      ttl,
		inflight: make(map[string]*call),
	}
}

func (p *CachedProvider) ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	// Only cache deterministic requests (temperature == 0).
	if req.Options.Temperature > 0 {
		return p.inner.ChatCompletion(ctx, req)
	}

	key := CacheKey(req)

	// Check cache.
	if resp, ok := p.cache.Get(key); ok {
		return resp, nil
	}

	// Singleflight: if another goroutine is already making the same call, wait.
	p.mu.Lock()
	if c, ok := p.inflight[key]; ok {
		p.mu.Unlock()
		c.wg.Wait()
		if c.err != nil {
			return ChatResponse{}, c.err
		}
		return c.resp, nil
	}

	c := &call{}
	c.wg.Add(1)
	p.inflight[key] = c
	p.mu.Unlock()

	// Make the actual call.
	c.resp, c.err = p.inner.ChatCompletion(ctx, req)
	if c.err == nil {
		p.cache.Set(key, c.resp, p.ttl)
	}

	c.wg.Done()

	// Clean up singleflight entry.
	p.mu.Lock()
	delete(p.inflight, key)
	p.mu.Unlock()

	return c.resp, c.err
}

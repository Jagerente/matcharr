package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

type entry[V any] struct {
	value     V
	expiresAt time.Time
}

type TTL[V any] struct {
	mu    sync.Mutex
	ttl   time.Duration
	now   func() time.Time
	items map[string]entry[V]

	hits, misses atomic.Uint64
}

func (c *TTL[V]) Hits() uint64   { return c.hits.Load() }
func (c *TTL[V]) Misses() uint64 { return c.misses.Load() }

func NewTTL[V any](ttl time.Duration) *TTL[V] {
	return &TTL[V]{
		ttl:   ttl,
		now:   time.Now,
		items: make(map[string]entry[V]),
	}
}

func (c *TTL[V]) Get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok || c.now().After(e.expiresAt) {
		delete(c.items, key)
		c.misses.Add(1)
		var zero V
		return zero, false
	}
	c.hits.Add(1)
	return e.value, true
}

func (c *TTL[V]) Set(key string, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get expires an entry only when its own key comes back; a key nobody
	// asks for again would pin one forever. Set runs on a cache miss, right
	// after a network round trip, so the sweep rides along for free.
	now := c.now()
	for k, e := range c.items {
		if now.After(e.expiresAt) {
			delete(c.items, k)
		}
	}
	c.items[key] = entry[V]{value: value, expiresAt: now.Add(c.ttl)}
}

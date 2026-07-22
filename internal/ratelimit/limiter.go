package ratelimit

import (
	"sync"
	"time"
)

const sweepInterval = 10 * time.Minute

type Limiter struct {
	burst float64
	rate  float64 // tokens per second

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
	now       func() time.Time // swapped in tests
}

type bucket struct {
	tokens float64
	seen   time.Time
}

func New(burst int, every time.Duration) *Limiter {
	return &Limiter{
		burst:   float64(burst),
		rate:    1 / every.Seconds(),
		buckets: make(map[string]*bucket),
		now:     time.Now,
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweepLocked(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst}
		l.buckets[key] = b
	} else {
		b.tokens = min(l.burst, b.tokens+now.Sub(b.seen).Seconds()*l.rate)
	}
	b.seen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *Limiter) sweepLocked(now time.Time) {
	if now.Sub(l.lastSweep) < sweepInterval {
		return
	}
	l.lastSweep = now

	full := time.Duration(l.burst / l.rate * float64(time.Second))
	for key, b := range l.buckets {
		if now.Sub(b.seen) >= full {
			delete(l.buckets, key)
		}
	}
}

package ratelimit

import (
	"testing"
	"time"
)

func newTestLimiter(burst int, every time.Duration, clock *time.Time) *Limiter {
	l := New(burst, every)
	l.now = func() time.Time { return *clock }
	return l
}

func TestBurstThenRefill(t *testing.T) {
	clock := time.Unix(0, 0)
	l := newTestLimiter(2, time.Second, &clock)

	for i := range 2 {
		if !l.Allow("a") {
			t.Fatalf("request %d within the burst was rejected", i)
		}
	}
	if l.Allow("a") {
		t.Fatal("the burst was not exhausted")
	}

	clock = clock.Add(time.Second)
	if !l.Allow("a") {
		t.Fatal("a token was not restored after the refill interval")
	}
}

func TestKeysAreIndependent(t *testing.T) {
	clock := time.Unix(0, 0)
	l := newTestLimiter(1, time.Second, &clock)

	if !l.Allow("a") || !l.Allow("b") {
		t.Fatal("one key exhausted another key's budget")
	}
	if l.Allow("a") {
		t.Fatal("key a kept spending past its burst")
	}
}

func TestRefilledBucketsAreDropped(t *testing.T) {
	clock := time.Unix(0, 0)
	l := newTestLimiter(2, time.Second, &clock)

	l.Allow("a")
	clock = clock.Add(sweepInterval + time.Second)
	l.Allow("b")

	if _, kept := l.buckets["a"]; kept {
		t.Fatal("a refilled bucket was kept, the map grows without bound")
	}
	if _, kept := l.buckets["b"]; !kept {
		t.Fatal("the active bucket was dropped")
	}
}

func TestSpentBucketSurvivesSweep(t *testing.T) {
	clock := time.Unix(0, 0)
	l := newTestLimiter(2, time.Hour, &clock)

	l.Allow("a")
	l.Allow("a")
	clock = clock.Add(sweepInterval + time.Second)

	if l.Allow("a") {
		t.Fatal("the sweep reset a bucket that had not refilled")
	}
}

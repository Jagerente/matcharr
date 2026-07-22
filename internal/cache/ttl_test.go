package cache

import (
	"testing"
	"time"
)

func TestTTL(t *testing.T) {
	c := NewTTL[string](time.Minute)

	now := time.Now()
	c.now = func() time.Time { return now }

	if _, ok := c.Get("missing"); ok {
		t.Fatal("empty cache returned a value")
	}

	c.Set("key", "value")
	if v, ok := c.Get("key"); !ok || v != "value" {
		t.Fatalf("Get = %q, %v; want value, true", v, ok)
	}

	now = now.Add(time.Minute + time.Second)
	if _, ok := c.Get("key"); ok {
		t.Fatal("expired entry was not evicted")
	}
}

func TestTTLSetSweepsEntriesNobodyAsksFor(t *testing.T) {
	c := NewTTL[string](time.Minute)

	now := time.Now()
	c.now = func() time.Time { return now }

	c.Set("forgotten", "value")
	now = now.Add(time.Minute + time.Second)
	c.Set("fresh", "value")

	if _, ok := c.items["forgotten"]; ok {
		t.Fatal("the expired entry survived a Set under another key")
	}
	if len(c.items) != 1 {
		t.Fatalf("entries after the sweep: %d, want 1", len(c.items))
	}
	if v, ok := c.Get("fresh"); !ok || v != "value" {
		t.Fatalf("Get = %q, %v; want value, true", v, ok)
	}
}

package ratelimit

import "testing"

func TestLimiterBurstsThenRefills(t *testing.T) {
	limiter := New(1, 3)
	for range 3 {
		if !limiter.Allow("key") {
			t.Fatal("expected first three requests to be allowed")
		}
	}
	if limiter.Allow("key") {
		t.Fatal("expected fourth request to exceed burst")
	}
}

func TestLimiterKeysAreIndependent(t *testing.T) {
	limiter := New(1, 1)
	if !limiter.Allow("a") || !limiter.Allow("b") {
		t.Fatal("distinct keys must not share buckets")
	}
}

func TestConcurrencyBoundsSlots(t *testing.T) {
	concurrency := NewConcurrency(2)
	if !concurrency.Acquire("key") || !concurrency.Acquire("key") {
		t.Fatal("expected two slots to be available")
	}
	if concurrency.Acquire("key") {
		t.Fatal("expected third slot to be rejected")
	}
	concurrency.Release("key")
	if !concurrency.Acquire("key") {
		t.Fatal("expected a slot after release")
	}
}

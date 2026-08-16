package ratelimit

import (
	"sync"
	"time"
)

// Limiter is an in-memory token-bucket limiter keyed by an arbitrary string
// (API key id, account id, or client IP). It is safe for concurrent use and
// lazily evicts keys that have been idle for several minutes.
//
// ponytail: in-memory, per-instance. Replace with a shared store (Redis) only
// if the broker scales past a single instance.
type Limiter struct {
	mu          sync.Mutex
	buckets     map[string]*bucket
	rate        float64
	burst       float64
	lastCleanup time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

const idleEviction = 5 * time.Minute

func New(perSecond float64, burst int) *Limiter {
	if perSecond <= 0 || burst <= 0 {
		panic("ratelimit: per-second rate and burst must be positive")
	}
	return &Limiter{buckets: make(map[string]*bucket), rate: perSecond, burst: float64(burst)}
}

// Allow reports whether key may proceed with one request right now.
func (l *Limiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cleanup(now)
	b := l.buckets[key]
	if b == nil {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *Limiter) cleanup(now time.Time) {
	if now.Sub(l.lastCleanup) < time.Minute {
		return
	}
	l.lastCleanup = now
	for key, b := range l.buckets {
		if now.Sub(b.last) > idleEviction {
			delete(l.buckets, key)
		}
	}
}

// Concurrency bounds the number of in-flight requests per key.
type Concurrency struct {
	mu    sync.Mutex
	slots map[string]int
	max   int
}

func NewConcurrency(max int) *Concurrency {
	if max <= 0 {
		panic("ratelimit: max concurrency must be positive")
	}
	return &Concurrency{slots: make(map[string]int), max: max}
}

// Acquire reserves a slot for key, reporting whether one was available.
func (c *Concurrency) Acquire(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.slots[key] >= c.max {
		return false
	}
	c.slots[key]++
	return true
}

// Release returns a previously acquired slot for key.
func (c *Concurrency) Release(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.slots[key] <= 1 {
		delete(c.slots, key)
		return
	}
	c.slots[key]--
}

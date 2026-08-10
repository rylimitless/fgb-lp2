package middlewares

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// In-memory token-bucket rate limiter (fixes audit item H3).
//
// Dependency-free and single-instance: it lives in the backend process's
// memory, which is sufficient for this deployment (one backend container).
// For a horizontally-scaled deployment, swap this for a Redis-backed limiter
// keyed the same way (client IP).
//
// Each bucket holds up to `burst` tokens and refills at `perMinute` tokens per
// minute. Idle buckets are GC'd periodically so memory stays bounded.

type bucket struct {
	tokens float64
	last   time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	refillPS float64 // tokens added per second
	burst    float64
}

func newRateLimiter(burst int, perMinute int) *rateLimiter {
	rl := &rateLimiter{
		buckets:  make(map[string]*bucket),
		refillPS: float64(perMinute) / 60.0,
		burst:    float64(burst),
	}
	// Sweep idle buckets every 10 minutes so a rotating-IP attacker can't grow
	// the map without bound. Buckets untouched for 30 min are reclaimed.
	go rl.gc(10*time.Minute, 30*time.Minute)
	return rl
}

// allow reports whether a request from key is permitted under the limit.
func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.burst, last: now}
		rl.buckets[key] = b
	}

	// Refill based on elapsed time, capped at burst.
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * rl.refillPS
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.last = now

	if b.tokens >= 1 {
		b.tokens -= 1
		return true
	}
	return false
}

// gc periodically drops buckets that haven't been touched within `maxIdle`.
// It runs on its own ticker for the lifetime of the limiter.
func (rl *rateLimiter) gc(interval, maxIdle time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-maxIdle)
		rl.mu.Lock()
		for k, b := range rl.buckets {
			if b.last.Before(cutoff) {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitByIP returns a middleware limiting requests per client IP.
//   - burst:     maximum requests allowed in an immediate burst
//   - perMinute: steady-state allowed requests per minute (sustained)
//
// Suggested settings from the audit:
//   - login:        5 / min,  burst 10
//   - setup:        10 / hour, burst 5  (use 1/min)
//   - forgot-password: 3 / hour, burst 3 (use 1/min)
func RateLimitByIP(burst int, perMinute int) gin.HandlerFunc {
	rl := newRateLimiter(burst, perMinute)
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again shortly.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

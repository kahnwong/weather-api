package main

import (
	"math"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

const (
	defaultRateLimitRequestsPerMinute = 60
	defaultRateLimitBurst             = 60
)

type rateLimitBucket struct {
	tokens float64
	last   time.Time
}

type rateLimiter struct {
	requestsPerSecond float64
	burst             float64

	mu          sync.Mutex
	buckets     map[string]*rateLimitBucket
	lastCleanup time.Time
}

func newRateLimiter(requestsPerMinute, burst int) *rateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = defaultRateLimitRequestsPerMinute
	}
	if burst <= 0 {
		burst = requestsPerMinute
	}

	now := time.Now()
	return &rateLimiter{
		requestsPerSecond: float64(requestsPerMinute) / 60,
		burst:             float64(burst),
		buckets:           make(map[string]*rateLimitBucket),
		lastCleanup:       now,
	}
}

func (l *rateLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[key]
	if !ok {
		bucket = &rateLimitBucket{tokens: l.burst, last: now}
		l.buckets[key] = bucket
	}

	elapsed := now.Sub(bucket.last).Seconds()
	bucket.tokens = math.Min(l.burst, bucket.tokens+elapsed*l.requestsPerSecond)
	bucket.last = now

	if bucket.tokens >= 1 {
		bucket.tokens--
		l.cleanup(now)
		return true, 0
	}

	retryAfter := time.Duration(math.Ceil((1-bucket.tokens)/l.requestsPerSecond)) * time.Second
	l.cleanup(now)
	return false, retryAfter
}

func (l *rateLimiter) cleanup(now time.Time) {
	if now.Sub(l.lastCleanup) < time.Minute {
		return
	}

	maxIdle := time.Duration(math.Ceil(l.burst/l.requestsPerSecond))*time.Second + time.Minute
	for key, bucket := range l.buckets {
		if now.Sub(bucket.last) > maxIdle {
			delete(l.buckets, key)
		}
	}
	l.lastCleanup = now
}

func rateLimitMiddleware(requestsPerMinute, burst int) fiber.Handler {
	limiter := newRateLimiter(requestsPerMinute, burst)

	return func(c fiber.Ctx) error {
		allowed, retryAfter := limiter.allow(c.IP(), time.Now())
		if !allowed {
			seconds := int(math.Ceil(retryAfter.Seconds()))
			c.Set("Retry-After", strconv.Itoa(seconds))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Rate limit exceeded"})
		}

		return c.Next()
	}
}

func rateLimitMiddlewareFromEnv() fiber.Handler {
	requestsPerMinute := envInt("RATE_LIMIT_REQUESTS_PER_MINUTE", defaultRateLimitRequestsPerMinute)
	burst := envInt("RATE_LIMIT_BURST", defaultRateLimitBurst)
	return rateLimitMiddleware(requestsPerMinute, burst)
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

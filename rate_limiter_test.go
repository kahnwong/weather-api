package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestRateLimiterAllowRefillsTokens(t *testing.T) {
	limiter := newRateLimiter(2, 2)
	now := time.Unix(0, 0)

	if allowed, _ := limiter.allow("client", now); !allowed {
		t.Fatal("first request should be allowed")
	}
	if allowed, _ := limiter.allow("client", now); !allowed {
		t.Fatal("second request should be allowed")
	}
	if allowed, retryAfter := limiter.allow("client", now); allowed || retryAfter != 30*time.Second {
		t.Fatalf("third request should be rejected with 30s retry, got allowed=%v retryAfter=%s", allowed, retryAfter)
	}
	if allowed, _ := limiter.allow("client", now.Add(30*time.Second)); !allowed {
		t.Fatal("request should be allowed after token refill")
	}
}

func TestRateLimitMiddlewareReturnsTooManyRequests(t *testing.T) {
	app := fiber.New()
	app.Use(rateLimitMiddleware(1, 1))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	first, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("perform first request: %v", err)
	}
	defer first.Body.Close()
	if first.StatusCode != http.StatusOK {
		t.Fatalf("first response status = %d, want %d", first.StatusCode, http.StatusOK)
	}

	second, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("perform second request: %v", err)
	}
	defer second.Body.Close()
	if second.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("second response status = %d, want %d", second.StatusCode, http.StatusTooManyRequests)
	}
	if got := second.Header.Get("Retry-After"); got == "" {
		t.Fatal("Retry-After header should be set")
	}
	body, err := io.ReadAll(second.Body)
	if err != nil {
		t.Fatalf("read second response: %v", err)
	}
	if got := string(body); got != `{"error":"Rate limit exceeded"}` {
		t.Fatalf("second response body = %s", got)
	}
}

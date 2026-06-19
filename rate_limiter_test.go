package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(rateLimitMiddleware(1, 1))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first response status = %d, want %d", first.Code, http.StatusOK)
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second response status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}
	if got := second.Header().Get("Retry-After"); got == "" {
		t.Fatal("Retry-After header should be set")
	}
}

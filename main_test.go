package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
)

func TestAPIKeyAuthMiddleware(t *testing.T) {
	previousAPIKey := apiKey
	apiKey = "expected-key"
	t.Cleanup(func() {
		apiKey = previousAPIKey
	})

	tests := map[string]struct {
		key        string
		wantStatus int
		wantBody   string
	}{
		"missing key": {
			wantStatus: fiber.StatusUnauthorized,
			wantBody:   `{"error":"Missing or malformed API key"}`,
		},
		"invalid key": {
			key:        "wrong-key",
			wantStatus: fiber.StatusUnauthorized,
			wantBody:   `{"error":"Invalid API key"}`,
		},
		"valid key": {
			key:        "expected-key",
			wantStatus: fiber.StatusNoContent,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/", apiKeyAuthMiddleware(), func(c fiber.Ctx) error {
				return c.SendStatus(fiber.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if test.key != "" {
				request.Header.Set("X-API-Key", test.key)
			}
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("perform request: %v", err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("read response: %v", err)
			}

			if response.StatusCode != test.wantStatus {
				t.Errorf("status = %d, want %d", response.StatusCode, test.wantStatus)
			}
			if got := string(body); got != test.wantBody {
				t.Errorf("body = %q, want %q", got, test.wantBody)
			}
		})
	}
}

func TestSetupLoggerLevel(t *testing.T) {
	tests := map[string]struct {
		value        string
		debugEnabled bool
		infoEnabled  bool
		errorEnabled bool
	}{
		"default": {
			debugEnabled: true,
			infoEnabled:  true,
			errorEnabled: true,
		},
		"invalid defaults to debug": {
			value:        "verbose",
			debugEnabled: true,
			infoEnabled:  true,
			errorEnabled: true,
		},
		"info": {
			value:        "info",
			infoEnabled:  true,
			errorEnabled: true,
		},
		"error": {
			value:        "error",
			errorEnabled: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", test.value)
			previousLogger := slog.Default()
			previousLevel := zerolog.GlobalLevel()
			t.Cleanup(func() {
				slog.SetDefault(previousLogger)
				zerolog.SetGlobalLevel(previousLevel)
			})

			logger := setupLogger()
			ctx := context.Background()
			if got := logger.Enabled(ctx, slog.LevelDebug); got != test.debugEnabled {
				t.Errorf("debug enabled = %v, want %v", got, test.debugEnabled)
			}
			if got := logger.Enabled(ctx, slog.LevelInfo); got != test.infoEnabled {
				t.Errorf("info enabled = %v, want %v", got, test.infoEnabled)
			}
			if got := logger.Enabled(ctx, slog.LevelError); got != test.errorEnabled {
				t.Errorf("error enabled = %v, want %v", got, test.errorEnabled)
			}
		})
	}
}

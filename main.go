package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"log/slog"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/rs/zerolog"
	slogfiber "github.com/samber/slog-fiber"
	slogzerolog "github.com/samber/slog-zerolog/v2"
)

var apiKey = os.Getenv("WEATHER_API_KEY")

func validateAPIKey(key string) bool {
	hashedAPIKey := sha256.Sum256([]byte(apiKey))
	hashedKey := sha256.Sum256([]byte(key))

	return subtle.ConstantTimeCompare(hashedAPIKey[:], hashedKey[:]) == 1
}

func apiKeyAuthMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		key := c.Get("X-API-Key")
		if key == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing or malformed API key"})
		}

		if !validateAPIKey(key) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid API key"})
		}

		return c.Next()
	}
}

func setupLogger() *slog.Logger {
	level, err := zerolog.ParseLevel(strings.ToLower(os.Getenv("LOG_LEVEL")))
	if err != nil || level == zerolog.NoLevel {
		level = zerolog.DebugLevel
	}

	zerolog.SetGlobalLevel(level)
	output := zerolog.ConsoleWriter{Out: os.Stderr}
	zeroLogger := zerolog.New(output).Level(level).With().Timestamp().Logger()
	logger := slog.New(slogzerolog.Option{
		Level:  slogzerolog.ZeroLogLeveler{Logger: &zeroLogger},
		Logger: &zeroLogger,
	}.NewZerologHandler())
	slog.SetDefault(logger)
	return logger
}

func main() {
	// init
	logger := setupLogger()
	initWeatherConfig()

	app := fiber.New()

	// middleware
	app.Use(slogfiber.New(logger))
	app.Use(recover.New())
	app.Use(rateLimitMiddlewareFromEnv())

	// routes
	app.Get("/weather", apiKeyAuthMiddleware(), WeatherGetController)

	// start server
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	slog.Info("Starting server", "address", listenAddr)
	if err := app.Listen(listenAddr, fiber.ListenConfig{DisableStartupMessage: true}); err != nil {
		slog.Error("Error starting server", "error", err)
		os.Exit(1)
	}
}

package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	slogzerolog "github.com/samber/slog-zerolog/v2"
)

var apiKey = os.Getenv("WEATHER_API_KEY")

func validateAPIKey(key string) bool {
	hashedAPIKey := sha256.Sum256([]byte(apiKey))
	hashedKey := sha256.Sum256([]byte(key))

	return subtle.ConstantTimeCompare(hashedAPIKey[:], hashedKey[:]) == 1
}

func apiKeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or malformed API key"})
			c.Abort()
			return
		}

		if !validateAPIKey(key) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func setupLogger() {
	level, err := zerolog.ParseLevel(strings.ToLower(os.Getenv("LOG_LEVEL")))
	if err != nil || level == zerolog.NoLevel {
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)
	output := zerolog.ConsoleWriter{Out: os.Stderr}
	logger := zerolog.New(output).Level(level).With().Timestamp().Logger()
	slog.SetDefault(slog.New(slogzerolog.Option{Logger: &logger}.NewZerologHandler()))
}

func main() {
	// init
	gin.SetMode(gin.ReleaseMode)
	setupLogger()
	initWeatherConfig()

	router := gin.New()

	// logging
	router.Use(logger.SetLogger())
	router.Use(gin.Recovery())

	// routes
	router.GET("/weather", apiKeyAuthMiddleware(), WeatherGetController)

	// start server
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	slog.Info("Starting server", "address", listenAddr)
	if err := router.Run(listenAddr); err != nil {
		slog.Error("Error starting server", "error", err)
		os.Exit(1)
	}
}

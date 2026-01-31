package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/gin-contrib/logger"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

func main() {
	// init
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// logging
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	router.Use(logger.SetLogger())
	router.Use(gin.Recovery())

	// routes
	router.GET("/weather", apiKeyAuthMiddleware(), WeatherGetController)

	// start server
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	log.Info().Str("address", listenAddr).Msg("Starting server")
	if err := router.Run(listenAddr); err != nil {
		log.Fatal().Err(err).Msg("Error starting server")
	}
}

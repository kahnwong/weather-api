package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"os"

	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/rs/zerolog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/keyauth"
)

var (
	apiKey = os.Getenv("QRCODE_API_KEY")
)

func validateAPIKey(c *fiber.Ctx, key string) (bool, error) {
	hashedAPIKey := sha256.Sum256([]byte(apiKey))
	hashedKey := sha256.Sum256([]byte(key))

	if subtle.ConstantTimeCompare(hashedAPIKey[:], hashedKey[:]) == 1 {
		return true, nil
	}
	return false, keyauth.ErrMissingOrMalformedAPIKey
}

func main() {
	// init
	app := fiber.New()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	app.Use(fiberzerolog.New(fiberzerolog.Config{
		Logger: &logger,
	}))

	// auth
	app.Use(keyauth.New(keyauth.Config{
		KeyLookup: "header:X-API-Key",
		Validator: validateAPIKey,
	}))

	// routes
	app.Get("/title", GetTitleController)
	app.Get("/images/qrcode.png", GetPngController)

	// start server
	err := app.Listen(os.Getenv("LISTEN_ADDR"))
	if err != nil {
		fmt.Println("Error starting server", err)
	}
}

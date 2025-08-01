package main

import (
	"bytes"
	"encoding/base64"
	"image/png"

	"github.com/nfnt/resize"
	"github.com/rs/zerolog/log"

	"github.com/gofiber/fiber/v2"
)

type TitleResponse struct {
	Name string `json:"name"`
}

type QrcodeRequestItem struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"` // base64
}

func TitleGetController(c *fiber.Ctx) error {
	qrcode, err := Qrcode.GetTitle(1)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Error obtaining qrcode data")
	}

	return c.JSON(TitleResponse{
		Name: qrcode.Name,
	})
}

func PngGetController(c *fiber.Ctx) error {
	qrcode, err := Qrcode.GetImage(1)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Error obtaining qrcode data")
	}

	// because for some reason garmin sdk can't forward header on image request
	reqApiKey := c.Query("apiKey")
	if reqApiKey != apiKey {
		return c.SendString("Nope")
	}

	c.Type("png")
	return c.Send(qrcode.Image)
}

func AddPostController(c *fiber.Ctx) error {
	// parse request
	p := new(QrcodeRequestItem)
	if err := c.BodyParser(p); err != nil {
		log.Error().Err(err).Msg("Error parsing request body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON request body",
		})
	}

	// insert
	//// Decode the Base64 string to a byte slice
	imageBytes, err := base64.StdEncoding.DecodeString(p.Image)
	if err != nil {
		log.Error().Err(err).Msg("Error decoding base64 image")
	}

	//// resize to 90x90 so garmin doesn't choke
	img, err := png.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		log.Error().Err(err).Msg("failed to decode PNG image")
	}
	resizedImg := resize.Resize(90, 90, img, resize.Lanczos3)

	//// encode back to bytes
	var buf bytes.Buffer
	err = png.Encode(&buf, resizedImg)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode resized PNG image")
	}

	//// insert to db
	err = Qrcode.Add(QrcodeItem{
		ID:    p.ID,
		Name:  p.Name,
		Image: buf.Bytes(),
	})
	if err != nil {
		log.Printf("Error adding image: %v", err)
	}

	c.Status(fiber.StatusOK)
	return c.SendString("Success")
}

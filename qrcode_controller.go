package main

import (
	"encoding/base64"
	"log"

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
		log.Printf("Error parsing request body: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON request body",
		})
	}

	// insert
	//// Decode the Base64 string to a byte slice
	imageBytes, err := base64.StdEncoding.DecodeString(p.Image)
	if err != nil {
		log.Fatalf("Error decoding Base64 string: %v", err)
	}

	err = Qrcode.Add(QrcodeItem{
		ID:    p.ID,
		Name:  p.Name,
		Image: imageBytes,
	})
	if err != nil {
		log.Printf("Error adding image: %v", err)
	}

	c.Status(fiber.StatusOK)
	return c.SendString("Success")
}

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
)

type Title struct {
	Name string `json:"name"`
}

func GetTitleController(c *fiber.Ctx) error {
	title := Title{
		Name: "Foo",
	}
	return c.JSON(title)
}

func GetPngController(c *fiber.Ctx) error {
	filePath := "./assets/qrcode.png"
	imageData, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading image file '%s': %v", filePath, err)
		return c.Status(fiber.StatusNotFound).SendString(fmt.Sprintf("Image '%s' not found or could not be read.", filePath))
	}

	c.Set(fiber.HeaderContentType, "img/png")
	return c.Send(imageData)
}

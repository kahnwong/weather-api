package qrcode

import (
	"encoding/base64"
	"os"
	"strconv"

	"github.com/rs/zerolog/log"

	"github.com/gofiber/fiber/v2"
)

var (
	apiKey = os.Getenv("QRCODE_API_KEY")
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
	qrcode, err := Qrcode.GetTitle(_stringToInt(c.Params("id")))
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Error obtaining qrcode data")
	}

	return c.JSON(TitleResponse{
		Name: qrcode.Name,
	})
}

func PngGetController(c *fiber.Ctx) error {
	qrcode, err := Qrcode.GetImage(_stringToInt(c.Params("id")))
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
	imageBytes, err := base64.StdEncoding.DecodeString(p.Image)
	if err != nil {
		log.Error().Err(err).Msg("Error decoding base64 image")
	}

	//// image processing
	imageGrayScaleBytes, _ := pngToGrayScale(imageBytes)
	imageCropBorderBytes, _ := pngCropBorder(imageGrayScaleBytes)
	//// -- resize to 90x90 so garmin doesn't choke
	imageResizedBytes, _ := pngResize(imageCropBorderBytes)

	//// insert to db
	err = Qrcode.Add(QrcodeItem{
		ID:    p.ID,
		Name:  p.Name,
		Image: imageResizedBytes,
	})
	if err != nil {
		log.Printf("Error adding image: %v", err)
	}

	c.Status(fiber.StatusOK)
	return c.SendString("Success")
}

func _stringToInt(s string) int {
	id, err := strconv.Atoi(s)
	if err != nil {
		log.Error().Err(err).Msgf("Error converting to int: %s", s)
	}

	return id
}

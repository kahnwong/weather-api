package main

import (
	"fmt"
	"log"
	"os"

	owm "github.com/briandowns/openweathermap"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type WeatherResponse struct {
	Description   string  `json:"description"`
	Temperature   float32 `json:"temperature"`
	RainOneHour   float64 `json:"rain_one_hour"`
	RainThreeHour float64 `json:"rain_three_hour"`
}

func WeatherGetController(c *fiber.Ctx) error {
	w, err := owm.NewCurrent("C", "en", os.Getenv("OPENWEATHER_API_KEY"))
	if err != nil {
		log.Fatalln(err)
	}

	err = w.CurrentByName(os.Getenv("CURRENT_CITY"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot obtain weather data",
		})
	}

	fmt.Println()

	caser := cases.Title(language.English)
	return c.JSON(WeatherResponse{
		Description:   caser.String(w.Weather[0].Description),
		Temperature:   float32(w.Main.Temp),
		RainOneHour:   w.Rain.OneH,
		RainThreeHour: w.Rain.ThreeH,
	})
}

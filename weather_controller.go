package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	owm "github.com/briandowns/openweathermap"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	CityName  string
	Latitude  float64
	Longitude float64
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
		log.Fatal().Err(err).Msg("Failed to init OpenWeatherMap")
	}

	// prioritize lat/lng over city name
	if Latitude != 0 {
		log.Info().Msg("Using latitude/longitude for location")
		err = w.CurrentByCoordinates(&owm.Coordinates{
			Longitude: Longitude,
			Latitude:  Latitude,
		})
	} else {
		log.Info().Msg("Using city name for location")
		err = w.CurrentByName(CityName)
	}

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

func stringToFloat(s string) (float64, error) {
	vInt, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, err
	} else {
		return vInt, nil
	}
}

func init() {
	var err error

	CityName = os.Getenv("CURRENT_CITY")
	Latitude, err = stringToFloat(os.Getenv("LATITUDE"))
	if err != nil {
		log.Warn().Msg("Error converting latitude to float")
		Latitude = 0
	}
	Longitude, err = stringToFloat(os.Getenv("LONGITUDE"))
	if err != nil {
		log.Warn().Msg("Error converting longitude to float")
		Longitude = 0
	}
}

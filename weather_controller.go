package main

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jdotcurs/pirateweather-go/pkg/pirateweather"
	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
)

var (
	CityName  string
	Latitude  float64
	Longitude float64
)

type WeatherResponse struct {
	Description   string  `json:"description"`
	Temperature   float64 `json:"temperature"`
	RainOneHour   float64 `json:"rain_one_hour"`
	RainThreeHour float64 `json:"rain_three_hour"`
}

func WeatherGetController(c *gin.Context) {
	client := pirateweather.NewClient(os.Getenv("PIRATEWEATHER_API_KEY"))
	forecast, err := client.Forecast(Latitude, Longitude,
		pirateweather.WithUnits("si"),
		pirateweather.WithExclude([]string{"minutely"}),
	)
	if err != nil {
		log.Error().Err(err).Msg("Error getting forecast")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get forecast"})
		return
	}

	c.JSON(http.StatusOK, WeatherResponse{
		Description:   forecast.Currently.Summary,
		Temperature:   forecast.Currently.Temperature,
		RainOneHour:   forecast.Hourly.Data[1].PrecipProbability * 100,
		RainThreeHour: forecast.Hourly.Data[3].PrecipProbability * 100,
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

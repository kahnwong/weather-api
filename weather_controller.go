package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const openMeteoForecastURL = "https://api.open-meteo.com/v1/forecast"

var weatherHTTPClient = &http.Client{Timeout: 10 * time.Second}

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

type openMeteoResponse struct {
	Current struct {
		Temperature *float64 `json:"temperature_2m"`
		WeatherCode *int     `json:"weather_code"`
	} `json:"current"`
	Hourly struct {
		PrecipitationProbability []*float64 `json:"precipitation_probability"`
	} `json:"hourly"`
}

func WeatherGetController(c *gin.Context) {
	weatherGetController(c, weatherHTTPClient, openMeteoForecastURL)
}

func weatherGetController(c *gin.Context, client *http.Client, endpoint string) {
	forecast, err := fetchOpenMeteoForecast(c.Request.Context(), client, endpoint, Latitude, Longitude)
	if err != nil {
		slog.Error("Error getting forecast", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get forecast"})
		return
	}

	c.JSON(http.StatusOK, forecast)
}

func fetchOpenMeteoForecast(ctx context.Context, client *http.Client, endpoint string, latitude, longitude float64) (WeatherResponse, error) {
	requestURL, err := url.Parse(endpoint)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("parse Open-Meteo URL: %w", err)
	}

	query := requestURL.Query()
	query.Set("latitude", strconv.FormatFloat(latitude, 'f', -1, 64))
	query.Set("longitude", strconv.FormatFloat(longitude, 'f', -1, 64))
	query.Set("models", "ecmwf_ifs")
	query.Set("current", "temperature_2m,weather_code")
	query.Set("hourly", "precipitation_probability")
	query.Set("forecast_hours", "4")
	requestURL.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("create Open-Meteo request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("request Open-Meteo forecast: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return WeatherResponse{}, fmt.Errorf("Open-Meteo returned status %s", response.Status)
	}

	var forecast openMeteoResponse
	if err := json.NewDecoder(response.Body).Decode(&forecast); err != nil {
		return WeatherResponse{}, fmt.Errorf("decode Open-Meteo forecast: %w", err)
	}
	if forecast.Current.Temperature == nil || forecast.Current.WeatherCode == nil {
		return WeatherResponse{}, fmt.Errorf("Open-Meteo response is missing current conditions")
	}
	if len(forecast.Hourly.PrecipitationProbability) < 4 ||
		forecast.Hourly.PrecipitationProbability[1] == nil ||
		forecast.Hourly.PrecipitationProbability[3] == nil {
		return WeatherResponse{}, fmt.Errorf("Open-Meteo response is missing hourly precipitation probabilities")
	}

	return WeatherResponse{
		Description:   weatherDescription(*forecast.Current.WeatherCode),
		Temperature:   *forecast.Current.Temperature,
		RainOneHour:   *forecast.Hourly.PrecipitationProbability[1],
		RainThreeHour: *forecast.Hourly.PrecipitationProbability[3],
	}, nil
}

func weatherDescription(code int) string {
	switch code {
	case 0:
		return "Clear Sky"
	case 1:
		return "Mainly Clear"
	case 2:
		return "Partly Cloudy"
	case 3:
		return "Overcast"
	case 45:
		return "Fog"
	case 48:
		return "Rime Fog"
	case 51:
		return "Light Drizzle"
	case 53:
		return "Moderate Drizzle"
	case 55:
		return "Heavy Drizzle"
	case 56, 57:
		return "Freezing Drizzle"
	case 61:
		return "Light Rain"
	case 63:
		return "Moderate Rain"
	case 65:
		return "Heavy Rain"
	case 66, 67:
		return "Freezing Rain"
	case 71:
		return "Light Snow"
	case 73:
		return "Moderate Snow"
	case 75:
		return "Heavy Snow"
	case 77:
		return "Snow Grains"
	case 80:
		return "Light Showers"
	case 81:
		return "Moderate Showers"
	case 82:
		return "Heavy Showers"
	case 85, 86:
		return "Snow Showers"
	case 95:
		return "Thunderstorm"
	case 96, 99:
		return "Hail Thunderstorm"
	default:
		return "Unknown Weather"
	}
}

func stringToFloat(s string) (float64, error) {
	vInt, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, err
	} else {
		return vInt, nil
	}
}

func initWeatherConfig() {
	var err error

	CityName = os.Getenv("CURRENT_CITY")
	Latitude, err = stringToFloat(os.Getenv("LATITUDE"))
	if err != nil {
		slog.Warn("Error converting latitude to float", "error", err)
		Latitude = 0
	}
	Longitude, err = stringToFloat(os.Getenv("LONGITUDE"))
	if err != nil {
		slog.Warn("Error converting longitude to float", "error", err)
		Longitude = 0
	}
}

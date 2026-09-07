package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

const (
	openMeteoForecastURL  = "https://api.open-meteo.com/v1/forecast"
	weatherRainThreshold  = 50
	weatherRelativeCutoff = 4 * time.Hour
	weatherRainHorizon    = 6 * time.Hour
)

var weatherHTTPClient = &http.Client{Timeout: 10 * time.Second}

var (
	CityName  string
	Latitude  float64
	Longitude float64
	Timezone  string
)

type WeatherResponse struct {
	Description string  `json:"description"`
	Temperature float64 `json:"temperature"`
	RainStart   *string `json:"rain_start"`
	RainStop    *string `json:"rain_stop"`
}

type openMeteoResponse struct {
	Current struct {
		Temperature *float64 `json:"temperature_2m"`
		WeatherCode *int     `json:"weather_code"`
	} `json:"current"`
	Hourly struct {
		Time                     []int64   `json:"time"`
		Rain                     []float64 `json:"rain"`
		PrecipitationProbability []int     `json:"precipitation_probability"`
	} `json:"hourly"`
}

type weatherRainWindow struct {
	Start   time.Time
	Stop    time.Time
	HasStop bool
}

func WeatherGetController(c fiber.Ctx) error {
	return weatherGetController(c, weatherHTTPClient, openMeteoForecastURL)
}

func weatherGetController(c fiber.Ctx, client *http.Client, endpoint string) error {
	forecast, err := fetchOpenMeteoForecast(c.RequestCtx(), client, endpoint, Latitude, Longitude)
	if err != nil {
		slog.Error("Error getting forecast", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get forecast"})
	}

	return c.Status(fiber.StatusOK).JSON(forecast)
}

func fetchOpenMeteoForecast(ctx context.Context, client *http.Client, endpoint string, latitude, longitude float64) (WeatherResponse, error) {
	location, err := time.LoadLocation(Timezone)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("load weather timezone %q: %w", Timezone, err)
	}

	requestURL, err := url.Parse(endpoint)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("parse Open-Meteo URL: %w", err)
	}

	query := requestURL.Query()
	query.Set("latitude", strconv.FormatFloat(latitude, 'f', -1, 64))
	query.Set("longitude", strconv.FormatFloat(longitude, 'f', -1, 64))
	query.Set("models", "ecmwf_ifs")
	query.Set("current", "temperature_2m,weather_code")
	query.Set("hourly", "rain,precipitation_probability")
	query.Set("forecast_hours", "24")
	query.Set("timeformat", "unixtime")
	query.Set("timezone", Timezone)
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
	if min(len(forecast.Hourly.Time), len(forecast.Hourly.Rain), len(forecast.Hourly.PrecipitationProbability)) == 0 {
		return WeatherResponse{}, fmt.Errorf("Open-Meteo response is missing hourly rain forecast")
	}
	rainStart, rainStop := rainTimes(time.Now().In(location), forecast.Hourly.Time, forecast.Hourly.Rain, forecast.Hourly.PrecipitationProbability)

	return WeatherResponse{
		Description: weatherDescription(*forecast.Current.WeatherCode),
		Temperature: *forecast.Current.Temperature,
		RainStart:   rainStart,
		RainStop:    rainStop,
	}, nil
}

func findWeatherRainWindow(now time.Time, times []int64, rain []float64, probabilities []int) (weatherRainWindow, bool) {
	length := min(len(times), len(rain), len(probabilities))
	var window weatherRainWindow
	found := false
	for index := range length {
		intervalStop := time.Unix(times[index], 0)
		intervalStart := intervalStop.Add(-time.Hour)
		if !intervalStop.After(now) {
			continue
		}
		qualifies := rain[index] > 0 && probabilities[index] >= weatherRainThreshold
		if !found {
			if !qualifies {
				continue
			}
			window.Start = intervalStart
			if window.Start.Before(now) {
				window.Start = now
			}
			window.Stop = intervalStop
			found = true
			continue
		}
		if qualifies && !intervalStart.After(window.Stop) {
			window.Stop = intervalStop
			continue
		}
		window.HasStop = true
		return window, true
	}
	return window, found
}

func weatherTimeLabel(now, target time.Time) string {
	if !target.After(now) {
		return "now"
	}
	remaining := target.Sub(now)
	if remaining < weatherRelativeCutoff {
		return fmt.Sprintf("+%dH", int(math.Ceil(remaining.Hours())))
	}
	return target.In(now.Location()).Format("15")

}

func rainTimes(now time.Time, times []int64, rain []float64, probabilities []int) (*string, *string) {
	window, found := findWeatherRainWindow(now, times, rain, probabilities)
	if !found || window.Start.Sub(now) > weatherRainHorizon {
		return nil, nil
	}
	start := new(weatherTimeLabel(now, window.Start))
	if !window.HasStop {
		return start, nil
	}
	return start, new(weatherTimeLabel(now, window.Stop))
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
	Timezone = os.Getenv("TIMEZONE")
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

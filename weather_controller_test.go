package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestWeatherGetControllerUsesECMWFIFSAndPreservesResponse(t *testing.T) {
	var receivedQuery url.Values
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		now := time.Now()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"current": map[string]any{"temperature_2m": 32.8, "weather_code": 2},
			"hourly": map[string]any{
				"time":                      []int64{now.Add(30 * time.Minute).Unix(), now.Add(90 * time.Minute).Unix(), now.Add(150 * time.Minute).Unix()},
				"rain":                      []float64{0.2, 0.1, 0},
				"precipitation_probability": []int{60, 70, 80},
			},
		})
	}))
	defer upstream.Close()

	previousLatitude, previousLongitude, previousTimezone := Latitude, Longitude, Timezone
	Latitude, Longitude = 1.3521, 103.8198
	Timezone = "Asia/Bangkok"
	t.Cleanup(func() {
		Latitude, Longitude, Timezone = previousLatitude, previousLongitude, previousTimezone
	})

	app := fiber.New()
	app.Get("/weather", func(c fiber.Ctx) error {
		return weatherGetController(c, upstream.Client(), upstream.URL)
	})
	request := httptest.NewRequest(http.MethodGet, "/weather", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.StatusCode, http.StatusOK, body)
	}

	wantQuery := map[string]string{
		"latitude":       "1.3521",
		"longitude":      "103.8198",
		"models":         "ecmwf_ifs",
		"current":        "temperature_2m,weather_code",
		"hourly":         "rain,precipitation_probability",
		"forecast_hours": "24",
		"timeformat":     "unixtime",
		"timezone":       Timezone,
	}
	for key, want := range wantQuery {
		if got := receivedQuery.Get(key); got != want {
			t.Errorf("query %q = %q, want %q", key, got, want)
		}
	}

	var got WeatherResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	rainStart, rainStop := "now", "+2H"
	want := WeatherResponse{
		Description: "Partly Cloudy",
		Temperature: 32.8,
		RainStart:   &rainStart,
		RainStop:    &rainStop,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("response = %+v, want %+v", got, want)
	}

	var fields map[string]any
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatalf("decode response fields: %v", err)
	}
	for _, key := range []string{"description", "temperature", "rain_start", "rain_stop"} {
		if _, ok := fields[key]; !ok {
			t.Errorf("response is missing field %q", key)
		}
	}
	if len(fields) != 4 {
		t.Errorf("response has %d fields, want 4", len(fields))
	}
}

func TestWeatherGetControllerHandlesUpstreamErrors(t *testing.T) {
	previousTimezone := Timezone
	Timezone = "UTC"
	t.Cleanup(func() {
		Timezone = previousTimezone
	})

	tests := map[string]struct {
		status int
		body   string
	}{
		"non-OK status": {
			status: http.StatusBadGateway,
			body:   `{}`,
		},
		"invalid JSON": {
			status: http.StatusOK,
			body:   `{`,
		},
		"missing current conditions": {
			status: http.StatusOK,
			body:   `{"hourly":{"time":[1],"rain":[0],"precipitation_probability":[1]}}`,
		},
		"missing hourly forecast": {
			status: http.StatusOK,
			body:   `{"current":{"temperature_2m":20,"weather_code":0},"hourly":{"precipitation_probability":[]}}`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer upstream.Close()

			app := fiber.New()
			app.Get("/weather", func(c fiber.Ctx) error {
				return weatherGetController(c, upstream.Client(), upstream.URL)
			})
			response, err := app.Test(httptest.NewRequest(http.MethodGet, "/weather", nil))
			if err != nil {
				t.Fatalf("perform request: %v", err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("read response: %v", err)
			}

			if response.StatusCode != http.StatusInternalServerError {
				t.Errorf("status = %d, want %d", response.StatusCode, http.StatusInternalServerError)
			}
			if got := strings.TrimSpace(string(body)); got != `{"error":"Failed to get forecast"}` {
				t.Errorf("body = %s", got)
			}
		})
	}
}

func TestRainTimesUsesRainAmountAndProbability(t *testing.T) {
	now := time.Date(2026, time.September, 7, 16, 15, 0, 0, time.FixedZone("Asia/Bangkok", 7*60*60))
	times := []int64{
		now.Add(45 * time.Minute).Unix(),
		now.Add(105 * time.Minute).Unix(),
		now.Add(165 * time.Minute).Unix(),
	}

	start, stop := rainTimes(now, times, []float64{0.4, 0.1, 0}, []int{80, 60, 80})
	if start == nil || *start != "now" {
		t.Fatalf("start = %v, want now", start)
	}
	if stop == nil || *stop != "+2H" {
		t.Fatalf("stop = %v, want +2H", stop)
	}

	start, stop = rainTimes(now, times, []float64{0, 0, 0}, []int{90, 90, 90})
	if start != nil || stop != nil {
		t.Fatalf("rain window with no rain = (%v, %v), want (nil, nil)", start, stop)
	}
}

func TestRainTimesIgnoresRainBeyondHorizon(t *testing.T) {
	now := time.Date(2026, time.September, 7, 16, 0, 0, 0, time.UTC)
	start, stop := rainTimes(
		now,
		[]int64{now.Add(8 * time.Hour).Unix()},
		[]float64{1},
		[]int{90},
	)
	if start != nil || stop != nil {
		t.Fatalf("rain window beyond horizon = (%v, %v), want (nil, nil)", start, stop)
	}
}

func TestWeatherDescriptionsAreTitleCaseAndAtMostTwoWords(t *testing.T) {
	tests := map[int]string{
		0:  "Clear Sky",
		1:  "Mainly Clear",
		2:  "Partly Cloudy",
		3:  "Overcast",
		45: "Fog",
		48: "Rime Fog",
		51: "Light Drizzle",
		53: "Moderate Drizzle",
		55: "Heavy Drizzle",
		56: "Freezing Drizzle",
		57: "Freezing Drizzle",
		61: "Light Rain",
		63: "Moderate Rain",
		65: "Heavy Rain",
		66: "Freezing Rain",
		67: "Freezing Rain",
		71: "Light Snow",
		73: "Moderate Snow",
		75: "Heavy Snow",
		77: "Snow Grains",
		80: "Light Showers",
		81: "Moderate Showers",
		82: "Heavy Showers",
		85: "Snow Showers",
		86: "Snow Showers",
		95: "Thunderstorm",
		96: "Hail Thunderstorm",
		99: "Hail Thunderstorm",
		-1: "Unknown Weather",
	}

	for code, want := range tests {
		got := weatherDescription(code)
		if got != want {
			t.Errorf("weatherDescription(%d) = %q, want %q", code, got, want)
		}
		words := strings.Fields(got)
		if len(words) > 2 {
			t.Errorf("weatherDescription(%d) has more than two words: %q", code, got)
		}
		for _, word := range words {
			if word[0] < 'A' || word[0] > 'Z' {
				t.Errorf("weatherDescription(%d) is not title case: %q", code, got)
			}
		}
	}
}

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWeatherGetControllerUsesECMWFIFSAndPreservesResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var receivedQuery url.Values
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"current":{"temperature_2m":32.8,"weather_code":2},
			"hourly":{"precipitation_probability":[16,12,8,6]}
		}`))
	}))
	defer upstream.Close()

	previousLatitude, previousLongitude := Latitude, Longitude
	Latitude, Longitude = 1.3521, 103.8198
	t.Cleanup(func() {
		Latitude, Longitude = previousLatitude, previousLongitude
	})

	request := httptest.NewRequest(http.MethodGet, "/weather", nil)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = request

	weatherGetController(context, upstream.Client(), upstream.URL)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	wantQuery := map[string]string{
		"latitude":       "1.3521",
		"longitude":      "103.8198",
		"models":         "ecmwf_ifs",
		"current":        "temperature_2m,weather_code",
		"hourly":         "precipitation_probability",
		"forecast_hours": "4",
	}
	for key, want := range wantQuery {
		if got := receivedQuery.Get(key); got != want {
			t.Errorf("query %q = %q, want %q", key, got, want)
		}
	}

	var got WeatherResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := WeatherResponse{
		Description:   "Partly Cloudy",
		Temperature:   32.8,
		RainOneHour:   12,
		RainThreeHour: 6,
	}
	if got != want {
		t.Errorf("response = %+v, want %+v", got, want)
	}

	var fields map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
		t.Fatalf("decode response fields: %v", err)
	}
	for _, key := range []string{"description", "temperature", "rain_one_hour", "rain_three_hour"} {
		if _, ok := fields[key]; !ok {
			t.Errorf("response is missing field %q", key)
		}
	}
	if len(fields) != 4 {
		t.Errorf("response has %d fields, want 4", len(fields))
	}
}

func TestWeatherGetControllerHandlesUpstreamErrors(t *testing.T) {
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
			body:   `{"hourly":{"precipitation_probability":[1,2,3,4]}}`,
		},
		"short hourly forecast": {
			status: http.StatusOK,
			body:   `{"current":{"temperature_2m":20,"weather_code":0},"hourly":{"precipitation_probability":[1,2]}}`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer upstream.Close()

			response := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(response)
			context.Request = httptest.NewRequest(http.MethodGet, "/weather", nil)

			weatherGetController(context, upstream.Client(), upstream.URL)

			if response.Code != http.StatusInternalServerError {
				t.Errorf("status = %d, want %d", response.Code, http.StatusInternalServerError)
			}
			if got := strings.TrimSpace(response.Body.String()); got != `{"error":"Failed to get forecast"}` {
				t.Errorf("body = %s", got)
			}
		})
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

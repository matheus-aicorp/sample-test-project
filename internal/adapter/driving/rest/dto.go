package rest

import (
	"encoding/json"
	"time"

	"github.com/matheus-aicorp/sample-test-project/internal/domain"
)

// Source identifies the upstream provider in the response body, as Open-Meteo's
// terms require attribution. Injected by the composition root so this package
// never imports the driven adapter.
type Source struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type weatherResponse struct {
	Count       int                 `json:"count"`
	RetrievedAt time.Time           `json:"retrieved_at"`
	Source      Source              `json:"source"`
	Capitals    []capitalWeatherDTO `json:"capitals"`
}

type capitalWeatherDTO struct {
	Slug            string    `json:"slug"`
	Name            string    `json:"name"`
	StateCode       string    `json:"state_code"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	TemperatureC    float64   `json:"temperature_c"`
	FeelsLikeC      float64   `json:"feels_like_c"`
	HumidityPercent int       `json:"humidity_percent"`
	WindSpeedKmh    float64   `json:"wind_speed_kmh"`
	Condition       string    `json:"condition"`
	ConditionCode   int       `json:"condition_code"`
	ObservedAt      time.Time `json:"observed_at"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details,omitempty"`
}

func newWeatherResponse(readings []domain.CurrentWeather, retrievedAt time.Time, source Source) weatherResponse {
	capitals := make([]capitalWeatherDTO, 0, len(readings))
	for _, reading := range readings {
		capitals = append(capitals, capitalWeatherDTO{
			Slug:            reading.Capital.Slug,
			Name:            reading.Capital.Name,
			StateCode:       reading.Capital.StateCode,
			Latitude:        reading.Capital.Latitude,
			Longitude:       reading.Capital.Longitude,
			TemperatureC:    reading.TemperatureCelsius,
			FeelsLikeC:      reading.FeelsLikeCelsius,
			HumidityPercent: reading.HumidityPercent,
			WindSpeedKmh:    reading.WindSpeedKmh,
			Condition:       reading.Condition(),
			ConditionCode:   int(reading.ConditionCode),
			ObservedAt:      reading.ObservedAt,
		})
	}

	return weatherResponse{
		Count:       len(capitals),
		RetrievedAt: retrievedAt,
		Source:      source,
		Capitals:    capitals,
	}
}

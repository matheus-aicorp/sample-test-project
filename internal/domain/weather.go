package domain

import "time"

// CurrentWeather is the condition observed at a capital at a given instant.
type CurrentWeather struct {
	Capital            Capital
	TemperatureCelsius float64
	FeelsLikeCelsius   float64
	HumidityPercent    int
	WindSpeedKmh       float64
	ConditionCode      WeatherCode
	ObservedAt         time.Time
}

// Condition renders ConditionCode as text. Derived rather than stored so the
// two can never disagree.
func (w CurrentWeather) Condition() string { return w.ConditionCode.Description() }

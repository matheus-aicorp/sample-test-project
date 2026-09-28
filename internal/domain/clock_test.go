package domain_test

import (
	"testing"
	"time"

	"github.com/matheus-aicorp/sample-test-project/internal/domain"
)

func TestSystemClockReturnsTheCurrentInstant(t *testing.T) {
	before := time.Now()
	got := domain.SystemClock{}.Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("Now() = %v, expected between %v and %v", got, before, after)
	}
}

func TestSystemClockSatisfiesClock(t *testing.T) {
	var clock domain.Clock = domain.SystemClock{}
	if clock.Now().IsZero() {
		t.Error("SystemClock returned the zero time")
	}
}

func TestCurrentWeatherConditionDerivesFromTheCode(t *testing.T) {
	tests := []struct {
		name string
		code domain.WeatherCode
		want string
	}{
		{name: "catalogued", code: domain.Thunderstorm, want: "Thunderstorm"},
		{name: "uncatalogued", code: domain.WeatherCode(4242), want: "Uncatalogued condition"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reading := domain.CurrentWeather{ConditionCode: tt.code}
			if got := reading.Condition(); got != tt.want {
				t.Errorf("Condition() = %q, want %q", got, tt.want)
			}
		})
	}
}

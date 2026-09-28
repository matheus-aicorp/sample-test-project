package domain_test

import (
	"testing"

	"github.com/matheus-aicorp/sample-test-project/internal/domain"
)

// declaredCodes must stay in sync with the constants in weather_code.go. It
// guards against adding a code and forgetting its description.
var declaredCodes = []domain.WeatherCode{
	domain.ClearSky, domain.MainlyClear, domain.PartlyCloudy, domain.Overcast,
	domain.Fog, domain.DepositingRimeFog,
	domain.LightDrizzle, domain.ModerateDrizzle, domain.DenseDrizzle,
	domain.LightFreezingDrizzle, domain.DenseFreezingDrizzle,
	domain.SlightRain, domain.ModerateRain, domain.HeavyRain,
	domain.LightFreezingRain, domain.HeavyFreezingRain,
	domain.SlightSnowFall, domain.ModerateSnowFall, domain.HeavySnowFall, domain.SnowGrains,
	domain.SlightRainShowers, domain.ModerateRainShowers, domain.ViolentRainShowers,
	domain.SlightSnowShowers, domain.HeavySnowShowers,
	domain.Thunderstorm, domain.ThunderstormSlightHail, domain.ThunderstormHeavyHail,
}

func TestWeatherCodeDescription(t *testing.T) {
	tests := []struct {
		name string
		code domain.WeatherCode
		want string
	}{
		{name: "clear sky", code: domain.ClearSky, want: "Clear sky"},
		{name: "partly cloudy", code: domain.PartlyCloudy, want: "Partly cloudy"},
		{name: "heavy rain", code: domain.HeavyRain, want: "Heavy rain"},
		{name: "thunderstorm with heavy hail", code: domain.ThunderstormHeavyHail, want: "Thunderstorm with heavy hail"},
		{name: "uncatalogued code falls back", code: domain.WeatherCode(4242), want: "Uncatalogued condition"},
		{name: "negative code falls back", code: domain.WeatherCode(-1), want: "Uncatalogued condition"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.Description(); got != tt.want {
				t.Errorf("Description() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEveryDeclaredCodeHasADescription(t *testing.T) {
	if len(declaredCodes) != 28 {
		t.Errorf("declaredCodes has %d entries, want 28 (WMO table)", len(declaredCodes))
	}

	seen := make(map[domain.WeatherCode]struct{}, len(declaredCodes))
	for _, code := range declaredCodes {
		if _, duplicate := seen[code]; duplicate {
			t.Errorf("code %d listed twice", code)
		}
		seen[code] = struct{}{}

		if !code.Known() {
			t.Errorf("code %d is not catalogued", code)
		}
		if got := code.Description(); got == "Uncatalogued condition" {
			t.Errorf("code %d has no description", code)
		}
	}
}

func TestWeatherCodeKnown(t *testing.T) {
	if !domain.ClearSky.Known() {
		t.Error("ClearSky should be known")
	}
	if domain.WeatherCode(4242).Known() {
		t.Error("uncatalogued code should not be known")
	}
}

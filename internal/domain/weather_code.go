package domain

// WeatherCode is a WMO 4677 interpretation code, the table Open-Meteo uses.
type WeatherCode int

const (
	ClearSky               WeatherCode = 0
	MainlyClear            WeatherCode = 1
	PartlyCloudy           WeatherCode = 2
	Overcast               WeatherCode = 3
	Fog                    WeatherCode = 45
	DepositingRimeFog      WeatherCode = 48
	LightDrizzle           WeatherCode = 51
	ModerateDrizzle        WeatherCode = 53
	DenseDrizzle           WeatherCode = 55
	LightFreezingDrizzle   WeatherCode = 56
	DenseFreezingDrizzle   WeatherCode = 57
	SlightRain             WeatherCode = 61
	ModerateRain           WeatherCode = 63
	HeavyRain              WeatherCode = 65
	LightFreezingRain      WeatherCode = 66
	HeavyFreezingRain      WeatherCode = 67
	SlightSnowFall         WeatherCode = 71
	ModerateSnowFall       WeatherCode = 73
	HeavySnowFall          WeatherCode = 75
	SnowGrains             WeatherCode = 77
	SlightRainShowers      WeatherCode = 80
	ModerateRainShowers    WeatherCode = 81
	ViolentRainShowers     WeatherCode = 82
	SlightSnowShowers      WeatherCode = 85
	HeavySnowShowers       WeatherCode = 86
	Thunderstorm           WeatherCode = 95
	ThunderstormSlightHail WeatherCode = 96
	ThunderstormHeavyHail  WeatherCode = 99
)

const unknownCondition = "Uncatalogued condition"

var wmoDescriptions = map[WeatherCode]string{
	ClearSky:               "Clear sky",
	MainlyClear:            "Mainly clear",
	PartlyCloudy:           "Partly cloudy",
	Overcast:               "Overcast",
	Fog:                    "Fog",
	DepositingRimeFog:      "Depositing rime fog",
	LightDrizzle:           "Light drizzle",
	ModerateDrizzle:        "Moderate drizzle",
	DenseDrizzle:           "Dense drizzle",
	LightFreezingDrizzle:   "Light freezing drizzle",
	DenseFreezingDrizzle:   "Dense freezing drizzle",
	SlightRain:             "Slight rain",
	ModerateRain:           "Moderate rain",
	HeavyRain:              "Heavy rain",
	LightFreezingRain:      "Light freezing rain",
	HeavyFreezingRain:      "Heavy freezing rain",
	SlightSnowFall:         "Slight snow fall",
	ModerateSnowFall:       "Moderate snow fall",
	HeavySnowFall:          "Heavy snow fall",
	SnowGrains:             "Snow grains",
	SlightRainShowers:      "Slight rain showers",
	ModerateRainShowers:    "Moderate rain showers",
	ViolentRainShowers:     "Violent rain showers",
	SlightSnowShowers:      "Slight snow showers",
	HeavySnowShowers:       "Heavy snow showers",
	Thunderstorm:           "Thunderstorm",
	ThunderstormSlightHail: "Thunderstorm with slight hail",
	ThunderstormHeavyHail:  "Thunderstorm with heavy hail",
}

// Description returns readable text for the code. Uncatalogued codes fall back
// to a generic string instead of failing: the WMO can publish new codes at any
// time and that must not break the API.
func (c WeatherCode) Description() string {
	if description, ok := wmoDescriptions[c]; ok {
		return description
	}
	return unknownCondition
}

// Known reports whether the code is in the catalogued WMO table.
func (c WeatherCode) Known() bool {
	_, ok := wmoDescriptions[c]
	return ok
}

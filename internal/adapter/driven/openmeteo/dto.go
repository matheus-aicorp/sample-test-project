package openmeteo

// Shared by query building and decoding, so the two cannot drift apart.
const currentFields = "temperature_2m,apparent_temperature,relative_humidity_2m,wind_speed_10m,weather_code"

// forecastResponse models /v1/forecast. Open-Meteo returns a bare object for
// a single coordinate and an array for several, so decoding goes through
// json.RawMessage before picking a shape.
type forecastResponse struct {
	Latitude         float64         `json:"latitude"`
	Longitude        float64         `json:"longitude"`
	UTCOffsetSeconds int             `json:"utc_offset_seconds"`
	Timezone         string          `json:"timezone"`
	Current          *currentWeather `json:"current"`
	Failed           bool            `json:"error"`
	Reason           string          `json:"reason"`
}

// Pointers keep a missing variable distinguishable from a legitimate zero.
type currentWeather struct {
	Time                string   `json:"time"`
	Temperature2m       *float64 `json:"temperature_2m"`
	ApparentTemperature *float64 `json:"apparent_temperature"`
	RelativeHumidity2m  *int     `json:"relative_humidity_2m"`
	WindSpeed10m        *float64 `json:"wind_speed_10m"`
	WeatherCode         *int     `json:"weather_code"`
}

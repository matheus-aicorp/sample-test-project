package openmeteo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/matheus-aicorp/sample-test-project/internal/domain"
)

const (
	DefaultBaseURL = "https://api.open-meteo.com"

	forecastPath = "/v1/forecast"

	// Open-Meteo snaps coordinates to its model grid, usually under 0.2°.
	// 0.5° leaves room for that while still catching a reordered batch:
	// São Paulo and Rio de Janeiro are ~0.65° apart.
	coordinateToleranceDegrees = 0.5

	maxResponseBytes = 1 << 20
)

var (
	ErrUnexpectedStatus = errors.New("open-meteo returned an unexpected status")
	ErrMalformedPayload = errors.New("open-meteo returned a malformed payload")
	ErrMismatchedBatch  = errors.New("open-meteo response does not match the requested capitals")
)

type Config struct {
	BaseURL   string
	Timeout   time.Duration
	Timezone  string
	UserAgent string

	// Doer optionally replaces the default *http.Client, letting callers bring
	// their own transport (tracing, retries, custom TLS) and tests inject a
	// mock to exercise transport-level failures.
	Doer Doer
}

// Doer is the slice of *http.Client this adapter actually needs.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is the driven adapter for the Open-Meteo HTTP API and the only place
// that knows the provider's URLs, query vocabulary and JSON shape.
type Client struct {
	baseURL   string
	timezone  string
	userAgent string
	doer      Doer
}

var _ domain.WeatherProvider = (*Client)(nil)

func NewClient(cfg Config) (*Client, error) {
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("invalid open-meteo base url %q: %w", cfg.BaseURL, err)
	}

	timezone := cfg.Timezone
	if timezone == "" {
		timezone = "auto"
	}

	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = "capitals-weather/1.0 (+https://github.com/matheus-aicorp/sample-test-project)"
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	doer := cfg.Doer
	if doer == nil {
		doer = &http.Client{Timeout: timeout}
	}

	return &Client{
		baseURL:   baseURL,
		timezone:  timezone,
		userAgent: userAgent,
		doer:      doer,
	}, nil
}

// CurrentWeather fetches every capital in a single batched request.
func (c *Client) CurrentWeather(ctx context.Context, capitals []domain.Capital) ([]domain.CurrentWeather, error) {
	if len(capitals) == 0 {
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint(capitals), nil)
	if err != nil {
		return nil, fmt.Errorf("building open-meteo request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling open-meteo: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("reading open-meteo response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d %s", ErrUnexpectedStatus, resp.StatusCode, upstreamReason(body))
	}

	payloads, err := decodeBatch(body, len(capitals))
	if err != nil {
		return nil, err
	}

	readings := make([]domain.CurrentWeather, 0, len(payloads))
	for i, payload := range payloads {
		reading, err := toCurrentWeather(capitals[i], payload)
		if err != nil {
			return nil, err
		}
		readings = append(readings, reading)
	}
	return readings, nil
}

func (c *Client) endpoint(capitals []domain.Capital) string {
	latitudes := make([]string, 0, len(capitals))
	longitudes := make([]string, 0, len(capitals))
	for _, capital := range capitals {
		latitudes = append(latitudes, strconv.FormatFloat(capital.Latitude, 'f', 4, 64))
		longitudes = append(longitudes, strconv.FormatFloat(capital.Longitude, 'f', 4, 64))
	}

	query := url.Values{
		"latitude":  {strings.Join(latitudes, ",")},
		"longitude": {strings.Join(longitudes, ",")},
		"current":   {currentFields},
		"timezone":  {c.timezone},
	}

	return c.baseURL + forecastPath + "?" + query.Encode()
}

func decodeBatch(body []byte, expected int) ([]forecastResponse, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("%w: empty body", ErrMalformedPayload)
	}

	var batch []forecastResponse
	if err := json.Unmarshal(trimmed, &batch); err != nil {
		var single forecastResponse
		if singleErr := json.Unmarshal(trimmed, &single); singleErr != nil {
			return nil, fmt.Errorf("%w: %w", ErrMalformedPayload, err)
		}
		batch = []forecastResponse{single}
	}

	if len(batch) != expected {
		return nil, fmt.Errorf("%w: got %d entries, want %d", ErrMismatchedBatch, len(batch), expected)
	}
	return batch, nil
}

func toCurrentWeather(capital domain.Capital, payload forecastResponse) (domain.CurrentWeather, error) {
	if payload.Failed {
		return domain.CurrentWeather{}, fmt.Errorf(
			"%w: open-meteo reported an error for %s: %s", ErrMalformedPayload, capital.Slug, payload.Reason,
		)
	}
	if payload.Current == nil {
		return domain.CurrentWeather{}, fmt.Errorf("%w: missing current block for %s", ErrMalformedPayload, capital.Slug)
	}
	// A reordered batch would otherwise attribute one city's temperature to
	// another with no visible symptom.
	if !coordinatesMatch(capital, payload) {
		return domain.CurrentWeather{}, fmt.Errorf(
			"%w: entry for %s came back at %.4f,%.4f, expected around %.4f,%.4f",
			ErrMismatchedBatch, capital.Slug, payload.Latitude, payload.Longitude, capital.Latitude, capital.Longitude,
		)
	}

	current := payload.Current
	if current.Temperature2m == nil {
		// Temperature is the one field this endpoint promises, so its absence
		// is a hard failure rather than a silent zero.
		return domain.CurrentWeather{}, fmt.Errorf("%w: missing temperature for %s", ErrMalformedPayload, capital.Slug)
	}

	observedAt, err := parseTimestamp(current.Time, payload.Timezone, payload.UTCOffsetSeconds)
	if err != nil {
		return domain.CurrentWeather{}, fmt.Errorf("%w: %s for %s: %w", ErrMalformedPayload, current.Time, capital.Slug, err)
	}

	return domain.CurrentWeather{
		Capital:            capital,
		TemperatureCelsius: *current.Temperature2m,
		FeelsLikeCelsius:   floatOr(current.ApparentTemperature, 0),
		HumidityPercent:    intOr(current.RelativeHumidity2m, 0),
		WindSpeedKmh:       floatOr(current.WindSpeed10m, 0),
		ConditionCode:      domain.WeatherCode(intOr(current.WeatherCode, -1)),
		ObservedAt:         observedAt,
	}, nil
}

func coordinatesMatch(capital domain.Capital, payload forecastResponse) bool {
	return math.Abs(payload.Latitude-capital.Latitude) <= coordinateToleranceDegrees &&
		math.Abs(payload.Longitude-capital.Longitude) <= coordinateToleranceDegrees
}

// parseTimestamp reads Open-Meteo's local wall-clock time ("2026-09-27T20:30")
// using the offset the response itself reports, so no tzdata is needed at runtime.
func parseTimestamp(value, timezone string, utcOffsetSeconds int) (time.Time, error) {
	location := time.FixedZone(timezone, utcOffsetSeconds)
	for _, layout := range []string{"2006-01-02T15:04", time.RFC3339} {
		if parsed, err := time.ParseInLocation(layout, value, location); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp format %q", value)
}

func floatOr(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}

func intOr(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func upstreamReason(body []byte) string {
	var envelope struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Reason == "" {
		return strings.TrimSpace(string(body))
	}
	return envelope.Reason
}

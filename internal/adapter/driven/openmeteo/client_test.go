package openmeteo_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/matheus-aicorp/sample-test-project/internal/adapter/driven/openmeteo"
	"github.com/matheus-aicorp/sample-test-project/internal/domain"
	"github.com/matheus-aicorp/sample-test-project/internal/mocks"
)

var (
	saoPaulo = domain.Capital{Slug: "sao-paulo", Name: "São Paulo", StateCode: "SP", Latitude: -23.5505, Longitude: -46.6333}
	curitiba = domain.Capital{Slug: "curitiba", Name: "Curitiba", StateCode: "PR", Latitude: -25.4284, Longitude: -49.2733}
)

func newClient(t *testing.T, handler http.HandlerFunc) *openmeteo.Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := openmeteo.NewClient(openmeteo.Config{
		BaseURL:  server.URL,
		Timeout:  5 * time.Second,
		Timezone: "America/Sao_Paulo",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func jsonResponder(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestCurrentWeatherBatchArray(t *testing.T) {
	var gotQuery string

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		jsonResponder(http.StatusOK, `[
			{"latitude":-23.5505,"longitude":-46.6333,"utc_offset_seconds":-10800,"timezone":"America/Sao_Paulo",
			 "current":{"time":"2026-09-27T20:30","temperature_2m":21.4,"apparent_temperature":22.1,
			            "relative_humidity_2m":61,"wind_speed_10m":10.4,"weather_code":2}},
			{"latitude":-25.4284,"longitude":-49.2733,"utc_offset_seconds":-10800,"timezone":"America/Sao_Paulo",
			 "current":{"time":"2026-09-27T20:30","temperature_2m":14.8,"apparent_temperature":13.9,
			            "relative_humidity_2m":88,"wind_speed_10m":7.2,"weather_code":61}}
		]`)(w, r)
	})

	readings, err := client.CurrentWeather(context.Background(), []domain.Capital{saoPaulo, curitiba})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(readings) != 2 {
		t.Fatalf("got %d readings, want 2", len(readings))
	}

	first := readings[0]
	if first.Capital.Slug != "sao-paulo" {
		t.Errorf("first reading slug = %q, want sao-paulo", first.Capital.Slug)
	}
	if first.TemperatureCelsius != 21.4 {
		t.Errorf("TemperatureCelsius = %v, want 21.4", first.TemperatureCelsius)
	}
	if first.FeelsLikeCelsius != 22.1 {
		t.Errorf("FeelsLikeCelsius = %v, want 22.1", first.FeelsLikeCelsius)
	}
	if first.HumidityPercent != 61 {
		t.Errorf("HumidityPercent = %d, want 61", first.HumidityPercent)
	}
	if first.WindSpeedKmh != 10.4 {
		t.Errorf("WindSpeedKmh = %v, want 10.4", first.WindSpeedKmh)
	}
	if first.ConditionCode != domain.PartlyCloudy {
		t.Errorf("ConditionCode = %d, want %d", first.ConditionCode, domain.PartlyCloudy)
	}
	if got := first.Condition(); got != "Partly cloudy" {
		t.Errorf("Condition() = %q, want %q", got, "Partly cloudy")
	}

	// 20:30 at UTC-3 is 23:30 UTC.
	wantObserved := time.Date(2026, 9, 27, 23, 30, 0, 0, time.UTC)
	if !first.ObservedAt.Equal(wantObserved) {
		t.Errorf("ObservedAt = %v, want %v", first.ObservedAt, wantObserved)
	}

	if readings[1].Capital.Slug != "curitiba" || readings[1].TemperatureCelsius != 14.8 {
		t.Errorf("second reading = %+v", readings[1])
	}

	for _, expected := range []string{"latitude=-23.5505%2C-25.4284", "longitude=-46.6333%2C-49.2733", "temperature_2m"} {
		if !strings.Contains(gotQuery, expected) {
			t.Errorf("query %q does not contain %q", gotQuery, expected)
		}
	}
}

func TestCurrentWeatherSingleObjectPayload(t *testing.T) {
	client := newClient(t, jsonResponder(http.StatusOK,
		`{"latitude":-23.5505,"longitude":-46.6333,"utc_offset_seconds":-10800,"timezone":"America/Sao_Paulo",
		  "current":{"time":"2026-09-27T20:30","temperature_2m":19.2,"apparent_temperature":20.1,
		             "relative_humidity_2m":86,"wind_speed_10m":9.7,"weather_code":3}}`))

	readings, err := client.CurrentWeather(context.Background(), []domain.Capital{saoPaulo})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(readings) != 1 {
		t.Fatalf("got %d readings, want 1 (bare object must be normalised)", len(readings))
	}
	if readings[0].TemperatureCelsius != 19.2 {
		t.Errorf("TemperatureCelsius = %v, want 19.2", readings[0].TemperatureCelsius)
	}
}

func TestCurrentWeatherOptionalFieldsMissing(t *testing.T) {
	client := newClient(t, jsonResponder(http.StatusOK,
		`{"latitude":-23.5505,"longitude":-46.6333,"utc_offset_seconds":0,"timezone":"UTC",
		  "current":{"time":"2026-09-27T20:30","temperature_2m":18.0}}`))

	readings, err := client.CurrentWeather(context.Background(), []domain.Capital{saoPaulo})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if readings[0].HumidityPercent != 0 || readings[0].WindSpeedKmh != 0 {
		t.Errorf("missing optional fields should default to zero, got %+v", readings[0])
	}
	if readings[0].ConditionCode.Known() {
		t.Error("absent weather_code must not resolve to a catalogued condition")
	}
}

func TestCurrentWeatherNoCapitalsSkipsTheCall(t *testing.T) {
	client := newClient(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("upstream must not be called for an empty capital list")
	})

	readings, err := client.CurrentWeather(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if readings != nil {
		t.Errorf("got %v, want nil", readings)
	}
}

func TestCurrentWeatherFailures(t *testing.T) {
	tests := []struct {
		name    string
		capital []domain.Capital
		status  int
		body    string
		want    error
	}{
		{
			name:    "rate limited",
			capital: []domain.Capital{saoPaulo},
			status:  http.StatusTooManyRequests,
			body:    `{"error":true,"reason":"Rate limit exceeded"}`,
			want:    openmeteo.ErrUnexpectedStatus,
		},
		{
			name:    "provider error envelope with 200",
			capital: []domain.Capital{saoPaulo},
			status:  http.StatusOK,
			body:    `{"latitude":-23.5505,"longitude":-46.6333,"error":true,"reason":"Cannot initialize WeatherAPI"}`,
			want:    openmeteo.ErrMalformedPayload,
		},
		{
			name:    "malformed json",
			capital: []domain.Capital{saoPaulo},
			status:  http.StatusOK,
			body:    `<html>not json</html>`,
			want:    openmeteo.ErrMalformedPayload,
		},
		{
			name:    "empty body",
			capital: []domain.Capital{saoPaulo},
			status:  http.StatusOK,
			body:    `   `,
			want:    openmeteo.ErrMalformedPayload,
		},
		{
			name:    "missing temperature",
			capital: []domain.Capital{saoPaulo},
			status:  http.StatusOK,
			body: `{"latitude":-23.5505,"longitude":-46.6333,"utc_offset_seconds":0,
			       "current":{"time":"2026-09-27T20:30","relative_humidity_2m":61}}`,
			want: openmeteo.ErrMalformedPayload,
		},
		{
			name:    "missing current block",
			capital: []domain.Capital{saoPaulo},
			status:  http.StatusOK,
			body:    `{"latitude":-23.5505,"longitude":-46.6333,"utc_offset_seconds":0,"current":null}`,
			want:    openmeteo.ErrMalformedPayload,
		},
		{
			name:    "unparseable timestamp",
			capital: []domain.Capital{saoPaulo},
			status:  http.StatusOK,
			body: `{"latitude":-23.5505,"longitude":-46.6333,"utc_offset_seconds":0,
			       "current":{"time":"not-a-time","temperature_2m":18.0}}`,
			want: openmeteo.ErrMalformedPayload,
		},
		{
			name:    "batch shorter than requested",
			capital: []domain.Capital{saoPaulo, curitiba},
			status:  http.StatusOK,
			body: `[{"latitude":-23.5505,"longitude":-46.6333,"utc_offset_seconds":0,
			        "current":{"time":"2026-09-27T20:30","temperature_2m":18.0}}]`,
			want: openmeteo.ErrMismatchedBatch,
		},
		{
			name:    "batch reordered, would silently misattribute temperatures",
			capital: []domain.Capital{saoPaulo, curitiba},
			status:  http.StatusOK,
			body: `[{"latitude":-25.4284,"longitude":-49.2733,"utc_offset_seconds":0,
			        "current":{"time":"2026-09-27T20:30","temperature_2m":14.8}},
			       {"latitude":-23.5505,"longitude":-46.6333,"utc_offset_seconds":0,
			        "current":{"time":"2026-09-27T20:30","temperature_2m":21.4}}]`,
			want: openmeteo.ErrMismatchedBatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newClient(t, jsonResponder(tt.status, tt.body))

			_, err := client.CurrentWeather(context.Background(), tt.capital)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestCurrentWeatherToleratesGridSnapping(t *testing.T) {
	client := newClient(t, jsonResponder(http.StatusOK,
		`{"latitude":-23.5149,"longitude":-46.6105,"utc_offset_seconds":-10800,
		  "current":{"time":"2026-09-27T20:30","temperature_2m":19.2}}`))

	if _, err := client.CurrentWeather(context.Background(), []domain.Capital{saoPaulo}); err != nil {
		t.Errorf("coordinates snapped to the model grid should still match: %v", err)
	}
}

func TestCurrentWeatherPropagatesCancellation(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	client := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := client.CurrentWeather(ctx, []domain.Capital{saoPaulo}); err == nil {
		t.Fatal("expected an error from a canceled context")
	}
}

func TestNewClient(t *testing.T) {
	t.Run("zero config falls back to defaults", func(t *testing.T) {
		if _, err := openmeteo.NewClient(openmeteo.Config{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects an unparsable base url", func(t *testing.T) {
		if _, err := openmeteo.NewClient(openmeteo.Config{BaseURL: "://invalid"}); err == nil {
			t.Fatal("expected an error for an invalid base URL")
		}
	})
}

// newMockClient injects a gomock Doer, which is the only way to exercise
// transport-level failures (dial errors, timeouts) without a real socket.
func newMockClient(t *testing.T, doer openmeteo.Doer) *openmeteo.Client {
	t.Helper()

	client, err := openmeteo.NewClient(openmeteo.Config{
		BaseURL:  openmeteo.DefaultBaseURL,
		Timezone: "America/Sao_Paulo",
		Doer:     doer,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestCurrentWeatherTransportFailures(t *testing.T) {
	tests := []struct {
		name    string
		doerErr error
	}{
		{name: "connection refused", doerErr: errors.New("dial tcp 216.239.32.5:443: connect: connection refused")},
		{name: "dns failure", doerErr: errors.New("dial tcp: lookup api.open-meteo.com: no such host")},
		{name: "client timeout", doerErr: &url.Error{Op: "Get", URL: openmeteo.DefaultBaseURL, Err: context.DeadlineExceeded}},
		{name: "tls handshake failure", doerErr: errors.New("remote error: tls: internal error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			doer := mocks.NewMockDoer(ctrl)
			doer.EXPECT().Do(gomock.Any()).Return(nil, tt.doerErr)

			_, err := newMockClient(t, doer).CurrentWeather(context.Background(), []domain.Capital{saoPaulo})
			if err == nil {
				t.Fatal("expected an error")
			}
			if !errors.Is(err, tt.doerErr) {
				t.Errorf("cause %v should stay reachable, got %v", tt.doerErr, err)
			}
			// A transport failure must not be reported as a payload problem.
			if errors.Is(err, openmeteo.ErrMalformedPayload) || errors.Is(err, openmeteo.ErrMismatchedBatch) {
				t.Errorf("transport failure misclassified as a payload error: %v", err)
			}
		})
	}
}

// The HTTP layer maps DeadlineExceeded to 504, so the client must preserve it
// rather than swallowing it into an opaque error.
func TestCurrentWeatherTimeoutStaysDetectable(t *testing.T) {
	ctrl := gomock.NewController(t)
	doer := mocks.NewMockDoer(ctrl)
	doer.EXPECT().Do(gomock.Any()).Return(nil, &url.Error{Op: "Get", URL: openmeteo.DefaultBaseURL, Err: context.DeadlineExceeded})

	_, err := newMockClient(t, doer).CurrentWeather(context.Background(), []domain.Capital{saoPaulo})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("errors.Is(err, context.DeadlineExceeded) = false for %v", err)
	}
}

// brokenBody fails on the first Read, simulating a connection dropped mid-response.
type brokenBody struct{}

func (brokenBody) Read([]byte) (int, error) { return 0, errors.New("unexpected EOF") }
func (brokenBody) Close() error             { return nil }

func TestCurrentWeatherBodyReadFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	doer := mocks.NewMockDoer(ctrl)
	doer.EXPECT().Do(gomock.Any()).Return(&http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       brokenBody{},
	}, nil)

	_, err := newMockClient(t, doer).CurrentWeather(context.Background(), []domain.Capital{saoPaulo})
	if err == nil {
		t.Fatal("expected an error when the response body cannot be read")
	}
	if !strings.Contains(err.Error(), "unexpected EOF") {
		t.Errorf("cause not surfaced: %v", err)
	}
}

// A non-JSON error body must still reach the caller's logs verbatim instead of
// being replaced by an empty reason.
func TestCurrentWeatherNonJSONErrorBody(t *testing.T) {
	client := newClient(t, jsonResponder(http.StatusBadGateway, "502 Bad Gateway: upstream nginx"))

	_, err := client.CurrentWeather(context.Background(), []domain.Capital{saoPaulo})
	if !errors.Is(err, openmeteo.ErrUnexpectedStatus) {
		t.Fatalf("error = %v, want ErrUnexpectedStatus", err)
	}
	if !strings.Contains(err.Error(), "upstream nginx") {
		t.Errorf("raw body should be surfaced, got %v", err)
	}
}

func TestCurrentWeatherBuildsTheExpectedRequest(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		wantHost string
	}{
		{name: "plain base url", baseURL: "https://api.open-meteo.com", wantHost: "api.open-meteo.com"},
		{name: "trailing slash is trimmed", baseURL: "https://api.open-meteo.com/", wantHost: "api.open-meteo.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			doer := mocks.NewMockDoer(ctrl)

			var captured *http.Request
			doer.EXPECT().Do(gomock.Any()).DoAndReturn(func(req *http.Request) (*http.Response, error) {
				captured = req
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(
						`{"latitude":-23.5505,"longitude":-46.6333,"utc_offset_seconds":0,
						  "current":{"time":"2026-09-27T20:30","temperature_2m":18.0}}`)),
				}, nil
			})

			client, err := openmeteo.NewClient(openmeteo.Config{
				BaseURL:   tt.baseURL,
				Timezone:  "America/Sao_Paulo",
				UserAgent: "test-agent",
				Doer:      doer,
			})
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			ctx := context.Background()
			if _, err := client.CurrentWeather(ctx, []domain.Capital{saoPaulo}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if captured.Method != http.MethodGet {
				t.Errorf("method = %q, want GET", captured.Method)
			}
			if captured.URL.Host != tt.wantHost {
				t.Errorf("host = %q, want %q", captured.URL.Host, tt.wantHost)
			}
			if captured.URL.Path != "/v1/forecast" {
				t.Errorf("path = %q, want /v1/forecast", captured.URL.Path)
			}
			if captured.Header.Get("Accept") != "application/json" {
				t.Errorf("Accept = %q", captured.Header.Get("Accept"))
			}
			if captured.Header.Get("User-Agent") != "test-agent" {
				t.Errorf("User-Agent = %q", captured.Header.Get("User-Agent"))
			}
			if captured.Context() != ctx {
				t.Error("request must carry the caller context so cancellation propagates")
			}

			query := captured.URL.Query()
			if got := query.Get("latitude"); got != "-23.5505" {
				t.Errorf("latitude = %q", got)
			}
			if got := query.Get("longitude"); got != "-46.6333" {
				t.Errorf("longitude = %q", got)
			}
			if got := query.Get("timezone"); got != "America/Sao_Paulo" {
				t.Errorf("timezone = %q", got)
			}
			wantFields := "temperature_2m,apparent_temperature,relative_humidity_2m,wind_speed_10m,weather_code"
			if got := query.Get("current"); got != wantFields {
				t.Errorf("current = %q, want %q", got, wantFields)
			}
		})
	}
}

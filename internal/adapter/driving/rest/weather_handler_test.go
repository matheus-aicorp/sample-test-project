package rest_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/matheus-aicorp/sample-test-project/internal/adapter/driving/rest"
	"github.com/matheus-aicorp/sample-test-project/internal/application"
	"github.com/matheus-aicorp/sample-test-project/internal/domain"
	"github.com/matheus-aicorp/sample-test-project/internal/mocks"
)

var (
	saoPaulo = domain.Capital{Slug: "sao-paulo", Name: "São Paulo", StateCode: "SP", Latitude: -23.5505, Longitude: -46.6333}
	curitiba = domain.Capital{Slug: "curitiba", Name: "Curitiba", StateCode: "PR", Latitude: -25.4284, Longitude: -49.2733}

	observedAt  = time.Date(2026, 9, 27, 20, 30, 0, 0, time.FixedZone("America/Sao_Paulo", -3*60*60))
	retrievedAt = time.Date(2026, 9, 27, 23, 30, 5, 0, time.UTC)
)

type weatherBody struct {
	Count       int       `json:"count"`
	RetrievedAt time.Time `json:"retrieved_at"`
	Source      struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"source"`
	Capitals []struct {
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
	} `json:"capitals"`
}

type errorBody struct {
	Error struct {
		Code    string          `json:"code"`
		Message string          `json:"message"`
		Details json.RawMessage `json:"details"`
	} `json:"error"`
}

func newRouter(t *testing.T) (*mocks.MockWeatherQuery, http.Handler) {
	t.Helper()

	ctrl := gomock.NewController(t)
	query := mocks.NewMockWeatherQuery(ctrl)

	handler := rest.NewRouter(rest.Deps{
		Query:          query,
		Source:         rest.Source{Name: "Open-Meteo", URL: "https://open-meteo.com/"},
		RequestTimeout: 2 * time.Second,
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	return query, handler
}

func get(t *testing.T, handler http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	return recorder
}

// captureSlugs records the filter the handler extracted from the query string.
func captureSlugs(query *mocks.MockWeatherQuery, result *application.Result, err error) *[]string {
	var slugs []string
	query.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, got []string) (*application.Result, error) {
			slugs = got
			return result, err
		}).
		AnyTimes()
	return &slugs
}

func decode(t *testing.T, recorder *httptest.ResponseRecorder, into any) {
	t.Helper()

	if err := json.Unmarshal(recorder.Body.Bytes(), into); err != nil {
		t.Fatalf("decoding response %q: %v", recorder.Body.String(), err)
	}
}

func TestWeatherEndpointReturnsEveryCapital(t *testing.T) {
	query, handler := newRouter(t)

	result := &application.Result{
		Readings: []domain.CurrentWeather{
			{
				Capital:            saoPaulo,
				TemperatureCelsius: 21.4,
				FeelsLikeCelsius:   22.1,
				HumidityPercent:    61,
				WindSpeedKmh:       10.4,
				ConditionCode:      domain.PartlyCloudy,
				ObservedAt:         observedAt,
			},
			{
				Capital:            curitiba,
				TemperatureCelsius: 14.8,
				FeelsLikeCelsius:   13.9,
				HumidityPercent:    88,
				WindSpeedKmh:       7.2,
				ConditionCode:      domain.SlightRain,
				ObservedAt:         observedAt,
			},
		},
		RetrievedAt: retrievedAt,
	}
	slugs := captureSlugs(query, result, nil)

	recorder := get(t, handler, rest.WeatherPath)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
	if len(*slugs) != 0 {
		t.Errorf("expected no filter, got %v", *slugs)
	}

	var body weatherBody
	decode(t, recorder, &body)

	if body.Count != 2 || len(body.Capitals) != 2 {
		t.Fatalf("count = %d, capitals = %d, want 2 and 2", body.Count, len(body.Capitals))
	}
	if !body.RetrievedAt.Equal(retrievedAt) {
		t.Errorf("retrieved_at = %v, want %v", body.RetrievedAt, retrievedAt)
	}
	if body.Source.Name != "Open-Meteo" || body.Source.URL != "https://open-meteo.com/" {
		t.Errorf("source = %+v", body.Source)
	}

	first := body.Capitals[0]
	if first.Slug != "sao-paulo" || first.Name != "São Paulo" || first.StateCode != "SP" {
		t.Errorf("identity fields = %+v", first)
	}
	if first.TemperatureC != 21.4 || first.FeelsLikeC != 22.1 {
		t.Errorf("temperatures = %v / %v", first.TemperatureC, first.FeelsLikeC)
	}
	if first.HumidityPercent != 61 || first.WindSpeedKmh != 10.4 {
		t.Errorf("humidity/wind = %d / %v", first.HumidityPercent, first.WindSpeedKmh)
	}
	if first.Condition != "Partly cloudy" || first.ConditionCode != int(domain.PartlyCloudy) {
		t.Errorf("condition = %q (%d)", first.Condition, first.ConditionCode)
	}
	if !first.ObservedAt.Equal(observedAt) {
		t.Errorf("observed_at = %v, want %v", first.ObservedAt, observedAt)
	}
	if body.Capitals[1].Slug != "curitiba" || body.Capitals[1].Condition != "Slight rain" {
		t.Errorf("second capital = %+v", body.Capitals[1])
	}
}

func TestWeatherEndpointParsesFilters(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   []string
	}{
		{name: "comma separated", target: rest.WeatherPath + "?capital=sao-paulo,curitiba", want: []string{"sao-paulo", "curitiba"}},
		{name: "repeated parameter", target: rest.WeatherPath + "?capital=sao-paulo&capital=curitiba", want: []string{"sao-paulo", "curitiba"}},
		{name: "mixed and blank entries dropped", target: rest.WeatherPath + "?capital=%20sao-paulo%20,,&capital=curitiba", want: []string{"sao-paulo", "curitiba"}},
		{name: "single capital", target: rest.WeatherPath + "?capital=recife", want: []string{"recife"}},
		{name: "empty value means no filter", target: rest.WeatherPath + "?capital=", want: nil},
		{name: "case is forwarded as typed", target: rest.WeatherPath + "?capital=Sao-Paulo", want: []string{"Sao-Paulo"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, handler := newRouter(t)
			slugs := captureSlugs(query, &application.Result{}, nil)

			recorder := get(t, handler, tt.target)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body: %s)", recorder.Code, recorder.Body.String())
			}
			if fmt.Sprint(*slugs) != fmt.Sprint(tt.want) {
				t.Errorf("slugs = %v, want %v", *slugs, tt.want)
			}
		})
	}
}

func TestWeatherEndpointAppliesRequestTimeout(t *testing.T) {
	query, handler := newRouter(t)

	var gotDeadline time.Time
	query.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ []string) (*application.Result, error) {
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Error("context handed to the use case has no deadline")
			}
			gotDeadline = deadline
			return &application.Result{}, nil
		})

	get(t, handler, rest.WeatherPath)

	if remaining := time.Until(gotDeadline); remaining > 2*time.Second || remaining < time.Second {
		t.Errorf("deadline leaves %v, want roughly the configured 2s", remaining)
	}
}

func TestWeatherEndpointEmptyReadingsRendersEmptyArray(t *testing.T) {
	query, handler := newRouter(t)
	captureSlugs(query, &application.Result{Readings: nil, RetrievedAt: retrievedAt}, nil)

	recorder := get(t, handler, rest.WeatherPath)

	if !strings.Contains(recorder.Body.String(), `"capitals":[]`) {
		t.Errorf("empty readings must serialise as [], got %s", recorder.Body.String())
	}
}

func TestErrorResponses(t *testing.T) {
	upstream := errors.New("connection refused")

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantDetail string
	}{
		{
			name: "unknown capital",
			err: &domain.UnknownCapitalsError{
				Slugs:      []string{"campinas"},
				ValidSlugs: []string{"sao-paulo", "curitiba"},
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "unknown_capital",
			wantDetail: "campinas",
		},
		{
			name:       "invalid request",
			err:        &domain.InvalidRequestError{Message: "at most 50 capitals per request, got 51"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "upstream failure",
			err:        fmt.Errorf("%w: %w", domain.ErrWeatherUnavailable, upstream),
			wantStatus: http.StatusBadGateway,
			wantCode:   "upstream_failure",
		},
		{
			name:       "upstream timeout wins over the generic upstream failure",
			err:        fmt.Errorf("%w: %w", domain.ErrWeatherUnavailable, context.DeadlineExceeded),
			wantStatus: http.StatusGatewayTimeout,
			wantCode:   "upstream_timeout",
		},
		{
			name:       "canceled request",
			err:        context.Canceled,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "request_canceled",
		},
		{
			name:       "unhandled error is not disclosed",
			err:        errors.New("secret internal detail"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, handler := newRouter(t)
			captureSlugs(query, nil, tt.err)

			recorder := get(t, handler, rest.WeatherPath)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q", got)
			}

			var body errorBody
			decode(t, recorder, &body)
			if body.Error.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", body.Error.Code, tt.wantCode)
			}
			if body.Error.Message == "" {
				t.Error("message must not be empty")
			}
			if strings.Contains(body.Error.Message, "secret internal detail") {
				t.Error("internal error details must not leak to the client")
			}
			if tt.wantDetail != "" && !strings.Contains(string(body.Error.Details), tt.wantDetail) {
				t.Errorf("details = %s, want it to mention %q", body.Error.Details, tt.wantDetail)
			}
		})
	}
}

func TestUnknownCapitalDetailsListValidSlugs(t *testing.T) {
	query, handler := newRouter(t)
	captureSlugs(query, nil, &domain.UnknownCapitalsError{
		Slugs:      []string{"campinas", "xyz"},
		ValidSlugs: []string{"sao-paulo", "curitiba"},
	})

	recorder := get(t, handler, rest.WeatherPath+"?capital=campinas,xyz")

	var body errorBody
	decode(t, recorder, &body)

	var details struct {
		InvalidSlugs []string `json:"invalid_slugs"`
		ValidSlugs   []string `json:"valid_slugs"`
	}
	if err := json.Unmarshal(body.Error.Details, &details); err != nil {
		t.Fatalf("decoding details: %v", err)
	}
	if len(details.InvalidSlugs) != 2 || details.InvalidSlugs[0] != "campinas" {
		t.Errorf("invalid_slugs = %v", details.InvalidSlugs)
	}
	if len(details.ValidSlugs) != 2 {
		t.Errorf("valid_slugs = %v", details.ValidSlugs)
	}
}

func TestPanicBecomesA500(t *testing.T) {
	query, handler := newRouter(t)
	query.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ []string) (*application.Result, error) {
			panic("boom")
		})

	recorder := get(t, handler, rest.WeatherPath)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}

	var body errorBody
	decode(t, recorder, &body)
	if body.Error.Code != "internal_error" {
		t.Errorf("code = %q, want internal_error", body.Error.Code)
	}
	if strings.Contains(recorder.Body.String(), "boom") {
		t.Error("panic value must not leak to the client")
	}
}

func TestRouterRouting(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantCode   string
	}{
		{name: "health probe", method: http.MethodGet, target: rest.HealthPath, wantStatus: http.StatusOK},
		{name: "post to the weather route", method: http.MethodPost, target: rest.WeatherPath, wantStatus: http.StatusMethodNotAllowed, wantCode: "method_not_allowed"},
		{name: "unknown route", method: http.MethodGet, target: "/v1/unknown", wantStatus: http.StatusNotFound, wantCode: "not_found"},
		{name: "root", method: http.MethodGet, target: "/", wantStatus: http.StatusNotFound, wantCode: "not_found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, handler := newRouter(t)
			// The weather route is the only one that reaches the use case, and
			// these cases never call it successfully.
			captureSlugs(query, &application.Result{}, nil)

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(tt.method, tt.target, nil))

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if tt.wantStatus == http.StatusMethodNotAllowed {
				if allow := recorder.Header().Get("Allow"); !strings.Contains(allow, http.MethodGet) {
					t.Errorf("Allow header = %q, want it to advertise GET", allow)
				}
			}
			if tt.wantCode != "" {
				var body errorBody
				decode(t, recorder, &body)
				if body.Error.Code != tt.wantCode {
					t.Errorf("code = %q, want %q", body.Error.Code, tt.wantCode)
				}
			}
		})
	}
}

func TestHealthEndpointBody(t *testing.T) {
	_, handler := newRouter(t)

	recorder := get(t, handler, rest.HealthPath)

	if got := strings.TrimSpace(recorder.Body.String()); got != `{"status":"ok"}` {
		t.Errorf("body = %s", got)
	}
}

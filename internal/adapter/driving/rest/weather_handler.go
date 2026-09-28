package rest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/matheus-aicorp/sample-test-project/internal/application"
	"github.com/matheus-aicorp/sample-test-project/internal/domain"
)

// WeatherQuery is the application port this transport consumes. Declared by
// the consumer so the handler is testable with a stub.
type WeatherQuery interface {
	Execute(ctx context.Context, slugs []string) (*application.Result, error)
}

// WeatherHandler serves GET /v1/capitals/weather. It only translates between
// HTTP and the use case: parsing query parameters in, rendering JSON out.
type WeatherHandler struct {
	query   WeatherQuery
	source  Source
	timeout time.Duration
	logger  *slog.Logger
}

func NewWeatherHandler(query WeatherQuery, source Source, timeout time.Duration, logger *slog.Logger) *WeatherHandler {
	return &WeatherHandler{query: query, source: source, timeout: timeout, logger: logger}
}

func (h *WeatherHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.timeout)
		defer cancel()
	}

	result, err := h.query.Execute(ctx, parseCapitalFilter(r.URL.Query()))
	if err != nil {
		writeError(w, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, newWeatherResponse(result.Readings, result.RetrievedAt, h.source))
}

// parseCapitalFilter accepts ?capital=sao-paulo,curitiba and repeated
// ?capital=a&capital=b, returning an empty slice when no filter was given.
func parseCapitalFilter(query map[string][]string) []string {
	var slugs []string
	for _, value := range query["capital"] {
		for _, raw := range strings.Split(value, ",") {
			if slug := strings.TrimSpace(raw); slug != "" {
				slugs = append(slugs, slug)
			}
		}
	}
	return slugs
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		// Marshalling these DTOs cannot fail; if it ever does the headers are
		// still unwritten, so a plain 500 is safe.
		http.Error(w, `{"error":{"code":"internal_error","message":"failed to encode response"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// writeError maps domain and application errors onto HTTP statuses. Anything
// unrecognised becomes a 500 whose cause is logged but never disclosed.
func writeError(w http.ResponseWriter, logger *slog.Logger, err error) {
	var (
		unknown *domain.UnknownCapitalsError
		invalid *domain.InvalidRequestError

		status  int
		code    string
		message string
		details json.RawMessage
	)

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		status, code, message = http.StatusGatewayTimeout, "upstream_timeout", "the weather source did not answer in time"

	case errors.Is(err, context.Canceled):
		status, code, message = http.StatusServiceUnavailable, "request_canceled", "request canceled"

	case errors.As(err, &unknown):
		status, code, message = http.StatusBadRequest, "unknown_capital", unknown.Error()
		details = marshalDetails(map[string]any{
			"invalid_slugs": unknown.Slugs,
			"valid_slugs":   unknown.ValidSlugs,
		})

	case errors.As(err, &invalid):
		status, code, message = http.StatusBadRequest, "invalid_request", invalid.Message

	case errors.Is(err, domain.ErrWeatherUnavailable):
		status, code, message = http.StatusBadGateway, "upstream_failure", "could not fetch live weather data"
		logger.Error("upstream weather failure", slog.Any("error", err))

	default:
		status, code, message = http.StatusInternalServerError, "internal_error", "unexpected internal error"
		logger.Error("unhandled request error", slog.Any("error", err))
	}

	writeJSON(w, status, errorResponse{Error: errorDetail{Code: code, Message: message, Details: details}})
}

func marshalDetails(details map[string]any) json.RawMessage {
	encoded, err := json.Marshal(details)
	if err != nil {
		return nil
	}
	return encoded
}

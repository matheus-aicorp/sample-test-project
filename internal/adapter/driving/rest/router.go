package rest

import (
	"log/slog"
	"net/http"
	"time"
)

const (
	WeatherPath = "/v1/capitals/weather"
	HealthPath  = "/healthz"
)

// Deps is filled in by the composition root, which keeps this package free of
// infrastructure imports.
type Deps struct {
	Query          WeatherQuery
	Source         Source
	RequestTimeout time.Duration
	Logger         *slog.Logger
}

// routes maps each exposed path to the methods it accepts. Single source for
// both the 404 and the 405 answers.
var routes = map[string]string{
	WeatherPath: "GET, HEAD",
	HealthPath:  "GET, HEAD",
}

// NewRouter assembles the HTTP handler chain.
func NewRouter(deps Deps) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET "+WeatherPath, NewWeatherHandler(deps.Query, deps.Source, deps.RequestTimeout, deps.Logger))
	mux.HandleFunc("GET "+HealthPath, healthHandler)

	return WithRecovery(WithRequestLog(route(mux), deps.Logger), deps.Logger)
}

// route keeps every response inside the API's JSON envelope. A "/" catch-all
// pattern cannot: it matches wrong-method requests too and hides the 405, while
// the mux's own 405 answers in plain text.
func route(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allow, known := routes[r.URL.Path]
		if !known {
			notFoundHandler(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			methodNotAllowedHandler(w, allow)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

// healthHandler is a liveness probe. It touches no dependency, so a healthy
// answer means only that the process is up and accepting connections.
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func notFoundHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotFound, errorResponse{
		Error: errorDetail{Code: "not_found", Message: "unknown route"},
	})
}

func methodNotAllowedHandler(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeJSON(w, http.StatusMethodNotAllowed, errorResponse{
		Error: errorDetail{Code: "method_not_allowed", Message: "method not allowed on this route"},
	})
}

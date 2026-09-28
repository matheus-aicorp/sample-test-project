package rest

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// WithRequestLog logs one structured line per request.
func WithRequestLog(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		logger.Info("request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
			slog.Int("status", recorder.status),
			slog.Int("bytes", recorder.written),
			slog.Duration("duration", time.Since(started)),
			slog.String("remote_addr", r.RemoteAddr),
		)
	})
}

// WithRecovery turns a panic into a 500 so one bad request cannot kill the
// process. It must wrap everything else to also catch panics in middleware.
func WithRecovery(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered",
					slog.Any("panic", recovered),
					slog.String("path", r.URL.Path),
					slog.String("stack", string(debug.Stack())),
				)
				writeJSON(w, http.StatusInternalServerError, errorResponse{
					Error: errorDetail{Code: "internal_error", Message: "unexpected internal error"},
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status  int
	written int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	written, err := r.ResponseWriter.Write(body)
	r.written += written
	return written, err
}

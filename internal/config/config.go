package config

import (
	"fmt"
	"time"
)

// Config holds every runtime setting. Nothing else in the codebase reads the
// environment.
type Config struct {
	HTTPAddr        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	RequestTimeout  time.Duration

	UpstreamBaseURL  string
	UpstreamTimeout  time.Duration
	UpstreamTimezone string

	SourceName string
	SourceURL  string

	LogLevel  string
	LogFormat string
}

// Lookup reads a setting by name. os.Getenv satisfies it, and tests can pass
// a map-backed function instead.
type Lookup func(key string) string

func Load(lookup Lookup) (Config, error) {
	requestTimeout, err := duration(lookup, "REQUEST_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	upstreamTimeout, err := duration(lookup, "OPEN_METEO_TIMEOUT", 8*time.Second)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := duration(lookup, "SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	// The write deadline has to outlast the upstream call it is waiting on,
	// otherwise the server closes the connection before the handler answers.
	return Config{
		HTTPAddr:        text(lookup, "HTTP_ADDR", ":8080"),
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    requestTimeout + 5*time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: shutdownTimeout,
		RequestTimeout:  requestTimeout,
		UpstreamBaseURL: text(lookup, "OPEN_METEO_BASE_URL", "https://api.open-meteo.com"),
		UpstreamTimeout: upstreamTimeout,
		// "auto" resolves each capital's own IANA zone. Brazil spans UTC-5 to
		// UTC-2, so a single named zone would render wrong local times for
		// Rio Branco, Manaus, Cuiabá and others. It is also ~30x faster:
		// Open-Meteo takes 20-30s to batch 27 locations against a named zone,
		// under 1.5s with auto.
		UpstreamTimezone: text(lookup, "OPEN_METEO_TIMEZONE", "auto"),
		SourceName:       text(lookup, "SOURCE_NAME", "Open-Meteo"),
		SourceURL:        text(lookup, "SOURCE_URL", "https://open-meteo.com/"),
		LogLevel:         text(lookup, "LOG_LEVEL", "info"),
		LogFormat:        text(lookup, "LOG_FORMAT", "json"),
	}, nil
}

func text(lookup Lookup, key, fallback string) string {
	if value := lookup(key); value != "" {
		return value
	}
	return fallback
}

func duration(lookup Lookup, key string, fallback time.Duration) (time.Duration, error) {
	raw := lookup(key)
	if raw == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s=%q: %w", key, raw, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be positive, got %q", key, raw)
	}
	return parsed, nil
}

package config_test

import (
	"testing"
	"time"

	"github.com/matheus-aicorp/sample-test-project/internal/config"
)

func lookupFrom(values map[string]string) config.Lookup {
	return func(key string) string { return values[key] }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := config.Load(lookupFrom(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.RequestTimeout != 10*time.Second {
		t.Errorf("RequestTimeout = %v, want 10s", cfg.RequestTimeout)
	}
	if cfg.UpstreamBaseURL != "https://api.open-meteo.com" {
		t.Errorf("UpstreamBaseURL = %q", cfg.UpstreamBaseURL)
	}
	if cfg.UpstreamTimeout != 8*time.Second {
		t.Errorf("UpstreamTimeout = %v, want 8s", cfg.UpstreamTimeout)
	}
	if cfg.UpstreamTimezone != "auto" {
		t.Errorf("UpstreamTimezone = %q, want auto (Brazil spans four UTC offsets)", cfg.UpstreamTimezone)
	}
	if cfg.SourceName != "Open-Meteo" || cfg.SourceURL != "https://open-meteo.com/" {
		t.Errorf("source = %q / %q", cfg.SourceName, cfg.SourceURL)
	}
	if cfg.LogLevel != "info" || cfg.LogFormat != "json" {
		t.Errorf("logging = %q / %q", cfg.LogLevel, cfg.LogFormat)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 10s", cfg.ShutdownTimeout)
	}
}

func TestWriteTimeoutOutlastsTheRequestTimeout(t *testing.T) {
	cfg, err := config.Load(lookupFrom(map[string]string{"REQUEST_TIMEOUT": "30s"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.WriteTimeout <= cfg.RequestTimeout {
		t.Errorf("WriteTimeout %v must exceed RequestTimeout %v, or the server closes the connection mid-request",
			cfg.WriteTimeout, cfg.RequestTimeout)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := config.Load(lookupFrom(map[string]string{
		"HTTP_ADDR":           ":9090",
		"REQUEST_TIMEOUT":     "3s",
		"OPEN_METEO_BASE_URL": "https://example.test",
		"OPEN_METEO_TIMEOUT":  "2s",
		"OPEN_METEO_TIMEZONE": "UTC",
		"SOURCE_NAME":         "Custom",
		"SOURCE_URL":          "https://custom.test",
		"LOG_LEVEL":           "debug",
		"LOG_FORMAT":          "text",
		"SHUTDOWN_TIMEOUT":    "4s",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.RequestTimeout != 3*time.Second {
		t.Errorf("RequestTimeout = %v", cfg.RequestTimeout)
	}
	if cfg.UpstreamBaseURL != "https://example.test" || cfg.UpstreamTimeout != 2*time.Second {
		t.Errorf("upstream = %q / %v", cfg.UpstreamBaseURL, cfg.UpstreamTimeout)
	}
	if cfg.UpstreamTimezone != "UTC" {
		t.Errorf("UpstreamTimezone = %q", cfg.UpstreamTimezone)
	}
	if cfg.SourceName != "Custom" || cfg.SourceURL != "https://custom.test" {
		t.Errorf("source = %q / %q", cfg.SourceName, cfg.SourceURL)
	}
	if cfg.LogLevel != "debug" || cfg.LogFormat != "text" {
		t.Errorf("logging = %q / %q", cfg.LogLevel, cfg.LogFormat)
	}
	if cfg.ShutdownTimeout != 4*time.Second {
		t.Errorf("ShutdownTimeout = %v", cfg.ShutdownTimeout)
	}
}

func TestLoadRejectsBadDurations(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "unparsable request timeout", key: "REQUEST_TIMEOUT", value: "ten seconds"},
		{name: "bare number request timeout", key: "REQUEST_TIMEOUT", value: "10"},
		{name: "zero request timeout", key: "REQUEST_TIMEOUT", value: "0s"},
		{name: "negative upstream timeout", key: "OPEN_METEO_TIMEOUT", value: "-5s"},
		{name: "negative shutdown timeout", key: "SHUTDOWN_TIMEOUT", value: "-1s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := config.Load(lookupFrom(map[string]string{tt.key: tt.value})); err == nil {
				t.Fatalf("%s=%q should have been rejected", tt.key, tt.value)
			}
		})
	}
}

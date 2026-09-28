package logging_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/matheus-aicorp/sample-test-project/internal/logging"
)

func TestNewRejectsInvalidLevel(t *testing.T) {
	if _, err := logging.NewWithOutput("chatty", "json", &bytes.Buffer{}); err == nil {
		t.Fatal("expected an error for an invalid level")
	}
}

func TestNewDefaultsToStdout(t *testing.T) {
	logger, err := logging.New("info", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logger == nil {
		t.Fatal("New returned a nil logger")
	}
	if !logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("info level should be enabled")
	}

	if _, err := logging.New("nope", "json"); err == nil {
		t.Error("New should reject an invalid level too")
	}
}

func TestNewHonoursLevel(t *testing.T) {
	var out bytes.Buffer
	logger, err := logging.NewWithOutput("warn", "json", &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logger.Info("should be dropped")
	logger.Warn("should be kept")

	if strings.Contains(out.String(), "should be dropped") {
		t.Errorf("info message leaked through a warn-level logger: %s", out.String())
	}
	if !strings.Contains(out.String(), "should be kept") {
		t.Errorf("warn message missing: %s", out.String())
	}
}

func TestNewFormats(t *testing.T) {
	tests := []struct {
		format   string
		wantJSON bool
	}{
		{format: "json", wantJSON: true},
		{format: "JSON", wantJSON: true},
		{format: "", wantJSON: true},
		{format: "text", wantJSON: false},
		{format: "Text", wantJSON: false},
		{format: "anything-else", wantJSON: true},
	}

	for _, tt := range tests {
		t.Run("format="+tt.format, func(t *testing.T) {
			var out bytes.Buffer
			logger, err := logging.NewWithOutput("info", tt.format, &out)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			logger.Info("hello", slog.String("key", "value"))
			line := out.String()

			if tt.wantJSON && !strings.HasPrefix(strings.TrimSpace(line), "{") {
				t.Errorf("expected JSON output, got %q", line)
			}
			if !tt.wantJSON && strings.HasPrefix(strings.TrimSpace(line), "{") {
				t.Errorf("expected text output, got %q", line)
			}
			if !strings.Contains(line, "hello") {
				t.Errorf("message missing from output: %q", line)
			}
		})
	}
}

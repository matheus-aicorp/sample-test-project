package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// New builds the process logger. Format is either "json" (default) or "text".
func New(level, format string) (*slog.Logger, error) {
	return NewWithOutput(level, format, os.Stdout)
}

// NewWithOutput is New with an injectable sink, which keeps the level and
// format parsing testable without capturing stdout.
func NewWithOutput(level, format string, out io.Writer) (*slog.Logger, error) {
	var severity slog.Level
	if err := severity.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}

	options := &slog.HandlerOptions{Level: severity}
	if strings.EqualFold(format, "text") {
		return slog.New(slog.NewTextHandler(out, options)), nil
	}
	return slog.New(slog.NewJSONHandler(out, options)), nil
}

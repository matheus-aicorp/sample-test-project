package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/matheus-aicorp/sample-test-project/internal/adapter/driven/memory"
	"github.com/matheus-aicorp/sample-test-project/internal/adapter/driven/openmeteo"
	"github.com/matheus-aicorp/sample-test-project/internal/adapter/driving/rest"
	"github.com/matheus-aicorp/sample-test-project/internal/application"
	"github.com/matheus-aicorp/sample-test-project/internal/config"
	"github.com/matheus-aicorp/sample-test-project/internal/domain"
	"github.com/matheus-aicorp/sample-test-project/internal/logging"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

// run is the composition root: the only place allowed to know concrete types.
// Everything inward is wired through domain interfaces.
func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	logger, err := logging.New(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		return err
	}

	weatherProvider, err := openmeteo.NewClient(openmeteo.Config{
		BaseURL:  cfg.UpstreamBaseURL,
		Timeout:  cfg.UpstreamTimeout,
		Timezone: cfg.UpstreamTimezone,
	})
	if err != nil {
		return err
	}

	query := application.NewGetCapitalsWeather(memory.NewCapitalRepository(), weatherProvider, domain.SystemClock{})

	router := rest.NewRouter(rest.Deps{
		Query:          query,
		Source:         rest.Source{Name: cfg.SourceName, URL: cfg.SourceURL},
		RequestTimeout: cfg.RequestTimeout,
		Logger:         logger,
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- server.ListenAndServe() }()
	logger.Info("server listening", slog.String("addr", cfg.HTTPAddr), slog.String("route", rest.WeatherPath))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serveErr:
		return err
	case sig := <-stop:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	// Shutdown makes ListenAndServe return ErrServerClosed; drain the channel
	// so the goroutine exits cleanly.
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	logger.Info("server stopped")
	return nil
}

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"survive-bro/apps/backend/internal/app"
	"survive-bro/apps/backend/internal/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	application := app.New(cfg, logger)

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server starting", "address", cfg.HTTPAddress)
		serverErrors <- application.HTTP.Listen(cfg.HTTPAddress)
	}()

	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-signalContext.Done():
		logger.Info("server draining")
	case err := <-serverErrors:
		if err != nil {
			return fmt.Errorf("listen: %w", err)
		}
		return nil
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := application.Shutdown(shutdownContext); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("shutdown: %w", err)
	}
	logger.Info("server stopped")
	return nil
}

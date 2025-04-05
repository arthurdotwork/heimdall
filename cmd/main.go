package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/arthurdotwork/heimdall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	gateway, err := heimdall.New("./config.yaml")
	if err != nil {
		slog.ErrorContext(ctx, "failed to initialize gateway", "error", err)
		return
	}

	slog.InfoContext(ctx, "Starting gateway...")
	if err := gateway.Start(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to start gateway", "error", err)
		return
	}
}

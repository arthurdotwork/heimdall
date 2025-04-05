// cmd/heimdall/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/arthurdotwork/alog"
	"github.com/arthurdotwork/heimdall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slog.SetDefault(alog.Logger(alog.WithAttrs(slog.String("app", "heimdall"))))

	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	gateway, err := heimdall.New(*configPath)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create gateway", "error", err)
		return
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.InfoContext(ctx, "starting heimdall", "addr", fmt.Sprintf("0.0.0.0:%d", gateway.Config().Gateway.Port))
	if err := gateway.Start(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to start gateway", "error", err)
		return
	}
}

// cmd/heimdall/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/arthurdotwork/alog"
	"github.com/arthurdotwork/heimdall"
	"github.com/arthurdotwork/heimdall/middleware"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slog.SetDefault(alog.Logger(alog.WithAttrs(slog.String("app", "heimdall"))))

	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Register default middleware with the global registry
	middleware.RegisterDefaults()

	_ = heimdall.RegisterMiddleware("basicAuth", requireBasicAuthMiddleware())

	gateway, err := heimdall.New(*configPath)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create gateway", "error", err)
		return
	}

	// Add a programmatic middleware (not from config)
	gateway.Use(createCustomMiddleware())

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.InfoContext(ctx, "starting heimdall", "addr", fmt.Sprintf("0.0.0.0:%d", gateway.Config().Gateway.Port))
	if err := gateway.Start(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to start gateway", "error", err)
		return
	}
}

type requestIDKey struct{}

// Example of programmatically creating a custom middleware
func createCustomMiddleware() heimdall.Middleware {
	return heimdall.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add a custom request ID
			requestID := fmt.Sprintf("%d", time.Now().UnixNano())
			ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)

			// Add request ID to response headers
			w.Header().Set("X-Request-Id", requestID)
			r.Header.Set("X-Request-Id", requestID)

			// Continue with the modified context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
}

func requireBasicAuthMiddleware() heimdall.Middleware {
	return heimdall.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok || username != "admin" || password != "password" {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
}

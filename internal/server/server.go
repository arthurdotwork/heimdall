package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/arthurdotwork/heimdall/internal/config"
	"github.com/arthurdotwork/heimdall/internal/middleware"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	cfg         config.GatewayConfig
	handler     http.Handler
	middlewares *middleware.MiddlewareChain
}

func NewServer(cfg config.GatewayConfig, handler http.Handler) *Server {
	return &Server{
		cfg:         cfg,
		handler:     handler,
		middlewares: middleware.NewMiddlewareChain(),
	}
}

func (s *Server) Start(ctx context.Context) error {
	reqCtx, cancelRequest := context.WithCancel(context.Background())

	// Apply middleware chain to the handler
	finalHandler := s.middlewares.Then(s.handler)

	// Create the HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.cfg.Port),
		Handler:      finalHandler,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
		BaseContext: func(_ net.Listener) context.Context {
			return reqCtx
		},
	}

	gr, ctx := errgroup.WithContext(ctx)

	gr.Go(func() error {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
		defer cancel()

		timer := time.AfterFunc(s.cfg.ShutdownTimeout, cancelRequest)
		defer timer.Stop()

		return srv.Shutdown(shutdownCtx)
	})

	gr.Go(func() error {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	// Wait for both goroutines to complete
	return gr.Wait()
}

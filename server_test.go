package heimdall_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/arthurdotwork/heimdall"
	"github.com/stretchr/testify/require"
)

func TestServer_Start(t *testing.T) {
	t.Parallel()

	t.Run("it should start and shutdown the server gracefully", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		server := heimdall.NewServer(heimdall.GatewayConfig{Port: 8081}, handler)

		// We start the server in a goroutine
		done := make(chan error, 1)
		go func() {
			done <- server.Start(ctx)
		}()

		// Wait for server to be ready
		time.Sleep(100 * time.Millisecond)

		// Cancel the context to trigger shutdown
		cancel()

		select {
		case err := <-done:
			require.NoError(t, err, "server should shut down without errors")
		case <-time.After(2 * time.Second):
			t.Fatal("server did not shut down within expected time")
		}
	})

	t.Run("it should handle context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		server := heimdall.NewServer(heimdall.GatewayConfig{Port: 8082}, handler)

		startTime := time.Now()
		err := server.Start(ctx)
		require.NoError(t, err, "server should returns an http.ErrServerClosed error which is skipped")
		duration := time.Since(startTime)

		require.Less(t, duration, 1*time.Second, "server should exit quickly with canceled context")
	})

	t.Run("it should handle abrupt shutdown with long-running requests", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		requestProcessingStarted := make(chan bool)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestProcessingStarted <- true

			select {
			case <-time.After(15 * time.Second):
				w.WriteHeader(http.StatusOK)
			case <-r.Context().Done():
				w.WriteHeader(http.StatusGatewayTimeout)
			}
		})

		server := heimdall.NewServer(heimdall.GatewayConfig{Port: 8083, ShutdownTimeout: 2 * time.Second}, handler)

		serverDone := make(chan error, 1)
		go func() {
			serverDone <- server.Start(ctx)
		}()

		time.Sleep(100 * time.Millisecond)

		requestDone := make(chan bool)
		go func() {
			defer func() { requestDone <- true }()

			client := &http.Client{Timeout: 20 * time.Second}
			_, err := client.Get("http://localhost:8083")
			require.NoError(t, err)
		}()

		<-time.After(1 * time.Second)
		require.True(t, <-requestProcessingStarted)

		cancel()

		serverError := <-serverDone
		require.Error(t, context.DeadlineExceeded, serverError)
	})
}

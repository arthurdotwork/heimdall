package middleware_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	internalMiddleware "github.com/arthurdotwork/heimdall/internal/middleware"
	"github.com/arthurdotwork/heimdall/middleware"
	"github.com/stretchr/testify/require"
)

func TestLoggerMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("it should log the request and response", func(t *testing.T) {
		var logBuffer bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logBuffer, nil))
		slog.SetDefault(logger)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK")) //nolint:errcheck
		})

		chain := internalMiddleware.NewMiddlewareChain().
			Add(middleware.Logger())

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		chain.Then(handler).ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "OK", rec.Body.String())

		logs := logBuffer.String()
		require.Contains(t, logs, "request started")
		require.Contains(t, logs, "path=/test")
		require.Contains(t, logs, "method=GET")
		require.Contains(t, logs, "request completed")
		require.Contains(t, logs, "status=200")
	})

	t.Run("it should log the correct status code", func(t *testing.T) {
		var logBuffer bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logBuffer, nil))
		slog.SetDefault(logger)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		chain := internalMiddleware.NewMiddlewareChain().
			Add(middleware.Logger())

		req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
		rec := httptest.NewRecorder()

		chain.Then(handler).ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)

		logs := logBuffer.String()
		require.Contains(t, logs, "status=404")
	})

	t.Run("it should correctly log the response size", func(t *testing.T) {
		var logBuffer bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logBuffer, nil))
		slog.SetDefault(logger)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello, World!")) //nolint:errcheck
		})

		chain := internalMiddleware.NewMiddlewareChain().
			Add(middleware.Logger())

		req := httptest.NewRequest(http.MethodGet, "/size-test", nil)
		rec := httptest.NewRecorder()

		chain.Then(handler).ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "Hello, World!", rec.Body.String())

		logs := logBuffer.String()
		require.Contains(t, logs, "bytes=13")
	})

	t.Run("it should preserve the request context", func(t *testing.T) {
		type contextKey string
		const testKey contextKey = "test-key"

		var contextValue string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contextValue = r.Context().Value(testKey).(string)
			w.WriteHeader(http.StatusOK)
		})

		chain := internalMiddleware.NewMiddlewareChain().
			Add(middleware.Logger())

		req := httptest.NewRequest(http.MethodGet, "/context-test", nil)
		req = req.WithContext(context.WithValue(req.Context(), testKey, "test-value"))
		rec := httptest.NewRecorder()

		chain.Then(handler).ServeHTTP(rec, req)

		require.Equal(t, "test-value", contextValue)
	})
}

package heimdall_test

import (
	"net/http"
	"testing"

	"github.com/arthurdotwork/heimdall"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareRegistry(t *testing.T) {
	t.Parallel()

	t.Run("it should register and retrieve middleware", func(t *testing.T) {
		registry := heimdall.NewMiddlewareRegistry()

		middleware := heimdall.MiddlewareFunc(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Test", "test-value")
				next.ServeHTTP(w, r)
			})
		})

		err := registry.Register("test", middleware)
		require.NoError(t, err)

		retrieved, exists := registry.Get("test")
		require.True(t, exists)
		require.NotNil(t, retrieved)
	})

	t.Run("it should prevent duplicate registration", func(t *testing.T) {
		registry := heimdall.NewMiddlewareRegistry()

		middleware := heimdall.MiddlewareFunc(func(next http.Handler) http.Handler {
			return next
		})

		err := registry.Register("test", middleware)
		require.NoError(t, err)

		err = registry.Register("test", middleware)
		require.Error(t, err)
		require.Contains(t, err.Error(), "already registered")
	})

	t.Run("it should return false for non-existent middleware", func(t *testing.T) {
		registry := heimdall.NewMiddlewareRegistry()

		_, exists := registry.Get("non-existent")
		require.False(t, exists)
	})

	t.Run("it should retrieve multiple middleware by name", func(t *testing.T) {
		registry := heimdall.NewMiddlewareRegistry()

		middleware1 := heimdall.MiddlewareFunc(func(next http.Handler) http.Handler {
			return next
		})

		middleware2 := heimdall.MiddlewareFunc(func(next http.Handler) http.Handler {
			return next
		})

		_ = registry.Register("middleware1", middleware1)
		_ = registry.Register("middleware2", middleware2)

		middlewares, missing := registry.GetMultiple([]string{"middleware1", "middleware2", "non-existent"})

		require.Len(t, middlewares, 2)
		require.Len(t, missing, 1)
		require.Equal(t, "non-existent", missing[0])
	})

	t.Run("it should use the default registry for global functions", func(t *testing.T) {
		// Reset the default registry by re-creating it
		// Note: This is a test-only approach and might need adaptation based on how the default registry is implemented
		heimdall.ResetDefaultRegistry()

		middleware := heimdall.MiddlewareFunc(func(next http.Handler) http.Handler {
			return next
		})

		// Register using the global function
		err := heimdall.RegisterMiddleware("global-test", middleware)
		require.NoError(t, err)

		// Retrieve using the global function
		retrieved, exists := heimdall.GetMiddleware("global-test")
		require.True(t, exists)
		require.NotNil(t, retrieved)

		// Retrieve multiple using the global function
		middlewares, missing := heimdall.GetMiddlewares([]string{"global-test", "non-existent"})
		require.Len(t, middlewares, 1)
		require.Len(t, missing, 1)
		require.Equal(t, "non-existent", missing[0])
	})
}

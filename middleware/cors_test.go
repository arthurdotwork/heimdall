package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	internalMiddleware "github.com/arthurdotwork/heimdall/internal/middleware"
	"github.com/arthurdotwork/heimdall/middleware"
	"github.com/stretchr/testify/require"
)

func TestCORSMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("it should handle preflight OPTIONS requests", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called for preflight requests")
		})

		corsMiddleware := middleware.CORS(nil)
		chain := internalMiddleware.NewMiddlewareChain().Add(corsMiddleware)
		finalHandler := chain.Then(handler)

		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()

		finalHandler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
		require.Equal(t, "http://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
		require.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Methods"))
		require.NotEmpty(t, rec.Header().Get("Access-Control-Allow-Headers"))
		require.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("it should add CORS headers to normal requests", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		config := &middleware.CORSConfig{
			AllowCredentials: true,
		}
		corsMiddleware := middleware.CORS(config)
		chain := internalMiddleware.NewMiddlewareChain().Add(corsMiddleware)
		finalHandler := chain.Then(handler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()

		finalHandler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "http://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
		require.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("it should use default config when nil is provided", func(t *testing.T) {
		corsMiddleware := middleware.CORS(nil)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		chain := internalMiddleware.NewMiddlewareChain().Add(corsMiddleware)
		finalHandler := chain.Then(handler)

		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()

		finalHandler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Code)
		require.Equal(t, "http://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
		require.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "GET")
		require.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
		require.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	})
}

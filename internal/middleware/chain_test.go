package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arthurdotwork/heimdall/internal/middleware"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareChain(t *testing.T) {
	t.Parallel()

	t.Run("it should apply middleware in the correct order", func(t *testing.T) {
		var order []string

		middleware1 := middleware.MiddlewareFunc(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "before_middleware1")
				next.ServeHTTP(w, r)
				order = append(order, "after_middleware1")
			})
		})

		middleware2 := middleware.MiddlewareFunc(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "before_middleware2")
				next.ServeHTTP(w, r)
				order = append(order, "after_middleware2")
			})
		})

		final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "handler")
			w.WriteHeader(http.StatusOK)
		})

		chain := middleware.NewMiddlewareChain().
			Add(middleware1).
			Add(middleware2)

		handler := chain.Then(final)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		require.Equal(t, []string{
			"before_middleware1",
			"before_middleware2",
			"handler",
			"after_middleware2",
			"after_middleware1",
		}, order)
	})

	t.Run("it should work with MiddlewareFunc", func(t *testing.T) {
		var handlerCalled bool

		middlewareFunc := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Test", "test-value")
				next.ServeHTTP(w, r)
			})
		}

		final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		chain := middleware.NewMiddlewareChain().
			AddFunc(middlewareFunc)

		handler := chain.Then(final)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.True(t, handlerCalled)

		require.Equal(t, "test-value", rec.Header().Get("X-Test"))
	})

	t.Run("it should work with ThenFunc", func(t *testing.T) {
		var handlerCalled bool

		mw := middleware.MiddlewareFunc(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Test", "test-value")
				next.ServeHTTP(w, r)
			})
		})

		chain := middleware.NewMiddlewareChain().
			Add(mw)

		handler := chain.ThenFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		require.True(t, handlerCalled)

		require.Equal(t, "test-value", rec.Header().Get("X-Test"))
	})

	t.Run("it should default to http.DefaultServeMux when no final handler is provided", func(t *testing.T) {
		mw := middleware.MiddlewareFunc(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Test", "test-value")
				next.ServeHTTP(w, r)
			})
		})

		chain := middleware.NewMiddlewareChain().
			Add(mw)

		handler := chain.Then(nil)

		require.NotNil(t, handler)
	})

	t.Run("it should create a clone of the middleware chain", func(t *testing.T) {
		middleware1 := middleware.MiddlewareFunc(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware1", "true")
				next.ServeHTTP(w, r)
			})
		})

		middleware2 := middleware.MiddlewareFunc(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Middleware2", "true")
				next.ServeHTTP(w, r)
			})
		})

		chain1 := middleware.NewMiddlewareChain().Add(middleware1)

		chain2 := chain1.Clone().Add(middleware2)

		handler1 := chain1.ThenFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler2 := chain2.ThenFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		rec1 := httptest.NewRecorder()

		handler1.ServeHTTP(rec1, req1)

		require.Equal(t, "true", rec1.Header().Get("X-Middleware1"))
		require.Empty(t, rec1.Header().Get("X-Middleware2"))

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		rec2 := httptest.NewRecorder()

		handler2.ServeHTTP(rec2, req2)

		require.Equal(t, "true", rec2.Header().Get("X-Middleware1"))
		require.Equal(t, "true", rec2.Header().Get("X-Middleware2"))
	})
}

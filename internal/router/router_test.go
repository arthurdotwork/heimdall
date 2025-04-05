package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/arthurdotwork/heimdall/internal/config"
	"github.com/arthurdotwork/heimdall/internal/middleware"
	"github.com/arthurdotwork/heimdall/internal/router"
	"github.com/stretchr/testify/require"
)

func TestNewRouter(t *testing.T) {
	t.Parallel()

	t.Run("it should return an error if it can not parse the target URL", func(t *testing.T) {
		endpoints := []config.EndpointConfig{{Target: "://invalid"}}

		_, err := router.New(endpoints)
		require.Error(t, err)
	})

	t.Run("it should build the router", func(t *testing.T) {
		endpoints := []config.EndpointConfig{{Path: "/", Target: "https://www.google.com/", Method: "GET"}}

		router, err := router.New(endpoints)
		require.NoError(t, err)
		require.NotNil(t, router)
		require.NotEmpty(t, router.Routes)
		require.Len(t, router.Routes, 1)
		require.Len(t, router.Routes["/"], 1)
		require.Equal(t, "GET", router.Routes["/"]["GET"].Method)
		require.Equal(t, "https://www.google.com/", router.Routes["/"]["GET"].Target.String())
	})

	t.Run("it should build router with middleware", func(t *testing.T) {
		middleware.ResetDefaultRegistry()

		testMiddleware := middleware.Func(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Test", "test-value")
				next.ServeHTTP(w, r)
			})
		})

		_ = middleware.RegisterMiddleware("test-middleware", testMiddleware)

		endpoints := []config.EndpointConfig{
			{
				Path:        "/",
				Target:      "https://www.example.com/",
				Method:      "GET",
				Middlewares: []string{"test-middleware"},
			},
		}

		router, err := router.New(endpoints)
		require.NoError(t, err)
		require.NotNil(t, router)

		route, exists := router.GetRoute("/", "GET")
		require.True(t, exists)
		require.Equal(t, []string{"test-middleware"}, route.Middleware)
	})
}

func TestRouter_GetRoute(t *testing.T) {
	t.Parallel()

	endpoints := []config.EndpointConfig{{Path: "/foo", Method: "GET"}}
	router, err := router.New(endpoints)
	require.NoError(t, err)

	t.Run("it should return an error if it can not find the route by path", func(t *testing.T) {
		route, ok := router.GetRoute("/bar", "GET")
		require.False(t, ok)
		require.Nil(t, route)
	})

	t.Run("it should return an error if it can not find the route by method", func(t *testing.T) {
		route, ok := router.GetRoute("/foo", "POST")
		require.False(t, ok)
		require.Nil(t, route)
	})

	t.Run("it should return the route", func(t *testing.T) {
		route, ok := router.GetRoute("/foo", "GET")
		require.True(t, ok)
		require.NotNil(t, route)
		require.Equal(t, "/foo", route.OriginalPath)
		require.Equal(t, "GET", route.Method)
	})
}

func TestRouter_ApplyGlobalMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("it should apply global middleware to all routes", func(t *testing.T) {
		endpoints := []config.EndpointConfig{
			{Path: "/foo", Target: "http://example.com", Method: "GET"},
			{Path: "/bar", Target: "http://example.com", Method: "POST"},
		}

		router, err := router.New(endpoints)
		require.NoError(t, err)

		globalMiddleware := middleware.NewChain()
		globalMiddleware.AddFunc(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Global", "true")
				next.ServeHTTP(w, r)
			})
		})

		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router.ApplyGlobalMiddleware(globalMiddleware, finalHandler)

		fooRoute, _ := router.GetRoute("/foo", "GET")
		require.NotNil(t, fooRoute.Handler)

		barRoute, _ := router.GetRoute("/bar", "POST")
		require.NotNil(t, barRoute.Handler)

		req := httptest.NewRequest(http.MethodGet, "/foo", nil)
		rec := httptest.NewRecorder()

		fooRoute.Handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "true", rec.Header().Get("X-Global"))
	})

	t.Run("it should combine global and route-specific middleware", func(t *testing.T) {
		middleware.ResetDefaultRegistry()

		routeMiddleware := middleware.Func(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Route", "true")
				next.ServeHTTP(w, r)
			})
		})

		_ = middleware.RegisterMiddleware("route-middleware", routeMiddleware)

		endpoints := []config.EndpointConfig{
			{
				Path:        "/with-middleware",
				Target:      "http://example.com",
				Method:      "GET",
				Middlewares: []string{"route-middleware"},
			},
			{
				Path:   "/without-middleware",
				Target: "http://example.com",
				Method: "GET",
			},
		}

		router, err := router.New(endpoints)
		require.NoError(t, err)

		globalMiddleware := middleware.NewChain()
		globalMiddleware.AddFunc(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Global", "true")
				next.ServeHTTP(w, r)
			})
		})

		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router.ApplyGlobalMiddleware(globalMiddleware, finalHandler)

		withRoute, _ := router.GetRoute("/with-middleware", "GET")
		req1 := httptest.NewRequest(http.MethodGet, "/with-middleware", nil)
		rec1 := httptest.NewRecorder()

		withRoute.Handler.ServeHTTP(rec1, req1)

		require.Equal(t, http.StatusOK, rec1.Code)
		require.Equal(t, "true", rec1.Header().Get("X-Global"))
		require.Equal(t, "true", rec1.Header().Get("X-Route"))

		withoutRoute, _ := router.GetRoute("/without-middleware", "GET")
		req2 := httptest.NewRequest(http.MethodGet, "/without-middleware", nil)
		rec2 := httptest.NewRecorder()

		withoutRoute.Handler.ServeHTTP(rec2, req2)

		require.Equal(t, http.StatusOK, rec2.Code)
		require.Equal(t, "true", rec2.Header().Get("X-Global"))
		require.Empty(t, rec2.Header().Get("X-Route"))
	})
}

func TestRouter_SetHandler(t *testing.T) {
	t.Parallel()

	t.Run("it should set a handler for a route", func(t *testing.T) {
		endpoints := []config.EndpointConfig{
			{Path: "/test", Target: "http://example.com", Method: "GET"},
		}

		router, err := router.New(endpoints)
		require.NoError(t, err)

		route, exists := router.GetRoute("/test", "GET")
		require.True(t, exists)

		require.Nil(t, route.Handler)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Handler", "called")
			w.WriteHeader(http.StatusOK)
		})

		router.SetHandler(route, handler)

		require.NotNil(t, route.Handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		route.Handler.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "called", rec.Header().Get("X-Handler"))
	})
}

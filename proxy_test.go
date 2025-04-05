package heimdall_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/arthurdotwork/heimdall"
	"github.com/stretchr/testify/require"
)

type mockRouter struct {
	routes map[string]map[string]*heimdall.Route
}

func (m *mockRouter) addRoute(path, method string, route *heimdall.Route) {
	if m.routes == nil {
		m.routes = make(map[string]map[string]*heimdall.Route)
	}

	if m.routes[path] == nil {
		m.routes[path] = make(map[string]*heimdall.Route)
	}

	m.routes[path][method] = route
}

func (m *mockRouter) GetRoute(path, method string) (*heimdall.Route, bool) {
	if route, ok := m.routes[path][method]; ok {
		return route, true
	}

	return nil, false
}

func (m *mockRouter) ApplyGlobalMiddleware(middlewareChain *heimdall.MiddlewareChain, finalHandler http.Handler) {
	for _, methodRoutes := range m.routes {
		for _, route := range methodRoutes {
			route.Handler = middlewareChain.Then(finalHandler)
		}
	}
}

func TestProxyHandler_Serve(t *testing.T) {
	t.Parallel()

	t.Run("it should return an error if the route is not found", func(t *testing.T) {
		mockRouter := &mockRouter{}

		proxy := heimdall.NewProxyHandler(mockRouter)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()

		proxy.ServeHTTP(recorder, req)
		require.Equal(t, http.StatusNotFound, recorder.Code)
		require.Equal(t, "Route Not Found\n", recorder.Body.String())
	})

	t.Run("it should return an error if the method is not allowed", func(t *testing.T) {
		mockRouter := &mockRouter{}
		mockRouter.addRoute("/test", http.MethodGet, &heimdall.Route{})

		proxy := heimdall.NewProxyHandler(mockRouter)
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		recorder := httptest.NewRecorder()

		proxy.ServeHTTP(recorder, req)
		require.Equal(t, http.StatusNotFound, recorder.Code)
		require.Equal(t, "Route Not Found\n", recorder.Body.String())
	})

	t.Run("it should proxy the request", func(t *testing.T) {
		targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAgent := r.Header.Get("User-Agent")
			require.Contains(t, userAgent, "Heimdall")

			customHeader := r.Header.Values("X-Custom-Header")
			require.Len(t, customHeader, 2)
			require.Equal(t, "value1", customHeader[0])
			require.Equal(t, "value2", customHeader[1])

			forwardedHeader := r.Header.Get("X-Forwarded-Header")
			require.Equal(t, "forwarded-value", forwardedHeader)

			forbiddenHeader := r.Header.Get("X-Forbidden-Header")
			require.Empty(t, forbiddenHeader)

			w.WriteHeader(http.StatusOK)
		}))
		defer targetServer.Close()

		targetURL, err := url.Parse(targetServer.URL)
		require.NoError(t, err)

		mockRouter := &mockRouter{}
		mockRouter.addRoute("/test", http.MethodGet, &heimdall.Route{
			Target: targetURL,
			Method: http.MethodGet,
			Headers: http.Header{
				"X-Custom-Header": []string{"value1", "value2"},
			},
			AllowedHeaders: []string{"X-Forwarded-Header"},
		})

		proxy := heimdall.NewProxyHandler(mockRouter)
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-Header", "forwarded-value")
		req.Header.Set("X-Forbidden-Header", "forbidden-value")

		recorder := httptest.NewRecorder()
		proxy.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("it should handle transport errors", func(t *testing.T) {
		mockRouter := &mockRouter{}
		targetURL, _ := url.Parse("http://invalid.example.test:1")
		mockRouter.addRoute("/test", http.MethodGet, &heimdall.Route{
			Target: targetURL,
			Method: http.MethodGet,
		})

		// Create the proxy handler
		proxy := heimdall.NewProxyHandler(mockRouter)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		recorder := httptest.NewRecorder()

		proxy.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusBadGateway, recorder.Code)
		require.Contains(t, recorder.Body.String(), "Gateway error")
	})

	t.Run("it should return 503 when context is canceled", func(t *testing.T) {
		mockRouter := &mockRouter{}
		targetURL, _ := url.Parse("http://example.com")
		mockRouter.addRoute("/test", http.MethodGet, &heimdall.Route{
			Target: targetURL,
			Method: http.MethodGet,
		})

		proxy := heimdall.NewProxyHandler(mockRouter)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		// Serve the request with canceled context
		proxy.ServeHTTP(recorder, req)

		// Should return a 503 Service Unavailable
		require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
		require.Equal(t, "close", recorder.Header().Get("Connection"))
	})

	t.Run("it should handle context cancellation during request", func(t *testing.T) {
		targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
				// Context was canceled, don't respond
				return
			case <-time.After(100 * time.Millisecond):
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer targetServer.Close()

		targetURL, err := url.Parse(targetServer.URL)
		require.NoError(t, err)

		mockRouter := &mockRouter{}
		mockRouter.addRoute("/test", http.MethodGet, &heimdall.Route{
			Target: targetURL,
			Method: http.MethodGet,
		})

		proxy := heimdall.NewProxyHandler(mockRouter)

		ctx, cancel := context.WithCancel(context.Background())
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)
		recorder := httptest.NewRecorder()

		done := make(chan struct{})
		go func() {
			defer close(done)
			proxy.ServeHTTP(recorder, req)
		}()

		time.Sleep(50 * time.Millisecond)
		cancel()

		<-done

		require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
		require.Equal(t, "close", recorder.Header().Get("Connection"))
		require.Contains(t, recorder.Body.String(), "Gateway is shutting down")
	})

	t.Run("it should use route's handler if set", func(t *testing.T) {
		mockRouter := &mockRouter{}

		route := &heimdall.Route{
			Method: http.MethodGet,
		}

		customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Custom-Handler", "true")
			w.WriteHeader(http.StatusOK)
		})

		middlewareChain := heimdall.NewMiddlewareChain()

		route.Handler = middlewareChain.Then(customHandler)

		mockRouter.addRoute("/custom", http.MethodGet, route)

		proxy := heimdall.NewProxyHandler(mockRouter)

		req := httptest.NewRequest(http.MethodGet, "/custom", nil)
		rec := httptest.NewRecorder()

		proxy.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "true", rec.Header().Get("X-Custom-Handler"))
	})

	t.Run("it should initialize route handlers with middleware", func(t *testing.T) {
		// Create a test backend server that returns a specific response
		testBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Echo back any headers with X-Echo- prefix
			for name, values := range r.Header {
				if len(name) >= 7 && name[:7] == "X-Echo-" {
					for _, value := range values {
						w.Header().Set(name, value)
					}
				}
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK from backend")) //nolint:errcheck
		}))
		defer testBackend.Close()

		backendURL, _ := url.Parse(testBackend.URL)

		// Create a mock router with our test route
		mockRouter := &mockRouter{}
		route := &heimdall.Route{
			Target:         backendURL,
			Method:         http.MethodGet,
			Middlewares:    heimdall.NewMiddlewareChain(),
			AllowedHeaders: []string{"X-Echo-Global-Middleware"}, // Allow our test header!
		}
		mockRouter.addRoute("/test", http.MethodGet, route)

		// Create the proxy handler
		proxy := heimdall.NewProxyHandler(mockRouter)

		// Create middleware that adds headers to the request
		middlewareChain := heimdall.NewMiddlewareChain()
		middlewareChain.AddFunc(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Add a header that will be forwarded to the backend
				r.Header.Set("X-Echo-Global-Middleware", "true")
				next.ServeHTTP(w, r)
			})
		})

		// Initialize route handlers with middleware
		proxy.InitializeRouteHandlers(middlewareChain)

		// Ensure the route has a handler
		require.NotNil(t, route.Handler, "Route should have a handler after initialization")

		// Now make a request through the route's handler
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		route.Handler.ServeHTTP(rec, req)

		// Verify the response
		require.Equal(t, http.StatusOK, rec.Code, "Should return OK status")
		require.Equal(t, "OK from backend", rec.Body.String(), "Should return the backend response")

		// Check that our middleware header was passed to the backend and echoed back
		require.Equal(t, "true", rec.Header().Get("X-Echo-Global-Middleware"),
			"Middleware header should be passed to backend and echoed back")
	})
}

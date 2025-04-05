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
}

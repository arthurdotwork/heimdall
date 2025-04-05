package proxy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/arthurdotwork/heimdall/internal/middleware"
	"github.com/arthurdotwork/heimdall/internal/router"
)

const defaultUserAgent = "Heimdall/0.1"

type Router interface {
	GetRoute(path, method string) (*router.Route, bool)
	ApplyGlobalMiddleware(middlewareChain *middleware.Chain, finalHandler http.Handler)
}

type Handler struct {
	router    Router
	proxyFunc func(target *url.URL) *httputil.ReverseProxy
}

func NewHandler(router Router) *Handler {
	return &Handler{
		router: router,
		proxyFunc: func(target *url.URL) *httputil.ReverseProxy {
			return httputil.NewSingleHostReverseProxy(target)
		},
	}
}

func (p *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	select {
	case <-req.Context().Done():
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	default:
		// normal processing..
	}

	route, ok := p.router.GetRoute(req.URL.Path, req.Method)
	if !ok {
		http.Error(w, "Route Not Found", http.StatusNotFound)
		return
	}

	// If the route has a handler (with middleware), use it
	if route.Handler != nil {
		route.Handler.ServeHTTP(w, req)
		return
	}

	// Otherwise, use the default proxy behavior
	p.proxyRequest(w, req, route)
}

// InitializeRouteHandlers initializes handlers for all routes with middleware
func (p *Handler) InitializeRouteHandlers(globalMiddleware *middleware.Chain) {
	p.router.ApplyGlobalMiddleware(globalMiddleware, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		route, ok := p.router.GetRoute(req.URL.Path, req.Method)
		if !ok {
			http.Error(w, "Route Not Found", http.StatusNotFound)
			return
		}

		p.proxyRequest(w, req, route)
	}))
}

func (p *Handler) proxyRequest(w http.ResponseWriter, req *http.Request, route *router.Route) {
	// Create a new URL based on the target
	targetURL := &url.URL{
		Scheme:   route.Target.Scheme,
		Host:     route.Target.Host,
		Path:     route.Target.Path,
		RawQuery: route.Target.RawQuery,
		Opaque:   route.Target.Opaque,
	}

	proxy := p.proxyFunc(targetURL)

	// Modify the director function
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetURL.Host
		req.URL.Path = targetURL.Path

		// Process headers
		p.processHeaders(req, route)
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		if !errors.Is(r.Context().Err(), context.Canceled) {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("Gateway error")) //nolint:errcheck
			return
		}

		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Gateway is shutting down")) //nolint:errcheck
	}

	proxy.ServeHTTP(w, req)
}

func (p *Handler) processHeaders(req *http.Request, route *router.Route) {
	allowedHeaderValues := make(map[string][]string)
	for _, allowedHeader := range route.AllowedHeaders {
		if values, exists := req.Header[allowedHeader]; exists {
			allowedHeaderValues[allowedHeader] = values
		}
	}

	req.Header = make(http.Header)

	for header, values := range allowedHeaderValues {
		for _, value := range values {
			req.Header.Add(header, value)
		}
	}

	for key, values := range route.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	req.Header.Set("User-Agent", defaultUserAgent)
}

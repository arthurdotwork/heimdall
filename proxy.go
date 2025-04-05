package heimdall

import (
	"context"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
)

const defaultUserAgent = "Heimdall/0.1"

type ProxyRouter interface {
	GetRoute(path, method string) (*Route, bool)
}

type ProxyHandler struct {
	router ProxyRouter
}

func NewProxyHandler(router ProxyRouter) *ProxyHandler {
	return &ProxyHandler{
		router: router,
	}
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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

	// Create a new URL based on the target
	targetURL := &url.URL{
		Scheme:   route.Target.Scheme,
		Host:     route.Target.Host,
		Path:     route.Target.Path,
		RawQuery: route.Target.RawQuery,
		Opaque:   route.Target.Opaque,
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

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

func (p *ProxyHandler) processHeaders(req *http.Request, route *Route) {
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

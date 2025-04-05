package router

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/arthurdotwork/heimdall/internal/config"
	"github.com/arthurdotwork/heimdall/internal/middleware"
)

type Route struct {
	OriginalPath   string
	Target         *url.URL
	Method         string
	Headers        map[string][]string
	AllowedHeaders []string
	Middleware     []string          // Middleware names for this route
	Middlewares    *middleware.Chain // Resolved middleware chain
	Handler        http.Handler      // Final handler after middleware (now exported)
}

type Router struct {
	// Routes is a map that index the route by path and method.
	Routes map[string]map[string]*Route
	// Registry for middleware
	registry *middleware.Registry
}

func New(endpoints []config.EndpointConfig) (*Router, error) {
	return NewWithRegistry(endpoints, middleware.DefaultRegistry())
}

func NewWithRegistry(endpoints []config.EndpointConfig, registry *middleware.Registry) (*Router, error) {
	routes := make(map[string]map[string]*Route)

	for _, endpoint := range endpoints {
		targetURL, err := url.Parse(endpoint.Target)
		if err != nil {
			return nil, err
		}

		if _, ok := routes[endpoint.Path]; !ok {
			routes[endpoint.Path] = make(map[string]*Route)
		}

		routes[endpoint.Path][endpoint.Method] = &Route{
			OriginalPath:   endpoint.Path,
			Target:         targetURL,
			Method:         endpoint.Method,
			Headers:        endpoint.Headers,
			AllowedHeaders: endpoint.AllowedHeaders,
			Middleware:     endpoint.Middlewares,
			Middlewares:    middleware.NewChain(),
		}
	}

	router := &Router{
		Routes:   routes,
		registry: registry,
	}

	// Initialize middleware for each route
	for _, methodRoutes := range routes {
		for _, route := range methodRoutes {
			if len(route.Middleware) > 0 {
				middlewares, missing := registry.GetMultiple(route.Middleware)
				if len(missing) > 0 {
					slog.Warn("some middleware not found for route",
						"path", route.OriginalPath,
						"method", route.Method,
						"missing", missing)
				}

				for _, mw := range middlewares {
					route.Middlewares.Add(mw)
				}
			}
		}
	}

	return router, nil
}

func (r *Router) GetRoute(path string, method string) (*Route, bool) {
	methodRoutes, ok := r.Routes[path]
	if !ok {
		return nil, false
	}

	route, ok := methodRoutes[method]
	if !ok {
		return nil, false
	}

	return route, true
}

// SetHandler sets the final handler for a route after applying its middleware
func (r *Router) SetHandler(route *Route, handler http.Handler) {
	route.Handler = route.Middlewares.Then(handler)
}

// ApplyGlobalMiddleware applies global middleware to all routes
func (r *Router) ApplyGlobalMiddleware(middlewareChain *middleware.Chain, finalHandler http.Handler) {
	for _, methodRoutes := range r.Routes {
		for _, route := range methodRoutes {
			// Clone the global middleware chain
			chain := middlewareChain.Clone()

			// Add route-specific middleware
			chain = chain.Add(route.Middlewares)

			// Set the final handler
			route.Handler = chain.Then(finalHandler)
		}
	}
}

package heimdall

import (
	"log/slog"
	"net/http"
	"net/url"
)

type Route struct {
	OriginalPath   string
	Target         *url.URL
	Method         string
	Headers        map[string][]string
	AllowedHeaders []string
	Middleware     []string         // Middleware names for this route
	Middlewares    *MiddlewareChain // Resolved middleware chain
	Handler        http.Handler     // Final handler after middleware (now exported)
}

type Router struct {
	// Routes is a map that index the route by path and method.
	Routes map[string]map[string]*Route
	// Registry for middleware
	registry *MiddlewareRegistry
}

func NewRouter(endpoints []EndpointConfig) (*Router, error) {
	return NewRouterWithRegistry(endpoints, defaultRegistry)
}

func NewRouterWithRegistry(endpoints []EndpointConfig, registry *MiddlewareRegistry) (*Router, error) {
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
			Middlewares:    NewMiddlewareChain(),
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

				for _, middleware := range middlewares {
					route.Middlewares.Add(middleware)
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
func (r *Router) ApplyGlobalMiddleware(middlewareChain *MiddlewareChain, finalHandler http.Handler) {
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

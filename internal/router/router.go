package router

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"

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

// patternRoute stores a parsed parameterized pattern and its method-to-route mapping.
type patternRoute struct {
	p      *pattern
	routes map[string]*Route
}

type Router struct {
	// Routes is a map that index the route by path and method.
	Routes map[string]map[string]*Route
	// patterns holds parameterized routes sorted by specificity (highest first).
	patterns []*patternRoute
	// Registry for middleware
	registry *middleware.Registry
}

func New(endpoints []config.EndpointConfig) (*Router, error) {
	return NewWithRegistry(endpoints, middleware.DefaultRegistry())
}

func NewWithRegistry(endpoints []config.EndpointConfig, registry *middleware.Registry) (*Router, error) {
	routes := make(map[string]map[string]*Route)
	patterns := make([]*patternRoute, 0)

	for _, endpoint := range endpoints {
		targetURL, err := url.Parse(endpoint.Target)
		if err != nil {
			return nil, err
		}

		route := &Route{
			OriginalPath:   endpoint.Path,
			Target:         targetURL,
			Method:         endpoint.Method,
			Headers:        endpoint.Headers,
			AllowedHeaders: endpoint.AllowedHeaders,
			Middleware:     endpoint.Middlewares,
			Middlewares:    middleware.NewChain(),
		}

		if strings.Contains(endpoint.Path, ":") {
			// Parameterized route
			p, err := parsePattern(endpoint.Path)
			if err != nil {
				return nil, err
			}
			if err := p.validate(); err != nil {
				return nil, err
			}
			// Find or create patternRoute for this pattern
			var pr *patternRoute
			for _, existing := range patterns {
				if existing.p.raw == p.raw {
					pr = existing
					break
				}
			}
			if pr == nil {
				pr = &patternRoute{p: p, routes: make(map[string]*Route)}
				patterns = append(patterns, pr)
			}
			pr.routes[endpoint.Method] = route
		} else if strings.Contains(endpoint.Path, "*") {
			// Wildcard patterns are not supported — validate via parsePattern to get consistent error
			p, err := parsePattern(endpoint.Path)
			if err != nil {
				return nil, err
			}
			if err := p.validate(); err != nil {
				return nil, err
			}
			// parsePattern+validate should have caught the wildcard, but just in case:
			return nil, fmt.Errorf("wildcard * not supported in pattern %q", endpoint.Path)
		} else {
			// Exact match route — existing behavior
			if _, ok := routes[endpoint.Path]; !ok {
				routes[endpoint.Path] = make(map[string]*Route)
			}
			routes[endpoint.Path][endpoint.Method] = route
		}
	}

	// Sort patterns by specificity (highest first)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].p.specificity() > patterns[j].p.specificity()
	})

	router := &Router{
		Routes:   routes,
		patterns: patterns,
		registry: registry,
	}

	// Initialize middleware for exact routes
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

	// Initialize middleware for parameterized routes
	for _, pr := range patterns {
		for _, route := range pr.routes {
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

func (r *Router) GetRoute(path string, method string) (*Route, map[string]string, bool) {
	// 1. Try exact match first (O(1))
	if methodRoutes, ok := r.Routes[path]; ok {
		if route, ok := methodRoutes[method]; ok {
			return route, map[string]string{}, true
		}
	}

	// 2. Try parameterized patterns (sorted by specificity, highest first)
	for _, pr := range r.patterns {
		params, matched := pr.p.match(path)
		if !matched {
			continue
		}
		route, ok := pr.routes[method]
		if !ok {
			continue
		}
		return route, params, true
	}

	return nil, nil, false
}

// SetHandler sets the final handler for a route after applying its middleware
func (r *Router) SetHandler(route *Route, handler http.Handler) {
	route.Handler = route.Middlewares.Then(handler)
}

// ApplyGlobalMiddleware applies global middleware to all routes
func (r *Router) ApplyGlobalMiddleware(middlewareChain *middleware.Chain, finalHandler http.Handler) {
	// Exact match routes
	for _, methodRoutes := range r.Routes {
		for _, route := range methodRoutes {
			chain := middlewareChain.Clone()
			chain = chain.Add(route.Middlewares)
			route.Handler = chain.Then(finalHandler)
		}
	}
	// Parameterized routes
	for _, pr := range r.patterns {
		for _, route := range pr.routes {
			chain := middlewareChain.Clone()
			chain = chain.Add(route.Middlewares)
			route.Handler = chain.Then(finalHandler)
		}
	}
}

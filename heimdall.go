package heimdall

import (
	"context"
	"log/slog"
)

type Gateway struct {
	config            *Config
	router            *Router
	proxy             *ProxyHandler
	server            *Server
	globalMiddlewares *MiddlewareChain
	registry          *MiddlewareRegistry
}

func New(configPath string) (*Gateway, error) {
	return NewWithRegistry(configPath, defaultRegistry)
}

func NewWithRegistry(configPath string, registry *MiddlewareRegistry) (*Gateway, error) {
	cfg, err := LoadFromFile(configPath)
	if err != nil {
		return nil, err
	}
	cfg = cfg.WithDefaults()

	router, err := NewRouterWithRegistry(cfg.Endpoints, registry)
	if err != nil {
		return nil, err
	}

	proxy := NewProxyHandler(router)
	server := NewServer(cfg.Gateway, proxy)

	// Initialize global middleware
	globalMiddlewares := NewMiddlewareChain()

	// Add global middleware from config
	if len(cfg.Gateway.Middlewares) > 0 {
		middlewares, missing := registry.GetMultiple(cfg.Gateway.Middlewares)
		if len(missing) > 0 {
			slog.Warn("some global middleware not found", "missing", missing)
		}

		for _, middleware := range middlewares {
			globalMiddlewares.Add(middleware)
		}
	}

	// Initialize route handlers with middleware
	proxy.InitializeRouteHandlers(globalMiddlewares)

	return &Gateway{
		config:            cfg,
		router:            router,
		proxy:             proxy,
		server:            server,
		globalMiddlewares: globalMiddlewares,
		registry:          registry,
	}, nil
}

func (g *Gateway) Start(ctx context.Context) error {
	return g.server.Start(ctx)
}

func (g *Gateway) Config() *Config {
	return g.config
}

// Use adds a middleware to the global middleware chain
func (g *Gateway) Use(middleware Middleware) *Gateway {
	g.globalMiddlewares.Add(middleware)

	// Update route handlers with the new middleware chain
	g.proxy.InitializeRouteHandlers(g.globalMiddlewares)

	return g
}

// UseFunc adds a middleware function to the global middleware chain
func (g *Gateway) UseFunc(middleware MiddlewareFunc) *Gateway {
	return g.Use(middleware)
}

// RegisterMiddleware registers a middleware with the gateway's registry
func (g *Gateway) RegisterMiddleware(name string, middleware Middleware) error {
	return g.registry.Register(name, middleware)
}

// GetMiddleware gets a middleware from the gateway's registry
func (g *Gateway) GetMiddleware(name string) (Middleware, bool) {
	return g.registry.Get(name)
}

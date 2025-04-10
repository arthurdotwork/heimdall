// Package heimdall provides an API Gateway implementation
package heimdall

import (
	"context"
	"log/slog"

	"github.com/arthurdotwork/heimdall/internal/config"
	internalMiddleware "github.com/arthurdotwork/heimdall/internal/middleware"
	"github.com/arthurdotwork/heimdall/internal/proxy"
	"github.com/arthurdotwork/heimdall/internal/router"
	"github.com/arthurdotwork/heimdall/internal/server"
)

// Gateway represents an API gateway that can route and proxy requests
type Gateway struct {
	config            *config.Config
	router            *router.Router
	proxy             *proxy.Handler
	server            *server.Server
	globalMiddlewares *internalMiddleware.Chain
	registry          *internalMiddleware.Registry
}

type (
	Config         = config.Config
	GatewayConfig  = config.GatewayConfig
	EndpointConfig = config.EndpointConfig
	Middleware     = internalMiddleware.Middleware
	MiddlewareFunc = internalMiddleware.Func
)

// New creates a new gateway instance
func New(configPath string) (*Gateway, error) {
	return NewWithRegistry(configPath, defaultRegistry)
}

// NewWithRegistry creates a new gateway with a custom middleware registry
func NewWithRegistry(configPath string, registry *internalMiddleware.Registry) (*Gateway, error) {
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return nil, err
	}
	cfg = cfg.WithDefaults()

	r, err := router.NewWithRegistry(cfg.Endpoints, registry)
	if err != nil {
		return nil, err
	}

	p := proxy.NewHandler(r)
	s := server.New(cfg.Gateway, p)

	// Initialize global middleware
	globalMiddlewares := internalMiddleware.NewChain()

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
	p.InitializeRouteHandlers(globalMiddlewares)

	return &Gateway{
		config:            cfg,
		router:            r,
		proxy:             p,
		server:            s,
		globalMiddlewares: globalMiddlewares,
		registry:          registry,
	}, nil
}

// Start starts the gateway
func (g *Gateway) Start(ctx context.Context) error {
	return g.server.Start(ctx)
}

// Config returns the gateway configuration
func (g *Gateway) Config() *config.Config {
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

// defaultRegistry is the default middleware registry
var defaultRegistry = internalMiddleware.NewRegistry()

// RegisterMiddleware registers a middleware with the default registry
func RegisterMiddleware(name string, middleware Middleware) error {
	return defaultRegistry.Register(name, middleware)
}

// LoadFromFile loads configuration from a file
func LoadFromFile(path string) (*Config, error) {
	return config.LoadFromFile(path)
}

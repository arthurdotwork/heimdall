package heimdall

import "context"

type Gateway struct {
	config *Config
	router *Router
	proxy  *ProxyHandler
	server *Server
}

func New(configPath string) (*Gateway, error) {
	cfg, err := LoadFromFile(configPath)
	if err != nil {
		return nil, err
	}
	cfg = cfg.WithDefaults()

	router, err := NewRouter(cfg.Endpoints)
	if err != nil {
		return nil, err
	}

	proxy := NewProxyHandler(router)
	server := NewServer(cfg.Gateway, proxy)

	return &Gateway{
		config: cfg,
		router: router,
		proxy:  proxy,
		server: server,
	}, nil
}

func (g *Gateway) Start(ctx context.Context) error {
	return g.server.Start(ctx)
}

func (g *Gateway) Config() *Config {
	return g.config
}

package heimdall

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Gateway struct {
		Port         int           `yaml:"port"`
		ReadTimeout  time.Duration `yaml:"readTimeout"`
		WriteTimeout time.Duration `yaml:"writeTimeout"`
	} `yaml:"gateway"`
	Endpoints []Endpoint `yaml:"endpoints"`
}

type Endpoint struct {
	Name           string            `yaml:"name"`
	Path           string            `yaml:"path"`
	Target         string            `yaml:"target"`
	Method         string            `yaml:"method"`
	Headers        map[string]string `yaml:"headers"`
	AllowedHeaders []string          `yaml:"allowed_headers"`
}

type Gateway struct {
	config Config
	routes map[string]*Route
}

type Route struct {
	URL            *url.URL
	Method         string
	Headers        map[string]string
	AllowedHeaders []string
	Name           string
}

func New(configPath string) (*Gateway, error) {
	// Read configuration from YAML file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, err
	}

	// Build routes map from configuration
	routes := make(map[string]*Route)
	for _, endpoint := range config.Endpoints {
		targetURL, err := url.Parse(endpoint.Target)
		if err != nil {
			return nil, err
		}

		routes[endpoint.Path] = &Route{
			URL:            targetURL,
			Method:         endpoint.Method,
			Headers:        endpoint.Headers,
			AllowedHeaders: endpoint.AllowedHeaders,
			Name:           endpoint.Name,
		}
	}

	return &Gateway{
		config: config,
		routes: routes,
	}, nil
}

func (g *Gateway) Start(ctx context.Context) error {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, ok := g.routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}

		// Check if method is allowed
		if route.Method != "" && r.Method != route.Method {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Create a new URL based on the target
		targetURL := &url.URL{
			Scheme:   route.URL.Scheme,
			Host:     route.URL.Host,
			Path:     route.URL.Path,
			RawQuery: route.URL.RawQuery,
			Opaque:   route.URL.Opaque,
		}

		slog.InfoContext(ctx, "Serving request",
			slog.Any("path", r.URL.Path),
			slog.Any("target", targetURL),
			slog.String("endpoint", route.Name))

		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// Modify the director function to correctly set the URL, strip headers, and add custom headers
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.Host = targetURL.Host
			req.URL.Path = targetURL.Path

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

			for key, value := range route.Headers {
				req.Header.Set(key, value)
			}

			req.Header.Set("User-Agent", "Heimdall/0.1")
		}

		proxy.ServeHTTP(w, r)
	})

	port := g.config.Gateway.Port
	if port == 0 {
		port = 8080
	}

	readTimeout := g.config.Gateway.ReadTimeout
	if readTimeout == 0 {
		readTimeout = 5 * time.Second
	}

	writeTimeout := g.config.Gateway.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 5 * time.Second
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	gr, ctx := errgroup.WithContext(ctx)

	gr.Go(func() error {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}

		return nil
	})

	gr.Go(func() error {
		slog.InfoContext(ctx, "Starting gateway server", slog.Int("port", port))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		return nil
	})

	if err := gr.Wait(); err != nil {
		return err
	}

	return nil
}

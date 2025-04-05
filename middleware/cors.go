package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/arthurdotwork/heimdall"
)

// CORSConfig holds configuration for CORS middleware
type CORSConfig struct {
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowMethods: []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPut,
			http.MethodPatch,
			http.MethodPost,
			http.MethodDelete,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
		},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// CORS creates a middleware for handling Cross-Origin Resource Sharing
func CORS(config *CORSConfig) heimdall.Middleware {
	if config == nil {
		config = DefaultCORSConfig()
	}

	allowMethods := config.AllowMethods
	allowHeaders := config.AllowHeaders
	allowCredentials := config.AllowCredentials
	maxAge := config.MaxAge

	return heimdall.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Set CORS headers
			header := w.Header()

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				header.Set("Access-Control-Allow-Origin", origin)

				if len(allowMethods) > 0 {
					header.Set("Access-Control-Allow-Methods", strings.Join(allowMethods, ", "))
				}

				if len(allowHeaders) > 0 {
					header.Set("Access-Control-Allow-Headers", strings.Join(allowHeaders, ", "))
				}

				if allowCredentials {
					header.Set("Access-Control-Allow-Credentials", "true")
				}

				if maxAge > 0 {
					header.Set("Access-Control-Max-Age", fmt.Sprintf("%d", maxAge))
				}

				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Handle normal requests
			if origin != "" {
				header.Set("Access-Control-Allow-Origin", origin)
			}

			if allowCredentials {
				header.Set("Access-Control-Allow-Credentials", "true")
			}

			next.ServeHTTP(w, r)
		})
	})
}

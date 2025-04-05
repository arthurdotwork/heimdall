package middleware

import (
	"github.com/arthurdotwork/heimdall"
	"github.com/arthurdotwork/heimdall/internal/middleware"
)

// RegisterDefaults registers the default middleware with the global registry
func RegisterDefaults() {
	// Register Logger middleware
	_ = heimdall.RegisterMiddleware("logger", Logger())

	// Register CORS middleware with default configuration
	_ = heimdall.RegisterMiddleware("cors", CORS(DefaultCORSConfig()))
}

// Register registers the middleware with a custom registry
func Register(registry *middleware.MiddlewareRegistry) {
	// Register Logger middleware
	_ = registry.Register("logger", Logger())

	// Register CORS middleware with default configuration
	_ = registry.Register("cors", CORS(DefaultCORSConfig()))
}

package heimdall

import (
	"fmt"
	"sync"
)

// MiddlewareRegistry is a registry of middleware that can be referenced by name
type MiddlewareRegistry struct {
	mutex       sync.RWMutex
	middlewares map[string]Middleware
}

// NewMiddlewareRegistry creates a new middleware registry
func NewMiddlewareRegistry() *MiddlewareRegistry {
	return &MiddlewareRegistry{
		middlewares: make(map[string]Middleware),
	}
}

// Register registers a middleware with the registry
func (r *MiddlewareRegistry) Register(name string, middleware Middleware) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.middlewares[name]; exists {
		return fmt.Errorf("middleware with name '%s' already registered", name)
	}

	r.middlewares[name] = middleware
	return nil
}

// Get retrieves a middleware by name
func (r *MiddlewareRegistry) Get(name string) (Middleware, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	middleware, exists := r.middlewares[name]
	return middleware, exists
}

// GetMultiple retrieves multiple middleware by name
func (r *MiddlewareRegistry) GetMultiple(names []string) ([]Middleware, []string) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []Middleware
	var missing []string

	for _, name := range names {
		if middleware, exists := r.middlewares[name]; exists {
			result = append(result, middleware)
		} else {
			missing = append(missing, name)
		}
	}

	return result, missing
}

// defaultRegistry is the default middleware registry
var defaultRegistry = NewMiddlewareRegistry()

// RegisterMiddleware registers a middleware with the default registry
func RegisterMiddleware(name string, middleware Middleware) error {
	return defaultRegistry.Register(name, middleware)
}

// GetMiddleware retrieves a middleware from the default registry
func GetMiddleware(name string) (Middleware, bool) {
	return defaultRegistry.Get(name)
}

// GetMiddlewares retrieves multiple middleware from the default registry
func GetMiddlewares(names []string) ([]Middleware, []string) {
	return defaultRegistry.GetMultiple(names)
}

// ResetDefaultRegistry resets the default registry (primarily for testing)
func ResetDefaultRegistry() {
	defaultRegistry = NewMiddlewareRegistry()
}

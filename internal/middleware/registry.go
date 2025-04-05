package middleware

import (
	"fmt"
	"sync"
)

// Registry is a registry of middleware that can be referenced by name
type Registry struct {
	mutex       sync.RWMutex
	middlewares map[string]Middleware
}

// NewRegistry creates a new middleware registry
func NewRegistry() *Registry {
	return &Registry{
		middlewares: make(map[string]Middleware),
	}
}

// Register registers a middleware with the registry
func (r *Registry) Register(name string, middleware Middleware) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.middlewares[name]; exists {
		return fmt.Errorf("middleware with name '%s' already registered", name)
	}

	r.middlewares[name] = middleware
	return nil
}

// Get retrieves a middleware by name
func (r *Registry) Get(name string) (Middleware, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	middleware, exists := r.middlewares[name]
	return middleware, exists
}

// GetMultiple retrieves multiple middleware by name
func (r *Registry) GetMultiple(names []string) ([]Middleware, []string) {
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
	defaultRegistry = NewRegistry()
}

// DefaultRegistry returns the default middleware registry
func DefaultRegistry() *Registry {
	// Use a sync.Once to initialize only once
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}

// The actual variable is private
var (
	defaultRegistry     *Registry
	defaultRegistryOnce sync.Once
)

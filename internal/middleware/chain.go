package middleware

import (
	"net/http"
)

// Middleware defines the contract for middleware components in Heimdall
type Middleware interface {
	// Wrap takes an http.Handler and returns a new http.Handler with additional behavior
	Wrap(next http.Handler) http.Handler
}

// MiddlewareFunc is a function type that implements the Middleware interface
type MiddlewareFunc func(http.Handler) http.Handler

// Wrap implements the Middleware interface for MiddlewareFunc
func (f MiddlewareFunc) Wrap(next http.Handler) http.Handler {
	return f(next)
}

// MiddlewareChain represents a chain of middleware that can be applied to a handler
type MiddlewareChain struct {
	middlewares []Middleware
}

// NewMiddlewareChain creates a new empty middleware chain
func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: []Middleware{},
	}
}

// Add appends a middleware to the chain
func (c *MiddlewareChain) Add(m Middleware) *MiddlewareChain {
	c.middlewares = append(c.middlewares, m)
	return c
}

// AddFunc appends a middleware function to the chain
func (c *MiddlewareChain) AddFunc(f MiddlewareFunc) *MiddlewareChain {
	return c.Add(f)
}

// Wrap implements the Middleware interface for MiddlewareChain
func (c *MiddlewareChain) Wrap(final http.Handler) http.Handler {
	return c.Then(final)
}

// Then applies the middleware chain to a final handler
func (c *MiddlewareChain) Then(final http.Handler) http.Handler {
	if final == nil {
		final = http.DefaultServeMux
	}

	// Apply Middlewares in reverse order, so the first middleware in the chain
	// is the outermost one when handling a request
	handler := final
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i].Wrap(handler)
	}

	return handler
}

// ThenFunc applies the middleware chain to a final handler function
func (c *MiddlewareChain) ThenFunc(final http.HandlerFunc) http.Handler {
	return c.Then(final)
}

// Clone creates a copy of the middleware chain
func (c *MiddlewareChain) Clone() *MiddlewareChain {
	clone := NewMiddlewareChain()
	clone.middlewares = append(clone.middlewares, c.middlewares...)
	return clone
}

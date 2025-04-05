package middleware

import (
	"net/http"
)

// Middleware defines the contract for middleware components in Heimdall
type Middleware interface {
	// Wrap takes an http.Handler and returns a new http.Handler with additional behavior
	Wrap(next http.Handler) http.Handler
}

// Func is a function type that implements the Middleware interface
type Func func(http.Handler) http.Handler

// Wrap implements the Middleware interface for Func
func (f Func) Wrap(next http.Handler) http.Handler {
	return f(next)
}

// Chain represents a chain of middleware that can be applied to a handler
type Chain struct {
	middlewares []Middleware
}

// NewChain creates a new empty middleware chain
func NewChain() *Chain {
	return &Chain{
		middlewares: []Middleware{},
	}
}

// Add appends a middleware to the chain
func (c *Chain) Add(m Middleware) *Chain {
	c.middlewares = append(c.middlewares, m)
	return c
}

// AddFunc appends a middleware function to the chain
func (c *Chain) AddFunc(f Func) *Chain {
	return c.Add(f)
}

// Wrap implements the Middleware interface for Chain
func (c *Chain) Wrap(final http.Handler) http.Handler {
	return c.Then(final)
}

// Then applies the middleware chain to a final handler
func (c *Chain) Then(final http.Handler) http.Handler {
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
func (c *Chain) ThenFunc(final http.HandlerFunc) http.Handler {
	return c.Then(final)
}

// Clone creates a copy of the middleware chain
func (c *Chain) Clone() *Chain {
	clone := NewChain()
	clone.middlewares = append(clone.middlewares, c.middlewares...)
	return clone
}

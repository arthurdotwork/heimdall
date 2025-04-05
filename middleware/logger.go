package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/arthurdotwork/heimdall"
)

// Logger creates a middleware for logging HTTP requests
func Logger() heimdall.Middleware {
	return heimdall.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response wrapper to capture status code
			rw := newResponseWriter(w)

			// Log the request
			slog.InfoContext(r.Context(), "request started",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)

			// Call the next handler
			next.ServeHTTP(rw, r)

			// Calculate request duration
			duration := time.Since(start)

			// Log the response
			slog.InfoContext(r.Context(), "request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration", duration.Seconds(),
				"bytes", rw.size,
			)
		})
	})
}

// responseWriter is a wrapper for http.ResponseWriter that captures the status code and body size
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

// newResponseWriter creates a new responseWriter
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		status:         http.StatusOK, // Default status code
	}
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size
func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

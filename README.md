# Heimdall

[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8.svg)](https://go.dev/)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI Status](https://github.com/arthurdotwork/heimdall/actions/workflows/heimdall.yml/badge.svg)](https://github.com/arthurdotwork/heimdall/actions)

Heimdall is a lightweight, extensible API Gateway for Go applications, designed to handle routing, proxying, and middleware management with ease.

## ✨ Features

- **Simple Configuration**: YAML-based setup for quick deployment
- **Middleware Architecture**: Pre-built middleware for logging, CORS and easy extension points
- **Flexible Routing**: Route HTTP requests to backend services with custom headers and path mapping
- **Docker Support**: Ready-to-use container with minimal footprint
- **Graceful Shutdown**: Clean handling of termination signals

## 🚀 Getting Started

### Prerequisites

- Go 1.24 or later (for development)
- Docker (optional, for containerized deployment)

### Installation

#### Using Go

```bash
# Clone the repository
git clone https://github.com/arthurdotwork/heimdall.git
cd heimdall

# Build the binary
go build -o heimdall ./cmd/heimdall
```

#### Using Docker

```bash
# Pull the image
docker pull ghcr.io/arthurdotwork/heimdall:latest

# Or build locally
docker build -t heimdall:local .
```

### Configuration

Create a `config.yaml` file based on the provided example:

```yaml
gateway:
  port: 8080
  readTimeout: 5s
  writeTimeout: 5s
  middlewares:
    - logger
    - cors

endpoints:
  - name: MyAPI
    path: /api
    target: https://api.example.com
    method: GET
    headers:
      X-My-Header: 
        - CustomValue
    allowed_headers:
      - X-Request-Id
```

### Running Heimdall

#### Standalone

```bash
./heimdall --config config.yaml
```

#### With Docker

```bash
docker run -p 8080:8080 -v $(pwd)/config.yaml:/etc/heimdall/config.yaml ghcr.io/arthurdotwork/heimdall:latest
```

## 🔌 Extending with Middleware

Heimdall's power comes from its middleware architecture. You can register and chain multiple middleware components to customize the gateway's behavior.

### Using Built-in Middleware

Heimdall comes with pre-configured middleware that you can reference in your config:

```yaml
gateway:
  middlewares:
    - logger  # Log all requests and responses
    - cors    # Add Cross-Origin Resource Sharing headers
```

### Creating Custom Middleware

Creating your own middleware is straightforward:

1. Define your middleware function:

```go
package main

import (
    "net/http"
    "github.com/arthurdotwork/heimdall"
)

func myCustomMiddleware() heimdall.Middleware {
    return heimdall.MiddlewareFunc(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Your custom logic here
            w.Header().Set("X-Custom-Header", "CustomValue")
            
            // Continue to the next middleware or handler
            next.ServeHTTP(w, r)
        })
    })
}
```

2. Register your middleware in your application:

```go
func main() {
    // Register the middleware with a name
    err := heimdall.RegisterMiddleware("custom", myCustomMiddleware())
    if err != nil {
        // Handle error
    }
    
    gateway, err := heimdall.New("config.yaml")
    if err != nil {
        // Handle error
    }
    
    // You can also add middleware programmatically
    gateway.Use(myCustomMiddleware())
    
    gateway.Start(context.Background())
}
```

3. Reference your middleware in the config file:

```yaml
endpoints:
  - name: ProtectedAPI
    path: /protected
    target: https://api.example.com/protected
    method: GET
    middlewares:
      - custom  # Your registered middleware
```

### Middleware Chain

Middlewares execute in order, with global middlewares running before endpoint-specific ones:

```
Request → Global Middlewares → Endpoint Middlewares → Backend Service
```

## 🧪 Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with code coverage
go test -cover ./...
```

### Using the Taskfile

Heimdall includes a Taskfile for common development tasks:

```bash
# Install task if you don't have it
go install github.com/go-task/task/v3/cmd/task@latest

# Run linters
task lint

# Run tests with better formatting
task test
```

## 📖 Reference

### Config File Structure

```yaml
gateway:
  port: 8080                # Port to listen on
  readTimeout: 5s           # Timeout for reading requests
  writeTimeout: 5s          # Timeout for writing responses
  shutdownTimeout: 10s      # Timeout for graceful shutdown
  middlewares: []           # Global middlewares

endpoints:
  - name: String            # Endpoint name (for logging)
    path: /path             # URL path to match
    target: http://backend  # Target backend URL
    method: GET             # HTTP method to match
    headers: {}             # Headers to add to proxied requests
    allowed_headers: []     # Headers to forward from client requests
    middlewares: []         # Endpoint-specific middlewares
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

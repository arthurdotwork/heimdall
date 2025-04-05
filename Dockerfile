FROM golang:1.24-alpine AS builder

# Install certificates for potential downloads during build
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy and download dependencies first (better layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with maximum optimizations
RUN CGO_ENABLED=0 go build -ldflags="-s -w -extldflags '-static'" -trimpath -o heimdall ./cmd/heimdall

FROM gcr.io/distroless/static:nonroot

# Copy binary and certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/heimdall /opt/heimdall

# Document that the image expects a config file to be mounted
VOLUME ["/etc/heimdall"]

# Use unprivileged user
USER nonroot:nonroot

# Expose the default port
EXPOSE 8080

# Command to run (users should mount their config file)
ENTRYPOINT ["/opt/heimdall", "--config", "/etc/heimdall/config.yaml"]

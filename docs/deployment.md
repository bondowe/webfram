---
layout: default
title: Deployment
nav_order: 15
description: "Production deployment guide"
---

# Production Deployment
{: .no_toc }

Guide for deploying WebFram applications to production.
{: .fs-6 .fw-300 }

## Table of contents

{: .no_toc .text-delta }

1. TOC
{:toc}

## Build Optimization

Build with optimizations:

```bash
# Strip debug info and disable symbol table
go build -ldflags="-s -w" -o app

# Further compression with upx (optional)
upx --best --lzma app
```

## Docker Deployment

### Dockerfile

```dockerfile
# Multi-stage build for minimal image size
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o webfram-app

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/webfram-app .
COPY --from=builder /app/assets ./assets

EXPOSE 8080
CMD ["./webfram-app"]
```

### Build and Run

```bash
# Build image
docker build -t webfram-app .

# Run container
docker run -p 8080:8080 webfram-app
```

## Environment Configuration

```go
package main

import (
    "os"
    "strconv"
)

type Config struct {
    Port           string
    Environment    string
    EnableOpenAPI  bool
    EnableJSONP    bool
}

func loadConfig() Config {
    return Config{
        Port:          getEnv("PORT", "8080"),
        Environment:   getEnv("ENVIRONMENT", "production"),
        EnableOpenAPI: getEnvBool("ENABLE_OPENAPI", false),
        EnableJSONP:   getEnvBool("ENABLE_JSONP", false),
    }
}

func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}

func getEnvBool(key string, fallback bool) bool {
    if value := os.Getenv(key); value != "" {
        b, err := strconv.ParseBool(value)
        if err == nil {
            return b
        }
    }
    return fallback
}
```

## Graceful Shutdown

**Automatic with ListenAndServe:**

```go
func main() {
    cfg := loadConfig()
    
    app.Configure(getWebFramConfig(cfg))
    mux := app.NewServeMux()
    registerRoutes(mux)
    
    serverCfg := &app.ServerConfig{
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  120 * time.Second,
    }
    
    // Automatic graceful shutdown on SIGINT/SIGTERM
    app.ListenAndServe(":"+cfg.Port, mux, serverCfg)
}
```

**Manual Graceful Shutdown:**

```go
func main() {
    server := &http.Server{
        Addr:         ":8080",
        Handler:      mux,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  120 * time.Second,
    }
    
    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()
    
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("Shutting down server...")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced to shutdown: %v", err)
    }
}
```

## Health Checks

```go
mux.HandleFunc("GET /health", func(w app.ResponseWriter, r *app.Request) {
    w.JSON(r.Context(), map[string]string{
        "status":  "healthy",
        "version": version,
    })
})

mux.HandleFunc("GET /readiness", func(w app.ResponseWriter, r *app.Request) {
    if !isReady() {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.JSON(r.Context(), map[string]string{"status": "not ready"})
        return
    }
    
    w.JSON(r.Context(), map[string]string{"status": "ready"})
})
```

## Monitoring & Telemetry

WebFram includes built-in Prometheus metrics:

```go
app.Configure(&app.Config{
    Telemetry: &app.Telemetry{
        Enabled: true,
        Addr:    ":9090", // Separate telemetry server
        HandlerOpts: promhttp.HandlerOpts{
            EnableOpenMetrics: true,
        },
    },
    // ... other config
})
```

**Metrics available:**

- `http_requests_total` - Request count by method, path, status
- `http_request_duration_seconds` - Request duration histogram

**Access metrics:**

```bash
curl http://localhost:9090/metrics
```

See full details in [Telemetry section](#telemetry--monitoring).

## Security Hardening

### Security Headers Middleware

```go
func securityMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        next.ServeHTTP(w, r)
    })
}

app.Use(securityMiddleware)
```

### HTTPS Configuration

```go
cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
if err != nil {
    log.Fatal(err)
}

tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
    MinVersion:   tls.VersionTLS13,
    CipherSuites: []uint16{
        tls.TLS_AES_128_GCM_SHA256,
        tls.TLS_AES_256_GCM_SHA384,
        tls.TLS_CHACHA20_POLY1305_SHA256,
    },
}

serverCfg := &app.ServerConfig{
    TLSConfig: tlsConfig,
}

app.ListenAndServe(":443", mux, serverCfg)
```

## Logging

```go
import "log/slog"

func loggingMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        start := time.Now()
        
        slog.Info("incoming request",
            "method", r.Method,
            "path", r.URL.Path,
            "remote", r.RemoteAddr,
        )
        
        next.ServeHTTP(w, r)
        
        slog.Info("request completed",
            "method", r.Method,
            "path", r.URL.Path,
            "duration", time.Since(start),
        )
    })
}
```

## Performance Tuning

```go
func main() {
    runtime.GOMAXPROCS(runtime.NumCPU())
    
    serverCfg := &app.ServerConfig{
        ReadTimeout:       10 * time.Second,
        ReadHeaderTimeout: 5 * time.Second,
        WriteTimeout:      10 * time.Second,
        IdleTimeout:       120 * time.Second,
        MaxHeaderBytes:    1 << 20, // 1 MB
    }
    
    app.ListenAndServe(":8080", mux, serverCfg)
}
```

## Kubernetes Deployment

### Deployment YAML

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: webfram-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: webfram-app
  template:
    metadata:
      labels:
        app: webfram-app
    spec:
      containers:
      - name: webfram-app
        image: webfram-app:latest
        ports:
        - containerPort: 8080
        env:
        - name: PORT
          value: "8080"
        - name: ENVIRONMENT
          value: "production"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /readiness
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "128Mi"
            cpu: "250m"
          limits:
            memory: "256Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: webfram-app
spec:
  selector:
    app: webfram-app
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

## Best Practices

1. **Use environment variables** for configuration
2. **Enable health checks** for orchestrators
3. **Implement graceful shutdown** to handle in-flight requests
4. **Monitor with Prometheus** metrics
5. **Add structured logging** with slog
6. **Set appropriate timeouts** to prevent resource exhaustion
7. **Use HTTPS** in production
8. **Apply security headers** to all responses
9. **Run as non-root user** in containers
10. **Regular security updates** for dependencies

## Troubleshooting

### High Memory Usage

- Check for goroutine leaks
- Monitor connection pools
- Review caching strategies

### High CPU Usage

- Profile with pprof
- Check for inefficient algorithms
- Review middleware overhead

### Slow Responses

- Enable request tracing
- Check database query performance
- Review middleware execution time

## See Also

- [Configuration](configuration.html)
- [Testing](testing.html)
- [Middleware](middleware.html)

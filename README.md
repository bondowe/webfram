# WebFram

[![CI](https://github.com/bondowe/webfram/actions/workflows/ci.yml/badge.svg)](https://github.com/bondowe/webfram/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/bondowe/webfram/branch/main/graph/badge.svg)](https://codecov.io/gh/bondowe/webfram)
[![Go Report Card](https://goreportcard.com/badge/github.com/bondowe/webfram)](https://goreportcard.com/report/github.com/bondowe/webfram)
[![Go Reference](https://pkg.go.dev/badge/github.com/bondowe/webfram.svg)](https://pkg.go.dev/github.com/bondowe/webfram)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**WebFram** is a production-ready, lightweight, feature-rich Go web framework built on top of the standard library's `net/http` package. It provides enterprise-grade features like automatic template caching with layouts, comprehensive data binding with validation, internationalization (i18n), Server-Sent Events (SSE), JSON Patch support, JSONP, OpenAPI 3.2.0 documentation generation, and flexible middleware support—all while maintaining minimal dependencies and maximum performance.

## Table of Contents

- [Features](#features)
- [Why WebFram?](#why-webfram)
- [Architecture & Design](#architecture--design)
- [Performance](#performance)
- [Security](#security)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
  - [Configuration Options](#configuration-options)
  - [Configuration Best Practices](#configuration-best-practices)
- [Server Configuration](#server-configuration)
  - [ListenAndServe](#listenandserve)
  - [ServerConfig Options](#serverconfig-options)
  - [Server Configuration Examples](#server-configuration-examples)
- [Routing](#routing)
  - [Basic Routes](#basic-routes)
  - [Route Parameters](#route-parameters)
  - [Wildcard Routes](#wildcard-routes)
- [Middleware](#middleware)
  - [Global Middleware](#global-middleware)
  - [Mux-Level Middleware](#mux-level-middleware)
  - [Route-Specific Middleware](#route-specific-middleware)
  - [Middleware Execution Order](#middleware-execution-order)
  - [Standard HTTP Middleware Support](#standard-http-middleware-support)
- [Request Handling](#request-handling)
- [Response Handling](#response-handling)
- [Data Binding & Validation](#data-binding--validation)
  - [Form Binding](#form-binding)
  - [JSON Binding](#json-binding)
  - [XML Binding](#xml-binding)
  - [Nested Structs](#nested-structs)
  - [Map Binding (Form Only)](#map-binding-form-only)
  - [Validation Tags Reference](#validation-tags-reference)
  - [Custom Error Messages](#custom-error-messages)
  - [Validation Errors](#validation-errors)
  - [Supported Field Types](#supported-field-types)
- [JSON Patch Support](#json-patch-support)
- [JSONP Support](#jsonp-support)
- [OpenAPI Documentation](#openapi-documentation)
- [Server-Sent Events (SSE)](#server-sent-events-sse)
- [Templates](#templates)
- [Internationalization (i18n)](#internationalization-i18n)
- [Complete Example](#complete-example)
- [Testing](#testing)
- [Production Deployment](#production-deployment)
- [Contributing](#contributing)
- [License](#license)

## Features

### Core Features

- 🚀 **Lightweight & Fast**: Built directly on `net/http` with zero reflection overhead for routing
- 📝 **Smart Templates**: Automatic template caching with layout inheritance, partials, and hot-reload in development
- ✅ **Data Binding**: Type-safe Form, JSON, and XML binding with comprehensive validation
- 🗺️ **Map Support**: Form binding supports maps with `fieldname[key]=value` syntax for dynamic data
- 🔄 **JSON Patch**: Full RFC 6902 JSON Patch support for RESTful partial updates
- 🌐 **JSONP**: Secure cross-origin JSON requests with built-in callback validation
- 📡 **Server-Sent Events**: Production-ready SSE support for real-time server-to-client streaming
- 📚 **OpenAPI 3.2.0**: Automatic API documentation generation with schema inference from struct tags
- 🌍 **i18n Support**: First-class internationalization using `golang.org/x/text` with template integration
- 🔧 **Flexible Middleware**: Support for both custom and standard HTTP middleware with composability
- 📦 **Multiple Response Formats**: JSON, JSONP, XML, YAML, HTML, and plain text responses
- 🎯 **Type-Safe**: Generic-based binding ensures compile-time type safety
- 🔒 **Comprehensive Validation**: 20+ validation rules including required, min/max, regex, enum, uniqueItems, multipleOf, and more

### Security Features

- 🔐 **Input Validation**: Built-in sanitization and validation for all input types
- 🛡️ **JSONP Callback Validation**: Automatic validation to prevent XSS attacks
- 📋 **Content-Type Enforcement**: Strict content-type checking for JSON Patch and other endpoints
- 🔍 **Safe Template Execution**: Automatic HTML escaping in templates

### Developer Experience

- 📖 **Comprehensive Documentation**: Extensive examples and API documentation
- 🧪 **Testable**: Easy to write unit and integration tests
- 🔄 **Zero Breaking Changes**: Semantic versioning with backward compatibility guarantees
- 📊 **OpenAPI Integration**: Auto-generated API docs from code
- 🎨 **Clean API**: Intuitive, idiomatic Go interfaces

## Why WebFram?

WebFram bridges the gap between using the raw `net/http` package and heavyweight frameworks. It provides essential web development features while maintaining the simplicity and performance characteristics of the standard library.

### Comparison with Other Frameworks

| Feature | WebFram | Gin | Echo | Chi | net/http |
|---------|---------|-----|------|-----|----------|
| Built on stdlib | ✅ | ❌ | ❌ | ✅ | ✅ |
| Automatic OpenAPI | ✅ | ❌ | ❌ | ❌ | ❌ |
| Built-in i18n | ✅ | ❌ | ❌ | ❌ | ❌ |
| Template Layouts | ✅ | ❌ | ❌ | ❌ | ❌ |
| JSON Patch | ✅ | ❌ | ❌ | ❌ | ❌ |
| SSE Support | ✅ | ❌ | ❌ | ❌ | ❌ |
| Validation | ✅ | ✅ | ✅ | ❌ | ❌ |
| Learning Curve | Low | Medium | Medium | Low | Low |
| Dependencies | Minimal | Many | Many | Minimal | None |

### Use Cases

- **REST APIs**: Full CRUD operations with OpenAPI documentation
- **Web Applications**: Server-side rendering with i18n support
- **Real-Time Systems**: SSE for live updates and notifications
- **Microservices**: Lightweight footprint with comprehensive features
- **Internal Tools**: Rapid development with validation and documentation

## Architecture & Design

WebFram follows these core design principles:

1. **Standard Library First**: Built on `net/http` for maximum compatibility
2. **Type Safety**: Leverages Go generics for compile-time type checking
3. **Minimal Dependencies**: Only essential, well-maintained dependencies
4. **Composability**: Middleware and handlers can be combined flexibly
5. **Convention over Configuration**: Sensible defaults with override options

### Package Structure

```text
webfram/
├── app.go                  # Core application types and configuration
├── mux.go                  # Enhanced ServeMux with middleware support
├── responseWriter.go       # Extended ResponseWriter with helper methods
├── listener.go             # Custom listener with graceful shutdown
├── internal/
│   ├── bind/              # Data binding and validation
│   ├── i18n/              # Internationalization support
│   └── template/          # Template rendering engine
└── openapi/               # OpenAPI 3.2.0 schema generation
```

## Performance

WebFram is designed for high performance with minimal overhead:

- **Zero Reflection in Hot Path**: Routing uses standard library pattern matching
- **Template Caching**: Templates are parsed once and cached
- **Efficient Validation**: Validation rules are pre-parsed and reused
- **Minimal Allocations**: Careful memory management reduces GC pressure

### Benchmarks

```text
BenchmarkConfigure-8                          1000000         1042 ns/op
BenchmarkBindJSON-8                           500000          2854 ns/op
BenchmarkBindJSON_WithValidation-8            300000          3921 ns/op
BenchmarkPatchJSON-8                          200000          5234 ns/op
BenchmarkSSE_PayloadGeneration-8              2000000          782 ns/op
```

## Security

WebFram includes built-in security features to protect your applications:

### Input Validation

All data binding operations include optional validation:

```go
// Validation is mandatory for Form binding
user, valErrors, err := app.BindForm[User](r)

// Validation is optional for JSON/XML (second parameter)
user, valErrors, err := app.BindJSON[User](r, true)  // With validation
user, valErrors, err := app.BindJSON[User](r, false) // Skip validation
```

### JSONP Security

JSONP callback names are automatically validated to prevent XSS:

```go
// Valid: myCallback, callback_123, _private
// Invalid: 123callback, my-callback, alert('xss')
```

### Content-Type Validation

JSON Patch requires the correct content-type header:

```go
// Must use: application/json-patch+json
// Rejects: application/json, text/plain, etc.
```

### Template Security

Templates automatically escape HTML content:

```html
<!-- User input is automatically escaped -->
<div>{{.UserInput}}</div>
```

## Installation

```bash
go get github.com/bondowe/webfram
```

## Quick Start

```go
package main

import (
    app "github.com/bondowe/webfram"
)

func main() {
    // Create a new mux
    mux := app.NewServeMux()

    // Define a route
    mux.HandleFunc("GET /hello", func(w app.ResponseWriter, r *app.Request) {
        w.JSON(map[string]string{"message": "Hello, World!"})
    })

    // Start the server (nil for default server configuration)
    app.ListenAndServe(":8080", mux, nil)
}
```

## Configuration

WebFram can be configured with templates, i18n, JSONP, and OpenAPI settings:

```go
//go:embed assets
var assetsFS embed.FS

func main() {
    app.Configure(&app.Config{
        Assets: &app.Assets{
            FS: assetsFS,
            Templates: &app.Templates{
                Dir:                   "templates",
                LayoutBaseName:        "layout",
                HTMLTemplateExtension: ".go.html",
                TextTemplateExtension: ".go.txt",
            },
            I18nMessages: &app.I18nMessages{
                Dir: "locales",
            },
        },
        JSONPCallbackParamName: "callback", // Enable JSONP with custom param name
        OpenAPI: &app.OpenAPI{
            EndpointEnabled: true,
            URLPath:         "GET /openapi.json", // Optional, defaults to GET /openapi.json
            Config:          getOpenAPIConfig(),
        },
    })

    mux := app.NewServeMux()
    // ... register routes
    
    app.ListenAndServe(":8080", mux, nil)
}
```

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `Assets.FS` | `nil` (required) | Embedded file system (e.g., `//go:embed assets`) |
| `Assets.Templates.Dir` | `"templates"` | Path to templates directory **within** the embedded FS |
| `Assets.Templates.LayoutBaseName` | `"layout"` | Base name for layout files |
| `Assets.Templates.HTMLTemplateExtension` | `".go.html"` | Extension for HTML templates |
| `Assets.Templates.TextTemplateExtension` | `".go.txt"` | Extension for text templates |
| `Assets.I18nMessages.Dir` | `"i18n"` | Path to locales directory **within** the embedded FS |
| `JSONPCallbackParamName` | `""` (disabled) | Query parameter name for JSONP callbacks |
| `OpenAPI.EndpointEnabled` | `false` | Enable/disable OpenAPI endpoint |
| `OpenAPI.URLPath` | `"GET /openapi.json"` | Path for OpenAPI spec endpoint |
| `OpenAPI.Config` | `nil` | OpenAPI configuration |

**Note:** The i18n function name in templates is always `T` and cannot be configured.

**Important:** The `Templates.Dir` and `I18nMessages.Dir` are relative paths within the embedded filesystem. For example, if you embed `assets` directory with `//go:embed assets`, and your templates are in `assets/templates/`, then set `Templates.Dir` to `"templates"`.

### Configuration Best Practices

1. **Use Embedded Filesystems**: Always use `//go:embed` for your assets directory:

```go
// Project structure:
// assets/
//   ├── templates/
//   │   └── index.go.html
//   └── locales/
//       └── messages.en.json

//go:embed assets
var assetsFS embed.FS

app.Configure(&app.Config{
    Assets: &app.Assets{
        FS: assetsFS,
        // Paths are relative to the embedded FS root
        Templates: &app.Templates{Dir: "templates"},
        I18nMessages: &app.I18nMessages{Dir: "locales"},
    },
})
```

2. **Environment-Specific Configuration**: Use environment variables for deployment-specific settings:

```go
//go:embed assets
var assetsFS embed.FS

func getConfig() *app.Config {
    cfg := &app.Config{
        Assets: &app.Assets{
            FS: assetsFS,
            // Paths relative to embedded FS (assets/templates/ and assets/locales/)
            Templates: &app.Templates{Dir: "templates"},
            I18nMessages: &app.I18nMessages{Dir: "locales"},
        },
    }
    
    // Enable JSONP only in development
    if os.Getenv("ENV") == "development" {
        cfg.JSONPCallbackParamName = "callback"
    }
    
    // Enable OpenAPI in non-production
    if os.Getenv("ENV") != "production" {
        cfg.OpenAPI = &app.OpenAPI{
            EndpointEnabled: true,
            Config: getOpenAPIConfig(),
        }
    }
    
    return cfg
}
```

3. **Validate Configuration**: Check configuration errors early:

```go
func main() {
    // Configure will panic on invalid config
    defer func() {
        if r := recover(); r != nil {
            log.Fatalf("Configuration error: %v", r)
        }
    }()
    
    app.Configure(getConfig())
    // ... rest of app
}
```

4. **Single Configuration Call**: Only call `Configure()` once at startup:

```go
func main() {
    // Configure once before creating any mux
    app.Configure(getConfig())
    
    // Create mux after configuration
    mux := app.NewServeMux()
    // ... register routes
}
```

## Server Configuration

### ListenAndServe

The `ListenAndServe` function starts an HTTP server with the specified address, multiplexer, and optional server configuration. It provides automatic graceful shutdown, OpenAPI endpoint registration, and configurable server timeouts.

**Signature:**
```go
func ListenAndServe(addr string, mux *ServeMux, cfg *ServerConfig)
```

**Parameters:**
- `addr`: Server address (e.g., `:8080`, `localhost:3000`, `0.0.0.0:8080`)
- `mux`: ServeMux instance with registered routes
- `cfg`: Optional ServerConfig for customizing server behavior (can be `nil` for defaults)

**Features:**
- Automatically registers OpenAPI endpoint if configured
- Graceful shutdown on SIGINT/SIGTERM signals
- Configurable timeouts with sensible defaults
- Panics on startup or shutdown errors for fail-fast behavior

**Basic Usage:**
```go
mux := app.NewServeMux()
mux.HandleFunc("GET /hello", handleHello)

// Use default server configuration
app.ListenAndServe(":8080", mux, nil)
```

**With Custom Configuration:**
```go
mux := app.NewServeMux()
mux.HandleFunc("GET /hello", handleHello)

// Custom server configuration
serverCfg := &app.ServerConfig{
    ReadTimeout:       30 * time.Second,
    WriteTimeout:      30 * time.Second,
    IdleTimeout:       120 * time.Second,
    MaxHeaderBytes:    1 << 20, // 1 MB
    ReadHeaderTimeout: 10 * time.Second,
}

app.ListenAndServe(":8080", mux, serverCfg)
```

### ServerConfig Options

The `ServerConfig` struct allows fine-grained control over HTTP server behavior:

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `ReadTimeout` | `time.Duration` | `15s` | Maximum duration for reading entire request |
| `ReadHeaderTimeout` | `time.Duration` | `15s` | Maximum duration for reading request headers |
| `WriteTimeout` | `time.Duration` | `15s` | Maximum duration for writing response |
| `IdleTimeout` | `time.Duration` | `60s` | Maximum idle time for keep-alive connections |
| `MaxHeaderBytes` | `int` | `1048576` (1MB) | Maximum size of request headers |
| `TLSConfig` | `*tls.Config` | `nil` | TLS configuration for HTTPS |
| `ConnState` | `func(net.Conn, http.ConnState)` | `nil` | Callback for connection state changes |
| `BaseContext` | `func(net.Listener) context.Context` | `nil` | Base context for all requests |
| `ConnContext` | `func(context.Context, net.Conn) context.Context` | `nil` | Per-connection context function |
| `ErrorLog` | `*slog.Logger` | `nil` | Custom error logger |
| `HTTP2` | `*http.HTTP2Config` | `nil` | HTTP/2 server configuration |
| `Protocols` | `*http.Protocols` | `nil` | HTTP protocol configuration |
| `TLSNextProto` | `map[string]func(...)` | `nil` | NPN/ALPN protocol upgrade functions |
| `DisableGeneralOptionsHandler` | `bool` | `false` | Disable automatic OPTIONS handling |

### Server Configuration Examples

#### Production Server with Timeouts

```go
func main() {
    app.Configure(getConfig())
    mux := app.NewServeMux()
    registerRoutes(mux)

    serverCfg := &app.ServerConfig{
        ReadTimeout:       30 * time.Second,
        WriteTimeout:      30 * time.Second,
        IdleTimeout:       120 * time.Second,
        ReadHeaderTimeout: 10 * time.Second,
        MaxHeaderBytes:    2 << 20, // 2 MB
    }

    app.ListenAndServe(":8080", mux, serverCfg)
}
```

#### HTTPS Server with TLS

```go
func main() {
    // Load TLS certificates
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

    mux := app.NewServeMux()
    registerRoutes(mux)

    serverCfg := &app.ServerConfig{
        TLSConfig:         tlsConfig,
        ReadTimeout:       30 * time.Second,
        WriteTimeout:      30 * time.Second,
        IdleTimeout:       120 * time.Second,
        ReadHeaderTimeout: 10 * time.Second,
    }

    app.ListenAndServe(":443", mux, serverCfg)
}
```

#### Custom Connection Tracking

```go
func main() {
    var activeConnections atomic.Int32

    mux := app.NewServeMux()
    registerRoutes(mux)

    serverCfg := &app.ServerConfig{
        ConnState: func(conn net.Conn, state http.ConnState) {
            switch state {
            case http.StateNew:
                count := activeConnections.Add(1)
                log.Printf("New connection from %s (total: %d)", conn.RemoteAddr(), count)
            case http.StateClosed:
                count := activeConnections.Add(-1)
                log.Printf("Connection closed from %s (total: %d)", conn.RemoteAddr(), count)
            }
        },
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
    }

    app.ListenAndServe(":8080", mux, serverCfg)
}
```

#### Custom Request Context

```go
func main() {
    mux := app.NewServeMux()
    registerRoutes(mux)

    serverCfg := &app.ServerConfig{
        BaseContext: func(listener net.Listener) context.Context {
            // Create base context with common values
            ctx := context.Background()
            ctx = context.WithValue(ctx, "serverStartTime", time.Now())
            return ctx
        },
        ConnContext: func(ctx context.Context, conn net.Conn) context.Context {
            // Add per-connection values
            return context.WithValue(ctx, "remoteAddr", conn.RemoteAddr().String())
        },
    }

    app.ListenAndServe(":8080", mux, serverCfg)
}
```

#### HTTP/2 Configuration

```go
func main() {
    mux := app.NewServeMux()
    registerRoutes(mux)

    serverCfg := &app.ServerConfig{
        HTTP2: &http.HTTP2Config{
            MaxConcurrentStreams:         250,
            MaxUploadBufferPerConnection: 1 << 20, // 1 MB
        },
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
    }

    app.ListenAndServe(":8080", mux, serverCfg)
}
```

#### Development vs Production Configuration

```go
func getServerConfig() *app.ServerConfig {
    if os.Getenv("ENV") == "production" {
        return &app.ServerConfig{
            ReadTimeout:       30 * time.Second,
            WriteTimeout:      30 * time.Second,
            IdleTimeout:       120 * time.Second,
            ReadHeaderTimeout: 10 * time.Second,
            MaxHeaderBytes:    2 << 20, // 2 MB
        }
    }
    
    // Development: more lenient timeouts for debugging
    return &app.ServerConfig{
        ReadTimeout:       5 * time.Minute,
        WriteTimeout:      5 * time.Minute,
        IdleTimeout:       10 * time.Minute,
        ReadHeaderTimeout: 1 * time.Minute,
    }
}

func main() {
    app.Configure(getConfig())
    mux := app.NewServeMux()
    registerRoutes(mux)

    app.ListenAndServe(":8080", mux, getServerConfig())
}
```

**Note:** `ListenAndServe` blocks until the server is shut down. It automatically handles graceful shutdown on receiving SIGINT or SIGTERM signals, ensuring all active connections are properly closed before the process exits.

## Routing

WebFram uses Go 1.22+ routing patterns with HTTP method prefixes, providing powerful and flexible route matching.

### Basic Routes

Define routes with HTTP method prefixes:

```go
mux := app.NewServeMux()

// Simple routes with different HTTP methods
mux.HandleFunc("GET /users", listUsers)
mux.HandleFunc("POST /users", createUser)
mux.HandleFunc("PUT /users/{id}", updateUser)
mux.HandleFunc("PATCH /users/{id}", patchUser)
mux.HandleFunc("DELETE /users/{id}", deleteUser)

// Multiple methods for same path
mux.HandleFunc("GET /health", healthCheck)
mux.HandleFunc("POST /health", healthCheckDetailed)
```

### Route Parameters

Access path parameters using `r.PathValue()`:

```go
mux.HandleFunc("GET /users/{id}", func(w app.ResponseWriter, r *app.Request) {
    userID := r.PathValue("id")
    
    // Use the parameter
    user, err := getUserByID(userID)
    if err != nil {
        w.Error(http.StatusNotFound, "User not found")
        return
    }
    
    w.JSON(user)
})

// Multiple parameters
mux.HandleFunc("GET /posts/{year}/{month}/{slug}", func(w app.ResponseWriter, r *app.Request) {
    year := r.PathValue("year")
    month := r.PathValue("month")
    slug := r.PathValue("slug")
    
    post := getPost(year, month, slug)
    w.JSON(post)
})
```

### Wildcard Routes

Use wildcards to match remaining path segments:

```go
// Serve static files
mux.HandleFunc("GET /static/{path...}", func(w app.ResponseWriter, r *app.Request) {
    filepath := r.PathValue("path")
    http.ServeFile(w.ResponseWriter, r.Request, filepath)
})

// API versioning
mux.HandleFunc("GET /api/v1/{resource...}", apiV1Handler)
mux.HandleFunc("GET /api/v2/{resource...}", apiV2Handler)
```

### Route Patterns

Go 1.22 routing supports these patterns:

```go
// Exact match
mux.HandleFunc("GET /exact", handler)

// Single segment parameter
mux.HandleFunc("GET /users/{id}", handler)

// Multiple parameters
mux.HandleFunc("GET /posts/{year}/{month}/{day}", handler)

// Wildcard (remaining path)
mux.HandleFunc("GET /files/{path...}", handler)

// Host-based routing
mux.HandleFunc("api.example.com/", apiHandler)
mux.HandleFunc("www.example.com/", wwwHandler)
```

## Middleware

WebFram supports both custom and standard HTTP middleware.

### Global Middleware

Applied to all routes:

```go
// Custom middleware
app.Use(func(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        // Before handler
        log.Printf("Request: %s %s", r.Method, r.URL.Path)
        
        next.ServeHTTP(w, r)
        
        // After handler (if needed)
    })
})

// Standard HTTP middleware
app.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Standard middleware logic
        next.ServeHTTP(w, r)
    })
})
```

### Mux-Level Middleware

Applied to all routes in a specific mux:

```go
mux := app.NewServeMux()

mux.Use(func(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        // Mux-level middleware logic
        next.ServeHTTP(w, r)
    })
})
```

### Route-Specific Middleware

Applied to individual routes:

```go
mux.HandleFunc("GET /admin", adminHandler, 
    authMiddleware,
    loggingMiddleware,
)
```

### Middleware Execution Order

Middleware executes in this order:

1. Global middleware (registered with `app.Use()`)
2. Mux-level middleware (registered with `mux.Use()`)
3. Route-specific middleware
4. i18n middleware (automatic)
5. Handler

### Standard HTTP Middleware Support

WebFram seamlessly integrates with standard `http.Handler` middleware from the ecosystem:

```go
import (
    "github.com/gorilla/csrf"
    "github.com/rs/cors"
    app "github.com/bondowe/webfram"
)

func main() {
    app.Configure(nil)
    
    // Use standard HTTP middleware
    app.Use(cors.Default().Handler)
    
    mux := app.NewServeMux()
    
    // CSRF protection (standard middleware)
    csrfMiddleware := csrf.Protect(
        []byte("32-byte-long-auth-key"),
        csrf.Secure(false), // Set to true in production
    )
    app.Use(csrfMiddleware)
    
    // Custom middleware works too
    app.Use(loggingMiddleware)
    
    mux.HandleFunc("GET /", handler)
    
    app.ListenAndServe(":8080", mux, nil)
}

func loggingMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("%s %s - %v", r.Method, r.URL.Path, time.Since(start))
    })
}
```

**Compatible Libraries:**

- CORS: `github.com/rs/cors`
- CSRF: `github.com/gorilla/csrf`
- Compression: `github.com/nytimes/gziphandler`
- Rate Limiting: `golang.org/x/time/rate`
- Authentication: `github.com/golang-jwt/jwt`
- Session: `github.com/gorilla/sessions`

## Request Handling

### Query Parameters

```go
mux.HandleFunc("GET /search", func(w app.ResponseWriter, r *app.Request) {
    query := r.URL.Query().Get("q")
    page := r.URL.Query().Get("page")
    // ... handle request
})
```

### Request Body

Access the raw request body:

```go
body, err := io.ReadAll(r.Body)
defer r.Body.Close()
```

## Response Handling

WebFram provides multiple response methods:

### JSON Response

```go
w.JSON(map[string]string{"message": "Success"})
```

The `JSON` method automatically handles JSONP requests if configured (see [JSONP Support](#jsonp-support)).

### HTML Response

```go
// Render a template
data := map[string]interface{}{"Name": "John"}
err := w.HTML("users/profile", data)
```

### HTML String Response

```go
err := w.HTMLString("<h1>{{.Title}}</h1>", map[string]string{"Title": "Hello"})
```

### Text Response

```go
// Render a text template
err := w.Text("users/email", data)
```

### Text String Response

```go
err := w.TextString("Hello {{.Name}}", map[string]string{"Name": "John"})
```

### XML Response

```go
type User struct {
    XMLName xml.Name `xml:"user"`
    Name    string   `xml:"name"`
}
err := w.XML(User{Name: "John"})
```

### YAML Response

```go
err := w.YAML(map[string]string{"name": "John"})
```

### Binary Response

```go
data := []byte{...}
err := w.Bytes(data, "application/pdf")
```

### No Content

```go
w.NoContent() // Returns 204 No Content
```

### Redirect

```go
w.Redirect(r.Request, "/login", http.StatusSeeOther)
```

### File Download

```go
// Inline display
w.ServeFile(r.Request, "/path/to/file.pdf", true)

// Force download
w.ServeFile(r.Request, "/path/to/file.pdf", false)
```

### Error Response

```go
w.Error(http.StatusBadRequest, "Invalid request")
```

### Custom Headers

```go
w.Header().Set("X-Custom-Header", "value")
w.WriteHeader(http.StatusOK)
w.JSON(data)
```

## Data Binding & Validation

WebFram provides type-safe data binding with comprehensive validation for Form, JSON, and XML formats.

### Form Binding

Form binding automatically parses form data and validates it according to struct tags.

```go
type CreateUserRequest struct {
    Name      string    `form:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"`
    Email     string    `form:"email" validate:"required,format=email" errmsg:"required=Email is required;format=Invalid email address"`
    Age       int       `form:"age" validate:"min=18,max=120" errmsg:"min=Must be at least 18;max=Must be at most 120"`
    Role      string    `form:"role" validate:"enum=admin|user|guest" errmsg:"enum=Must be admin, user, or guest"`
    Birthdate time.Time `form:"birthdate" validate:"required" format:"2006-01-02" errmsg:"required=Birthdate is required"`
    Hobbies   []string  `form:"hobbies" validate:"minItems=1,maxItems=5,uniqueItems" errmsg:"minItems=At least one hobby required;maxItems=Maximum 5 hobbies allowed"`
}

mux.HandleFunc("POST /users", func(w app.ResponseWriter, r *app.Request) {
    user, valErrors, err := app.BindForm[CreateUserRequest](r)
    
    if err != nil {
        w.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    if valErrors.Any() {
        w.WriteHeader(http.StatusBadRequest)
        w.JSON(valErrors)
        return
    }
    
    // Process valid user data
    w.JSON(user)
})
```

**Form data example:**

```
name=John+Doe&email=john@example.com&age=30&role=admin&birthdate=1993-01-15&hobbies=reading&hobbies=coding
```

### JSON Binding

JSON binding parses JSON request bodies with optional validation.

```go
type CreateUserRequest struct {
    Name    string   `json:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"`
    Email   string   `json:"email" validate:"required,format=email" errmsg:"format=Invalid email address"`
    Hobbies []string `json:"hobbies" validate:"minItems=1,maxItems=5,uniqueItems" errmsg:"minItems=At least one hobby required;maxItems=Maximum 5 hobbies allowed"`
    Age     int      `json:"age" validate:"min=18,max=120,multipleOf=1"`
}

mux.HandleFunc("POST /api/users", func(w app.ResponseWriter, r *app.Request) {
    // Second parameter controls validation (true = validate, false = skip validation)
    user, valErrors, err := app.BindJSON[CreateUserRequest](r, true)
    
    if err != nil {
        w.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    if valErrors.Any() {
        w.WriteHeader(http.StatusBadRequest)
        w.JSON(valErrors)
        return
    }
    
    // Process valid user data
    w.JSON(user)
})
```

**JSON request example:**

```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "hobbies": ["reading", "coding"],
  "age": 30
}
```

### XML Binding

WebFram supports XML binding with the same validation features as JSON:

```go
type CreateUserRequest struct {
    XMLName   xml.Name  `xml:"user"`
    Name      string    `xml:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"`
    Email     string    `xml:"email" validate:"required,format=email" errmsg:"format=Invalid email address"`
    Age       int       `xml:"age" validate:"min=18,max=120"`
    Role      string    `xml:"role" validate:"enum=admin|user|guest" errmsg:"enum=Must be admin, user, or guest"`
}

mux.HandleFunc("POST /api/users", func(w app.ResponseWriter, r *app.Request) {
    // Second parameter controls validation (true = validate, false = skip validation)
    user, valErrors, err := app.BindXML[CreateUserRequest](r, true)
    
    if err != nil {
        w.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    if valErrors.Any() {
        w.WriteHeader(http.StatusBadRequest)
        w.XML(valErrors)
        return
    }
    
    // Process valid user data
    w.XML(user)
})
```

**Example XML request:**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<user>
    <name>John Doe</name>
    <email>john@example.com</email>
    <age>30</age>
    <role>admin</role>
</user>
```

### Nested Structs

All binding types support nested structs:

```go
type Address struct {
    Street string `json:"street" xml:"street" form:"street" validate:"required" errmsg:"required=Street is required"`
    City   string `json:"city" xml:"city" form:"city" validate:"required" errmsg:"required=City is required"`
    Zip    int    `json:"zip" xml:"zip" form:"zip" validate:"min=10000,max=99999" errmsg:"min=Invalid zip;max=Invalid zip"`
}

type User struct {
    Name    string  `json:"name" xml:"name" form:"name" validate:"required"`
    Address Address `json:"address" xml:"address" form:"address" validate:"required"`
}

// Form fields: name, address.street, address.city, address.zip
// JSON/XML: nested objects
```

**Form data example:**

```
name=John+Doe&address.street=123+Main+St&address.city=Springfield&address.zip=12345
```

**JSON example:**

```json
{
  "name": "John Doe",
  "address": {
    "street": "123 Main St",
    "city": "Springfield",
    "zip": 12345
  }
}
```

### Map Binding (Form Only)

Form binding supports maps with the syntax `fieldname[key]=value`:

```go
type Config struct {
    Metadata map[string]string `form:"metadata" validate:"minItems=1,maxItems=10"`
    Scores   map[string]int    `form:"scores"`
    Settings map[int]string    `form:"settings"`
}

mux.HandleFunc("POST /config", func(w app.ResponseWriter, r *app.Request) {
    config, valErrors, err := app.BindForm[Config](r)
    
    if err != nil {
        w.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    if valErrors.Any() {
        w.WriteHeader(http.StatusBadRequest)
        w.JSON(valErrors)
        return
    }
    
    w.JSON(config)
})
```

**Form data example:**

```
metadata[color]=red&metadata[size]=large&scores[math]=95&scores[science]=87&settings[1]=enabled&settings[2]=disabled
```

**Supported map types:**

- `map[string]string`
- `map[string]int`
- `map[int]string`
- `map[int]int`
- Other basic types (uint, float, bool)
- `map[string]time.Time`
- `map[string]uuid.UUID`

### Validation Tags Reference

WebFram supports 20+ validation tags for comprehensive data validation:

| Tag | Applies To | Description | Example |
|-----|------------|-------------|---------|
| `required` | All types | Field must be present and non-empty | `validate:"required"` |
| `min=N` | int, uint, float | Minimum value (inclusive) | `validate:"min=18"` |
| `max=N` | int, uint, float | Maximum value (inclusive) | `validate:"max=120"` |
| `multipleOf=N` | int, float | Value must be a multiple of N | `validate:"multipleOf=5"` |
| `minlength=N` | string | Minimum length in characters | `validate:"minlength=3"` |
| `maxlength=N` | string | Maximum length in characters | `validate:"maxlength=50"` |
| `minItems=N` | slice, map | Minimum number of items | `validate:"minItems=1"` |
| `maxItems=N` | slice, map | Maximum number of items | `validate:"maxItems=10"` |
| `uniqueItems` | slice | All items must be unique | `validate:"uniqueItems"` |
| `emptyItemsAllowed` | slice | Allow empty items in slice | `validate:"emptyItemsAllowed"` |
| `regexp=PATTERN` | string | Must match regular expression | `validate:"regexp=^\\w+@\\w+\\.com$"` |
| `pattern=PATTERN` | string | Alias for regexp | `validate:"pattern=^[A-Z]{3}-\\d{4}$"` |
| `enum=val1\|val2` | string | Must be one of specified values | `validate:"enum=admin\|user\|guest"` |
| `format=email` | string (form) | Must be a valid email (IDN supported) | `validate:"format=email"` |
| `format=LAYOUT` | time.Time | Time parsing layout | `format:"2006-01-02"` |

**Validation rules can be combined:**

```go
type Product struct {
    Name  string `json:"name" validate:"required,minlength=2,maxlength=100"`
    SKU   string `json:"sku" validate:"required,regexp=^[A-Z]{3}-\\d{4}$"`
    Price int    `json:"price" validate:"required,min=0,max=1000000,multipleOf=100"`
    Tags  []string `json:"tags" validate:"minItems=1,maxItems=20,uniqueItems"`
}
```

### Custom Error Messages

Use the `errmsg` tag to provide custom validation error messages:

```go
type User struct {
    Name  string `json:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"`
    Email string `json:"email" validate:"required,format=email" errmsg:"required=Email is required;format=Please provide a valid email address"`
    Age   int    `json:"age" validate:"min=18,max=120" errmsg:"min=Must be at least 18 years old;max=Must be at most 120 years old"`
}
```

**Format:** `errmsg:"validationRule1=Message1;validationRule2=Message2"`

### Validation Errors

WebFram provides a structured `ValidationErrors` type for handling validation errors:

```go
type ValidationErrors struct {
    Errors []ValidationError `json:"errors" xml:"errors"`
}

type ValidationError struct {
    Field string `json:"field" xml:"field"`
    Error string `json:"error" xml:"error"`
}
```

**Methods:**

- `Any() bool` - Returns true if there are any validation errors

**Example usage:**

```go
user, valErrors, err := app.BindJSON[CreateUserRequest](r, true)

if err != nil {
    // Binding error (malformed JSON, etc.)
    w.Error(http.StatusBadRequest, err.Error())
    return
}

if valErrors.Any() {
    // Validation errors - return structured error response
    w.WriteHeader(http.StatusBadRequest)
    w.JSON(valErrors)
    return
}

// No errors - proceed with valid data
```

**Validation error response (JSON format):**

```json
{
  "errors": [
    {
      "field": "name",
      "error": "Name is required"
    },
    {
      "field": "email",
      "error": "Invalid email address"
    },
    {
      "field": "age",
      "error": "Must be at least 18 years old"
    }
  ]
}
```

**Validation error response (XML format):**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<validationErrors>
    <errors>
        <validationError>
            <field>name</field>
            <error>Name is required</error>
        </validationError>
        <validationError>
            <field>email</field>
            <error>Invalid email address</error>
        </validationError>
        <validationError>
            <field>age</field>
            <error>Must be at least 18 years old</error>
        </validationError>
    </errors>
</validationErrors>
```

### Supported Field Types

- **Primitives**: `string`, `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`, `bool`
- **Time**: `time.Time`
- **UUID**: `uuid.UUID` (from `github.com/google/uuid`)
- **Slices**: `[]string`, `[]int`, `[]time.Time`, `[]uuid.UUID`, etc.
- **Maps** (form only): `map[string]string`, `map[string]int`, `map[int]string`, etc.
- **Nested structs**: Any struct type
- **Pointers**: All types support pointer variants

### Skipping Validation

For JSON and XML binding, you can skip validation by passing `false` as the second parameter:

```go
// Skip validation - useful when you trust the data source
user, valErrors, err := app.BindJSON[User](r, false)
// valErrors will be empty, only binding errors are checked

// With validation enabled
user, valErrors, err := app.BindJSON[User](r, true)
// Both binding and validation errors are checked
```

**Note:** Form binding always performs validation.

## JSON Patch Support

WebFram supports [RFC 6902 JSON Patch](https://tools.ietf.org/html/rfc6902) for partial resource updates using the `PATCH` HTTP method.

### Using JSON Patch

```go
type User struct {
    ID    uuid.UUID `json:"id"`
    Name  string    `json:"name" validate:"required,minlength=3"`
    Email string    `json:"email" validate:"required,format=email"`
    Role  string    `json:"role" validate:"enum=admin|user|guest"`
}

mux.HandleFunc("PATCH /users/{id}", func(w app.ResponseWriter, r *app.Request) {
    id := r.PathValue("id")
    
    // Fetch the existing user from database
    user, err := getUserFromDB(id)
    if err != nil {
        w.Error(http.StatusNotFound, "User not found")
        return
    }
    
    // Apply JSON Patch operations (with validation)
    valErrors, err := app.PatchJSON(r, &user, true)
    if err != nil {
        if err == app.ErrMethodNotAllowed {
            w.Error(http.StatusMethodNotAllowed, "PATCH method required")
            return
        }
        w.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    if len(valErrors) > 0 {
        w.WriteHeader(http.StatusBadRequest)
        w.JSON(app.ValidationErrors{Errors: valErrors})
        return
    }
    
    // Save the updated user to database
    err = saveUserToDB(user)
    if err != nil {
        w.Error(http.StatusInternalServerError, err.Error())
        return
    }
    
    w.JSON(user)
})
```

### JSON Patch Request Example

```bash
curl -X PATCH http://localhost:8080/users/123 \
  -H "Content-Type: application/json-patch+json" \
  -d '[
    {"op": "replace", "path": "/name", "value": "John Updated"},
    {"op": "replace", "path": "/email", "value": "john.updated@example.com"}
  ]'
```

### Supported JSON Patch Operations

The `PatchJSON` function supports all standard JSON Patch operations:

- **add**: Add a new value

  ```json
  {"op": "add", "path": "/email", "value": "new@example.com"}
  ```

- **remove**: Remove a value

  ```json
  {"op": "remove", "path": "/email"}
  ```

- **replace**: Replace an existing value

  ```json
  {"op": "replace", "path": "/name", "value": "New Name"}
  ```

- **move**: Move a value from one location to another

  ```json
  {"op": "move", "from": "/oldField", "path": "/newField"}
  ```

- **copy**: Copy a value to a new location

  ```json
  {"op": "copy", "from": "/name", "path": "/displayName"}
  ```

- **test**: Test that a value at the target location is equal to a specified value

  ```json
  {"op": "test", "path": "/name", "value": "Expected Name"}
  ```

### JSON Patch with Validation

The third parameter to `PatchJSON` controls whether validation is performed after applying the patch:

```go
// With validation - recommended for user input
valErrors, err := app.PatchJSON(r, &user, true)
if err != nil {
    // Handle patch errors
}
if len(valErrors) > 0 {
    // Handle validation errors
}

// Without validation - for trusted operations
valErrors, err := app.PatchJSON(r, &user, false)
// Only patch errors are checked, validation is skipped
```

### Error Handling

The `PatchJSON` function returns specific errors:

- `app.ErrMethodNotAllowed`: When called on non-PATCH requests
- Content-Type validation: Requires `application/json-patch+json` header
- Patch errors: Invalid JSON or malformed patch operations
- Validation errors: Returned as `[]ValidationError` when validation is enabled

```go
valErrors, err := app.PatchJSON(r, &resource, true)
if err != nil {
    if err == app.ErrMethodNotAllowed {
        w.Error(http.StatusMethodNotAllowed, "Only PATCH method is allowed")
        return
    }
    // Other errors (invalid JSON, invalid patch, wrong Content-Type)
    w.Error(http.StatusBadRequest, err.Error())
    return
}

if len(valErrors) > 0 {
    // Validation failed after applying patch
    w.WriteHeader(http.StatusBadRequest)
    w.JSON(app.ValidationErrors{Errors: valErrors})
    return
}
```

## JSONP Support

WebFram provides built-in support for JSONP (JSON with Padding) to enable cross-origin requests from browsers that don't support CORS.

### Configuring JSONP

Enable JSONP by setting the `JSONPCallbackParamName` in your application configuration:

```go
app.Configure(&app.Config{
    JSONPCallbackParamName: "callback", // Enable JSONP with "callback" query parameter
    // ... other config options
})
```

If `JSONPCallbackParamName` is not set or is empty, JSONP is disabled and all JSON responses are returned as standard JSON.

**Note:** The callback parameter name itself is validated and must start with a letter or underscore and contain only alphanumeric characters and underscores.

### Using JSONP

Once configured, any route that uses `w.JSON()` will automatically support JSONP when the callback parameter is present in the query string:

```go
mux.HandleFunc("GET /api/users", func(w app.ResponseWriter, r *app.Request) {
    users := []User{
        {ID: uuid.New(), Name: "John Doe"},
        {ID: uuid.New(), Name: "Jane Smith"},
    }
    
    // Automatically handles both JSON and JSONP
    w.JSON(users)
})
```

### JSONP Request Examples

**Standard JSON request** (no callback parameter):

```bash
curl http://localhost:8080/api/users
```

Response:

```json
[
  {"id": "123...", "name": "John Doe"},
  {"id": "456...", "name": "Jane Smith"}
]
```

**JSONP request** (with callback parameter):

```bash
curl http://localhost:8080/api/users?callback=myCallback
```

Response:

```javascript
myCallback([
  {"id": "123...", "name": "John Doe"},
  {"id": "456...", "name": "Jane Smith"}
]);
```

### Client-Side JSONP Usage

**Using vanilla JavaScript:**

```html
<script>
function myCallback(data) {
    console.log('Received data:', data);
    // Process the data
}

// Create script tag to make JSONP request
var script = document.createElement('script');
script.src = 'http://localhost:8080/api/users?callback=myCallback';
document.body.appendChild(script);
</script>
```

**Using jQuery:**

```javascript
$.ajax({
    url: 'http://localhost:8080/api/users',
    dataType: 'jsonp',
    jsonpCallback: 'myCallback',
    success: function(data) {
        console.log('Received data:', data);
    }
});
```

### JSONP Response Details

When JSONP is enabled and a callback parameter is provided:

- Content-Type is set to `application/javascript`
- Response is wrapped with the callback function: `callbackName(jsonData);`
- The callback parameter name is configurable via `JSONPCallbackParamName`

### Callback Name Validation

🔒 **Built-in Security**: WebFram automatically validates JSONP callback names to prevent security vulnerabilities:

- **Allowed characters**: Only alphanumeric characters (a-z, A-Z, 0-9) and underscores (_)
- **Must start with**: Letter or underscore
- **Validation pattern**: `^[a-zA-Z_][a-zA-Z0-9_]*$`
- **Invalid callbacks**: Return a `400 Bad Request` error with a descriptive message

**Valid callback names:**

- `myCallback`
- `callback123`
- `my_callback_function`
- `_privateCallback`
- `jQuery123456789_callback`

**Invalid callback names:**

- `123callback` (starts with number)
- `my-callback` (contains hyphen)
- `callback()` (contains parentheses)
- `alert('xss')` (potential XSS attempt)
- `../../../etc/passwd` (path traversal attempt)

**Error response for invalid callback:**

```http
HTTP/1.1 400 Bad Request
Content-Type: text/plain

invalid JSONP callback method name: "my-callback()". Only alphanumeric characters and underscores are allowed
```

### Security Considerations

⚠️ **Important JSONP Security Notes:**

1. **Callback Validation**: WebFram automatically validates callback names to prevent XSS attacks and malicious code injection. Only alphanumeric characters and underscores are allowed, and the name must start with a letter or underscore.

2. **CORS Alternative**: If possible, use CORS instead of JSONP for modern browsers:

   ```go
   mux.Use(func(next app.Handler) app.Handler {
       return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
           w.Header().Set("Access-Control-Allow-Origin", "*")
           next.ServeHTTP(w, r)
       })
   })
   ```

3. **Sensitive Data**: Avoid exposing sensitive data through JSONP endpoints as they can be requested from any origin.

4. **Read-Only Operations**: Use JSONP only for read-only GET requests. Never use JSONP for state-changing operations (POST, PUT, DELETE, PATCH).

## OpenAPI Documentation

WebFram automatically generates OpenAPI 3.2.0 documentation from your route definitions, validation tags, and API configurations.

### Enabling OpenAPI

Configure OpenAPI in your application:

```go
app.Configure(&app.Config{
    OpenAPI: &app.OpenAPI{
        EndpointEnabled: true,
        URLPath:         "GET /openapi.json", // Optional, defaults to GET /openapi.json
        Config:          getOpenAPIConfig(),
    },
})

func getOpenAPIConfig() *openapi.Config {
    return &openapi.Config{
        Info: &openapi.Info{
            Title:          "My API",
            Summary:        "API for my awesome application",
            Description:    "This API provides endpoints for managing users and products.",
            TermsOfService: "https://example.com/terms/",
            Contact: &openapi.Contact{
                Name:  "API Support",
                URL:   "https://example.com/support",
                Email: "support@example.com",
            },
            License: &openapi.License{
                Name:       "MIT",
                Identifier: "MIT",
                URL:        "https://opensource.org/licenses/MIT",
            },
            Version: "1.0.0",
        },
        Servers: []openapi.Server{
            {
                URL:         "http://localhost:8080",
                Description: "Local development server",
                Name:        "local",
            },
            {
                URL:         "https://api.example.com",
                Description: "Production server",
                Name:        "production",
            },
        },
    }
}
```

Once configured, access your OpenAPI spec at: `http://localhost:8080/openapi.json`

### Documenting Routes

Use `WithAPIConfig()` to add OpenAPI documentation to individual routes:

```go
mux.HandleFunc("POST /users", createUserHandler).WithAPIConfig(&app.APIConfig{
    OperationID: "createUser",
    Summary:     "Create a new user",
    Description: "Creates a new user account with the provided information.",
    Tags:        []string{"Users"},
    Parameters: []app.Parameter{
        {
            Name:        "X-Request-ID",
            In:          "header",
            Description: "Unique request identifier",
            Required:    false,
            TypeHint:    "",
            Example:     "550e8400-e29b-41d4-a716-446655440000",
        },
    },
    RequestBody: &app.RequestBody{
        Description: "User creation data",
        Required:    true,
        Content: map[string]app.TypeInfo{
            "application/json": {
                TypeHint: &User{},
                Examples: map[string]app.Example{
                    "admin": {
                        Summary:   "Admin user",
                        DataValue: User{Name: "Admin User", Role: "admin"},
                    },
                    "regular": {
                        Summary:   "Regular user",
                        DataValue: User{Name: "Regular User", Role: "user"},
                    },
                },
            },
        },
    },
    Responses: map[string]app.Response{
        "201": {
            Summary:     "User created successfully",
            Description: "The user was created successfully",
            Content: map[string]app.TypeInfo{
                "application/json": {
                    TypeHint: &User{},
                },
            },
        },
        "400": {
            Description: "Invalid request data",
        },
        "500": {
            Description: "Internal server error",
        },
    },
})
```

### Path-Level Configuration

Configure documentation for entire paths that apply to all operations:

```go
app.SetOpenAPIPathInfo("/users/{id}", &app.PathInfo{
    Summary:     "User operations",
    Description: "Endpoints for managing individual users",
    Parameters: []app.Parameter{
        {
            Name:        "id",
            In:          "path",
            Description: "User ID",
            Required:    true,
            TypeHint:    "",
            MinLength:   36,
            MaxLength:   36,
            Example:     "550e8400-e29b-41d4-a716-446655440000",
        },
    },
    Servers: []app.Server{
        {Name: "Local", URL: "http://localhost:8080"},
    },
})
```

### Schema Generation from Struct Tags

WebFram automatically generates JSON schemas from struct tags, including validation rules:

```go
type User struct {
    Name    string   `json:"name" validate:"required,minlength=3,maxlength=50"`
    Email   string   `json:"email" validate:"required,format=email"`
    Age     int      `json:"age" validate:"min=18,max=120"`
    Role    string   `json:"role" validate:"enum=admin|user|guest"`
    Hobbies []string `json:"hobbies" validate:"minItems=1,maxItems=10,uniqueItems"`
}
```

This generates an OpenAPI schema with:

- Required fields
- String length constraints (minLength, maxLength)
- Numeric constraints (minimum, maximum)
- Enum values
- Array constraints (minItems, maxItems, uniqueItems)
- Format specifications (email, uuid, date-time)

### Viewing OpenAPI Documentation

After starting your server, access the OpenAPI spec at your configured endpoint:

```bash
curl http://localhost:8080/openapi.json
```

You can also use tools like Swagger UI or Redoc to visualize your API documentation:

```html
<!-- Swagger UI -->
<!DOCTYPE html>
<html>
<head>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@latest/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@latest/swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({
            url: 'http://localhost:8080/openapi.json',
            dom_id: '#swagger-ui',
        })
    </script>
</body>
</html>
```

## Server-Sent Events (SSE)

WebFram provides built-in support for Server-Sent Events (SSE), enabling real-time server-to-client communication over HTTP. SSE is ideal for push notifications, live updates, streaming data, and real-time dashboards.

### Creating an SSE Endpoint

Use the `app.SSE()` function to create an SSE handler:

```go
mux.Handle("GET /events", app.SSE(
    payloadFunc,      // Function that generates SSE payload
    disconnectFunc,   // Function called when client disconnects
    errorFunc,        // Function called on errors
    interval,         // Time interval between messages
    headers,          // Optional custom headers
))
```

### SSE Payload Structure

The `SSEPayload` struct defines the message format:

```go
type SSEPayload struct {
    Id       string        // Event ID (optional)
    Event    string        // Event type/name (optional)
    Comments []string      // Comments (optional, for debugging)
    Data     any          // Data payload (required)
    Retry    time.Duration // Retry interval (optional)
}
```

### Basic Example

```go
mux.Handle("GET /time", app.SSE(
    // Payload function - generates data to send
    func() app.SSEPayload {
        return app.SSEPayload{
            Data: fmt.Sprintf("Current time: %s", time.Now().Format(time.RFC3339)),
        }
    },
    // Disconnect function - called when client disconnects
    func() {
        log.Println("Client disconnected")
    },
    // Error function - called on errors
    func(err error) {
        log.Printf("SSE error: %v\n", err)
    },
    // Interval - send message every 2 seconds
    2*time.Second,
    // Custom headers - nil for default
    nil,
))
```

### Advanced Example with Event Types

```go
mux.Handle("GET /notifications", app.SSE(
    func() app.SSEPayload {
        // Simulate different notification types
        notificationType := getNextNotification()
        
        return app.SSEPayload{
            Id:       uuid.New().String(),           // Unique event ID
            Event:    notificationType,               // Event type (e.g., "message", "alert")
            Comments: []string{"Notification event"}, // Optional comments
            Data:     generateNotificationData(),     // Notification payload
            Retry:    5 * time.Second,               // Client retry interval
        }
    },
    func() {
        log.Println("Client stopped listening to notifications")
    },
    func(err error) {
        log.Printf("Notification stream error: %v\n", err)
    },
    1*time.Second,
    nil,
))
```

### Client-Side Usage

**Vanilla JavaScript:**

```javascript
const eventSource = new EventSource('http://localhost:8080/events');

// Listen for messages
eventSource.onmessage = function(event) {
    console.log('Received:', event.data);
};

// Listen for specific event types
eventSource.addEventListener('TIME_UPDATE', function(event) {
    console.log('Time update:', event.data);
});

// Handle errors
eventSource.onerror = function(error) {
    console.error('EventSource error:', error);
};

// Close connection when done
// eventSource.close();
```

### SSE Configuration

#### Required Parameters

- **`payloadFunc`**: Function that returns an `SSEPayload`. Called at each interval.
- **`interval`**: Must be greater than zero. Determines how often messages are sent.

#### Optional Parameters

- **`disconnectFunc`**: Called when client disconnects. Defaults to no-op if `nil`.
- **`errorFunc`**: Called on stream errors. Defaults to printing errors if `nil`.
- **`headers`**: Map of custom HTTP headers to include in the response. Can be `nil`.

### Use Cases

**Real-time Dashboards:**

```go
mux.Handle("GET /dashboard", app.SSE(dashboardMetrics, nil, nil, 1*time.Second, nil))
```

**Live Notifications:**

```go
mux.Handle("GET /notifications", app.SSE(userNotifications, nil, nil, 2*time.Second, nil))
```

**Stock Price Updates:**

```go
mux.Handle("GET /stocks/{symbol}", app.SSE(stockPriceUpdates, nil, nil, 1*time.Second, nil))
```

**Log Streaming:**

```go
mux.Handle("GET /logs", app.SSE(tailLogs, nil, nil, 500*time.Millisecond, nil))
```

## Templates

WebFram provides a powerful templating system with automatic caching, layout inheritance, and partials.

### Template Configuration

Templates must be provided via an embedded file system. The template directory path is relative to the embedded filesystem root:

```go
// Your project structure:
// assets/
//   └── templates/
//       ├── layout.go.html
//       └── index.go.html

//go:embed assets
var assetsFS embed.FS

app.Configure(&app.Config{
    Assets: &app.Assets{
        FS: assetsFS,
        Templates: &app.Templates{
            Dir:                   "templates",  // Path relative to embedded FS (assets/templates/)
            LayoutBaseName:        "layout",
            HTMLTemplateExtension: ".go.html",
            TextTemplateExtension: ".go.txt",
        },
    },
})
```

### Template Structure

Your project structure should have an `assets` directory containing both `templates` and `locales`:

```
assets/
├── templates/
│   ├── layout.go.html              # Root layout (inherited by all)
│   ├── users/
│   │   ├── layout.go.html          # Users layout (inherits from root)
│   │   ├── list.go.html            # Inherits from users layout
│   │   ├── details.go.html
│   │   └── manage/
│   │       ├── update.go.html
│   │       └── delete.go.html
│   ├── _partOne.go.html            # Partial template
│   └── openapi.html
└── locales/
    ├── messages.en.json
    ├── messages.fr.json
    └── messages.es.json
```

The `assets` directory is embedded with `//go:embed assets`, and the `Templates.Dir` and `I18nMessages.Dir` paths are relative to this embedded filesystem.

### Layout Files

Layouts are automatically detected and applied:

- `layout.go.html` - Standard layout in each directory (inherits from parent)

**Root layout** (`templates/layout.go.html`):

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{block "title" .}}Default Title{{end}}</title>
</head>
<body>
    {{block "content" .}}{{end}}
</body>
</html>
```

**Page template** (`templates/users/list.go.html`):

```html
{{define "title"}}Users List{{end}}

{{define "content"}}
<h1>Users</h1>
<ul>
    {{range .Users}}
    <li>{{.Name}}</li>
    {{end}}
</ul>
{{end}}
```

### Partials

Partials are reusable template components with names starting with `_`:

**Partial** (`templates/_partOne.go.html`):

```html
<header>
    <h1>{{.Title}}</h1>
</header>
```

**Using partials in templates**:

```html
{{define "content"}}
    <!-- Include a partial -->
    {{template "_partOne.go.html" .}}
    
    <div>Your main content here</div>
{{end}}
```

### Rendering Templates

```go
mux.HandleFunc("GET /users", func(w app.ResponseWriter, r *app.Request) {
    data := map[string]interface{}{
        "Users": []User{
            {Name: "John", Email: "john@example.com"},
            {Name: "Jane", Email: "jane@example.com"},
        },
    }
    
    err := w.HTML("users/list", data)
    if err != nil {
        w.Error(http.StatusInternalServerError, err.Error())
    }
})
```

## Internationalization (i18n)

WebFram provides built-in i18n support using `golang.org/x/text`. The i18n function is always available in templates as `T`.

### Message Files

Create message files in JSON format in your locales directory:

**locales/messages.en.json**:

```json
{
  "language": "en",
  "messages": [
    {
      "id": "Welcome to %s! Clap %d times.",
      "message": "Welcome to %s! Clap %d times.",
      "translation": "Welcome to %s! Clap %d times.",
      "placeholders": {
        "arg_1": {
          "id": "arg_1",
          "string": "%s",
          "type": "string",
          "underlyingType": "string",
          "argNum": 1,
          "expr": "arg1"
        },
        "arg_2": {
          "id": "arg_2",
          "string": "%d",
          "type": "int",
          "underlyingType": "int",
          "argNum": 2,
          "expr": "arg2"
        }
      }
    }
  ]
}
```

### Configure i18n

```go
//go:embed assets
var assetsFS embed.FS

app.Configure(&app.Config{
    Assets: &app.Assets{
        FS: assetsFS,
        I18nMessages: &app.I18nMessages{
            Dir: "locales",
        },
    },
})
```

### Using i18n in Handlers

```go
import "golang.org/x/text/language"

mux.HandleFunc("GET /greeting", func(w app.ResponseWriter, r *app.Request) {
    printer := app.GetI18nPrinter(language.Spanish)
    msg := printer.Sprintf("Welcome to %s! Clap %d times.", "WebFram", 5)
    w.JSON(map[string]string{"message": msg})
})
```

### Using i18n in Templates

The i18n function is automatically available in templates as `T` (not configurable):

```html
{{define "content"}}
<h1>{{T "Welcome to %s! Clap %d times." "WebFram" 5}}</h1>
{{end}}
```

## Complete Example

```go
package main

import (
    "embed"
    "fmt"
    "log"
    "net/http"
    "time"
    
    app "github.com/bondowe/webfram"
    "github.com/bondowe/webfram/openapi"
    "github.com/google/uuid"
    "golang.org/x/text/language"
)

// Project structure:
// assets/
//   ├── templates/
//   │   └── index.go.html
//   └── locales/
//       ├── messages.en.json
//       └── messages.fr.json

//go:embed assets
var assetsFS embed.FS

type User struct {
    ID        uuid.UUID   `json:"id" xml:"id" form:"id"`
    Name      string      `json:"name" xml:"name" form:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"`
    Email     string      `json:"email" xml:"email" form:"email" validate:"required,format=email" errmsg:"required=Email is required;format=Invalid email"`
    Role      string      `json:"role" xml:"role" form:"role" validate:"enum=admin|user|guest" errmsg:"enum=Must be admin, user, or guest"`
    Birthdate time.Time   `form:"birthdate" validate:"required" format:"2006-01-02"`
}

func loggingMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        duration := time.Since(start)
        log.Printf("%s %s - %v", r.Method, r.URL.Path, duration)
    })
}

func main() {
    // Configure the application
    app.Configure(&app.Config{
        Assets: &app.Assets{
            FS: assetsFS,
            Templates: &app.Templates{
                Dir: "templates",
            },
            I18nMessages: &app.I18nMessages{
                Dir: "locales",
            },
        },
        JSONPCallbackParamName: "callback", // Enable JSONP support
        OpenAPI: &app.OpenAPI{
            EndpointEnabled: true,
            Config:          getOpenAPIConfig(),
        },
    })

    // Global middleware
    app.Use(loggingMiddleware)

    // Create mux
    mux := app.NewServeMux()

    // Routes
    mux.HandleFunc("GET /", func(w app.ResponseWriter, r *app.Request) {
        err := w.HTML("index", nil)
        if err != nil {
            w.Error(http.StatusInternalServerError, err.Error())
        }
    })

    // JSON endpoint with JSONP support
    mux.HandleFunc("GET /users", func(w app.ResponseWriter, r *app.Request) {
        users := []User{
            {ID: uuid.New(), Name: "John Doe", Email: "john@example.com"},
            {ID: uuid.New(), Name: "Jane Smith", Email: "jane@example.com"},
        }
        w.JSON(users)
    }).WithAPIConfig(&app.APIConfig{
        OperationID: "listUsers",
        Summary:     "List all users",
        Tags:        []string{"Users"},
        Responses: map[string]app.Response{
            "200": {
                Description: "List of users",
                Content: map[string]app.TypeInfo{
                    "application/json": {TypeHint: &[]User{}},
                },
            },
        },
    })

    // Create user with JSON
    mux.HandleFunc("POST /api/users/json", func(w app.ResponseWriter, r *app.Request) {
        user, valErrors, err := app.BindJSON[User](r, true)
        
        if err != nil {
            w.Error(http.StatusBadRequest, err.Error())
            return
        }
        
        if valErrors.Any() {
            w.WriteHeader(http.StatusBadRequest)
            w.JSON(valErrors)
            return
        }

        user.ID = uuid.New()
        w.WriteHeader(http.StatusCreated)
        w.JSON(user)
    }).WithAPIConfig(&app.APIConfig{
        OperationID: "createUser",
        Summary:     "Create a new user",
        Tags:        []string{"Users"},
        RequestBody: &app.RequestBody{
            Required: true,
            Content: map[string]app.TypeInfo{
                "application/json": {TypeHint: &User{}},
            },
        },
        Responses: map[string]app.Response{
            "201": {
                Description: "User created",
                Content: map[string]app.TypeInfo{
                    "application/json": {TypeHint: &User{}},
                },
            },
            "400": {Description: "Validation error"},
        },
    })

    // Update user with JSON Patch
    mux.HandleFunc("PATCH /users/{id}", func(w app.ResponseWriter, r *app.Request) {
        id := r.PathValue("id")
        
        // Fetch existing user
        user := User{
            ID:    uuid.MustParse(id),
            Name:  "John Doe",
            Email: "john@example.com",
            Role:  "user",
        }
        
        // Apply JSON Patch with validation
        valErrors, err := app.PatchJSON(r, &user, true)
        if err != nil {
            if err == app.ErrMethodNotAllowed {
                w.Error(http.StatusMethodNotAllowed, err.Error())
                return
            }
            w.Error(http.StatusBadRequest, err.Error())
            return
        }
        
        if len(valErrors) > 0 {
            w.WriteHeader(http.StatusBadRequest)
            w.JSON(app.ValidationErrors{Errors: valErrors})
            return
        }
        
        w.JSON(user)
    })

    // SSE endpoint for real-time updates
    mux.Handle("GET /events", app.SSE(
        func() app.SSEPayload {
            return app.SSEPayload{
                ID:       uuid.New().String(),
                Event:    "TIME_UPDATE",
                Comments: []string{"Server time update"},
                Data:     fmt.Sprintf("Current server time: %s", time.Now().Format(time.RFC3339)),
            }
        },
        func() {
            log.Println("Client disconnected from events stream")
        },
        func(err error) {
            log.Printf("SSE error: %v\n", err)
        },
        5*time.Second,
        nil,
    ))

    // i18n example
    mux.HandleFunc("GET /greeting", func(w app.ResponseWriter, r *app.Request) {
        printer := app.GetI18nPrinter(language.Spanish)
        msg := printer.Sprintf("Welcome to %s! Clap %d times.", "WebFram", 5)
        w.JSON(map[string]string{"message": msg})
    })

    // Start server
    log.Println("Server starting on :8080")
    log.Println("OpenAPI docs: http://localhost:8080/openapi.json")
    app.ListenAndServe(":8080", mux, nil)
}

func getOpenAPIConfig() *openapi.Config {
    return &openapi.Config{
        Info: &openapi.Info{
            Title:       "WebFram Example API",
            Summary:     "An example API demonstrating WebFram features.",
            Description: "This is an example API documentation generated by WebFram.",
            Version:     "1.0.0",
        },
        Servers: []openapi.Server{
            {
                URL:         "http://localhost:8080",
                Description: "Local development server",
            },
        },
    }
}
```

## Testing

WebFram is designed to be easily testable with comprehensive test utilities.

### Testing Handlers

```go
package main

import (
    "net/http"
    "net/http/httptest"
    "testing"
    
    app "github.com/bondowe/webfram"
)

func TestUserHandler(t *testing.T) {
    // Setup
    app.Configure(nil)
    mux := app.NewServeMux()
    mux.HandleFunc("GET /users/{id}", getUserHandler)
    
    // Create request
    req := httptest.NewRequest("GET", "/users/123", nil)
    rec := httptest.NewRecorder()
    
    // Execute
    mux.ServeHTTP(rec, req)
    
    // Assert
    if rec.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", rec.Code)
    }
}
```

### Testing Middleware

```go
func TestLoggingMiddleware(t *testing.T) {
    var logged bool
    
    middleware := func(next app.Handler) app.Handler {
        return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
            logged = true
            next.ServeHTTP(w, r)
        })
    }
    
    handler := app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        w.WriteHeader(http.StatusOK)
    })
    
    wrapped := middleware(handler)
    
    req := httptest.NewRequest("GET", "/", nil)
    rec := httptest.NewRecorder()
    
    w := app.ResponseWriter{ResponseWriter: rec}
    r := &app.Request{Request: req}
    
    wrapped.ServeHTTP(w, r)
    
    if !logged {
        t.Error("Expected middleware to log request")
    }
}
```

### Testing Data Binding

```go
func TestBindJSON(t *testing.T) {
    type User struct {
        Name  string `json:"name" validate:"required"`
        Email string `json:"email" validate:"required,format=email"`
    }
    
    app.Configure(nil)
    
    jsonData := `{"name":"John","email":"john@example.com"}`
    req := httptest.NewRequest("POST", "/", strings.NewReader(jsonData))
    req.Header.Set("Content-Type", "application/json")
    
    r := &app.Request{Request: req}
    
    user, valErrors, err := app.BindJSON[User](r, true)
    
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    
    if valErrors.Any() {
        t.Fatalf("Validation errors: %v", valErrors)
    }
    
    if user.Name != "John" {
        t.Errorf("Expected name 'John', got '%s'", user.Name)
    }
}
```

### Testing SSE

```go
func TestSSE(t *testing.T) {
    payloadFunc := func() app.SSEPayload {
        return app.SSEPayload{Data: "test message"}
    }
    
    handler := app.SSE(payloadFunc, nil, nil, 1*time.Second, nil)
    
    req := httptest.NewRequest("GET", "/events", nil)
    rec := httptest.NewRecorder()
    
    w := app.ResponseWriter{ResponseWriter: rec}
    r := &app.Request{Request: req}
    
    // Test in goroutine with timeout
    done := make(chan bool)
    go func() {
        handler.ServeHTTP(w, r)
        done <- true
    }()
    
    select {
    case <-done:
        // Handler completed
    case <-time.After(2 * time.Second):
        // Test timeout
    }
    
    if rec.Header().Get("Content-Type") != "text/event-stream" {
        t.Error("Expected SSE content type")
    }
}
```

### Integration Testing

```go
func TestFullAPI(t *testing.T) {
    // Setup complete application
    app.Configure(&app.Config{
        Assets: &app.Assets{
            FS: testFS,
            Templates: &app.Templates{
                Dir: "templates",
            },
        },
    })
    
    mux := app.NewServeMux()
    mux.HandleFunc("GET /api/users", listUsers)
    mux.HandleFunc("POST /api/users", createUser)
    
    // Start test server
    server := httptest.NewServer(mux)
    defer server.Close()
    
    // Test API endpoints
    resp, err := http.Get(server.URL + "/api/users")
    if err != nil {
        t.Fatal(err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected 200, got %d", resp.StatusCode)
    }
}
```

## Production Deployment

### Build Optimization

Build with optimizations for production:

```bash
# Strip debug info and disable symbol table
go build -ldflags="-s -w" -o app

# Further compression with upx (optional)
upx --best --lzma app
```

### Docker Deployment

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
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/locales ./locales

EXPOSE 8080
CMD ["./webfram-app"]
```

### Environment Configuration

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
    TrustedProxies []string
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

### Graceful Shutdown

**Automatic Graceful Shutdown:**

The `ListenAndServe` function automatically handles graceful shutdown on SIGINT/SIGTERM:

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
    
    // ListenAndServe handles graceful shutdown automatically
    app.ListenAndServe(":"+cfg.Port, mux, serverCfg)
}
```

**Manual Graceful Shutdown (if needed):**

If you need custom shutdown logic, you can use the standard library directly:

```go
func main() {
    cfg := loadConfig()
    
    app.Configure(getWebFramConfig(cfg))
    mux := app.NewServeMux()
    registerRoutes(mux)
    
    server := &http.Server{
        Addr:         ":" + cfg.Port,
        Handler:      mux,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  120 * time.Second,
    }
    
    go func() {
        log.Printf("Server starting on port %s", cfg.Port)
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
    
    log.Println("Server exited")
}
```

### Health Checks

```go
mux.HandleFunc("GET /health", func(w app.ResponseWriter, r *app.Request) {
    w.JSON(map[string]string{
        "status": "healthy",
        "version": version,
    })
})

mux.HandleFunc("GET /readiness", func(w app.ResponseWriter, r *app.Request) {
    // Check database, cache, etc.
    if !isReady() {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.JSON(map[string]string{
            "status": "not ready",
        })
        return
    }
    
    w.JSON(map[string]string{
        "status": "ready",
    })
})
```

### Monitoring & Metrics

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "HTTP request duration in seconds",
        },
        []string{"method", "path", "status"},
    )
)

func init() {
    prometheus.MustRegister(requestDuration)
}

func metricsMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        start := time.Now()
        
        // Wrap response writer to capture status
        wrapped := &statusRecorder{ResponseWriter: w, status: 200}
        
        next.ServeHTTP(*wrapped, r)
        
        duration := time.Since(start).Seconds()
        requestDuration.WithLabelValues(
            r.Method,
            r.URL.Path,
            strconv.Itoa(wrapped.status),
        ).Observe(duration)
    })
}

// Expose metrics endpoint
mux.Handle("GET /metrics", promhttp.Handler())
```

### Performance Tuning

```go
func main() {
    // Adjust GOMAXPROCS for your environment
    runtime.GOMAXPROCS(runtime.NumCPU())
    
    // Configure server with appropriate timeouts
    server := &http.Server{
        Addr:              ":8080",
        Handler:           mux,
        ReadTimeout:       10 * time.Second,
        ReadHeaderTimeout: 5 * time.Second,
        WriteTimeout:      10 * time.Second,
        IdleTimeout:       120 * time.Second,
        MaxHeaderBytes:    1 << 20, // 1 MB
    }
    
    // Set connection limits if needed
    server.SetKeepAlivesEnabled(true)
    
    log.Fatal(server.ListenAndServe())
}
```

### Security Hardening

```go
func securityMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        // Security headers
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        next.ServeHTTP(w, r)
    })
}
```

### Logging

```go
import "log/slog"

func loggingMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        start := time.Now()
        
        // Log request
        slog.Info("incoming request",
            "method", r.Method,
            "path", r.URL.Path,
            "remote", r.RemoteAddr,
        )
        
        next.ServeHTTP(w, r)
        
        // Log response
        slog.Info("request completed",
            "method", r.Method,
            "path", r.URL.Path,
            "duration", time.Since(start),
        )
    })
}
```

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`go test ./...`)
5. Run linters (`golangci-lint run`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Development Setup

```bash
# Clone repository
git clone https://github.com/bondowe/webfram.git
cd webfram

# Install dependencies
go mod download

# Run tests
go test ./... -v -race -coverprofile=coverage.out

# Run linters
golangci-lint run

# View coverage
go tool cover -html=coverage.out
```

## License

This project is licensed under the MIT License.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

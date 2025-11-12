# Configuration

Learn how to configure WebFram for your application needs.

## Basic Configuration

WebFram can be configured with templates, i18n, JSONP, and OpenAPI settings:

```go
//go:embed all:assets
var assetsFS embed.FS

func main() {
    app.Configure(&app.Config{
        Assets: &app.Assets{
            FS: assetsFS,
            Templates: &app.Templates{
                Dir:                   "assets/templates",
                LayoutBaseName:        "layout",
                HTMLTemplateExtension: ".go.html",
                TextTemplateExtension: ".go.txt",
            },
            I18nMessages: &app.I18nMessages{
                Dir: "assets/locales",
            },
        },
        JSONPCallbackParamName: "callback", // Enable JSONP
        OpenAPI: &app.OpenAPI{
            EndpointEnabled: true,
            URLPath:         "GET /openapi.json",
            Config:          getOpenAPIConfig(),
        },
    })

    mux := app.NewServeMux()
    // ... register routes
    
    app.ListenAndServe(":8080", mux, nil)
}
```

## Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `Assets.FS` | `nil` (required) | Embedded file system (e.g., `//go:embed assets`) |
| `Assets.Templates.Dir` | `"assets/templates"` | Path to templates directory within the embedded FS |
| `Assets.Templates.LayoutBaseName` | `"layout"` | Base name for layout files |
| `Assets.Templates.HTMLTemplateExtension` | `".go.html"` | Extension for HTML templates |
| `Assets.Templates.TextTemplateExtension` | `".go.txt"` | Extension for text templates |
| `Assets.I18nMessages.Dir` | `"assets/locales"` | Path to locales directory within the embedded FS |
| `JSONPCallbackParamName` | `""` (disabled) | Query parameter name for JSONP callbacks |
| `OpenAPI.EndpointEnabled` | `false` | Enable/disable OpenAPI endpoint |
| `OpenAPI.URLPath` | `"GET /openapi.json"` | Path for OpenAPI spec endpoint |
| `OpenAPI.Config` | `nil` | OpenAPI configuration |

## Server Configuration

### ListenAndServe

Start an HTTP server with automatic graceful shutdown:

```go
func ListenAndServe(addr string, mux *ServeMux, cfg *ServerConfig)
```

**Basic Usage:**

```go
mux := app.NewServeMux()
mux.HandleFunc("GET /hello", handleHello)

// Use default configuration
app.ListenAndServe(":8080", mux, nil)
```

**With Custom Configuration:**

```go
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

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `ReadTimeout` | `time.Duration` | `15s` | Maximum duration for reading entire request |
| `ReadHeaderTimeout` | `time.Duration` | `15s` | Maximum duration for reading request headers |
| `WriteTimeout` | `time.Duration` | `15s` | Maximum duration for writing response |
| `IdleTimeout` | `time.Duration` | `60s` | Maximum idle time for keep-alive connections |
| `MaxHeaderBytes` | `int` | `1048576` (1MB) | Maximum size of request headers |
| `TLSConfig` | `*tls.Config` | `nil` | TLS configuration for HTTPS |
| `ErrorLog` | `*slog.Logger` | `nil` | Custom error logger |

## Configuration Best Practices

### 1. Use Embedded Filesystems

Always use `//go:embed` for your assets directory:

```go
// Project structure:
// assets/
//   ├── templates/
//   │   ├── _partial.go.html
//   │   └── index.go.html
//   └── locales/
//       └── messages.en.json

//go:embed all:assets
var assetsFS embed.FS

app.Configure(&app.Config{
    Assets: &app.Assets{
        FS: assetsFS,
        Templates: &app.Templates{Dir: "assets/templates"},
        I18nMessages: &app.I18nMessages{Dir: "assets/locales"},
    },
})
```

**Important:** Use the `all:` prefix to include files starting with `_` (partials).

### 2. Environment-Specific Configuration

Use environment variables for deployment-specific settings:

```go
func getConfig() *app.Config {
    cfg := &app.Config{
        Assets: &app.Assets{
            FS: assetsFS,
            Templates: &app.Templates{Dir: "assets/templates"},
            I18nMessages: &app.I18nMessages{Dir: "assets/locales"},
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

### 3. Validate Configuration

Check configuration errors early:

```go
func main() {
    defer func() {
        if r := recover(); r != nil {
            log.Fatalf("Configuration error: %v", r)
        }
    }()
    
    app.Configure(getConfig())
    // ... rest of app
}
```

### 4. Single Configuration Call

Only call `Configure()` once at startup:

```go
func main() {
    // Configure once before creating any mux
    app.Configure(getConfig())
    
    // Create mux after configuration
    mux := app.NewServeMux()
    // ... register routes
}
```

## Production Server Configuration

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

## HTTPS Configuration

```go
func main() {
    cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
    if err != nil {
        log.Fatal(err)
    }

    tlsConfig := &tls.Config{
        Certificates: []tls.Certificate{cert},
        MinVersion:   tls.VersionTLS13,
    }

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

## Development vs Production

```go
func getServerConfig() *app.ServerConfig {
    if os.Getenv("ENV") == "production" {
        return &app.ServerConfig{
            ReadTimeout:       30 * time.Second,
            WriteTimeout:      30 * time.Second,
            IdleTimeout:       120 * time.Second,
            ReadHeaderTimeout: 10 * time.Second,
            MaxHeaderBytes:    2 << 20,
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
```

## See Also

- [Getting Started](getting-started)
- [Routing](routing)
- [Deployment](deployment)

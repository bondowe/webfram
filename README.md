# WebFram

<div align="center">
  <img src="webfram-logo.svg" alt="WebFram Logo" width="200"/>
</div>

<div align="center">

[![CI](https://github.com/bondowe/webfram/actions/workflows/ci.yml/badge.svg)](https://github.com/bondowe/webfram/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/bondowe/webfram/branch/main/graph/badge.svg)](https://codecov.io/gh/bondowe/webfram)
[![Go Report Card](https://goreportcard.com/badge/github.com/bondowe/webfram)](https://goreportcard.com/report/github.com/bondowe/webfram)
[![Go Reference](https://pkg.go.dev/badge/github.com/bondowe/webfram.svg)](https://pkg.go.dev/github.com/bondowe/webfram)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

</div>

**WebFram** is a production-ready, lightweight, feature-rich Go web framework built on top of the standard library's `net/http` package. It provides enterprise-grade features like automatic template caching with layouts, comprehensive data binding with validation, internationalization (i18n), Server-Sent Events (SSE), JSON Patch support, JSONP, OpenAPI 3.2.0 documentation generation, and flexible middleware support—all while maintaining minimal dependencies and maximum performance.

---

## 📚 Full Documentation

**[View Complete Documentation →](https://bondowe.github.io/webfram/)**

For comprehensive guides, API reference, and detailed examples, visit our documentation:

- 📖 [Getting Started](https://bondowe.github.io/webfram/getting-started.html) - Installation and quick start
- ⚙️ [Configuration](https://bondowe.github.io/webfram/configuration.html) - App and server setup
- 🔗 [Routing](https://bondowe.github.io/webfram/routing.html) - URL patterns and parameters
- 🔧 [Middleware](https://bondowe.github.io/webfram/middleware.html) - Request/response interceptors
- 📨 [Request & Response](https://bondowe.github.io/webfram/request-response.html) - HTTP handling
- 📋 [Data Binding](https://bondowe.github.io/webfram/data-binding.html) - Form, JSON, XML binding with validation
- 🔄 [JSON Patch](https://bondowe.github.io/webfram/json-patch.html) - RFC 6902 partial updates
- 🌐 [JSONP](https://bondowe.github.io/webfram/jsonp.html) - Cross-origin requests
- 📚 [OpenAPI](https://bondowe.github.io/webfram/openapi.html) - Auto-generated API docs
- 📡 [Server-Sent Events](https://bondowe.github.io/webfram/sse.html) - Real-time streaming
- 🎨 [Templates](https://bondowe.github.io/webfram/templates.html) - Template system with layouts
- 🌍 [Internationalization](https://bondowe.github.io/webfram/i18n.html) - Multi-language support
- 🧪 [Testing](https://bondowe.github.io/webfram/testing.html) - Testing strategies
- ⚡ [Benchmarks](https://bondowe.github.io/webfram/benchmarks.html) - Performance benchmarks
- 🚀 [Deployment](https://bondowe.github.io/webfram/deployment.html) - Production deployment guide

---

## ✨ Features

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

---

## 🚀 Quick Start

### Installation

```bash
go get github.com/bondowe/webfram
```

### Basic Example

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
        w.JSON(r.Context(), map[string]string{"message": "Hello, World!"})
    })

    // Start the server (nil for default server configuration)
    app.ListenAndServe(":8080", mux, nil)
}
```

### With Data Binding

```go
type User struct {
    Name  string `form:"name" validate:"required,min=2,max=50"`
    Email string `form:"email" validate:"required,email"`
    Age   int    `form:"age" validate:"required,min=18,max=120"`
}

mux.HandleFunc("POST /users", func(w app.ResponseWriter, r *app.Request) {
    user, valErrors, err := app.BindForm[User](r)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        w.JSON(r.Context(), map[string]string{"error": err.Error()})
        return
    }
    
    if len(valErrors) > 0 {
        w.WriteHeader(http.StatusBadRequest)
        w.JSON(r.Context(), valErrors)
        return
    }
    
    // Process user...
    w.JSON(r.Context(), user)
})
```

### With Templates

```go
//go:embed all:assets
var assetsFS embed.FS

func main() {
    app.Configure(&app.Config{
        Assets: &app.Assets{
            FS: assetsFS,
            Templates: &app.Templates{
                Dir: "assets/templates",
            },
        },
    })

    mux := app.NewServeMux()
    
    mux.HandleFunc("GET /", func(w app.ResponseWriter, r *app.Request) {
        data := map[string]interface{}{
            "Title": "Welcome",
            "Message": "Hello from WebFram!",
        }
        w.Render(r.Context(), "index", data)
    })

    app.ListenAndServe(":8080", mux, nil)
}
```

---

## 📚 Learn More

For complete documentation including:

- Comprehensive guides and tutorials
- API reference and examples
- Best practices and patterns
- Production deployment strategies
- Testing approaches
- And much more...

**[Visit the Documentation →](https://bondowe.github.io/webfram/)**

---

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🔗 Resources

- [Documentation](https://bondowe.github.io/webfram/)
- [API Reference](https://pkg.go.dev/github.com/bondowe/webfram)
- [GitHub Repository](https://github.com/bondowe/webfram)
- [Issue Tracker](https://github.com/bondowe/webfram/issues)

---

**Built with ❤️ using Go's standard library**

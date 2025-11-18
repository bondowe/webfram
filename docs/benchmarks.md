---
layout: default
title: Benchmarks
nav_order: 16
description: "Performance benchmarks and comparisons"
---

# Benchmarks
{: .no_toc }

Performance benchmarks comparing WebFram with Go's standard library and insights into framework overhead.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Overview

WebFram is built on top of Go's `net/http` standard library, providing additional features while maintaining excellent performance. These benchmarks demonstrate the performance characteristics and overhead of WebFram compared to the standard library.

## Benchmark Environment

All benchmarks were run with:
- **Go Version**: 1.22+
- **Iterations**: 1,000,000 operations
- **Hardware**: Test results may vary based on hardware

## Simple Route Handling

Basic GET request without path parameters:

| Framework | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| **Go Standard Library** | 135.5 | 12 | 1 |
| **WebFram** | 2,911 | 1,593 | 20 |

**Analysis**: WebFram adds ~2.8 microseconds overhead per request for enhanced response writing capabilities, middleware support, and error handling.

## Path Parameter Extraction

GET request with path parameter (`/user/{id}`):

| Framework | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| **Go Standard Library** | 345.7 | 32 | 2 |
| **WebFram** | 3,189 | 1,609 | 21 |

**Analysis**: WebFram provides the same path parameter extraction as the standard library with minimal additional overhead while offering enhanced response methods.

## JSON Response Generation

Generating and sending JSON responses:

| Framework | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| **WebFram** | 3,914 | 2,230 | 28 |

**Analysis**: WebFram's `w.JSON()` method provides convenient JSON serialization with proper content-type headers and error handling.

## Multiple Routes

Performance with 5 registered routes:

| Framework | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| **WebFram** | 3,273 | 1,609 | 21 |

**Analysis**: Route matching performance remains consistent regardless of the number of routes, thanks to Go 1.22+'s efficient routing implementation.

## Middleware Overhead

Global middleware performance:

| Framework | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| **WebFram (with middleware)** | 4,462 | 2,426 | 26 |
| **WebFram (without middleware)** | 2,911 | 1,593 | 20 |

**Analysis**: Each middleware layer adds ~1.5 microseconds. Middleware is applied efficiently using Go's handler chaining pattern.

## Data Binding Performance

### JSON Binding with Validation

| Operation | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| **JSON Bind + Validation** | 114,154 | 7,185 | 36 |

**Analysis**: Comprehensive JSON binding with struct validation adds overhead but provides type safety, automatic validation, and clear error messages - essential for production APIs.

### JSON Patch (RFC 6902)

| Operation | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| **JSON Patch Application** | 17,259 | 8,663 | 84 |

**Analysis**: JSON Patch operations are optimized for partial updates, significantly reducing bandwidth compared to full resource replacements.

## Feature-Specific Benchmarks

### Server-Sent Events (SSE)

| Operation | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| **SSE Payload Generation** | 5.486 | 0 | 0 |

**Analysis**: Zero-allocation SSE payload formatting enables high-throughput real-time event streaming.

### HTTP Middleware Adaptation

| Operation | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| **Adapt Standard Middleware** | 49.23 | 16 | 1 |

**Analysis**: Converting standard `http.Handler` middleware to WebFram middleware has minimal overhead.

### ServeMux Creation

| Operation | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| **New ServeMux** | 207.6 | 224 | 1 |

**Analysis**: Mux initialization is a one-time operation with negligible impact on overall performance.

## Performance Recommendations

### 1. Choose the Right Tool

- **Use Standard Library** when you need absolute minimal overhead and only basic routing
- **Use WebFram** when you need enhanced response methods, middleware, data binding, templates, i18n, SSE, or JSON Patch

### 2. Middleware Optimization

```go
// Good: Apply middleware selectively
adminMux := app.NewServeMux()
adminMux.Use(authMiddleware)
adminMux.Use(loggingMiddleware)

publicMux := app.NewServeMux()
publicMux.Use(loggingMiddleware) // Only logging, no auth
```

### 3. Data Binding Best Practices

```go
// Good: Use binding for complex validation
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,format=email"`
}

// For simple cases, manual parsing may be faster
simpleID := r.PathValue("id")
```

### 4. JSON Response Caching

```go
// Cache frequently-used responses
var cachedResponse []byte

func handler(w app.ResponseWriter, r *app.Request) {
    if cachedResponse == nil {
        data := getExpensiveData()
        cachedResponse, _ = json.Marshal(data)
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(cachedResponse)
}
```

## Real-World Performance

In production scenarios, the overhead of WebFram (microseconds per request) is negligible compared to:

- **Database queries**: Milliseconds to seconds
- **External API calls**: Tens to hundreds of milliseconds
- **Business logic**: Variable, often milliseconds
- **Network latency**: Milliseconds to seconds

The productivity gains from WebFram's features (data binding, validation, templates, i18n, SSE, etc.) far outweigh the minimal performance overhead for most applications.

## Running Benchmarks

Run the included benchmarks yourself:

```bash
# Run all benchmarks
go test -bench=. -benchmem -run=^$

# Run specific benchmark
go test -bench=BenchmarkWebFram_Simple -benchmem -run=^$

# Run with more iterations for accuracy
go test -bench=. -benchmem -benchtime=5s -run=^$
```

## Comparison with Other Frameworks

While we've compared primarily with Go's standard library, here's how WebFram positions itself:

| Framework | Philosophy | Use Case |
|-----------|-----------|----------|
| **net/http (stdlib)** | Minimal, explicit | Maximum control, minimal abstraction |
| **WebFram** | Enhanced stdlib | Productive development with stdlib foundation |
| **Gin** | High performance, full-featured | Large-scale applications, extensive middleware |
| **Echo** | Optimized routing, extensible | RESTful APIs, microservices |
| **Fiber** | Express-inspired, fast | Node.js developers, rapid development |

**WebFram's Advantage**: Direct compatibility with `net/http`, zero reflection in routing, optional features that don't impact unused paths, and familiar Go patterns.

## Performance Metrics Explained

### ns/op (Nanoseconds per Operation)
Lower is better. Measures time taken for each operation.

### B/op (Bytes per Operation)
Lower is better. Measures memory allocated per operation.

### allocs/op (Allocations per Operation)
Lower is better. Measures number of heap allocations per operation.

## Continuous Improvement

We continuously monitor and optimize WebFram's performance. If you notice performance issues:

1. Run benchmarks in your specific environment
2. Profile your application: `go test -cpuprofile=cpu.prof -memprofile=mem.prof`
3. [Open an issue](https://github.com/bondowe/webfram/issues) with benchmark results
4. Consider contributing optimizations via pull request

## See Also

- [Getting Started](getting-started.md) - Begin building with WebFram
- [Configuration](configuration.md) - Optimize your application settings
- [Deployment](deployment.md) - Production deployment best practices
- [Testing](testing.md) - Performance testing strategies

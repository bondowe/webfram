---
layout: default
title: Home
nav_order: 1
description: "WebFram is a production-ready, lightweight Go web framework built on net/http"
permalink: /
---

# WebFram Documentation

{: .fs-9 }

Welcome to the WebFram documentation! This comprehensive guide will help you build production-ready web applications with Go.
{: .fs-6 .fw-300 }

## Table of Contents

### Getting Started

- **[Getting Started](getting-started.html)** - Installation, quick start, and your first WebFram application

### Core Concepts

- **[Configuration](configuration.html)** - Application configuration, server setup, and best practices
- **[Routing](routing.html)** - URL routing, path parameters, and route patterns
- **[Middleware](middleware.html)** - Global, mux-level, and route-specific middleware
- **[Request & Response](request-response.html)** - Handling requests and generating responses

### Data Handling

- **[Data Binding & Validation](data-binding.html)** - Form, JSON, and XML binding with comprehensive validation
- **[JSON Patch](json-patch.html)** - RFC 6902 JSON Patch support for partial updates
- **[JSONP](jsonp.html)** - Cross-origin requests with JSONP

### Advanced Features

- **[OpenAPI Documentation](openapi.html)** - Automatic API documentation generation
- **[Server-Sent Events (SSE)](sse.html)** - Real-time server-to-client streaming
- **[Templates](templates.html)** - Server-side rendering with layouts and partials
- **[Internationalization (i18n)](i18n.html)** - Multi-language support

### Production

- **[Testing](testing.html)** - Unit testing, integration testing, and best practices
- **[Deployment](deployment.html)** - Production deployment, Docker, monitoring, and security

## Quick Links

- [GitHub Repository](https://github.com/bondowe/webfram)
- [Go Package Documentation](https://pkg.go.dev/github.com/bondowe/webfram)
- [Examples](https://github.com/bondowe/webfram/tree/main/cmd/sample-app)

## Why WebFram?

WebFram bridges the gap between using the raw `net/http` package and heavyweight frameworks. It provides:

- 🚀 **Lightweight & Fast** - Built directly on `net/http`
- 📝 **Smart Templates** - Automatic caching with layout inheritance
- ✅ **Data Binding** - Type-safe Form, JSON, and XML binding
- 🔄 **JSON Patch** - Full RFC 6902 support
- 📡 **Server-Sent Events** - Production-ready SSE
- 📚 **OpenAPI 3.2.0** - Automatic documentation
- 🌍 **i18n Support** - First-class internationalization
- 🔧 **Flexible Middleware** - Custom and standard HTTP middleware

## Getting Help

If you encounter issues or have questions:

1. Check the [documentation](index.html)
2. Search [existing issues](https://github.com/bondowe/webfram/issues)
3. Open a [new issue](https://github.com/bondowe/webfram/issues/new)
4. Read the [contributing guide](../CONTRIBUTING.md)

## License

WebFram is licensed under the [MIT License](https://opensource.org/licenses/MIT).

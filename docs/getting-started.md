---
layout: default
title: Getting Started
nav_order: 2
description: "Installation and quick start guide for WebFram"
---

# Getting Started
{: .no_toc }

Get up and running with WebFram in minutes.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Installation

```bash
go get github.com/bondowe/webfram
```

## Requirements

- Go 1.22 or later (for routing pattern matching)
- Basic understanding of Go and `net/http`

## Quick Start

Create your first WebFram application:

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

Run your application:

```bash
go run main.go
```

Visit `http://localhost:8080/hello` to see your API in action!

## Basic Example with Multiple Routes

```go
package main

import (
    "net/http"
    app "github.com/bondowe/webfram"
)

func main() {
    mux := app.NewServeMux()

    // Home page
    mux.HandleFunc("GET /", func(w app.ResponseWriter, r *app.Request) {
        w.JSON(r.Context(), map[string]string{
            "message": "Welcome to WebFram!",
            "version": "1.0.0",
        })
    })

    // Users endpoint
    mux.HandleFunc("GET /users", func(w app.ResponseWriter, r *app.Request) {
        users := []map[string]string{
            {"id": "1", "name": "John Doe"},
            {"id": "2", "name": "Jane Smith"},
        }
        w.JSON(r.Context(), users)
    })

    // User detail with path parameter
    mux.HandleFunc("GET /users/{id}", func(w app.ResponseWriter, r *app.Request) {
        userID := r.PathValue("id")
        w.JSON(r.Context(), map[string]string{
            "id":   userID,
            "name": "User " + userID,
        })
    })

    // Health check
    mux.HandleFunc("GET /health", func(w app.ResponseWriter, r *app.Request) {
        w.JSON(r.Context(), map[string]string{"status": "healthy"})
    })

    app.ListenAndServe(":8080", mux, nil)
}
```

## Next Steps

Now that you have a basic application running, explore these topics:

1. **[Configuration](configuration.html)** - Learn about app configuration and server setup
2. **[Routing](routing.html)** - Understand routing patterns and parameters
3. **[Middleware](middleware.html)** - Add logging, auth, and custom middleware
4. **[Data Binding](data-binding.html)** - Handle form and JSON data with validation
5. **[Templates](templates.html)** - Build server-side rendered pages

## Complete Example Application

Check out the [sample application](https://github.com/bondowe/webfram/tree/main/cmd/sample-app) for a complete example with templates, i18n, and OpenAPI documentation.

## Common Patterns

### JSON API

```go
mux.HandleFunc("GET /api/data", func(w app.ResponseWriter, r *app.Request) {
    data := map[string]interface{}{
        "items": []string{"one", "two", "three"},
        "count": 3,
    }
    w.JSON(r.Context(), data)
})
```

### Error Handling

```go
mux.HandleFunc("GET /api/user/{id}", func(w app.ResponseWriter, r *app.Request) {
    id := r.PathValue("id")
    
    user, err := getUser(id)
    if err != nil {
        w.Error(http.StatusNotFound, "User not found")
        return
    }
    
    w.JSON(r.Context(), user)
})
```

### POST Request

```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,format=email"`
}

mux.HandleFunc("POST /api/users", func(w app.ResponseWriter, r *app.Request) {
    user, valErrors, err := app.BindJSON[CreateUserRequest](r, true)
    
    if err != nil {
        w.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    if valErrors.Any() {
        w.WriteHeader(http.StatusBadRequest)
        w.JSON(r.Context(), valErrors)
        return
    }
    
    // Save user...
    w.WriteHeader(http.StatusCreated)
    w.JSON(r.Context(), user)
})
```

## Project Structure

Recommended project structure for WebFram applications:

```text
myapp/
├── main.go
├── go.mod
├── go.sum
├── assets/
│   ├── templates/
│   │   ├── layout.go.html
│   │   └── index.go.html
│   └── locales/
│       ├── messages.en.json
│       └── messages.fr.json
├── handlers/
│   ├── users.go
│   └── health.go
└── models/
    └── user.go
```

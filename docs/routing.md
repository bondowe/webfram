---
layout: default
title: Routing
nav_order: 4
description: "URL routing patterns and parameters"
---

# Routing
{: .no_toc }

WebFram uses Go 1.22+ routing patterns with HTTP method prefixes for powerful and flexible route matching.
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Basic Routes

Define routes with HTTP method prefixes:

```go
mux := app.NewServeMux()

// Simple routes with different HTTP methods
mux.HandleFunc("GET /users", listUsers)
mux.HandleFunc("POST /users", createUser)
mux.HandleFunc("PUT /users/{id}", updateUser)
mux.HandleFunc("PATCH /users/{id}", patchUser)
mux.HandleFunc("DELETE /users/{id}", deleteUser)
```

## Route Parameters

Access path parameters using `r.PathValue()`:

```go
mux.HandleFunc("GET /users/{id}", func(w app.ResponseWriter, r *app.Request) {
    userID := r.PathValue("id")
    
    user, err := getUserByID(userID)
    if err != nil {
        w.Error(http.StatusNotFound, "User not found")
        return
    }
    
    w.JSON(r.Context(), user)
})

// Multiple parameters
mux.HandleFunc("GET /posts/{year}/{month}/{slug}", func(w app.ResponseWriter, r *app.Request) {
    year := r.PathValue("year")
    month := r.PathValue("month")
    slug := r.PathValue("slug")
    
    post := getPost(year, month, slug)
    w.JSON(r.Context(), post)
})
```

## Wildcard Routes

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

## Route Patterns

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

## RESTful Routes

Example of a complete RESTful resource:

```go
// List all users
mux.HandleFunc("GET /users", func(w app.ResponseWriter, r *app.Request) {
    users := getAllUsers()
    w.JSON(r.Context(), users)
})

// Get single user
mux.HandleFunc("GET /users/{id}", func(w app.ResponseWriter, r *app.Request) {
    id := r.PathValue("id")
    user := getUserByID(id)
    w.JSON(r.Context(), user)
})

// Create new user
mux.HandleFunc("POST /users", func(w app.ResponseWriter, r *app.Request) {
    user, valErrors, err := app.BindJSON[User](r, true)
    // ... handle creation
    w.WriteHeader(http.StatusCreated)
    w.JSON(r.Context(), user)
})

// Update user (full)
mux.HandleFunc("PUT /users/{id}", func(w app.ResponseWriter, r *app.Request) {
    id := r.PathValue("id")
    user, valErrors, err := app.BindJSON[User](r, true)
    // ... handle update
    w.JSON(r.Context(), user)
})

// Update user (partial)
mux.HandleFunc("PATCH /users/{id}", func(w app.ResponseWriter, r *app.Request) {
    id := r.PathValue("id")
    user := getUserByID(id)
    valErrors, err := app.PatchJSON(r, &user, true)
    // ... handle patch
    w.JSON(r.Context(), user)
})

// Delete user
mux.HandleFunc("DELETE /users/{id}", func(w app.ResponseWriter, r *app.Request) {
    id := r.PathValue("id")
    deleteUser(id)
    w.NoContent()
})
```

## Query Parameters

Access query parameters using standard `url.Values`:

```go
mux.HandleFunc("GET /search", func(w app.ResponseWriter, r *app.Request) {
    query := r.URL.Query().Get("q")
    page := r.URL.Query().Get("page")
    limit := r.URL.Query().Get("limit")
    
    results := search(query, page, limit)
    w.JSON(r.Context(), results)
})
```

## Organizing Routes

For larger applications, organize routes in separate functions:

```go
func main() {
    mux := app.NewServeMux()
    
    registerUserRoutes(mux)
    registerProductRoutes(mux)
    registerAuthRoutes(mux)
    
    app.ListenAndServe(":8080", mux, nil)
}

func registerUserRoutes(mux *app.ServeMux) {
    mux.HandleFunc("GET /users", listUsers)
    mux.HandleFunc("POST /users", createUser)
    mux.HandleFunc("GET /users/{id}", getUser)
    mux.HandleFunc("PUT /users/{id}", updateUser)
    mux.HandleFunc("DELETE /users/{id}", deleteUser)
}
```

## Route Groups with Middleware

Apply middleware to groups of routes:

```go
func registerAdminRoutes(mux *app.ServeMux) {
    // Admin routes with auth middleware
    mux.HandleFunc("GET /admin/users", listUsers, authMiddleware, adminMiddleware)
    mux.HandleFunc("DELETE /admin/users/{id}", deleteUser, authMiddleware, adminMiddleware)
}

func registerPublicRoutes(mux *app.ServeMux) {
    // Public routes without auth
    mux.HandleFunc("GET /", homePage)
    mux.HandleFunc("GET /about", aboutPage)
}
```

## See Also

- [Middleware](middleware.html)
- [Request & Response](request-response.html)
- [Data Binding](data-binding.html)

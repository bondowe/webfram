# Middleware

WebFram supports both custom and standard HTTP middleware with flexible composition.

## Middleware Types

### Global Middleware

Applied to all routes across all muxes:

```go
app.Use(func(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        // Before handler
        log.Printf("Request: %s %s", r.Method, r.URL.Path)
        
        next.ServeHTTP(w, r)
        
        // After handler
    })
})
```

### Mux-Level Middleware

Applied to all routes in a specific mux:

```go
mux := app.NewServeMux()

mux.Use(func(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        // Mux-specific middleware logic
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

## Middleware Execution Order

Middleware executes in this order:

1. Global middleware (registered with `app.Use()`)
2. Mux-level middleware (registered with `mux.Use()`)
3. Route-specific middleware
4. i18n middleware (automatic)
5. Handler

## Custom Middleware Examples

### Logging Middleware

```go
func loggingMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        start := time.Now()
        
        log.Printf("Started %s %s", r.Method, r.URL.Path)
        
        next.ServeHTTP(w, r)
        
        duration := time.Since(start)
        log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, duration)
    })
}

app.Use(loggingMiddleware)
```

### Authentication Middleware

```go
func authMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        token := r.Header.Get("Authorization")
        
        if token == "" {
            w.Error(http.StatusUnauthorized, "Missing authorization token")
            return
        }
        
        user, err := validateToken(token)
        if err != nil {
            w.Error(http.StatusUnauthorized, "Invalid token")
            return
        }
        
        // Add user to context
        ctx := context.WithValue(r.Context(), "user", user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Apply to specific routes
mux.HandleFunc("GET /profile", profileHandler, authMiddleware)
```

### Rate Limiting Middleware

```go
func rateLimitMiddleware(next app.Handler) app.Handler {
    limiter := rate.NewLimiter(10, 20) // 10 req/sec, burst of 20
    
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        if !limiter.Allow() {
            w.Error(http.StatusTooManyRequests, "Rate limit exceeded")
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

mux.Use(rateLimitMiddleware)
```

### CORS Middleware

```go
func corsMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusNoContent)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

app.Use(corsMiddleware)
```

### Recovery Middleware

```go
func recoveryMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("Panic recovered: %v\n%s", err, debug.Stack())
                w.Error(http.StatusInternalServerError, "Internal server error")
            }
        }()
        
        next.ServeHTTP(w, r)
    })
}

app.Use(recoveryMiddleware)
```

## Standard HTTP Middleware Support

WebFram seamlessly integrates with standard `http.Handler` middleware:

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
    
    // CSRF protection
    csrfMiddleware := csrf.Protect(
        []byte("32-byte-long-auth-key"),
        csrf.Secure(false),
    )
    app.Use(csrfMiddleware)
    
    app.ListenAndServe(":8080", mux, nil)
}
```

### Compatible Libraries

- **CORS**: `github.com/rs/cors`
- **CSRF**: `github.com/gorilla/csrf`
- **Compression**: `github.com/nytimes/gziphandler`
- **Rate Limiting**: `golang.org/x/time/rate`
- **Authentication**: `github.com/golang-jwt/jwt`
- **Session**: `github.com/gorilla/sessions`

## Middleware Composition

Chain multiple middleware together:

```go
func main() {
    app.Configure(nil)
    
    // Global middleware chain
    app.Use(recoveryMiddleware)
    app.Use(loggingMiddleware)
    app.Use(corsMiddleware)
    
    mux := app.NewServeMux()
    
    // Mux-level middleware
    mux.Use(rateLimitMiddleware)
    
    // Route with specific middleware
    mux.HandleFunc("GET /admin", adminHandler,
        authMiddleware,
        adminRoleMiddleware,
    )
    
    app.ListenAndServe(":8080", mux, nil)
}
```

## Context Values in Middleware

Pass values between middleware and handlers:

```go
func userMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        userID := r.Header.Get("X-User-ID")
        
        // Add to context
        ctx := context.WithValue(r.Context(), "userID", userID)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Access in handler
mux.HandleFunc("GET /profile", func(w app.ResponseWriter, r *app.Request) {
    userID := r.Context().Value("userID").(string)
    
    profile := getProfile(userID)
    w.JSON(r.Context(), profile)
})
```

## See Also

- [Routing](routing)
- [Request & Response](request-response)
- [Deployment](deployment)

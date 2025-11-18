---
layout: default
title: Testing
nav_order: 14
description: "Testing strategies and examples"
---

# Testing

WebFram is designed to be easily testable with comprehensive test utilities.

## Testing Handlers

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

## Testing Middleware

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

## Testing Data Binding

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

## Testing SSE

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

## Integration Testing

```go
func TestFullAPI(t *testing.T) {
    // Setup complete application
    app.Configure(&app.Config{
        Assets: &app.Assets{
            FS: testFS,
            Templates: &app.Templates{
                Dir: "assets/templates",
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

## Testing Templates

```go
func TestTemplateRendering(t *testing.T) {
    //go:embed testdata
    var testFS embed.FS
    
    app.Configure(&app.Config{
        Assets: &app.Assets{
            FS: testFS,
            Templates: &app.Templates{
                Dir: "testdata/templates",
            },
        },
    })
    
    mux := app.NewServeMux()
    mux.HandleFunc("GET /", func(w app.ResponseWriter, r *app.Request) {
        data := map[string]string{"Title": "Test"}
        err := w.HTML(r.Context(), "index", data)
        if err != nil {
            w.Error(http.StatusInternalServerError, err.Error())
        }
    })
    
    req := httptest.NewRequest("GET", "/", nil)
    rec := httptest.NewRecorder()
    
    mux.ServeHTTP(rec, req)
    
    if rec.Code != http.StatusOK {
        t.Errorf("Expected 200, got %d", rec.Code)
    }
    
    body := rec.Body.String()
    if !strings.Contains(body, "Test") {
        t.Error("Expected response to contain 'Test'")
    }
}
```

## Testing with Table-Driven Tests

{% raw %}

```go
func TestUserValidation(t *testing.T) {
    tests := []struct {
        name      string
        input     User
        wantError bool
    }{
        {
            name:      "valid user",
            input:     User{Name: "John", Email: "john@example.com", Age: 30},
            wantError: false,
        },
        {
            name:      "invalid email",
            input:     User{Name: "John", Email: "invalid", Age: 30},
            wantError: true,
        },
        {
            name:      "missing name",
            input:     User{Name: "", Email: "john@example.com", Age: 30},
            wantError: true,
        },
        {
            name:      "age too low",
            input:     User{Name: "John", Email: "john@example.com", Age: 10},
            wantError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            valErrors := validateUser(tt.input)
            hasError := len(valErrors) > 0
            
            if hasError != tt.wantError {
                t.Errorf("Expected error: %v, got: %v", tt.wantError, hasError)
            }
        })
    }
}
```

{% endraw %}

## Mocking Dependencies

{% raw %}

```go
type UserRepository interface {
    GetUser(id string) (*User, error)
    CreateUser(user User) error
}

type MockUserRepo struct {
    users map[string]*User
}

func (m *MockUserRepo) GetUser(id string) (*User, error) {
    if user, ok := m.users[id]; ok {
        return user, nil
    }
    return nil, errors.New("user not found")
}

func (m *MockUserRepo) CreateUser(user User) error {
    m.users[user.ID] = &user
    return nil
}

func TestGetUserHandler(t *testing.T) {
    mockRepo := &MockUserRepo{
        users: map[string]*User{
            "123": {ID: "123", Name: "John"},
        },
    }
    
    handler := NewUserHandler(mockRepo)
    
    req := httptest.NewRequest("GET", "/users/123", nil)
    rec := httptest.NewRecorder()
    
    handler.ServeHTTP(rec, req)
    
    if rec.Code != http.StatusOK {
        t.Errorf("Expected 200, got %d", rec.Code)
    }
}
```

{% endraw %}

## Testing Error Cases

```go
func TestErrorHandling(t *testing.T) {
    mux := app.NewServeMux()
    mux.HandleFunc("GET /error", func(w app.ResponseWriter, r *app.Request) {
        w.Error(http.StatusBadRequest, "Bad request")
    })
    
    req := httptest.NewRequest("GET", "/error", nil)
    rec := httptest.NewRecorder()
    
    mux.ServeHTTP(rec, req)
    
    if rec.Code != http.StatusBadRequest {
        t.Errorf("Expected 400, got %d", rec.Code)
    }
    
    if !strings.Contains(rec.Body.String(), "Bad request") {
        t.Error("Expected error message in response")
    }
}
```

## Coverage

Run tests with coverage:

```bash
go test ./... -v -race -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Benchmarks

{% raw %}

```go
func BenchmarkBindJSON(b *testing.B) {
    type User struct {
        Name  string `json:"name" validate:"required"`
        Email string `json:"email" validate:"required,format=email"`
    }
    
    app.Configure(nil)
    
    jsonData := `{"name":"John","email":"john@example.com"}`
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        req := httptest.NewRequest("POST", "/", strings.NewReader(jsonData))
        req.Header.Set("Content-Type", "application/json")
        r := &app.Request{Request: req}
        
        _, _, _ = app.BindJSON[User](r, true)
    }
}
```

{% endraw %}

## Best Practices

1. **Test all handlers** - Cover happy and error paths
2. **Use table-driven tests** - Test multiple scenarios
3. **Mock dependencies** - Isolate units under test
4. **Test middleware** - Verify execution order
5. **Integration tests** - Test complete request flows
6. **Test validation** - Cover all validation rules
7. **Measure coverage** - Aim for >80%
8. **Run with race detector** - `go test -race`
9. **Benchmark critical paths** - Monitor performance
10. **Test error handling** - Verify error responses

## See Also

- [Getting Started](getting-started.md)
- [Data Binding](data-binding.md)
- [Middleware](middleware.md)

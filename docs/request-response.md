---
layout: default
title: Request & Response
nav_order: 6
description: "HTTP request handling and response generation"
---

# Request & Response Handling

Learn how to handle HTTP requests and generate responses in WebFram.

## Request Handling

### Path Parameters

Access URL path parameters:

```go
mux.HandleFunc("GET /users/{id}", func(w app.ResponseWriter, r *app.Request) {
    userID := r.PathValue("id")
    // Use userID...
})
```

### Query Parameters

Access query string parameters:

```go
mux.HandleFunc("GET /search", func(w app.ResponseWriter, r *app.Request) {
    query := r.URL.Query().Get("q")
    page := r.URL.Query().Get("page")
    
    // Or get all values
    tags := r.URL.Query()["tags"]
})
```

### Request Headers

Access HTTP headers:

```go
mux.HandleFunc("GET /api/data", func(w app.ResponseWriter, r *app.Request) {
    authToken := r.Header.Get("Authorization")
    userAgent := r.Header.Get("User-Agent")
    contentType := r.Header.Get("Content-Type")
})
```

### Request Body

Access raw request body:

```go
body, err := io.ReadAll(r.Body)
defer r.Body.Close()
```

For structured data, use [Data Binding](data-binding.html) instead.

### Request Context

Access context for cancellation and values:

```go
mux.HandleFunc("GET /api/data", func(w app.ResponseWriter, r *app.Request) {
    ctx := r.Context()
    
    // Check if cancelled
    select {
    case <-ctx.Done():
        return
    default:
        // Continue processing
    }
})
```

## Response Methods

All response methods require `context.Context` as the first parameter (obtained from `r.Context()`). This enables JSONP support and internationalization.

### JSON Response

```go
w.JSON(r.Context(), map[string]string{"message": "Success"})
```

Automatically handles JSONP if configured.

### HTML Response

Render a template:

```go
data := map[string]interface{}{"Name": "John"}
err := w.HTML(r.Context(), "users/profile", data)
if err != nil {
    w.Error(http.StatusInternalServerError, err.Error())
}
```

### HTML String Response

Render inline HTML template:

{% raw %}
```go
err := w.HTMLString("<h1>{{.Title}}</h1>", map[string]string{"Title": "Hello"})
```
{% endraw %}

### Text Response

Render a text template:

```go
err := w.Text(r.Context(), "users/email", data)
```

### Text String Response

Render inline text template:

{% raw %}
```go
err := w.TextString("Hello {{.Name}}", map[string]string{"Name": "John"})
```
{% endraw %}

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
w.Error(http.StatusNotFound, "User not found")
w.Error(http.StatusInternalServerError, "Server error")
```

### Custom Headers

```go
w.Header().Set("X-Custom-Header", "value")
w.Header().Set("Cache-Control", "no-cache")
w.WriteHeader(http.StatusOK)
w.JSON(r.Context(), data)
```

### Status Codes

```go
// Success (2xx)
w.WriteHeader(http.StatusOK)                    // 200
w.WriteHeader(http.StatusCreated)               // 201
w.WriteHeader(http.StatusAccepted)              // 202
w.WriteHeader(http.StatusNoContent)             // 204

// Redirection (3xx)
w.WriteHeader(http.StatusMovedPermanently)      // 301
w.WriteHeader(http.StatusFound)                 // 302
w.WriteHeader(http.StatusSeeOther)              // 303
w.WriteHeader(http.StatusNotModified)           // 304

// Client Error (4xx)
w.WriteHeader(http.StatusBadRequest)            // 400
w.WriteHeader(http.StatusUnauthorized)          // 401
w.WriteHeader(http.StatusForbidden)             // 403
w.WriteHeader(http.StatusNotFound)              // 404
w.WriteHeader(http.StatusMethodNotAllowed)      // 405
w.WriteHeader(http.StatusConflict)              // 409
w.WriteHeader(http.StatusUnprocessableEntity)   // 422
w.WriteHeader(http.StatusTooManyRequests)       // 429

// Server Error (5xx)
w.WriteHeader(http.StatusInternalServerError)   // 500
w.WriteHeader(http.StatusNotImplemented)        // 501
w.WriteHeader(http.StatusBadGateway)            // 502
w.WriteHeader(http.StatusServiceUnavailable)    // 503
```

## Complete Examples

### API Endpoint with Error Handling

```go
mux.HandleFunc("GET /api/users/{id}", func(w app.ResponseWriter, r *app.Request) {
    id := r.PathValue("id")
    
    // Validate ID
    if id == "" {
        w.Error(http.StatusBadRequest, "Missing user ID")
        return
    }
    
    // Get user from database
    user, err := db.GetUser(id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            w.Error(http.StatusNotFound, "User not found")
            return
        }
        w.Error(http.StatusInternalServerError, "Database error")
        return
    }
    
    // Success
    w.JSON(r.Context(), user)
})
```

### File Upload Handler

```go
mux.HandleFunc("POST /upload", func(w app.ResponseWriter, r *app.Request) {
    // Parse multipart form (32 MB max)
    err := r.ParseMultipartForm(32 << 20)
    if err != nil {
        w.Error(http.StatusBadRequest, "Invalid form data")
        return
    }
    
    // Get file from form
    file, header, err := r.FormFile("file")
    if err != nil {
        w.Error(http.StatusBadRequest, "No file uploaded")
        return
    }
    defer file.Close()
    
    // Save file
    filename := filepath.Join("/uploads", header.Filename)
    dst, err := os.Create(filename)
    if err != nil {
        w.Error(http.StatusInternalServerError, "Failed to save file")
        return
    }
    defer dst.Close()
    
    io.Copy(dst, file)
    
    w.JSON(r.Context(), map[string]string{
        "message":  "File uploaded successfully",
        "filename": header.Filename,
    })
})
```

### Streaming Response

```go
mux.HandleFunc("GET /stream", func(w app.ResponseWriter, r *app.Request) {
    w.Header().Set("Content-Type", "text/plain")
    w.Header().Set("Transfer-Encoding", "chunked")
    
    flusher, ok := w.ResponseWriter.(http.Flusher)
    if !ok {
        w.Error(http.StatusInternalServerError, "Streaming not supported")
        return
    }
    
    for i := 0; i < 10; i++ {
        fmt.Fprintf(w, "Chunk %d\n", i)
        flusher.Flush()
        time.Sleep(time.Second)
    }
})
```

## See Also

- [Data Binding](data-binding.html)
- [Templates](templates.html)
- [JSONP](jsonp.html)
- [Middleware](middleware.html)

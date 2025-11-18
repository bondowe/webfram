---
layout: default
title: JSONP
nav_order: 9
description: "Cross-origin requests with JSONP"
---

# JSONP Support

WebFram provides built-in support for JSONP (JSON with Padding) to enable cross-origin requests from browsers that don't support CORS.

## Overview

JSONP wraps JSON responses in a JavaScript function call, allowing cross-origin requests. However, **CORS is preferred** for modern applications.

## Configuration

Enable JSONP in your application configuration:

```go
app.Configure(&app.Config{
    JSONPCallbackParamName: "callback", // Enable JSONP
    // ... other config
})
```

If `JSONPCallbackParamName` is not set or empty, JSONP is disabled.

## Usage

Once configured, any route using `w.JSON(r.Context(), data)` automatically supports JSONP when the callback parameter is present:

```go
mux.HandleFunc("GET /api/users", func(w app.ResponseWriter, r *app.Request) {
    users := []User{
        {ID: uuid.New(), Name: "John Doe"},
        {ID: uuid.New(), Name: "Jane Smith"},
    }
    
    // Automatically handles both JSON and JSONP
    w.JSON(r.Context(), users)
})
```

## Request Examples

**Standard JSON** (no callback parameter):

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

**JSONP** (with callback parameter):

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

## Client-Side Usage

**Vanilla JavaScript:**

```html
<script>
function myCallback(data) {
    console.log('Received data:', data);
}

var script = document.createElement('script');
script.src = 'http://localhost:8080/api/users?callback=myCallback';
document.body.appendChild(script);
</script>
```

**jQuery:**

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

## Response Details

When JSONP is enabled and callback parameter is provided:

- Content-Type: `application/javascript`
- Response format: `callbackName(jsonData);`
- Callback parameter name is configurable

## Callback Validation

🔒 **Built-in Security**: WebFram automatically validates JSONP callback names:

- **Allowed**: Only alphanumeric characters (a-z, A-Z, 0-9) and underscores (_)
- **Must start with**: Letter or underscore
- **Pattern**: `^[a-zA-Z_][a-zA-Z0-9_]*$`

**Valid callbacks:**

- `myCallback`
- `callback123`
- `my_callback_function`
- `_privateCallback`
- `jQuery123456789_callback`

**Invalid callbacks:**

- `123callback` (starts with number)
- `my-callback` (contains hyphen)
- `callback()` (contains parentheses)
- `alert('xss')` (XSS attempt)

**Error response for invalid callback:**

```http
HTTP/1.1 400 Bad Request
Content-Type: text/plain

invalid JSONP callback method name: "my-callback()". Only alphanumeric characters and underscores are allowed
```

## Security Considerations

⚠️ **Important Security Notes:**

### 1. Callback Validation

WebFram automatically validates callback names to prevent XSS attacks. Only alphanumeric characters and underscores are allowed.

### 2. Use CORS Instead

For modern browsers, use CORS:

```go
func corsMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
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

### 3. Sensitive Data

Avoid exposing sensitive data through JSONP endpoints as they can be requested from any origin.

### 4. Read-Only Operations

**Use JSONP only for read-only GET requests**. Never use JSONP for state-changing operations (POST, PUT, DELETE, PATCH).

### 5. Environment-Specific

Enable JSONP only in specific environments:

```go
func getConfig() *app.Config {
    cfg := &app.Config{
        Assets: &app.Assets{FS: assetsFS},
    }
    
    // Only enable JSONP in development
    if os.Getenv("ENV") == "development" {
        cfg.JSONPCallbackParamName = "callback"
    }
    
    return cfg
}
```

## Best Practices

1. **Prefer CORS** over JSONP for modern applications
2. **Use HTTPS** when serving JSONP endpoints
3. **Validate origins** if possible
4. **Read-only endpoints** - Never allow state changes via JSONP
5. **Audit log** JSONP requests in production
6. **Disable in production** unless absolutely necessary

## See Also

- [Request & Response](request-response)
- [Configuration](configuration)
- [Middleware](middleware)

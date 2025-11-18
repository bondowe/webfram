---
layout: default
title: JSON Patch
nav_order: 8
description: "RFC 6902 JSON Patch support for partial updates"
---

# JSON Patch Support

WebFram supports [RFC 6902 JSON Patch](https://tools.ietf.org/html/rfc6902) for partial resource updates.

## Overview

JSON Patch allows you to update resources partially using a standardized format, perfect for RESTful PATCH endpoints.

## Basic Usage

```go
type User struct {
    ID    uuid.UUID `json:"id"`
    Name  string    `json:"name" validate:"required,minlength=3"`
    Email string    `json:"email" validate:"required,format=email"`
    Role  string    `json:"role" validate:"enum=admin|user|guest"`
}

mux.HandleFunc("PATCH /users/{id}", func(w app.ResponseWriter, r *app.Request) {
    id := r.PathValue("id")
    
    // Fetch existing user
    user, err := getUserFromDB(id)
    if err != nil {
        w.Error(http.StatusNotFound, "User not found")
        return
    }
    
    // Apply JSON Patch with validation
    valErrors, err := app.PatchJSON(r, &user, true)
    if err != nil {
        if err == app.ErrMethodNotAllowed {
            w.Error(http.StatusMethodNotAllowed, "PATCH method required")
            return
        }
        w.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    if len(valErrors) > 0 {
        w.WriteHeader(http.StatusBadRequest)
        w.JSON(r.Context(), app.ValidationErrors{Errors: valErrors})
        return
    }
    
    // Save updated user
    saveUserToDB(user)
    w.JSON(r.Context(), user)
})
```

## Supported Operations

### Add

Add a new value:

```json
{"op": "add", "path": "/email", "value": "new@example.com"}
```

### Remove

Remove a value:

```json
{"op": "remove", "path": "/email"}
```

### Replace

Replace an existing value:

```json
{"op": "replace", "path": "/name", "value": "New Name"}
```

### Move

Move a value from one location to another:

```json
{"op": "move", "from": "/oldField", "path": "/newField"}
```

### Copy

Copy a value to a new location:

```json
{"op": "copy", "from": "/name", "path": "/displayName"}
```

### Test

Test that a value matches:

```json
{"op": "test", "path": "/name", "value": "Expected Name"}
```

## Request Example

```bash
curl -X PATCH http://localhost:8080/users/123 \
  -H "Content-Type: application/json-patch+json" \
  -d '[
    {"op": "replace", "path": "/name", "value": "John Updated"},
    {"op": "replace", "path": "/email", "value": "john.updated@example.com"}
  ]'
```

## With Validation

Control validation with the third parameter:

```go
// With validation (recommended)
valErrors, err := app.PatchJSON(r, &user, true)
if err != nil {
    w.Error(http.StatusBadRequest, err.Error())
    return
}
if len(valErrors) > 0 {
    w.WriteHeader(http.StatusBadRequest)
    w.JSON(r.Context(), app.ValidationErrors{Errors: valErrors})
    return
}

// Without validation
valErrors, err := app.PatchJSON(r, &user, false)
```

## Error Handling

`PatchJSON` returns specific errors:

- **`app.ErrMethodNotAllowed`** - Called on non-PATCH requests
- **Content-Type validation** - Requires `application/json-patch+json`
- **Patch errors** - Invalid JSON or malformed operations
- **Validation errors** - Returned when validation is enabled

```go
valErrors, err := app.PatchJSON(r, &resource, true)
if err != nil {
    if err == app.ErrMethodNotAllowed {
        w.Error(http.StatusMethodNotAllowed, "Only PATCH allowed")
        return
    }
    w.Error(http.StatusBadRequest, err.Error())
    return
}

if len(valErrors) > 0 {
    w.WriteHeader(http.StatusBadRequest)
    w.JSON(r.Context(), app.ValidationErrors{Errors: valErrors})
    return
}
```

## Complete Example

```go
type Product struct {
    ID          uuid.UUID `json:"id"`
    Name        string    `json:"name" validate:"required,minlength=3"`
    Description string    `json:"description" validate:"maxlength=500"`
    Price       int       `json:"price" validate:"min=0"`
    InStock     bool      `json:"in_stock"`
}

mux.HandleFunc("PATCH /products/{id}", func(w app.ResponseWriter, r *app.Request) {
    id := r.PathValue("id")
    
    product, err := getProduct(id)
    if err != nil {
        w.Error(http.StatusNotFound, "Product not found")
        return
    }
    
    valErrors, err := app.PatchJSON(r, &product, true)
    if err != nil {
        w.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    if len(valErrors) > 0 {
        w.WriteHeader(http.StatusBadRequest)
        w.JSON(r.Context(), app.ValidationErrors{Errors: valErrors})
        return
    }
    
    updateProduct(product)
    w.JSON(r.Context(), product)
})
```

**Request:**

```bash
curl -X PATCH http://localhost:8080/products/550e8400-e29b-41d4-a716-446655440000 \
  -H "Content-Type: application/json-patch+json" \
  -d '[
    {"op": "replace", "path": "/price", "value": 2999},
    {"op": "replace", "path": "/in_stock", "value": true}
  ]'
```

## Best Practices

1. **Always require `application/json-patch+json` content-type**
2. **Validate after applying patches** to ensure data integrity
3. **Fetch latest data** before applying patches to avoid conflicts
4. **Use optimistic locking** for concurrent updates
5. **Log patch operations** for audit trails

## See Also

- [Data Binding](data-binding.md)
- [Request & Response](request-response.md)
- [OpenAPI](openapi.md)

# WebFram

[![CI](https://github.com/bondowe/webfram/actions/workflows/ci.yml/badge.svg)](https://github.com/bondowe/webfram/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/bondowe/webfram/branch/main/graph/badge.svg)](https://codecov.io/gh/bondowe/webfram)
[![Go Report Card](https://goreportcard.com/badge/github.com/bondowe/webfram)](https://goreportcard.com/report/github.com/bondowe/webfram)
[![Go Reference](https://pkg.go.dev/badge/github.com/bondowe/webfram.svg)](https://pkg.go.dev/github.com/bondowe/webfram)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**WebFram** is a lightweight, feature-rich Go web framework built on top of the standard library's `net/http` package. It provides powerful features like automatic template caching with layouts, comprehensive data binding with validation, internationalization (i18n), Server-Sent Events (SSE), JSON Patch support, JSONP, OpenAPI documentation generation, and flexible middleware support.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Routing](#routing)
- [Middleware](#middleware)
- [Request Handling](#request-handling)
- [Response Handling](#response-handling)
- [Data Binding & Validation](#data-binding--validation)
  - [Form Binding](#form-binding)
  - [JSON Binding](#json-binding)
  - [XML Binding](#xml-binding)
  - [Nested Structs](#nested-structs)
  - [Map Binding (Form Only)](#map-binding-form-only)
  - [Validation Tags Reference](#validation-tags-reference)
  - [Custom Error Messages](#custom-error-messages)
  - [Validation Errors](#validation-errors)
  - [Supported Field Types](#supported-field-types)
- [JSON Patch Support](#json-patch-support)
- [JSONP Support](#jsonp-support)
- [OpenAPI Documentation](#openapi-documentation)
- [Server-Sent Events (SSE)](#server-sent-events-sse)
- [Templates](#templates)
- [Internationalization (i18n)](#internationalization-i18n)
- [Complete Example](#complete-example)
- [License](#license)

## Features

- 🚀 **Lightweight**: Built on top of `net/http` with minimal overhead
- 📝 **Smart Templates**: Automatic template caching with layout inheritance and partials
- ✅ **Data Binding**: Form, JSON, and XML binding with comprehensive validation
- 🗺️ **Map Support**: Form binding supports maps with `fieldname[key]=value` syntax
- 🔄 **JSON Patch**: RFC 6902 JSON Patch support for partial updates
- 🌐 **JSONP**: Secure cross-origin JSON requests with callback validation
- 📡 **Server-Sent Events**: Built-in SSE support for real-time server-to-client streaming
- 📚 **OpenAPI**: Automatic OpenAPI 3.1 documentation generation
- 🌍 **i18n Support**: First-class internationalization using `golang.org/x/text`
- 🔧 **Flexible Middleware**: Support for both custom and standard HTTP middleware
- 📦 **Multiple Response Formats**: JSON, JSONP, XML, YAML, HTML, and plain text
- 🎯 **Type-Safe**: Generic-based binding for type safety
- 🔒 **Comprehensive Validation**: 20+ validation rules including required, min/max, regex, enum, uniqueItems, multipleOf, and more

## Installation

```bash
go get github.com/bondowe/webfram
```

## Quick Start

```go
package main

import (
    "net/http"
    app "github.com/bondowe/webfram"
)

func main() {
    // Create a new mux
    mux := app.NewServeMux()

    // Define a route
    mux.HandleFunc("GET /hello", func(w app.ResponseWriter, r *app.Request) {
        w.JSON(map[string]string{"message": "Hello, World!"})
    })

    // Start the server
    app.ListenAndServe(":8080", mux)
}
```

## Configuration

WebFram can be configured with templates, i18n, JSONP, and OpenAPI settings:

```go
//go:embed templates
var templatesFS embed.FS

//go:embed locales/*.json
var i18nFS embed.FS

func main() {
    app.Configure(&app.Config{
        Templates: &app.TemplateConfig{
            FS:                    templatesFS,
            TemplatesPath:         "templates",
            LayoutBaseName:        "layout",
            HTMLTemplateExtension: ".go.html",
            TextTemplateExtension: ".go.txt",
        },
        I18n: &app.I18nConfig{
            FS: i18nFS,
        },
        JSONPCallbackParamName: "callback", // Enable JSONP with custom param name
        OpenAPI: &app.OpenAPIConfig{
            EndpointEnabled: true,
            URLPath:         "GET /openapi.json", // Optional, defaults to GET /openapi.json
            Config:          getOpenAPIConfig(),
        },
    })

    // ... rest of your app
}
```

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `Templates.FS` | `nil` (required) | File system containing templates |
| `Templates.TemplatesPath` | `""` | Root path for templates within FS |
| `Templates.LayoutBaseName` | `"layout"` | Base name for layout files |
| `Templates.HTMLTemplateExtension` | `".go.html"` | Extension for HTML templates |
| `Templates.TextTemplateExtension` | `".go.txt"` | Extension for text templates |
| `I18n.FS` | `os.DirFS("i18n")` | File system for i18n message files |
| `JSONPCallbackParamName` | `""` (disabled) | Query parameter name for JSONP callbacks |
| `OpenAPI.EndpointEnabled` | `false` | Enable/disable OpenAPI endpoint |
| `OpenAPI.URLPath` | `"GET /openapi.json"` | Path for OpenAPI spec endpoint |
| `OpenAPI.Config` | `nil` | OpenAPI configuration |

**Note:** The i18n function name in templates is always `T` and cannot be configured.

## Routing

WebFram uses Go 1.22+ routing patterns with HTTP method prefixes:

```go
mux := app.NewServeMux()

// Simple routes
mux.HandleFunc("GET /users", listUsers)
mux.HandleFunc("POST /users", createUser)
mux.HandleFunc("GET /users/{id}", getUser)
mux.HandleFunc("PUT /users/{id}", updateUser)
mux.HandleFunc("PATCH /users/{id}", patchUser)  // JSON Patch support
mux.HandleFunc("DELETE /users/{id}", deleteUser)

// Wildcard routes
mux.HandleFunc("GET /files/{path...}", serveFiles)
```

### Accessing Route Parameters

```go
mux.HandleFunc("GET /users/{id}", func(w app.ResponseWriter, r *app.Request) {
    id := r.PathValue("id")
    // ... handle request
})
```

## Middleware

WebFram supports both custom and standard HTTP middleware.

### Global Middleware

Applied to all routes:

```go
// Custom middleware
app.Use(func(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        // Before handler
        log.Printf("Request: %s %s", r.Method, r.URL.Path)
        
        next.ServeHTTP(w, r)
        
        // After handler (if needed)
    })
})

// Standard HTTP middleware
app.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Standard middleware logic
        next.ServeHTTP(w, r)
    })
})
```

### Mux-Level Middleware

Applied to all routes in a specific mux:

```go
mux := app.NewServeMux()

mux.Use(func(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        // Mux-level middleware logic
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

### Middleware Execution Order

Middleware executes in this order:

1. Global middleware (registered with `app.Use()`)
2. Mux-level middleware (registered with `mux.Use()`)
3. Route-specific middleware
4. i18n middleware (automatic)
5. Handler

## Request Handling

### Query Parameters

```go
mux.HandleFunc("GET /search", func(w app.ResponseWriter, r *app.Request) {
    query := r.URL.Query().Get("q")
    page := r.URL.Query().Get("page")
    // ... handle request
})
```

### Request Body

Access the raw request body:

```go
body, err := io.ReadAll(r.Body)
defer r.Body.Close()
```

## Response Handling

WebFram provides multiple response methods:

### JSON Response

```go
w.JSON(map[string]string{"message": "Success"})
```

The `JSON` method automatically handles JSONP requests if configured (see [JSONP Support](#jsonp-support)).

### HTML Response

```go
// Render a template
data := map[string]interface{}{"Name": "John"}
err := w.HTML("users/profile", data)
```

### HTML String Response

```go
err := w.HTMLString("<h1>{{.Title}}</h1>", map[string]string{"Title": "Hello"})
```

### Text Response

```go
// Render a text template
err := w.Text("users/email", data)
```

### Text String Response

```go
err := w.TextString("Hello {{.Name}}", map[string]string{"Name": "John"})
```

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
```

### Custom Headers

```go
w.Header().Set("X-Custom-Header", "value")
w.WriteHeader(http.StatusOK).JSON(data)
```

## Data Binding & Validation

WebFram provides type-safe data binding with comprehensive validation for Form, JSON, and XML formats.

### Form Binding

Form binding automatically parses form data and validates it according to struct tags.

```go
type CreateUserRequest struct {
    Name      string    `form:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"`
    Email     string    `form:"email" validate:"required,format=email" errmsg:"required=Email is required;format=Invalid email address"`
    Age       int       `form:"age" validate:"min=18,max=120" errmsg:"min=Must be at least 18;max=Must be at most 120"`
    Role      string    `form:"role" validate:"enum=admin|user|guest" errmsg:"enum=Must be admin, user, or guest"`
    Birthdate time.Time `form:"birthdate" validate:"required" format:"2006-01-02" errmsg:"required=Birthdate is required"`
    Hobbies   []string  `form:"hobbies" validate:"minItems=1,maxItems=5,uniqueItems" errmsg:"minItems=At least one hobby required;maxItems=Maximum 5 hobbies allowed"`
}

mux.HandleFunc("POST /users", func(w app.ResponseWriter, r *app.Request) {
    user, valErrors, err := app.BindForm[CreateUserRequest](r)
    
    if err != nil {
        w.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    if valErrors.Any() {
        w.WriteHeader(http.StatusBadRequest).JSON(valErrors)
        return
    }
    
    // Process valid user data
    w.JSON(user)
})
```

**Form data example:**

```
name=John+Doe&email=john@example.com&age=30&role=admin&birthdate=1993-01-15&hobbies=reading&hobbies=coding
```

### JSON Binding

JSON binding parses JSON request bodies with optional validation.

```go
type CreateUserRequest struct {
    Name    string   `json:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"`
    Email   string   `json:"email" validate:"required,format=email" errmsg:"format=Invalid email address"`
    Hobbies []string `json:"hobbies" validate:"minItems=1,maxItems=5,uniqueItems" errmsg:"minItems=At least one hobby required;maxItems=Maximum 5 hobbies allowed"`
    Age     int      `json:"age" validate:"min=18,max=120,multipleOf=1"`
}

mux.HandleFunc("POST /api/users", func(w app.ResponseWriter, r *app.Request) {
    // Second parameter controls validation (true = validate, false = skip validation)
    user, valErrors, err := app.BindJSON[CreateUserRequest](r, true)
    
    if err != nil {
        w.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    if valErrors.Any() {
        w.WriteHeader(http.StatusBadRequest).JSON(valErrors)
        return
    }
    
    // Process valid user data
    w.JSON(user)
})
```

**JSON request example:**

```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "hobbies": ["reading", "coding"],
  "age": 30
}
```

### XML Binding

WebFram supports XML binding with the same validation features as JSON:

```go
type CreateUserRequest struct {
    XMLName   xml.Name  `xml:"user"`
    Name      string    `xml:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"`
    Email     string    `xml:"email" validate:"required,format=email" errmsg:"format=Invalid email address"`
    Age       int       `xml:"age" validate:"min=18,max=120"`
    Role      string    `xml:"role" validate:"enum=admin|user|guest" errmsg:"enum=Must be admin, user, or guest"`
}

mux.HandleFunc("POST /api/users", func(w app.ResponseWriter, r *app.Request) {
    // Second parameter controls validation (true = validate, false = skip validation)
    user, valErrors, err := app.BindXML[CreateUserRequest](r, true)
    
    if err != nil {
        w.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    if valErrors.Any() {
        w.WriteHeader(http.StatusBadRequest).XML(valErrors)
        return
    }
    
    // Process valid user data
    w.XML(user)
})
```

**Example XML request:**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<user>
    <name>John Doe</name>
    <email>john@example.com</email>
    <age>30</age>
    <role>admin</role>
</user>
```

### Nested Structs

All binding types support nested structs:

```go
type Address struct {
    Street string `json:"street" xml:"street" form:"street" validate:"required" errmsg:"required=Street is required"`
    City   string `json:"city" xml:"city" form:"city" validate:"required" errmsg:"required=City is required"`
    Zip    int    `json:"zip" xml:"zip" form:"zip" validate:"min=10000,max=99999" errmsg:"min=Invalid zip;max=Invalid zip"`
}

type User struct {
    Name    string  `json:"name" xml:"name" form:"name" validate:"required"`
    Address Address `json:"address" xml:"address" form:"address" validate:"required"`
}

// Form fields: name, address.street, address.city, address.zip
// JSON/XML: nested objects
```

**Form data example:**

```
name=John+Doe&address.street=123+Main+St&address.city=Springfield&address.zip=12345
```

**JSON example:**

```json
{
  "name": "John Doe",
  "address": {
    "street": "123 Main St",
    "city": "Springfield",
    "zip": 12345
  }
}
```

### Map Binding (Form Only)

Form binding supports maps with the syntax `fieldname[key]=value`:

```go
type Config struct {
    Metadata map[string]string `form:"metadata" validate:"minItems=1,maxItems=10"`
    Scores   map[string]int    `form:"scores"`
    Settings map[int]string    `form:"settings"`
}

mux.HandleFunc("POST /config", func(w app.ResponseWriter, r *app.Request) {
    config, valErrors, err := app.BindForm[Config](r)
    
    if err != nil {
        w.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    if valErrors.Any() {
        w.WriteHeader(http.StatusBadRequest).JSON(valErrors)
        return
    }
    
    w.JSON(config)
})
```

**Form data example:**

```
metadata[color]=red&metadata[size]=large&scores[math]=95&scores[science]=87&settings[1]=enabled&settings[2]=disabled
```

**Supported map types:**

- `map[string]string`
- `map[string]int`
- `map[int]string`
- `map[int]int`
- Other basic types (uint, float, bool)
- `map[string]time.Time`
- `map[string]uuid.UUID`

### Validation Tags Reference

WebFram supports 20+ validation tags for comprehensive data validation:

| Tag | Applies To | Description | Example |
|-----|------------|-------------|---------|
| `required` | All types | Field must be present and non-empty | `validate:"required"` |
| `min=N` | int, uint, float | Minimum value (inclusive) | `validate:"min=18"` |
| `max=N` | int, uint, float | Maximum value (inclusive) | `validate:"max=120"` |
| `multipleOf=N` | int, float | Value must be a multiple of N | `validate:"multipleOf=5"` |
| `minlength=N` | string | Minimum length in characters | `validate:"minlength=3"` |
| `maxlength=N` | string | Maximum length in characters | `validate:"maxlength=50"` |
| `minItems=N` | slice, map | Minimum number of items | `validate:"minItems=1"` |
| `maxItems=N` | slice, map | Maximum number of items | `validate:"maxItems=10"` |
| `uniqueItems` | slice | All items must be unique | `validate:"uniqueItems"` |
| `emptyItemsAllowed` | slice | Allow empty items in slice | `validate:"emptyItemsAllowed"` |
| `regexp=PATTERN` | string | Must match regular expression | `validate:"regexp=^\\w+@\\w+\\.com$"` |
| `pattern=PATTERN` | string | Alias for regexp | `validate:"pattern=^[A-Z]{3}-\\d{4}$"` |
| `enum=val1\|val2` | string | Must be one of specified values | `validate:"enum=admin\|user\|guest"` |
| `format=email` | string (form) | Must be a valid email (IDN supported) | `validate:"format=email"` |
| `format=LAYOUT` | time.Time | Time parsing layout | `format:"2006-01-02"` |

**Validation rules can be combined:**

```go
type Product struct {
    Name  string `json:"name" validate:"required,minlength=2,maxlength=100"`
    SKU   string `json:"sku" validate:"required,regexp=^[A-Z]{3}-\\d{4}$"`
    Price int    `json:"price" validate:"required,min=0,max=1000000,multipleOf=100"`
    Tags  []string `json:"tags" validate:"minItems=1,maxItems=20,uniqueItems"`
}
```

### Custom Error Messages

Use the `errmsg` tag to provide custom validation error messages:

```go
type User struct {
    Name  string `json:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"`
    Email string `json:"email" validate:"required,format=email" errmsg:"required=Email is required;format=Please provide a valid email address"`
    Age   int    `json:"age" validate:"min=18,max=120" errmsg:"min=Must be at least 18 years old;max=Must be at most 120 years old"`
}
```

**Format:** `errmsg:"validationRule1=Message1;validationRule2=Message2"`

### Validation Errors

WebFram provides a structured `ValidationErrors` type for handling validation errors:

```go
type ValidationErrors struct {
    Errors []ValidationError `json:"errors" xml:"errors"`
}

type ValidationError struct {
    Field string `json:"field" xml:"field"`
    Error string `json:"error" xml:"error"`
}
```

**Methods:**

- `Any() bool` - Returns true if there are any validation errors

**Example usage:**

```go
user, valErrors, err := app.BindJSON[CreateUserRequest](r, true)

if err != nil {
    // Binding error (malformed JSON, etc.)
    w.Error(http.StatusBadRequest, err.Error())
    return
}

if valErrors.Any() {
    // Validation errors - return structured error response
    w.WriteHeader(http.StatusBadRequest).JSON(valErrors)
    return
}

// No errors - proceed with valid data
```

**Validation error response (JSON format):**

```json
{
  "errors": [
    {
      "field": "name",
      "error": "Name is required"
    },
    {
      "field": "email",
      "error": "Invalid email address"
    },
    {
      "field": "age",
      "error": "Must be at least 18 years old"
    }
  ]
}
```

**Validation error response (XML format):**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<validationErrors>
    <errors>
        <validationError>
            <field>name</field>
            <error>Name is required</error>
        </validationError>
        <validationError>
            <field>email</field>
            <error>Invalid email address</error>
        </validationError>
        <validationError>
            <field>age</field>
            <error>Must be at least 18 years old</error>
        </validationError>
    </errors>
</validationErrors>
```

### Supported Field Types

- **Primitives**: `string`, `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`, `bool`
- **Time**: `time.Time`
- **UUID**: `uuid.UUID` (from `github.com/google/uuid`)
- **Slices**: `[]string`, `[]int`, `[]time.Time`, `[]uuid.UUID`, etc.
- **Maps** (form only): `map[string]string`, `map[string]int`, `map[int]string`, etc.
- **Nested structs**: Any struct type
- **Pointers**: All types support pointer variants

### Skipping Validation

For JSON and XML binding, you can skip validation by passing `false` as the second parameter:

```go
// Skip validation - useful when you trust the data source
user, valErrors, err := app.BindJSON[User](r, false)
// valErrors will be empty, only binding errors are checked

// With validation enabled
user, valErrors, err := app.BindJSON[User](r, true)
// Both binding and validation errors are checked
```

**Note:** Form binding always performs validation.

## JSON Patch Support

WebFram supports [RFC 6902 JSON Patch](https://tools.ietf.org/html/rfc6902) for partial resource updates using the `PATCH` HTTP method.

### Using JSON Patch

```go
type User struct {
    ID    uuid.UUID `json:"id"`
    Name  string    `json:"name" validate:"required,minlength=3"`
    Email string    `json:"email" validate:"required,format=email"`
    Role  string    `json:"role" validate:"enum=admin|user|guest"`
}

mux.HandleFunc("PATCH /users/{id}", func(w app.ResponseWriter, r *app.Request) {
    id := r.PathValue("id")
    
    // Fetch the existing user from database
    user, err := getUserFromDB(id)
    if err != nil {
        w.Error(http.StatusNotFound, "User not found")
        return
    }
    
    // Apply JSON Patch operations (with validation)
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
        w.WriteHeader(http.StatusBadRequest).JSON(app.ValidationErrors{Errors: valErrors})
        return
    }
    
    // Save the updated user to database
    err = saveUserToDB(user)
    if err != nil {
        w.Error(http.StatusInternalServerError, err.Error())
        return
    }
    
    w.JSON(user)
})
```

### JSON Patch Request Example

```bash
curl -X PATCH http://localhost:8080/users/123 \
  -H "Content-Type: application/json-patch+json" \
  -d '[
    {"op": "replace", "path": "/name", "value": "John Updated"},
    {"op": "replace", "path": "/email", "value": "john.updated@example.com"}
  ]'
```

### Supported JSON Patch Operations

The `PatchJSON` function supports all standard JSON Patch operations:

- **add**: Add a new value

  ```json
  {"op": "add", "path": "/email", "value": "new@example.com"}
  ```

- **remove**: Remove a value

  ```json
  {"op": "remove", "path": "/email"}
  ```

- **replace**: Replace an existing value

  ```json
  {"op": "replace", "path": "/name", "value": "New Name"}
  ```

- **move**: Move a value from one location to another

  ```json
  {"op": "move", "from": "/oldField", "path": "/newField"}
  ```

- **copy**: Copy a value to a new location

  ```json
  {"op": "copy", "from": "/name", "path": "/displayName"}
  ```

- **test**: Test that a value at the target location is equal to a specified value

  ```json
  {"op": "test", "path": "/name", "value": "Expected Name"}
  ```

### JSON Patch with Validation

The third parameter to `PatchJSON` controls whether validation is performed after applying the patch:

```go
// With validation - recommended for user input
valErrors, err := app.PatchJSON(r, &user, true)
if err != nil {
    // Handle patch errors
}
if len(valErrors) > 0 {
    // Handle validation errors
}

// Without validation - for trusted operations
valErrors, err := app.PatchJSON(r, &user, false)
// Only patch errors are checked, validation is skipped
```

### Error Handling

The `PatchJSON` function returns specific errors:

- `app.ErrMethodNotAllowed`: When called on non-PATCH requests
- Content-Type validation: Requires `application/json-patch+json` header
- Patch errors: Invalid JSON or malformed patch operations
- Validation errors: Returned as `[]ValidationError` when validation is enabled

```go
valErrors, err := app.PatchJSON(r, &resource, true)
if err != nil {
    if err == app.ErrMethodNotAllowed {
        w.Error(http.StatusMethodNotAllowed, "Only PATCH method is allowed")
        return
    }
    // Other errors (invalid JSON, invalid patch, wrong Content-Type)
    w.Error(http.StatusBadRequest, err.Error())
    return
}

if len(valErrors) > 0 {
    // Validation failed after applying patch
    w.WriteHeader(http.StatusBadRequest).JSON(app.ValidationErrors{Errors: valErrors})
    return
}
```

## JSONP Support

WebFram provides built-in support for JSONP (JSON with Padding) to enable cross-origin requests from browsers that don't support CORS.

### Configuring JSONP

Enable JSONP by setting the `JSONPCallbackParamName` in your application configuration:

```go
app.Configure(&app.Config{
    JSONPCallbackParamName: "callback", // Enable JSONP with "callback" query parameter
    // ... other config options
})
```

If `JSONPCallbackParamName` is not set or is empty, JSONP is disabled and all JSON responses are returned as standard JSON.

**Note:** The callback parameter name itself is validated and must start with a letter or underscore and contain only alphanumeric characters and underscores.

### Using JSONP

Once configured, any route that uses `w.JSON()` will automatically support JSONP when the callback parameter is present in the query string:

```go
mux.HandleFunc("GET /api/users", func(w app.ResponseWriter, r *app.Request) {
    users := []User{
        {ID: uuid.New(), Name: "John Doe"},
        {ID: uuid.New(), Name: "Jane Smith"},
    }
    
    // Automatically handles both JSON and JSONP
    w.JSON(users)
})
```

### JSONP Request Examples

**Standard JSON request** (no callback parameter):

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

**JSONP request** (with callback parameter):

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

### Client-Side JSONP Usage

**Using vanilla JavaScript:**

```html
<script>
function myCallback(data) {
    console.log('Received data:', data);
    // Process the data
}

// Create script tag to make JSONP request
var script = document.createElement('script');
script.src = 'http://localhost:8080/api/users?callback=myCallback';
document.body.appendChild(script);
</script>
```

**Using jQuery:**

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

### JSONP Response Details

When JSONP is enabled and a callback parameter is provided:

- Content-Type is set to `application/javascript`
- Response is wrapped with the callback function: `callbackName(jsonData);`
- The callback parameter name is configurable via `JSONPCallbackParamName`

### Callback Name Validation

🔒 **Built-in Security**: WebFram automatically validates JSONP callback names to prevent security vulnerabilities:

- **Allowed characters**: Only alphanumeric characters (a-z, A-Z, 0-9) and underscores (_)
- **Must start with**: Letter or underscore
- **Validation pattern**: `^[a-zA-Z_][a-zA-Z0-9_]*$`
- **Invalid callbacks**: Return a `400 Bad Request` error with a descriptive message

**Valid callback names:**

- `myCallback`
- `callback123`
- `my_callback_function`
- `_privateCallback`
- `jQuery123456789_callback`

**Invalid callback names:**

- `123callback` (starts with number)
- `my-callback` (contains hyphen)
- `callback()` (contains parentheses)
- `alert('xss')` (potential XSS attempt)
- `../../../etc/passwd` (path traversal attempt)

**Error response for invalid callback:**

```http
HTTP/1.1 400 Bad Request
Content-Type: text/plain

invalid JSONP callback method name: "my-callback()". Only alphanumeric characters and underscores are allowed
```

### Security Considerations

⚠️ **Important JSONP Security Notes:**

1. **Callback Validation**: WebFram automatically validates callback names to prevent XSS attacks and malicious code injection. Only alphanumeric characters and underscores are allowed, and the name must start with a letter or underscore.

2. **CORS Alternative**: If possible, use CORS instead of JSONP for modern browsers:

   ```go
   mux.Use(func(next app.Handler) app.Handler {
       return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
           w.Header().Set("Access-Control-Allow-Origin", "*")
           next.ServeHTTP(w, r)
       })
   })
   ```

3. **Sensitive Data**: Avoid exposing sensitive data through JSONP endpoints as they can be requested from any origin.

4. **Read-Only Operations**: Use JSONP only for read-only GET requests. Never use JSONP for state-changing operations (POST, PUT, DELETE, PATCH).

## OpenAPI Documentation

WebFram automatically generates OpenAPI 3.1 documentation from your route definitions, validation tags, and API configurations.

### Enabling OpenAPI

Configure OpenAPI in your application:

```go
app.Configure(&app.Config{
    OpenAPI: &app.OpenAPIConfig{
        EndpointEnabled: true,
        URLPath:         "GET /openapi.json", // Optional, defaults to GET /openapi.json
        Config:          getOpenAPIConfig(),
    },
})

func getOpenAPIConfig() *openapi.Config {
    return &openapi.Config{
        Info: &openapi.Info{
            Title:          "My API",
            Summary:        "API for my awesome application",
            Description:    "This API provides endpoints for managing users and products.",
            TermsOfService: "https://example.com/terms/",
            Contact: &openapi.Contact{
                Name:  "API Support",
                URL:   "https://example.com/support",
                Email: "support@example.com",
            },
            License: &openapi.License{
                Name:       "MIT",
                Identifier: "MIT",
                URL:        "https://opensource.org/licenses/MIT",
            },
            Version: "1.0.0",
        },
        Servers: []openapi.Server{
            {
                URL:         "http://localhost:8080",
                Description: "Local development server",
                Name:        "local",
            },
            {
                URL:         "https://api.example.com",
                Description: "Production server",
                Name:        "production",
            },
        },
    }
}
```

Once configured, access your OpenAPI spec at: `http://localhost:8080/openapi.json`

### Documenting Routes

Use `WithAPIConfig()` to add OpenAPI documentation to individual routes:

```go
mux.HandleFunc("POST /users", createUserHandler).WithAPIConfig(&app.APIConfig{
    OperationID: "createUser",
    Summary:     "Create a new user",
    Description: "Creates a new user account with the provided information.",
    Tags:        []string{"Users"},
    Parameters: []app.Parameter{
        {
            Name:        "X-Request-ID",
            In:          "header",
            Description: "Unique request identifier",
            Required:    false,
            TypeHint:    "",
            Example:     "550e8400-e29b-41d4-a716-446655440000",
        },
    },
    RequestBody: &app.RequestBody{
        Description: "User creation data",
        Required:    true,
        Content: map[string]app.TypeInfo{
            "application/json": {
                TypeHint: &User{},
                Examples: map[string]app.Example{
                    "admin": {
                        Summary:   "Admin user",
                        DataValue: User{Name: "Admin User", Role: "admin"},
                    },
                    "regular": {
                        Summary:   "Regular user",
                        DataValue: User{Name: "Regular User", Role: "user"},
                    },
                },
            },
        },
    },
    Responses: map[string]app.Response{
        "201": {
            Summary:     "User created successfully",
            Description: "The user was created successfully",
            Content: map[string]app.TypeInfo{
                "application/json": {
                    TypeHint: &User{},
                },
            },
        },
        "400": {
            Description: "Invalid request data",
        },
        "500": {
            Description: "Internal server error",
        },
    },
})
```

### Path-Level Configuration

Configure documentation for entire paths that apply to all operations:

```go
app.SetOpenAPIPathInfo("/users/{id}", &app.PathInfo{
    Summary:     "User operations",
    Description: "Endpoints for managing individual users",
    Parameters: []app.Parameter{
        {
            Name:        "id",
            In:          "path",
            Description: "User ID",
            Required:    true,
            TypeHint:    "",
            MinLength:   36,
            MaxLength:   36,
            Example:     "550e8400-e29b-41d4-a716-446655440000",
        },
    },
    Servers: []app.Server{
        {Name: "Local", URL: "http://localhost:8080"},
    },
})
```

### Schema Generation from Struct Tags

WebFram automatically generates JSON schemas from struct tags, including validation rules:

```go
type User struct {
    Name    string   `json:"name" validate:"required,minlength=3,maxlength=50"`
    Email   string   `json:"email" validate:"required,format=email"`
    Age     int      `json:"age" validate:"min=18,max=120"`
    Role    string   `json:"role" validate:"enum=admin|user|guest"`
    Hobbies []string `json:"hobbies" validate:"minItems=1,maxItems=10,uniqueItems"`
}
```

This generates an OpenAPI schema with:

- Required fields
- String length constraints (minLength, maxLength)
- Numeric constraints (minimum, maximum)
- Enum values
- Array constraints (minItems, maxItems, uniqueItems)
- Format specifications (email, uuid, date-time)

### Viewing OpenAPI Documentation

After starting your server, access the OpenAPI spec at your configured endpoint:

```bash
curl http://localhost:8080/openapi.json
```

You can also use tools like Swagger UI or Redoc to visualize your API documentation:

```html
<!-- Swagger UI -->
<!DOCTYPE html>
<html>
<head>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@latest/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@latest/swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({
            url: 'http://localhost:8080/openapi.json',
            dom_id: '#swagger-ui',
        })
    </script>
</body>
</html>
```

## Server-Sent Events (SSE)

WebFram provides built-in support for Server-Sent Events (SSE), enabling real-time server-to-client communication over HTTP. SSE is ideal for push notifications, live updates, streaming data, and real-time dashboards.

### Creating an SSE Endpoint

Use the `app.SSE()` function to create an SSE handler:

```go
mux.Handle("GET /events", app.SSE(
    payloadFunc,      // Function that generates SSE payload
    disconnectFunc,   // Function called when client disconnects
    errorFunc,        // Function called on errors
    interval,         // Time interval between messages
    headers,          // Optional custom headers
))
```

### SSE Payload Structure

The `SSEPayload` struct defines the message format:

```go
type SSEPayload struct {
    Id       string        // Event ID (optional)
    Event    string        // Event type/name (optional)
    Comments []string      // Comments (optional, for debugging)
    Data     any          // Data payload (required)
    Retry    time.Duration // Retry interval (optional)
}
```

### Basic Example

```go
mux.Handle("GET /time", app.SSE(
    // Payload function - generates data to send
    func() app.SSEPayload {
        return app.SSEPayload{
            Data: fmt.Sprintf("Current time: %s", time.Now().Format(time.RFC3339)),
        }
    },
    // Disconnect function - called when client disconnects
    func() {
        log.Println("Client disconnected")
    },
    // Error function - called on errors
    func(err error) {
        log.Printf("SSE error: %v\n", err)
    },
    // Interval - send message every 2 seconds
    2*time.Second,
    // Custom headers - nil for default
    nil,
))
```

### Advanced Example with Event Types

```go
mux.Handle("GET /notifications", app.SSE(
    func() app.SSEPayload {
        // Simulate different notification types
        notificationType := getNextNotification()
        
        return app.SSEPayload{
            Id:       uuid.New().String(),           // Unique event ID
            Event:    notificationType,               // Event type (e.g., "message", "alert")
            Comments: []string{"Notification event"}, // Optional comments
            Data:     generateNotificationData(),     // Notification payload
            Retry:    5 * time.Second,               // Client retry interval
        }
    },
    func() {
        log.Println("Client stopped listening to notifications")
    },
    func(err error) {
        log.Printf("Notification stream error: %v\n", err)
    },
    1*time.Second,
    nil,
))
```

### Client-Side Usage

**Vanilla JavaScript:**

```javascript
const eventSource = new EventSource('http://localhost:8080/events');

// Listen for messages
eventSource.onmessage = function(event) {
    console.log('Received:', event.data);
};

// Listen for specific event types
eventSource.addEventListener('TIME_UPDATE', function(event) {
    console.log('Time update:', event.data);
});

// Handle errors
eventSource.onerror = function(error) {
    console.error('EventSource error:', error);
};

// Close connection when done
// eventSource.close();
```

### SSE Configuration

#### Required Parameters

- **`payloadFunc`**: Function that returns an `SSEPayload`. Called at each interval.
- **`interval`**: Must be greater than zero. Determines how often messages are sent.

#### Optional Parameters

- **`disconnectFunc`**: Called when client disconnects. Defaults to no-op if `nil`.
- **`errorFunc`**: Called on stream errors. Defaults to printing errors if `nil`.
- **`headers`**: Map of custom HTTP headers to include in the response. Can be `nil`.

### Use Cases

**Real-time Dashboards:**

```go
mux.Handle("GET /dashboard", app.SSE(dashboardMetrics, nil, nil, 1*time.Second, nil))
```

**Live Notifications:**

```go
mux.Handle("GET /notifications", app.SSE(userNotifications, nil, nil, 2*time.Second, nil))
```

**Stock Price Updates:**

```go
mux.Handle("GET /stocks/{symbol}", app.SSE(stockPriceUpdates, nil, nil, 1*time.Second, nil))
```

**Log Streaming:**

```go
mux.Handle("GET /logs", app.SSE(tailLogs, nil, nil, 500*time.Millisecond, nil))
```

## Templates

WebFram provides a powerful templating system with automatic caching, layout inheritance, and partials.

### Template Configuration

Templates must be provided via an embedded file system and a base path:

```go
//go:embed templates
var templatesFS embed.FS

app.Configure(&app.Config{
    Templates: &app.TemplateConfig{
        FS:                    templatesFS,
        TemplatesPath:         "templates",
        LayoutBaseName:        "layout",
        HTMLTemplateExtension: ".go.html",
        TextTemplateExtension: ".go.txt",
    },
})
```

### Template Structure

```
templates/
├── layout.go.html              # Root layout (inherited by all)
├── users/
│   ├── layout.go.html          # Users layout (inherits from root)
│   ├── list.go.html            # Inherits from users layout
│   ├── details.go.html
│   └── manage/
│       ├── update.go.html
│       └── delete.go.html
├── _partOne.go.html            # Partial template
└── openapi.html
```

### Layout Files

Layouts are automatically detected and applied:

- `layout.go.html` - Standard layout in each directory (inherits from parent)

**Root layout** (`templates/layout.go.html`):

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{block "title" .}}Default Title{{end}}</title>
</head>
<body>
    {{block "content" .}}{{end}}
</body>
</html>
```

**Page template** (`templates/users/list.go.html`):

```html
{{define "title"}}Users List{{end}}

{{define "content"}}
<h1>Users</h1>
<ul>
    {{range .Users}}
    <li>{{.Name}}</li>
    {{end}}
</ul>
{{end}}
```

### Partials

Partials are reusable template components with names starting with `_`:

**Partial** (`templates/_partOne.go.html`):

```html
<header>
    <h1>{{.Title}}</h1>
</header>
```

**Using partials in templates**:

```html
{{define "content"}}
    <!-- Include a partial -->
    {{template "_partOne.go.html" .}}
    
    <div>Your main content here</div>
{{end}}
```

### Rendering Templates

```go
mux.HandleFunc("GET /users", func(w app.ResponseWriter, r *app.Request) {
    data := map[string]interface{}{
        "Users": []User{
            {Name: "John", Email: "john@example.com"},
            {Name: "Jane", Email: "jane@example.com"},
        },
    }
    
    err := w.HTML("users/list", data)
    if err != nil {
        w.Error(http.StatusInternalServerError, err.Error())
    }
})
```

## Internationalization (i18n)

WebFram provides built-in i18n support using `golang.org/x/text`. The i18n function is always available in templates as `T`.

### Message Files

Create message files in JSON format in your locales directory:

**locales/messages.en.json**:

```json
{
  "language": "en",
  "messages": [
    {
      "id": "Welcome to %s! Clap %d times.",
      "message": "Welcome to %s! Clap %d times.",
      "translation": "Welcome to %s! Clap %d times.",
      "placeholders": {
        "arg_1": {
          "id": "arg_1",
          "string": "%s",
          "type": "string",
          "underlyingType": "string",
          "argNum": 1,
          "expr": "arg1"
        },
        "arg_2": {
          "id": "arg_2",
          "string": "%d",
          "type": "int",
          "underlyingType": "int",
          "argNum": 2,
          "expr": "arg2"
        }
      }
    }
  ]
}
```

### Configure i18n

```go
//go:embed locales/*.json
var i18nFS embed.FS

app.Configure(&app.Config{
    I18n: &app.I18nConfig{
        FS: i18nFS,
    },
})
```

### Using i18n in Handlers

```go
import "golang.org/x/text/language"

mux.HandleFunc("GET /greeting", func(w app.ResponseWriter, r *app.Request) {
    printer := app.GetI18nPrinter(language.Spanish)
    msg := printer.Sprintf("Welcome to %s! Clap %d times.", "WebFram", 5)
    w.JSON(map[string]string{"message": msg})
})
```

### Using i18n in Templates

The i18n function is automatically available in templates as `T` (not configurable):

```html
{{define "content"}}
<h1>{{T "Welcome to %s! Clap %d times." "WebFram" 5}}</h1>
{{end}}
```

## Complete Example

```go
package main

import (
    "embed"
    "fmt"
    "log"
    "net/http"
    "time"
    
    app "github.com/bondowe/webfram"
    "github.com/bondowe/webfram/openapi"
    "github.com/google/uuid"
    "golang.org/x/text/language"
)

//go:embed templates
var templatesFS embed.FS

//go:embed locales/*.json
var i18nFS embed.FS

type User struct {
    ID        uuid.UUID   `json:"id" xml:"id" form:"id"`
    Name      string      `json:"name" xml:"name" form:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"`
    Email     string      `json:"email" xml:"email" form:"email" validate:"required,format=email" errmsg:"required=Email is required;format=Invalid email"`
    Role      string      `json:"role" xml:"role" form:"role" validate:"enum=admin|user|guest" errmsg:"enum=Must be admin, user, or guest"`
    Birthdate time.Time   `form:"birthdate" validate:"required" format:"2006-01-02"`
}

func loggingMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        duration := time.Since(start)
        log.Printf("%s %s - %v", r.Method, r.URL.Path, duration)
    })
}

func main() {
    // Configure the application
    app.Configure(&app.Config{
        Templates: &app.TemplateConfig{
            FS:            templatesFS,
            TemplatesPath: "templates",
        },
        I18n: &app.I18nConfig{
            FS: i18nFS,
        },
        JSONPCallbackParamName: "callback", // Enable JSONP support
        OpenAPI: &app.OpenAPIConfig{
            EndpointEnabled: true,
            Config:          getOpenAPIConfig(),
        },
    })

    // Global middleware
    app.Use(loggingMiddleware)

    // Create mux
    mux := app.NewServeMux()

    // Routes
    mux.HandleFunc("GET /", func(w app.ResponseWriter, r *app.Request) {
        err := w.HTML("index", nil)
        if err != nil {
            w.Error(http.StatusInternalServerError, err.Error())
        }
    })

    // JSON endpoint with JSONP support
    mux.HandleFunc("GET /users", func(w app.ResponseWriter, r *app.Request) {
        users := []User{
            {ID: uuid.New(), Name: "John Doe", Email: "john@example.com"},
            {ID: uuid.New(), Name: "Jane Smith", Email: "jane@example.com"},
        }
        w.JSON(users)
    }).WithAPIConfig(&app.APIConfig{
        OperationID: "listUsers",
        Summary:     "List all users",
        Tags:        []string{"Users"},
        Responses: map[string]app.Response{
            "200": {
                Description: "List of users",
                Content: map[string]app.TypeInfo{
                    "application/json": {TypeHint: &[]User{}},
                },
            },
        },
    })

    // Create user with JSON
    mux.HandleFunc("POST /api/users/json", func(w app.ResponseWriter, r *app.Request) {
        user, valErrors, err := app.BindJSON[User](r, true)
        
        if err != nil {
            w.Error(http.StatusBadRequest, err.Error())
            return
        }
        
        if valErrors.Any() {
            w.WriteHeader(http.StatusBadRequest).JSON(valErrors)
            return
        }

        user.ID = uuid.New()
        w.WriteHeader(http.StatusCreated).JSON(user)
    }).WithAPIConfig(&app.APIConfig{
        OperationID: "createUser",
        Summary:     "Create a new user",
        Tags:        []string{"Users"},
        RequestBody: &app.RequestBody{
            Required: true,
            Content: map[string]app.TypeInfo{
                "application/json": {TypeHint: &User{}},
            },
        },
        Responses: map[string]app.Response{
            "201": {
                Description: "User created",
                Content: map[string]app.TypeInfo{
                    "application/json": {TypeHint: &User{}},
                },
            },
            "400": {Description: "Validation error"},
        },
    })

    // Update user with JSON Patch
    mux.HandleFunc("PATCH /users/{id}", func(w app.ResponseWriter, r *app.Request) {
        id := r.PathValue("id")
        
        // Fetch existing user
        user := User{
            ID:    uuid.MustParse(id),
            Name:  "John Doe",
            Email: "john@example.com",
            Role:  "user",
        }
        
        // Apply JSON Patch with validation
        valErrors, err := app.PatchJSON(r, &user, true)
        if err != nil {
            if err == app.ErrMethodNotAllowed {
                w.Error(http.StatusMethodNotAllowed, err.Error())
                return
            }
            w.Error(http.StatusBadRequest, err.Error())
            return
        }
        
        if len(valErrors) > 0 {
            w.WriteHeader(http.StatusBadRequest).JSON(app.ValidationErrors{Errors: valErrors})
            return
        }
        
        w.JSON(user)
    })

    // SSE endpoint for real-time updates
    mux.Handle("GET /events", app.SSE(
        func() app.SSEPayload {
            return app.SSEPayload{
                Id:       uuid.New().String(),
                Event:    "TIME_UPDATE",
                Comments: []string{"Server time update"},
                Data:     fmt.Sprintf("Current server time: %s", time.Now().Format(time.RFC3339)),
            }
        },
        func() {
            log.Println("Client disconnected from events stream")
        },
        func(err error) {
            log.Printf("SSE error: %v\n", err)
        },
        5*time.Second,
        nil,
    ))

    // i18n example
    mux.HandleFunc("GET /greeting", func(w app.ResponseWriter, r *app.Request) {
        printer := app.GetI18nPrinter(language.Spanish)
        msg := printer.Sprintf("Welcome to %s! Clap %d times.", "WebFram", 5)
        w.JSON(map[string]string{"message": msg})
    })

    // Start server
    log.Println("Server starting on :8080")
    log.Println("OpenAPI docs: http://localhost:8080/openapi.json")
    app.ListenAndServe(":8080", mux)
}

func getOpenAPIConfig() *openapi.Config {
    return &openapi.Config{
        Info: &openapi.Info{
            Title:       "WebFram Example API",
            Summary:     "An example API demonstrating WebFram features.",
            Description: "This is an example API documentation generated by WebFram.",
            Version:     "1.0.0",
        },
        Servers: []openapi.Server{
            {
                URL:         "http://localhost:8080",
                Description: "Local development server",
            },
        },
    }
}
```

## License

This project is licensed under the MIT License.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

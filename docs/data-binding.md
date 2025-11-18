---
layout: default
title: Data Binding & Validation
nav_order: 7
description: "Form, JSON, XML binding with comprehensive validation"
---

# Data Binding & Validation

WebFram provides type-safe data binding with comprehensive validation for Form, JSON, and XML formats.

## Overview

Data binding converts HTTP request data into strongly-typed Go structs with automatic validation. WebFram supports:

- **Form binding** - URL-encoded and multipart forms
- **JSON binding** - JSON request bodies
- **XML binding** - XML request bodies
- **Unified binding** - Bind from multiple sources simultaneously

## Form Binding

Form binding automatically parses form data and validates it:

```go
type CreateUserRequest struct {
    Name      string    `form:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"`
    Email     string    `form:"email" validate:"required,format=email" errmsg:"required=Email is required;format=Invalid email address"`
    Age       int       `form:"age" validate:"min=18,max=120"`
    Role      string    `form:"role" validate:"enum=admin|user|guest"`
    Birthdate time.Time `form:"birthdate" validate:"required" format:"2006-01-02"`
    Hobbies   []string  `form:"hobbies" validate:"minItems=1,maxItems=5,uniqueItems"`
}

mux.HandleFunc("POST /users", func(w app.ResponseWriter, r *app.Request) {
    user, valErrors, err := app.BindForm[CreateUserRequest](r)
    
    if err != nil {
        w.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    if valErrors.Any() {
        w.WriteHeader(http.StatusBadRequest)
        w.JSON(r.Context(), valErrors)
        return
    }
    
    w.JSON(r.Context(), user)
})
```

**Form data:**

```text
name=John+Doe&email=john@example.com&age=30&role=admin&hobbies=reading&hobbies=coding
```

## JSON Binding

Parse JSON request bodies with optional validation:

```go
type CreateUserRequest struct {
    Name    string   `json:"name" validate:"required,minlength=3"`
    Email   string   `json:"email" validate:"required,format=email"`
    Hobbies []string `json:"hobbies" validate:"minItems=1,maxItems=5,uniqueItems"`
    Age     int      `json:"age" validate:"min=18,max=120"`
}

mux.HandleFunc("POST /api/users", func(w app.ResponseWriter, r *app.Request) {
    // Second parameter: true = validate, false = skip validation
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
    
    w.JSON(r.Context(), user)
})
```

**JSON request:**

```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "hobbies": ["reading", "coding"],
  "age": 30
}
```

## XML Binding

Parse XML request bodies with validation:

```go
type CreateUserRequest struct {
    XMLName xml.Name  `xml:"user"`
    Name    string    `xml:"name" validate:"required,minlength=3"`
    Email   string    `xml:"email" validate:"required,format=email"`
    Age     int       `xml:"age" validate:"min=18,max=120"`
    Role    string    `xml:"role" validate:"enum=admin|user|guest"`
}

mux.HandleFunc("POST /api/users", func(w app.ResponseWriter, r *app.Request) {
    user, valErrors, err := app.BindXML[CreateUserRequest](r, true)
    
    if err != nil {
        w.Error(http.StatusBadRequest, err.Error())
        return
    }
    
    if valErrors.Any() {
        w.WriteHeader(http.StatusBadRequest)
        w.XML(valErrors)
        return
    }
    
    w.XML(user)
})
```

**XML request:**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<user>
    <name>John Doe</name>
    <email>john@example.com</email>
    <age>30</age>
    <role>admin</role>
</user>
```

## Validation Tags

WebFram supports 20+ validation tags:

| Tag | Applies To | Description | Example |
|-----|------------|-------------|---------|
| `required` | All types | Field must be present and non-empty | `validate:"required"` |
| `equals=VALUE` | string, int, uint, float | Value must exactly equal specified value | `validate:"equals=active"` |
| `min=N` | int, uint, float | Minimum value (inclusive) | `validate:"min=18"` |
| `max=N` | int, uint, float | Maximum value (inclusive) | `validate:"max=120"` |
| `multipleOf=N` | int, float | Value must be multiple of N | `validate:"multipleOf=5"` |
| `minlength=N` | string | Minimum length in characters | `validate:"minlength=3"` |
| `maxlength=N` | string | Maximum length in characters | `validate:"maxlength=50"` |
| `minItems=N` | slice, map | Minimum number of items | `validate:"minItems=1"` |
| `maxItems=N` | slice, map | Maximum number of items | `validate:"maxItems=10"` |
| `uniqueItems` | slice | All items must be unique | `validate:"uniqueItems"` |
| `emptyItemsAllowed` | slice | Allow empty items in slice | `validate:"emptyItemsAllowed"` |
| `regexp=PATTERN` | string | Must match regular expression | `validate:"regexp=^\\w+@\\w+\\.com$"` |
| `pattern=PATTERN` | string | Alias for regexp | `validate:"pattern=^[A-Z]{3}-\\d{4}$"` |
| `enum=val1\|val2` | string, int, float | Must be one of specified values | `validate:"enum=admin\|user\|guest"` |
| `format=email` | string | Must be valid email (IDN supported) | `validate:"format=email"` |
| `format=url` | string | Must be valid HTTP/HTTPS URL | `validate:"format=url"` |
| `format=LAYOUT` | time.Time | Time parsing layout | `format:"2006-01-02"` |

**Combine multiple rules:**

```go
type Product struct {
    Name  string `json:"name" validate:"required,minlength=2,maxlength=100"`
    SKU   string `json:"sku" validate:"required,regexp=^[A-Z]{3}-\\d{4}$"`
    Price int    `json:"price" validate:"required,min=0,max=1000000,multipleOf=100"`
    Tags  []string `json:"tags" validate:"minItems=1,maxItems=20,uniqueItems"`
}
```

### Equals Validation

The `equals` rule validates that a field value exactly matches a specified value:

```go
type Configuration struct {
    Environment string  `json:"environment" validate:"equals=production"`
    Version     string  `json:"version" validate:"equals=2.0.0"`
    Port        int     `json:"port" validate:"equals=8080"`
    Threshold   float64 `json:"threshold" validate:"equals=0.95"`
}
```

**Use cases:**

- Enforce specific configuration values
- Validate expected constants
- Check for exact status codes or states

**Examples:**

```go
// String equality
Status string `validate:"equals=active"` // Must be exactly "active"

// Integer equality  
Count int `validate:"equals=42"` // Must be exactly 42

// Float equality
Rate float64 `validate:"equals=1.5"` // Must be exactly 1.5

// Combined with other rules
State string `validate:"required,equals=confirmed"` // Required and must equal "confirmed"
```

### URL Format Validation

The `format=url` rule validates that a string is a valid HTTP or HTTPS URL:

```go
type WebResource struct {
    Website    string `json:"website" validate:"format=url"`
    APIBaseURL string `json:"api_base_url" validate:"required,format=url"`
    Callback   string `json:"callback" validate:"format=url"`
}
```

**Valid URLs:**

- `http://example.com`
- `https://example.com`
- `https://example.com/path/to/resource`
- `https://api.example.com:8080/v1/users?page=1`
- `http://subdomain.example.com#section`

**Invalid URLs:**

- `example.com` (missing protocol)
- `ftp://example.com` (only http/https allowed)
- `http://exa mple.com` (contains spaces)
- `not a url` (malformed)

**Examples:**

```go
type APIConfig struct {
    // Simple URL validation
    Endpoint string `json:"endpoint" validate:"format=url"`
    
    // Required URL
    CallbackURL string `json:"callback_url" validate:"required,format=url"`
    
    // Optional URL
    AvatarURL string `json:"avatar_url,omitempty" validate:"format=url"`
    
    // With custom error message
    RedirectURL string `json:"redirect_url" validate:"format=url" errmsg:"format=Please provide a valid HTTP/HTTPS URL"`
}
```

**Pattern used:** `^https?://[^\s/$.?#].[^\s]*$`

This pattern ensures:

- URL starts with `http://` or `https://`
- Contains valid domain/host
- No spaces in the URL
- Supports paths, query parameters, and fragments

## Custom Error Messages

Use `errmsg` tag for custom validation error messages:

```go
type User struct {
    Name  string `json:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"`
    Email string `json:"email" validate:"required,format=email" errmsg:"required=Email is required;format=Please provide a valid email address"`
    Age   int    `json:"age" validate:"min=18,max=120" errmsg:"min=Must be at least 18;max=Must be at most 120"`
}
```

**Format:** `errmsg:"rule1=Message1;rule2=Message2"`

## Validation Errors

Structured validation error type:

```go
type ValidationErrors struct {
    Errors []ValidationError `json:"errors" xml:"errors"`
}

type ValidationError struct {
    Field string `json:"field" xml:"field"`
    Error string `json:"error" xml:"error"`
}
```

**Check for errors:**

```go
if valErrors.Any() {
    w.WriteHeader(http.StatusBadRequest)
    w.JSON(r.Context(), valErrors)
    return
}
```

**Error response (JSON):**

```json
{
  "errors": [
    {"field": "name", "error": "Name is required"},
    {"field": "email", "error": "Invalid email address"},
    {"field": "age", "error": "Must be at least 18"}
  ]
}
```

## Nested Structs

All binding types support nested structs:

```go
type Address struct {
    Street string `json:"street" form:"street" validate:"required"`
    City   string `json:"city" form:"city" validate:"required"`
    Zip    int    `json:"zip" form:"zip" validate:"min=10000,max=99999"`
}

type User struct {
    Name    string  `json:"name" form:"name" validate:"required"`
    Address Address `json:"address" form:"address" validate:"required"`
}

// Form: name=John&address.street=123+Main&address.city=NYC&address.zip=10001
// JSON: {"name": "John", "address": {"street": "123 Main", "city": "NYC", "zip": 10001}}
```

## Map Binding (Form Only)

Form binding supports maps:

```go
type Config struct {
    Metadata map[string]string `form:"metadata" validate:"minItems=1,maxItems=10"`
    Scores   map[string]int    `form:"scores"`
    Settings map[int]string    `form:"settings"`
}

// Form data: metadata[color]=red&metadata[size]=large&scores[math]=95
```

**Supported map types:**

- `map[string]string`
- `map[string]int`
- `map[int]string`
- `map[string]time.Time`
- `map[string]uuid.UUID`

## Unified Bind Method

Bind from multiple sources simultaneously:

```go
type UserRequest struct {
    ID      string `form:"id" bindFrom:"path"`
    Query   string `form:"q" bindFrom:"query"`
    Token   string `form:"Authorization" bindFrom:"header"`
    Session string `form:"session_id" bindFrom:"cookie"`
    Name    string `form:"name" bindFrom:"form"`
}

mux.HandleFunc("POST /users/{id}", func(w app.ResponseWriter, r *app.Request) {
    req, valErrors, err := bind.Bind[UserRequest](r, true)
    // Each field bound from its specified source
})
```

**Binding sources:**

- `path` - URL path parameters
- `query` - Query parameters
- `header` - HTTP headers
- `cookie` - HTTP cookies
- `form` - Form data
- `body` - Request body (JSON/XML)
- `auto` - Use precedence rules

## Supported Types

- **Primitives**: `string`, `int`, `int8`-`int64`, `uint`, `uint8`-`uint64`, `float32`, `float64`, `bool`
- **Time**: `time.Time`
- **UUID**: `uuid.UUID` (from `github.com/google/uuid`)
- **Slices**: `[]string`, `[]int`, `[]time.Time`, etc.
- **Maps** (form only): `map[string]string`, `map[string]int`, etc.
- **Nested structs**: Any struct type
- **Pointers**: All types support pointer variants

## Skip Validation

Skip validation for trusted data:

```go
// Skip validation
user, valErrors, err := app.BindJSON[User](r, false)

// With validation
user, valErrors, err := app.BindJSON[User](r, true)
```

**Note:** Form binding always validates.

## See Also

- [Request & Response](request-response)
- [JSON Patch](json-patch)
- [Routing](routing)

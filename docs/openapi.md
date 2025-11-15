---
layout: default
title: OpenAPI Documentation
nav_order: 10
description: "Automatic API documentation generation"
---

WebFram automatically generates OpenAPI 3.2.0 documentation from your route definitions, validation tags, and API configurations.

## Enabling OpenAPI

Configure OpenAPI in your application:

```go
app.Configure(&app.Config{
    OpenAPI: &app.OpenAPI{
        Enabled: true,
        URLPath: "GET /openapi.json", // Optional, defaults to GET /openapi.json
        Config:  getOpenAPIConfig(),
    },
})

func getOpenAPIConfig() *app.OpenAPIConfig {
    return &app.OpenAPIConfig{
        Info: &app.Info{
            Title:          "My API",
            Summary:        "API for my awesome application",
            Description:    "This API provides endpoints for managing users and products.",
            TermsOfService: "https://example.com/terms/",
            Contact: &app.Contact{
                Name:  "API Support",
                URL:   "https://example.com/support",
                Email: "support@example.com",
            },
            License: &app.License{
                Name:       "MIT",
                Identifier: "MIT",
                URL:        "https://opensource.org/licenses/MIT",
            },
            Version: "1.0.0",
        },
        Servers: []app.Server{
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
        Tags: []app.Tag{
            {
                Name:        "Users",
                Summary:     "User management",
                Description: "Operations for managing users",
            },
        },
    }
}
```

Access your OpenAPI spec at: `http://localhost:8080/openapi.json`

## Built-in OpenAPI UI

WebFram automatically generates an interactive API documentation UI using [Scalar](https://github.com/scalar/scalar). When you enable OpenAPI, an HTML page is automatically created alongside your JSON spec.

For example, if your OpenAPI spec is at `/openapi.json`, the UI will be available at `/openapi.html`:

```bash
# View the JSON specification
curl http://localhost:8080/openapi.json

# Open the interactive UI in your browser
open http://localhost:8080/openapi.html
```

The UI provides:

- **Interactive API testing** - Try out API endpoints directly from the browser
- **Request/response examples** - See sample requests and responses
- **Schema visualization** - Browse data models and their properties
- **Authentication support** - Test authenticated endpoints
- **Dark/light mode** - Comfortable viewing in any environment

### Custom URL Paths

The UI automatically adapts to your custom OpenAPI paths:

```go
app.Configure(&app.Config{
    OpenAPI: &app.OpenAPI{
        Enabled: true,
        URLPath: "GET /api/v1/docs.json",
        Config:  getOpenAPIConfig(),
    },
})
```

With the above configuration:

- JSON spec: `http://localhost:8080/api/v1/docs.json`
- Interactive UI: `http://localhost:8080/api/v1/docs.html`

## Documenting Routes

Use `WithAPIConfig()` to add OpenAPI documentation:

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
                "application/json": {TypeHint: &User{}},
            },
        },
        "400": {Description: "Invalid request data"},
        "500": {Description: "Internal server error"},
    },
})
```

## Path-Level Configuration

Configure documentation for entire paths:

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

## Schema Generation

WebFram automatically generates JSON schemas from struct tags:

```go
type User struct {
    Name    string   `json:"name" validate:"required,minlength=3,maxlength=50"`
    Email   string   `json:"email" validate:"required,format=email"`
    Age     int      `json:"age" validate:"min=18,max=120"`
    Role    string   `json:"role" validate:"enum=admin|user|guest"`
    Hobbies []string `json:"hobbies" validate:"minItems=1,maxItems=10,uniqueItems"`
}
```

Generates OpenAPI schema with:

- Required fields
- String length constraints (minLength, maxLength)
- Numeric constraints (minimum, maximum)
- Enum values
- Array constraints (minItems, maxItems, uniqueItems)
- Format specifications (email, uuid, date-time)

## Complete Example

```go
type User struct {
    ID    uuid.UUID `json:"id"`
    Name  string    `json:"name" validate:"required,minlength=3"`
    Email string    `json:"email" validate:"required,format=email"`
    Role  string    `json:"role" validate:"enum=admin|user|guest"`
}

// List users
mux.HandleFunc("GET /users", listUsers).WithAPIConfig(&app.APIConfig{
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

// Get single user
mux.HandleFunc("GET /users/{id}", getUser).WithAPIConfig(&app.APIConfig{
    OperationID: "getUser",
    Summary:     "Get user by ID",
    Tags:        []string{"Users"},
    Parameters: []app.Parameter{
        {
            Name:        "id",
            In:          "path",
            Description: "User ID",
            Required:    true,
            Example:     "550e8400-e29b-41d4-a716-446655440000",
        },
    },
    Responses: map[string]app.Response{
        "200": {
            Description: "User details",
            Content: map[string]app.TypeInfo{
                "application/json": {TypeHint: &User{}},
            },
        },
        "404": {Description: "User not found"},
    },
})

// Create user
mux.HandleFunc("POST /users", createUser).WithAPIConfig(&app.APIConfig{
    OperationID: "createUser",
    Summary:     "Create a new user",
    Tags:        []string{"Users"},
    RequestBody: &app.RequestBody{
        Description: "User data",
        Required:    true,
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
        "400": {Description: "Invalid input"},
    },
})
```

## Viewing Documentation

WebFram provides a built-in interactive UI at the `.html` endpoint (e.g., `http://localhost:8080/openapi.html`).

You can also access the raw JSON specification:

```bash
curl http://localhost:8080/openapi.json
```

### Alternative Visualization Tools

If you prefer other API documentation tools, you can use them with the JSON spec:

#### Swagger UI

```html
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

#### Redoc

```html
<!DOCTYPE html>
<html>
<body>
    <redoc spec-url="http://localhost:8080/openapi.json"></redoc>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>
```

## Best Practices

1. **Use meaningful operation IDs** - Helps with code generation
2. **Provide examples** - Makes API easier to understand
3. **Document all responses** - Include error cases
4. **Use tags** - Organize endpoints logically
5. **Version your API** - Include version in info
6. **Add descriptions** - Explain complex endpoints
7. **Security schemes** - Document authentication requirements

## See Also

- [Data Binding](data-binding.html)
- [Routing](routing.html)
- [Configuration](configuration.html)

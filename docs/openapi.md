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

Use `WithOperationConfig()` to add OpenAPI documentation:

{% raw %}

```go
mux.HandleFunc("POST /users", createUserHandler).WithOperationConfig(&app.OperationConfig{
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

{% endraw %}

## Path-Level Configuration

Configure documentation for entire paths:

{% raw %}

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

{% endraw %}

## Schema Generation

WebFram automatically generates JSON and XML schemas from struct tags:

### JSON Schema Generation

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

### XML Schema Generation

For XML content types, WebFram generates XML-aware schemas with proper XML metadata:

```go
type User struct {
    ID    uuid.UUID `xml:"id,attr" validate:"required"`
    Name  string    `xml:"name" validate:"required,minlength=3"`
    Email string    `xml:"email,attr" validate:"required,format=email"`
    Age   int       `xml:"age" validate:"min=18,max=120"`
}
```

XML schemas include:

- XML element and attribute metadata
- Proper XML namespace and prefix support
- Automatic example generation with mock data
- Slice examples wrapped with xmlRootName for valid XML structure

See the [XML Schema Generation documentation](xml-schema-generation) for complete details.

## TypeHint Usage for Streaming Media Types

When documenting endpoints that produce streaming media types, the `TypeHint` behavior varies:

### Server-Sent Events (text/event-stream)

For SSE endpoints, **do not set a TypeHint**. The framework automatically uses the `SSEPayload` type:

```go
// ✅ Correct - no TypeHint needed
mux.Handle("GET /events", app.SSE(...)).WithOperationConfig(&app.OperationConfig{
    OperationID: "streamEvents",
    Summary:     "Stream server events",
    Responses: map[string]app.Response{
        "200": {
            Description: "Server-sent events stream",
            Content: map[string]app.TypeInfo{
                "text/event-stream": {}, // SSEPayload is automatically used
            },
        },
    },
})

// ❌ Incorrect - TypeHint is ignored for SSE
mux.Handle("GET /events", app.SSE(...)).WithOperationConfig(&app.OperationConfig{
    OperationID: "streamEvents",
    Summary:     "Stream server events",
    Responses: map[string]app.Response{
        "200": {
            Description: "Server-sent events stream",
            Content: map[string]app.TypeInfo{
                "text/event-stream": {TypeHint: &MyType{}}, // Will be ignored
            },
        },
    },
})
```

The `SSEPayload` structure is automatically documented with its `Data` field accepting `any` type.

### JSON Sequence (application/json-seq)

For JSON Sequence (RFC 7464) endpoints, **set the TypeHint to the line item type**:

```go
type Notification struct {
    ID      string    `json:"id"`
    Message string    `json:"message"`
    Time    time.Time `json:"time"`
}

// ✅ Correct - TypeHint points to line item type
mux.HandleFunc("GET /notifications", streamNotifications).WithOperationConfig(&app.OperationConfig{
    OperationID: "streamNotifications",
    Summary:     "Stream notifications",
    Responses: map[string]app.Response{
        "200": {
            Description: "Stream of notifications",
            Content: map[string]app.TypeInfo{
                "application/json-seq": {TypeHint: &Notification{}}, // Each line is a Notification
            },
        },
    },
})
```

The TypeHint specifies the structure of each record in the sequence.

### XML Streaming (application/xml, text/xml)

For XML streaming endpoints, **set the TypeHint to the struct or slice type**:

#### Single Struct (application/xml)

```go
type User struct {
    ID    uuid.UUID `xml:"id,attr"`
    Name  string    `xml:"name"`
    Email string    `xml:"email"`
}

// ✅ Correct - TypeHint points to struct type
mux.HandleFunc("GET /user/stream", streamUser).WithOperationConfig(&app.OperationConfig{
    OperationID: "streamUser",
    Summary:     "Stream user data as XML",
    Responses: map[string]app.Response{
        "200": {
            Description: "XML stream of user data",
            Content: map[string]app.TypeInfo{
                "application/xml": {TypeHint: &User{}}, // Describes the XML structure
            },
        },
    },
})
```

#### Slice/Array (application/xml)

```go
type User struct {
    ID    uuid.UUID `xml:"id,attr"`
    Name  string    `xml:"name"`
    Email string    `xml:"email"`
}

// ✅ Correct - TypeHint points to slice type for XML arrays
mux.HandleFunc("GET /users/stream", streamUsers).WithOperationConfig(&app.OperationConfig{
    OperationID: "streamUsers",
    Summary:     "Stream users as XML array",
    Responses: map[string]app.Response{
        "200": {
            Description: "XML stream of user array",
            Content: map[string]app.TypeInfo{
                "application/xml": {TypeHint: []User{}}, // Describes array of users
            },
        },
    },
})
```

#### Text/XML Variant

For `text/xml` media type, the TypeHint behavior is identical to `application/xml`:

```go
// ✅ Correct - text/xml uses same TypeHint rules as application/xml
mux.HandleFunc("GET /data.xml", getXMLData).WithOperationConfig(&app.OperationConfig{
    OperationID: "getXMLData",
    Summary:     "Get data as text/xml",
    Responses: map[string]app.Response{
        "200": {
            Description: "XML data in text format",
            Content: map[string]app.TypeInfo{
                "text/xml": {TypeHint: &User{}}, // Same as application/xml
            },
        },
    },
})
```

### Summary

| Media Type               | TypeHint Behavior                                        |
|--------------------------|----------------------------------------------------------|
| `text/event-stream`      | Don't set - automatically uses `SSEPayload`              |
| `application/json-seq`   | Set to line item type - describes each record in stream |
| `application/json`       | Set to response type - describes the entire response    |
| `application/xml`        | Set to struct/slice type - describes XML structure      |
| `text/xml`               | Set to struct/slice type - describes XML structure      |

See the [SSE documentation](sse.md) for more details on Server-Sent Events.

## Complete Example

{% raw %}

```go
type User struct {
    ID    uuid.UUID `json:"id" xml:"id,attr"`
    Name  string    `json:"name" xml:"name" validate:"required,minlength=3"`
    Email string    `json:"email" xml:"email,attr" validate:"required,format=email"`
    Role  string    `json:"role" xml:"role" validate:"enum=admin|user|guest"`
}

// List users (JSON response)
mux.HandleFunc("GET /users", listUsers).WithOperationConfig(&app.OperationConfig{
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

// List users (XML response)
mux.HandleFunc("GET /users.xml", listUsersXML).WithOperationConfig(&app.OperationConfig{
    OperationID: "listUsersXML",
    Summary:     "List all users (XML)",
    Tags:        []string{"Users"},
    Responses: map[string]app.Response{
        "200": {
            Description: "List of users in XML format",
            Content: map[string]app.TypeInfo{
                "application/xml": {TypeHint: &[]User{}},
            },
        },
    },
})

// Get single user
mux.HandleFunc("GET /users/{id}", getUser).WithOperationConfig(&app.OperationConfig{
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
mux.HandleFunc("POST /users", createUser).WithOperationConfig(&app.OperationConfig{
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

{% endraw %}

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

## Security Requirements

WebFram supports security requirements at both the global API level and individual operation level.

### Global Security Requirements

Define default security requirements that apply to all operations:

```go
func getOpenAPIConfig() *app.OpenAPIConfig {
    return &app.OpenAPIConfig{
        Info: &app.Info{
            Title:   "Secure API",
            Version: "1.0.0",
        },
        // Global security - applies to all operations by default
        Security: []map[string][]string{
            {"BearerAuth": {}}, // All operations require Bearer token
        },
        Components: &app.Components{
            SecuritySchemes: map[string]app.SecurityScheme{
                "BearerAuth": app.NewHTTPBearerSecurityScheme(&app.HTTPBearerSecuritySchemeOptions{
                    Description: "JWT Bearer Token",
                }),
            },
        },
    }
}
```

### Operation-Level Security

Override global security requirements for specific operations:

```go
// Public endpoint - no authentication required
mux.HandleFunc("GET /health", healthCheck).WithOperationConfig(&app.OperationConfig{
    OperationID: "healthCheck",
    Summary:     "Health check endpoint",
    Security:    []map[string][]string{}, // Empty array = no auth required
})

// Endpoint requiring multiple auth options (OR)
mux.HandleFunc("GET /users", listUsers).WithOperationConfig(&app.OperationConfig{
    OperationID: "listUsers",
    Summary:     "List users",
    Security: []map[string][]string{
        {"BearerAuth": {}},      // Accept Bearer token OR
        {"ApiKeyAuth": {}},       // Accept API key
    },
})

// Endpoint requiring specific OAuth scopes
mux.HandleFunc("DELETE /users/{id}", deleteUser).WithOperationConfig(&app.OperationConfig{
    OperationID: "deleteUser",
    Summary:     "Delete user",
    Security: []map[string][]string{
        {"OAuth2Auth": {"users:write", "users:delete"}}, // Requires specific scopes
    },
})

// Use global security (omit Security field or set to nil)
mux.HandleFunc("GET /profile", getProfile).WithOperationConfig(&app.OperationConfig{
    OperationID: "getProfile",
    Summary:     "Get user profile",
    // No Security field = uses global security
})
```

#### Security Requirement Behavior

- **`nil` (omitted)**: Operation uses global security requirements
- **Empty array `[]`**: No authentication required (public endpoint)
- **Multiple requirements**: Client can satisfy ANY of the requirements (OR logic)
- **Scopes in requirement**: Client must have ALL specified scopes (AND logic)

## Security Schemes

WebFram supports all OpenAPI 3.2.0 security scheme types. Define security schemes in your configuration, then reference them in global or operation-level security requirements.

### Configuring Security Schemes

Add security schemes to your OpenAPI configuration:

```go
func getOpenAPIConfig() *app.OpenAPIConfig {
    return &app.OpenAPIConfig{
        Info: &app.Info{
            Title:   "Secure API",
            Version: "1.0.0",
        },
        Components: &app.Components{
            SecuritySchemes: map[string]app.SecurityScheme{
                "BasicAuth": app.NewHTTPBasicSecurityScheme(&app.HTTPBasicSecuritySchemeOptions{
                    Description: "HTTP Basic Authentication",
                }),
                "BearerAuth": app.NewHTTPBearerSecurityScheme(&app.HTTPBearerSecuritySchemeOptions{
                    Description:  "JWT Bearer Token",
                    BearerFormat: "JWT",
                }),
                "ApiKeyAuth": app.NewAPIKeySecurityScheme(&app.APIKeySecuritySchemeOptions{
                    Name:        "X-API-Key",
                    In:          "header",
                    Description: "API Key in header",
                }),
                "OAuth2Auth": app.NewOAuth2SecurityScheme(&app.OAuth2SecuritySchemeOptions{
                    Description: "OAuth 2.0 Authentication",
                    Flows: []app.OAuthFlow{
                        app.NewAuthorizationCodeOAuthFlow(&app.AuthorizationCodeOAuthFlowOptions{
                            AuthorizationURL: "https://example.com/oauth/authorize",
                            TokenURL:         "https://example.com/oauth/token",
                            Scopes: map[string]string{
                                "read":  "Read access",
                                "write": "Write access",
                            },
                        }),
                    },
                }),
            },
        },
    }
}
```

### HTTP Authentication Schemes

#### Basic Authentication

```go
"BasicAuth": app.NewHTTPBasicSecurityScheme(&app.HTTPBasicSecuritySchemeOptions{
    Description: "HTTP Basic Authentication using username and password",
})
```

#### Digest Authentication

```go
"DigestAuth": app.NewHTTPDigestSecurityScheme(&app.HTTPDigestSecuritySchemeOptions{
    Description: "HTTP Digest Authentication",
})
```

#### Bearer Token (JWT)

```go
"BearerAuth": app.NewHTTPBearerSecurityScheme(&app.HTTPBearerSecuritySchemeOptions{
    Description:  "JWT Bearer Token Authentication",
    BearerFormat: "JWT",
    Extensions: map[string]interface{}{
        "x-example": "Bearer eyJhbGciOiJIUzI1NiIs...",
    },
})
```

### API Key Authentication

API keys can be sent in headers, query parameters, or cookies:

```go
// Header-based API Key
"ApiKeyAuth": app.NewAPIKeySecurityScheme(&app.APIKeySecuritySchemeOptions{
    Name:        "X-API-Key",
    In:          "header",
    Description: "API Key in custom header",
})

// Query parameter API Key
"ApiKeyQuery": app.NewAPIKeySecurityScheme(&app.APIKeySecuritySchemeOptions{
    Name:        "api_key",
    In:          "query",
    Description: "API Key in query parameter",
})

// Cookie-based API Key
"ApiKeyCookie": app.NewAPIKeySecurityScheme(&app.APIKeySecuritySchemeOptions{
    Name:        "session",
    In:          "cookie",
    Description: "Session cookie",
})
```

### OAuth 2.0 Flows

WebFram supports all OAuth 2.0 flows:

#### Authorization Code Flow

```go
app.NewOAuth2SecurityScheme(&app.OAuth2SecuritySchemeOptions{
    Description: "OAuth 2.0 Authorization Code Flow",
    Flows: []app.OAuthFlow{
        app.NewAuthorizationCodeOAuthFlow(&app.AuthorizationCodeOAuthFlowOptions{
            AuthorizationURL: "https://example.com/oauth/authorize",
            TokenURL:         "https://example.com/oauth/token",
            RefreshURL:       "https://example.com/oauth/refresh",
            Scopes: map[string]string{
                "read":  "Read access to protected resources",
                "write": "Write access to protected resources",
                "admin": "Admin access",
            },
        }),
    },
})
```

#### Implicit Flow

```go
app.NewImplicitOAuthFlow(&app.ImplicitOAuthFlowOptions{
    AuthorizationURL: "https://example.com/oauth/authorize",
    Scopes: map[string]string{
        "public": "Public access",
    },
})
```

#### Client Credentials Flow

```go
app.NewClientCredentialsOAuthFlow(&app.ClientCredentialsOAuthFlowOptions{
    TokenURL: "https://example.com/oauth/token",
    Scopes: map[string]string{
        "machine": "Machine-to-machine access",
    },
})
```

#### Device Authorization Flow

```go
app.NewDeviceAuthorizationOAuthFlow(&app.DeviceAuthorizationOAuthFlowOptions{
    DeviceAuthorizationURL: "https://example.com/oauth/device_authorize",
    TokenURL:               "https://example.com/oauth/token",
    Scopes: map[string]string{
        "device": "Device access",
    },
})
```

#### Multiple OAuth Flows

You can configure multiple OAuth flows for the same security scheme:

```go
"OAuth2Auth": app.NewOAuth2SecurityScheme(&app.OAuth2SecuritySchemeOptions{
    Description: "OAuth 2.0 with multiple flows",
    Flows: []app.OAuthFlow{
        app.NewAuthorizationCodeOAuthFlow(&app.AuthorizationCodeOAuthFlowOptions{
            AuthorizationURL: "https://example.com/oauth/authorize",
            TokenURL:         "https://example.com/oauth/token",
            Scopes: map[string]string{
                "read":  "Read access",
                "write": "Write access",
            },
        }),
        app.NewClientCredentialsOAuthFlow(&app.ClientCredentialsOAuthFlowOptions{
            TokenURL: "https://example.com/oauth/token",
            Scopes: map[string]string{
                "admin": "Admin access",
            },
        }),
        app.NewImplicitOAuthFlow(&app.ImplicitOAuthFlowOptions{
            AuthorizationURL: "https://example.com/oauth/authorize",
            Scopes: map[string]string{
                "public": "Public access",
            },
        }),
    },
})
```

### OpenID Connect

```go
"OpenIDConnect": app.NewOpenIdConnectSecurityScheme(&app.OpenIdConnectSecuritySchemeOptions{
    OpenIdConnectURL: "https://example.com/.well-known/openid-configuration",
    Description:      "OpenID Connect Authentication",
})
```

### Mutual TLS (mTLS)

```go
"MutualTLS": app.NewMutualTLSSecurityScheme(&app.MutualTLSSecuritySchemeOptions{
    Description: "Mutual TLS client certificate authentication",
})
```

### Deprecated Security Schemes

Mark security schemes as deprecated when transitioning to newer auth methods:

```go
"OldApiKey": app.NewAPIKeySecurityScheme(&app.APIKeySecuritySchemeOptions{
    Name:        "X-Old-API-Key",
    In:          "header",
    Description: "Deprecated API Key (use BearerAuth instead)",
    Deprecated:  true,
})
```

### Complete Security Example

```go
app.Configure(&app.Config{
    OpenAPI: &app.OpenAPI{
        Enabled: true,
        Config: &app.OpenAPIConfig{
            Info: &app.Info{
                Title:   "Secure API",
                Version: "2.0.0",
            },
            Components: &app.Components{
                SecuritySchemes: map[string]app.SecurityScheme{
                    "BasicAuth": app.NewHTTPBasicSecurityScheme(&app.HTTPBasicSecuritySchemeOptions{
                        Description: "Basic Authentication for development",
                    }),
                    "BearerAuth": app.NewHTTPBearerSecurityScheme(&app.HTTPBearerSecuritySchemeOptions{
                        Description:  "JWT Bearer Token (production)",
                        BearerFormat: "JWT",
                    }),
                    "ApiKeyAuth": app.NewAPIKeySecurityScheme(&app.APIKeySecuritySchemeOptions{
                        Name:        "X-API-Key",
                        In:          "header",
                        Description: "API Key for external integrations",
                    }),
                    "OAuth2Auth": app.NewOAuth2SecurityScheme(&app.OAuth2SecuritySchemeOptions{
                        Description: "OAuth 2.0 for third-party applications",
                        Flows: []app.OAuthFlow{
                            app.NewAuthorizationCodeOAuthFlow(&app.AuthorizationCodeOAuthFlowOptions{
                                AuthorizationURL: "https://api.example.com/oauth/authorize",
                                TokenURL:         "https://api.example.com/oauth/token",
                                Scopes: map[string]string{
                                    "users:read":  "Read user information",
                                    "users:write": "Modify user information",
                                },
                            }),
                        },
                    }),
                    "OpenIDConnect": app.NewOpenIdConnectSecurityScheme(&app.OpenIdConnectSecuritySchemeOptions{
                        OpenIdConnectURL: "https://accounts.example.com/.well-known/openid-configuration",
                        Description:      "OpenID Connect for SSO",
                    }),
                },
            },
            Tags: []app.Tag{
                {Name: "Users", Description: "User management operations"},
                {Name: "Admin", Description: "Administrative operations"},
            },
        },
    },
})
```

## Best Practices

1. **Use meaningful operation IDs** - Helps with code generation
2. **Provide examples** - Makes API easier to understand
3. **Document all responses** - Include error cases
4. **Use tags** - Organize endpoints logically
5. **Version your API** - Include version in info
6. **Add descriptions** - Explain complex endpoints
7. **Security schemes** - Document authentication requirements and choose appropriate security schemes
8. **OAuth scopes** - Clearly describe what each scope allows
9. **Deprecation** - Mark old security schemes as deprecated when migrating

## See Also

- [Data Binding](data-binding.md)
- [Routing](routing.md)
- [Configuration](configuration.md)

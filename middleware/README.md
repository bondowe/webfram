# WebFram Security Middlewares

This package provides configurable middleware for implementing various OpenAPI v3.2.0 security schemes in WebFram applications.

## Available Middlewares

### HTTP Basic Authentication

Enforces HTTP Basic Authentication as defined in RFC 7617.

```go
config := middleware.BasicAuthConfig{
    Authenticator: func(username, password string) bool {
        // Validate credentials
        return username == "admin" && password == "secret"
    },
    Realm: "MyApp",
}

mux.Use(middleware.BasicAuth(config))
```

### HTTP Digest Authentication

Enforces HTTP Digest Authentication as defined in RFC 7616.

```go
config := middleware.DigestAuthConfig{
    Realm: "MyApp",
    PasswordGetter: func(username, realm string) (password string, ok bool) {
        // Return password for user
        if username == "admin" {
            return "secret", true
        }
        return "", false
    },
    NonceTTL: 30 * time.Minute,
}

mux.Use(middleware.DigestAuth(config))
```

### HTTP Bearer Token Authentication

Enforces Bearer Token Authentication (commonly used for JWT).

```go
config := middleware.BearerAuthConfig{
    Authenticator: func(token string) bool {
        // Validate JWT token
        return validateJWT(token)
    },
}

mux.Use(middleware.BearerAuth(config))
```

### API Key Authentication

Supports API keys in headers, query parameters, or cookies.

```go
config := middleware.APIKeyAuthConfig{
    KeyName: "X-API-Key",
    In:      "header", // "header", "query", or "cookie"
    Authenticator: func(key string) bool {
        // Validate API key
        return key == "valid-key"
    },
}

mux.Use(middleware.APIKeyAuth(config))
```

### OAuth 2.0 Authentication

Validates OAuth 2.0 access tokens.

```go
config := middleware.OAuth2AuthConfig{
    TokenValidator: func(token string) bool {
        // Validate OAuth2 token
        return validateOAuth2Token(token)
    },
    Scopes: []string{"read", "write"},
}

mux.Use(middleware.OAuth2Auth(config))
```

### OpenID Connect Authentication

Validates OpenID Connect ID tokens.

```go
config := middleware.OpenIDConnectAuthConfig{
    TokenValidator: func(token string) bool {
        // Validate OIDC token
        return validateOIDCToken(token)
    },
}

mux.Use(middleware.OpenIDConnectAuth(config))
```

### Mutual TLS Authentication

Enforces client certificate authentication.

```go
config := middleware.MutualTLSAuthConfig{
    CertificateValidator: func(cert *x509.Certificate) bool {
        // Validate client certificate
        return cert.Subject.CommonName == "valid-client"
    },
}

mux.Use(middleware.MutualTLSAuth(config))
```

## Configuration Options

All middlewares support:

- **Authenticator/Validator functions**: Custom logic for credential validation
- **UnauthorizedHandler**: Optional custom handler for failed authentication
- **Scheme-specific options**: Realm, key names, TTL, etc.

## Usage with WebFram

```go
package main

import (
    "github.com/bondowe/webfram"
    "github.com/bondowe/webfram/middleware"
)

func main() {
    app := webfram.NewServeMux()

    // Apply authentication middleware
    app.Use(middleware.BasicAuth(middleware.BasicAuthConfig{
        Authenticator: authenticateUser,
        Realm: "MyApp",
    }))

    // Protected routes
    app.HandleFunc("GET /api/users", listUsers)
    app.HandleFunc("POST /api/users", createUser)

    webfram.ListenAndServe(":8080", app, nil)
}

func authenticateUser(username, password string) bool {
    // Implement authentication logic
    return true // or false
}
```

## Security Considerations

- Always use HTTPS in production
- Implement proper password hashing for basic auth
- Validate tokens securely for bearer/OAuth2/OIDC
- Regularly rotate API keys and nonces
- Use strong certificate validation for mutual TLS

## OpenAPI Integration

These middlewares work seamlessly with WebFram's OpenAPI documentation generation. Configure security requirements in your operation configs:

```go
mux.HandleFunc("GET /users", listUsers).WithOperationConfig(&webfram.OperationConfig{
    Security: []map[string][]string{
        {"BasicAuth": {}},
    },
})
```
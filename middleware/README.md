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

The OAuth2 middlewares support all major OAuth2 flows as defined in RFC 6749.

#### Authorization Code Flow

For web applications that can securely store client secrets.

```go
config := middleware.OAuth2AuthorizationCodeConfig{
    OAuth2BaseConfig: middleware.OAuth2BaseConfig{
        ClientID:     "your-client-id",
        TokenURL:     "https://auth.example.com/oauth/token",
        Scopes:       []string{"read", "write"},
        TokenValidator: func(token string) bool {
            return validateOAuth2Token(token)
        },
        UnauthorizedHandler: nil,
        RefreshBuffer: 5 * time.Minute, // Refresh tokens 5 minutes before expiry
    },
    ClientSecret: "your-client-secret",
    AuthorizationURL: "https://auth.example.com/oauth/authorize",
    RedirectURL:      "https://yourapp.com/oauth/callback",
    StateStore: func(state string) (redirectURL string, ok bool) {
        // Store/retrieve state for CSRF protection
        return "/", true
    },
    TokenStore: func(sessionID string) (*middleware.OAuth2Token, bool) {
        // Store/retrieve tokens by session
        return nil, false
    },
    SessionIDExtractor: func(r *http.Request) string {
        // Extract session ID from request
        return "session-id"
    },
    // Optional: Enable PKCE for enhanced security (recommended for public clients)
    PKCE: &middleware.OAuth2PKCEConfig{
        CodeVerifierStore: func(state string) (codeVerifier string, ok bool) {
            // Store/retrieve PKCE code verifier by state
            return getStoredVerifier(state)
        },
        ChallengeMethod: middleware.PKCES256, // or middleware.PKCEPlain
    },
}

mux.Use(middleware.OAuth2AuthorizationCodeAuth(config))
```

**PKCE (Proof Key for Code Exchange)**: When enabled, PKCE protects against authorization code interception attacks by requiring a code verifier during token exchange. Use `PKCES256` for production (recommended) or `PKCEPlain` for compatibility.

**Automatic Token Refresh**: Tokens are automatically refreshed when they expire or are close to expiring (based on `RefreshBuffer`). The middleware handles refresh token storage and validation.

#### Implicit Flow

For single-page applications (SPAs) where tokens are returned directly.

```go
config := middleware.OAuth2ImplicitConfig{
    OAuth2BaseConfig: middleware.OAuth2BaseConfig{
        ClientID:     "your-client-id",
        TokenURL:     "https://auth.example.com/oauth/token",
        Scopes:       []string{"read"},
        TokenValidator: func(token string) bool {
            return validateOAuth2Token(token)
        },
        UnauthorizedHandler: nil,
        RefreshBuffer: 5 * time.Minute, // Refresh tokens 5 minutes before expiry
    },
    AuthorizationURL: "https://auth.example.com/oauth/authorize",
    RedirectURL:      "https://yourapp.com/oauth/callback",
    StateStore: func(state string) (redirectURL string, ok bool) {
        // Store/retrieve state for CSRF protection
        return "/", true
    },
}

mux.Use(middleware.OAuth2ImplicitAuth(config))
```

#### Device Authorization Grant Flow

For devices without browsers or keyboards (IoT, smart TVs, etc.).

```go
config := middleware.OAuth2DeviceConfig{
    OAuth2BaseConfig: middleware.OAuth2BaseConfig{
        ClientID:     "your-client-id",
        TokenURL:     "https://auth.example.com/oauth/token",
        Scopes:       []string{"read"},
        TokenValidator: func(token string) bool {
            return validateOAuth2Token(token)
        },
        UnauthorizedHandler: nil,
        RefreshBuffer: 5 * time.Minute, // Refresh tokens 5 minutes before expiry
    },
}

mux.Use(middleware.OAuth2DeviceAuth(config))
```

Device Flow Usage:

1. Device makes POST request with `?request_device_code=true` to get user code
2. User visits verification URI and enters the code  
3. Device polls with `?device_code=...` until authorization completes

#### Client Credentials Flow

For service-to-service authentication.

```go
config := middleware.OAuth2ClientCredentialsConfig{
    OAuth2BaseConfig: middleware.OAuth2BaseConfig{
        ClientID:     "your-client-id",
        TokenURL:     "https://auth.example.com/oauth/token",
        Scopes:       []string{"read"},
        TokenValidator: func(token string) bool {
            return validateOAuth2Token(token)
        },
        UnauthorizedHandler: nil,
        RefreshBuffer: 5 * time.Minute, // Refresh tokens 5 minutes before expiry
    },
    ClientSecret: "your-client-secret",
}

mux.Use(middleware.OAuth2ClientCredentialsAuth(config))
```

#### OAuth2 Configuration Options

All OAuth2 flows share common configuration through `OAuth2BaseConfig`:

- **`RefreshBuffer`**: Time buffer before token expiration to trigger automatic refresh (default: 5 minutes). Tokens are refreshed when they expire or are within this buffer period.
- **`TokenValidator`**: Function to validate **access tokens** (opaque tokens for API authorization)
- **`UnauthorizedHandler`**: Custom handler for authentication failures
- **`Scopes`**: Requested OAuth2 scopes
- **`ClientID`**: OAuth2 client identifier

**Token Validator Types:**

- **OAuth2 TokenValidator**: Validates access tokens (used for API authorization)
- **OpenID Connect TokenValidator**: Validates ID tokens (JWTs containing user identity information)

**Automatic Token Refresh**: When `TokenStore` and `SessionIDExtractor` are configured, tokens are automatically refreshed using refresh tokens before they expire, providing seamless authentication for users.

#### Token Validation with Scope Checking

For advanced token validation that checks OAuth2 scopes, you can create scope-aware validators using closures:

```go
// Scope-aware validator using token introspection
func createScopeValidator(requiredScopes []string, tokenIntrospector func(string) (map[string]interface{}, error)) func(string) bool {
    return func(token string) bool {
        claims, err := tokenIntrospector(token)
        if err != nil {
            return false
        }
        
        tokenScopes, ok := claims["scope"].([]interface{})
        if !ok {
            return false
        }
        
        var scopes []string
        for _, scope := range tokenScopes {
            if s, ok := scope.(string); ok {
                scopes = append(scopes, s)
            }
        }
        
        return hasAllScopes(scopes, requiredScopes)
    }
}

func hasAllScopes(tokenScopes, requiredScopes []string) bool {
    scopeMap := make(map[string]bool)
    for _, scope := range tokenScopes {
        scopeMap[scope] = true
    }
    
    for _, required := range requiredScopes {
        if !scopeMap[required] {
            return false
        }
    }
    return true
}

// Usage with Authorization Code flow
config := middleware.OAuth2AuthorizationCodeConfig{
    OAuth2BaseConfig: middleware.OAuth2BaseConfig{
        ClientID:     "your-client-id",
        TokenURL:     "https://auth.example.com/oauth/token",
        Scopes:       []string{"read", "write", "profile"},
        TokenValidator: createScopeValidator(
            []string{"read"}, // Require at least 'read' scope
            func(token string) (map[string]interface{}, error) {
                // Your token introspection logic
                return introspectToken(token)
            },
        ),
        RefreshBuffer: 5 * time.Minute,
    },
    ClientSecret: "your-client-secret",
    AuthorizationURL: "https://auth.example.com/oauth/authorize",
    RedirectURL:      "https://yourapp.com/oauth/callback",
}

mux.Use(middleware.OAuth2AuthorizationCodeAuth(config))
```

#### JWT-Based Scope Validation

```go
import "github.com/golang-jwt/jwt/v5"

func createJWTValidator(requiredScopes []string, jwtSecret []byte) func(string) bool {
    return func(tokenString string) bool {
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return jwtSecret, nil
        })
        
        if err != nil || !token.Valid {
            return false
        }
        
        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            return false
        }
        
        var tokenScopes []string
        if scopeInterface, exists := claims["scope"]; exists {
            switch scopes := scopeInterface.(type) {
            case []interface{}:
                for _, scope := range scopes {
                    if s, ok := scope.(string); ok {
                        tokenScopes = append(tokenScopes, s)
                    }
                }
            case string:
                tokenScopes = strings.Split(scopes, " ")
            }
        }
        
        return hasAllScopes(tokenScopes, requiredScopes)
    }
}
```

#### Route-Specific Scope Validation

For granular access control, combine OAuth2 middleware with route-specific scope checking:

```go
// RequireAllScopes requires ALL specified scopes (AND logic)
func RequireAllScopes(requiredScopes ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token, ok := r.Context().Value(middleware.OAuth2TokenKey{}).(*middleware.OAuth2Token)
            if !ok {
                http.Error(w, "No OAuth2 token in context", http.StatusUnauthorized)
                return
            }
            
            tokenScopes := strings.Split(token.Scope, " ")
            if !hasAllScopes(tokenScopes, requiredScopes) {
                http.Error(w, "Insufficient scopes", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// RequireAnyScopes requires ANY of the specified scopes (OR logic)
func RequireAnyScopes(requiredScopes ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token, ok := r.Context().Value(middleware.OAuth2TokenKey{}).(*middleware.OAuth2Token)
            if !ok {
                http.Error(w, "No OAuth2 token in context", http.StatusUnauthorized)
                return
            }
            
            tokenScopes := strings.Split(token.Scope, " ")
            if !hasAnyScopes(tokenScopes, requiredScopes) {
                http.Error(w, "Insufficient scopes", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

// Usage with OAuth2 middleware
mux.Use(middleware.OAuth2AuthorizationCodeAuth(oauthConfig))

// Require ALL scopes (user must have both "read" AND "users")
readUsers := middleware.RequireAllScopes("read", "users")(http.HandlerFunc(listUsers))

// Require ANY scope (user must have either "write" OR "admin")
writeUsers := middleware.RequireAnyScopes("write", "admin")(http.HandlerFunc(createUser))

// Require specific admin scope
adminPanel := middleware.RequireAllScopes("admin")(http.HandlerFunc(adminDashboard))

mux.HandleFunc("GET /users", readUsers.ServeHTTP)
mux.HandleFunc("POST /users", writeUsers.ServeHTTP)
mux.HandleFunc("GET /admin", adminPanel.ServeHTTP)
```

**Scope Logic:**

- `RequireAllScopes("read", "users")`: Token must have BOTH "read" AND "users" scopes
- `RequireAnyScopes("write", "admin")`: Token must have EITHER "write" OR "admin" scope (or both)

#### Token Validation Only

For validating Bearer tokens when OAuth2 flow is handled elsewhere.

```go
config := middleware.OAuth2TokenConfig{
    TokenValidator: func(token string) bool {
        return validateOAuth2Token(token)
    },
    UnauthorizedHandler: nil,
}

mux.Use(middleware.OAuth2TokenAuth(config))
```

### OpenID Connect Authentication

Validates OpenID Connect **ID tokens** (JWTs containing user identity information like user ID, email, and profile data). Supports both simple token validation and full authentication flow with redirects.

#### Simple Token Validation

For validating existing ID tokens (when authentication is handled elsewhere):

```go
config := middleware.OpenIDConnectAuthConfig{
    TokenValidator: func(token string) bool {
        // Validate OIDC ID token (JWT)
        return validateOIDCToken(token)
    },
}

mux.Use(middleware.OpenIDConnectAuth(config))
```

#### Full Authentication Flow

For web applications that need to redirect users to authenticate:

```go
config := middleware.OpenIDConnectAuthConfig{
    IssuerURL:       "https://accounts.google.com",
    ClientID:        "your-client-id",
    ClientSecret:    "your-client-secret", 
    RedirectURL:     "https://yourapp.com/oidc/callback",
    Scopes:          []string{"openid", "profile", "email"},
    TokenValidator: func(token string) bool {
        return validateOIDCToken(token)
    },
    StateStore: func(state string) (redirectURL string, ok bool) {
        // Store/retrieve state for CSRF protection
        return "/", true
    },
}

mux.Use(middleware.OpenIDConnectAuth(config))
```

The middleware automatically detects which mode to use based on whether redirect fields are configured.

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

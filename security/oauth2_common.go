package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuth2FlowType represents the different OAuth2 flow types.
type OAuth2FlowType string

const (
	OAuth2FlowAuthorizationCode OAuth2FlowType = "authorization_code"
	OAuth2FlowImplicit          OAuth2FlowType = "implicit"
	OAuth2FlowDevice            OAuth2FlowType = "device_code"
	OAuth2FlowClientCredentials OAuth2FlowType = "client_credentials"
)

// OAuth2TokenKey is the context key for OAuth2 tokens.
type OAuth2TokenKey struct{}

// OAuth2Token represents an OAuth2 token response with expiration tracking.
type OAuth2Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	IssuedAt     time.Time `json:"issued_at,omitempty"`  // When token was issued
	ExpiresAt    time.Time `json:"expires_at,omitempty"` // When token expires
}

// IsExpired checks if the access token is expired with a buffer.
func (t *OAuth2Token) IsExpired(buffer time.Duration) bool {
	if t.ExpiresAt.IsZero() {
		return false // No expiration info, assume valid
	}
	return time.Now().Add(buffer).After(t.ExpiresAt)
}

// NeedsRefresh checks if the token should be refreshed (expired or close to expiring).
func (t *OAuth2Token) NeedsRefresh(buffer time.Duration) bool {
	return t.IsExpired(buffer) && t.RefreshToken != ""
}

// OAuth2DeviceCode represents a device authorization response.
type OAuth2DeviceCode struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval,omitempty"`
}

// OAuth2BaseConfig holds common OAuth2 configuration fields.
type OAuth2BaseConfig struct {
	// ClientID is the OAuth2 client identifier
	ClientID string
	// TokenURL is the OAuth2 token endpoint
	TokenURL string
	// Scopes are the requested OAuth2 scopes
	Scopes []string
	// TokenValidator validates access tokens
	TokenValidator func(token string) bool
	// UnauthorizedHandler is called when authentication fails
	UnauthorizedHandler http.Handler
	// RefreshBuffer is the time buffer before expiration to trigger refresh (default: 5 minutes)
	RefreshBuffer time.Duration
}

// OAuth2Config holds common OAuth2 configuration (deprecated - use flow-specific configs).
type OAuth2Config struct {
	// ClientID is the OAuth2 client identifier
	ClientID string
	// ClientSecret is the OAuth2 client secret
	ClientSecret string
	// AuthorizationURL is the OAuth2 authorization endpoint
	AuthorizationURL string
	// TokenURL is the OAuth2 token endpoint
	TokenURL string
	// RedirectURL is the OAuth2 redirect URI
	RedirectURL string
	// Scopes are the requested OAuth2 scopes
	Scopes []string
	// TokenValidator validates access tokens
	TokenValidator func(token string) bool
	// StateStore stores/retrieves OAuth2 state parameters
	StateStore func(state string) (redirectURL string, ok bool)
	// TokenStore stores/retrieves OAuth2 tokens
	TokenStore func(sessionID string) (*OAuth2Token, bool)
	// SessionIDExtractor extracts session ID from request
	SessionIDExtractor func(r *http.Request) string
	// UnauthorizedHandler is called when authentication fails
	UnauthorizedHandler http.Handler
	// RefreshBuffer is the time buffer before expiration to trigger refresh (default: 5 minutes)
	RefreshBuffer time.Duration
}

// Shared helper functions

// refreshOAuth2Token refreshes an OAuth2 token using the refresh token.
func refreshOAuth2Token(config OAuth2BaseConfig, clientID, clientSecret, refreshToken string) (*OAuth2Token, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}

	// Add client_secret if provided (for confidential clients)
	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		config.TokenURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &url.Error{Op: "POST", URL: config.TokenURL, Err: http.ErrNotSupported}
	}

	var token OAuth2Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	// Set expiration time
	now := time.Now()
	token.IssuedAt = now
	if token.ExpiresIn > 0 {
		token.ExpiresAt = now.Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	return &token, nil
}

// validateAndRefreshToken validates a token and refreshes it if needed.
func validateAndRefreshToken(r *http.Request, config OAuth2BaseConfig, clientID, clientSecret string, tokenStore func(sessionID string) (*OAuth2Token, bool), sessionIDExtractor func(r *http.Request) string) (*OAuth2Token, error) {
	sessionID := sessionIDExtractor(r)
	token, ok := tokenStore(sessionID)
	if !ok {
		return nil, http.ErrNoCookie // No token stored
	}

	// Check if token needs refresh
	buffer := config.RefreshBuffer
	if buffer == 0 {
		buffer = 5 * time.Minute // Default 5 minutes
	}

	if token.NeedsRefresh(buffer) {
		// Try to refresh the token
		newToken, err := refreshOAuth2Token(config, clientID, clientSecret, token.RefreshToken)
		if err != nil {
			return nil, err
		}
		// Update stored token
		tokenStore(sessionID) // This would need to be a setter function in real implementation
		token = newToken
	}

	return token, nil
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func generateState() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func unauthorizedOAuth2(w http.ResponseWriter, handler http.Handler) {
	if handler != nil {
		handler.ServeHTTP(w, nil)
		return
	}

	w.Header().Set("WWW-Authenticate", `Bearer`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("Unauthorized"))
}

// RequireAllScopes returns middleware that requires ALL of the specified scopes.
// The token must have every scope in the requiredScopes slice.
func RequireAllScopes(requiredScopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := r.Context().Value(OAuth2TokenKey{}).(*OAuth2Token)
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

// RequireAnyScopes returns middleware that requires ANY of the specified scopes.
// The token must have at least one scope from the requiredScopes slice.
func RequireAnyScopes(requiredScopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := r.Context().Value(OAuth2TokenKey{}).(*OAuth2Token)
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

// hasAllScopes checks if tokenScopes contains ALL requiredScopes
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

// hasAnyScopes checks if tokenScopes contains ANY of the requiredScopes
func hasAnyScopes(tokenScopes, requiredScopes []string) bool {
	scopeMap := make(map[string]bool)
	for _, scope := range tokenScopes {
		scopeMap[scope] = true
	}

	for _, required := range requiredScopes {
		if scopeMap[required] {
			return true
		}
	}
	return false
}

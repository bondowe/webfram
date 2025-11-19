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

// OIDCToken represents an OpenID Connect token response with expiration tracking.
type OIDCToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	IDToken      string    `json:"id_token"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	IssuedAt     time.Time `json:"issued_at,omitempty"`  // When token was issued
	ExpiresAt    time.Time `json:"expires_at,omitempty"` // When token expires
}

// IsExpired checks if the access token is expired with a buffer.
func (t *OIDCToken) IsExpired(buffer time.Duration) bool {
	if t.ExpiresAt.IsZero() {
		return false // No expiration info, assume valid
	}
	return time.Now().Add(buffer).After(t.ExpiresAt)
}

// NeedsRefresh checks if the token should be refreshed (expired or close to expiring).
func (t *OIDCToken) NeedsRefresh(buffer time.Duration) bool {
	return t.IsExpired(buffer) && t.RefreshToken != ""
}

// OIDCTokenKey is the context key for OIDC tokens.
type OIDCTokenKey struct{}

// OpenIDConnectAuthConfig holds configuration for OpenID Connect authentication middleware.
type OpenIDConnectAuthConfig struct {
	// TokenValidator validates ID tokens (for simple token validation)
	TokenValidator func(token string) bool
	// UnauthorizedHandler is called when authentication fails (optional)
	UnauthorizedHandler http.Handler

	// Fields for full OIDC flow (optional - if provided, enables redirect)
	IssuerURL    string   // OIDC provider issuer URL
	ClientID     string   // OIDC client ID
	ClientSecret string   // OIDC client secret
	RedirectURL  string   // Callback URL
	Scopes       []string // Requested scopes (default: ["openid"])

	// State management (required for redirect flow)
	StateStore func(state string) (redirectURL string, ok bool)
	// Token storage (optional)
	TokenStore         func(sessionID string) (*OIDCToken, bool)
	SessionIDExtractor func(r *http.Request) string
	// RefreshBuffer is the time buffer before expiration to trigger refresh (default: 5 minutes)
	RefreshBuffer time.Duration
}

// OpenIDConnectAuth returns a middleware that enforces OpenID Connect Authentication.
// If redirect fields are configured, it will redirect users to authenticate.
// Otherwise, it validates existing Bearer tokens.
func OpenIDConnectAuth(config OpenIDConnectAuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If redirect flow is configured, handle full OIDC flow
			if config.IssuerURL != "" && config.ClientID != "" {
				handleOIDCFlow(w, r, config, next)
				return
			}

			// Otherwise, just validate existing Bearer tokens
			auth := r.Header.Get("Authorization")
			if auth == "" {
				unauthorizedOIDC(w, config.UnauthorizedHandler)
				return
			}

			if !strings.HasPrefix(auth, "Bearer ") {
				unauthorizedOIDC(w, config.UnauthorizedHandler)
				return
			}

			token := strings.TrimPrefix(auth, "Bearer ")
			if !config.TokenValidator(token) {
				unauthorizedOIDC(w, config.UnauthorizedHandler)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// handleOIDCFlow handles the full OpenID Connect authentication flow with redirects.
func handleOIDCFlow(w http.ResponseWriter, r *http.Request, config OpenIDConnectAuthConfig, next http.Handler) {
	// Check if this is a callback from the OIDC provider
	if r.URL.Query().Get("code") != "" && r.URL.Query().Get("state") != "" {
		handleOIDCCallback(w, r, config, next)
		return
	}

	// Check for valid ID token in header
	if token := extractOIDCBearerToken(r); token != "" && config.TokenValidator(token) {
		next.ServeHTTP(w, r)
		return
	}

	// Try to get and refresh token from store
	if config.TokenStore != nil && config.SessionIDExtractor != nil {
		sessionID := config.SessionIDExtractor(r)
		if storedToken, ok := config.TokenStore(sessionID); ok {
			buffer := config.RefreshBuffer
			if buffer == 0 {
				buffer = 5 * time.Minute
			}

			// Check if token is still valid, doesn't need refresh, and validates
			if !storedToken.IsExpired(0) && !storedToken.NeedsRefresh(buffer) && config.TokenValidator(storedToken.IDToken) {
				ctx := context.WithValue(r.Context(), OIDCTokenKey{}, storedToken)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Try to refresh if needed and possible
			if storedToken.NeedsRefresh(buffer) && storedToken.RefreshToken != "" {
				if newToken, err := refreshOIDCToken(config, storedToken.RefreshToken); err == nil && config.TokenValidator(newToken.IDToken) {
					ctx := context.WithValue(r.Context(), OIDCTokenKey{}, newToken)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}
	}

	// Redirect to OIDC provider
	redirectToOIDCProvider(w, r, config)
}

// handleOIDCCallback handles the callback from the OIDC provider.
func handleOIDCCallback(w http.ResponseWriter, r *http.Request, config OpenIDConnectAuthConfig, next http.Handler) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	// Verify state
	if config.StateStore != nil {
		if _, ok := config.StateStore(state); !ok {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}
	}

	// Exchange code for tokens
	token, err := exchangeOIDCCodeForTokens(config, code)
	if err != nil {
		unauthorizedOIDC(w, config.UnauthorizedHandler)
		return
	}

	// Store token
	if config.TokenStore != nil && config.SessionIDExtractor != nil {
		sessionID := config.SessionIDExtractor(r)
		config.TokenStore(sessionID) // In real implementation, store the token
	}

	// Add token to context and proceed
	ctx := context.WithValue(r.Context(), OIDCTokenKey{}, token)
	next.ServeHTTP(w, r.WithContext(ctx))
}

// redirectToOIDCProvider redirects the user to the OIDC provider for authentication.
func redirectToOIDCProvider(w http.ResponseWriter, r *http.Request, config OpenIDConnectAuthConfig) {
	state := generateOIDCState()

	// Store state and original URL
	if config.StateStore != nil {
		config.StateStore(state) // In real implementation, store with TTL
	}

	// Build authorization URL
	authURL, _ := url.Parse(config.IssuerURL + "/authorize")
	q := authURL.Query()
	q.Set("response_type", "code")
	q.Set("client_id", config.ClientID)
	q.Set("redirect_uri", config.RedirectURL)
	q.Set("scope", strings.Join(getOIDCScopes(config.Scopes), " "))
	q.Set("state", state)
	authURL.RawQuery = q.Encode()

	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

// exchangeOIDCCodeForTokens exchanges the authorization code for OIDC tokens.
func exchangeOIDCCodeForTokens(config OpenIDConnectAuthConfig, code string) (*OIDCToken, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {config.RedirectURL},
		"client_id":     {config.ClientID},
		"client_secret": {config.ClientSecret},
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		config.IssuerURL+"/token",
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

	var token OIDCToken
	if decodeErr := json.NewDecoder(resp.Body).Decode(&token); decodeErr != nil {
		return nil, decodeErr
	}

	// Set expiration time
	token.IssuedAt = time.Now()
	if token.ExpiresIn > 0 {
		token.ExpiresAt = token.IssuedAt.Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	return &token, nil
}

// refreshOIDCToken refreshes an OIDC token using the refresh token.
func refreshOIDCToken(config OpenIDConnectAuthConfig, refreshToken string) (*OIDCToken, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {config.ClientID},
		"client_secret": {config.ClientSecret},
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		config.IssuerURL+"/token",
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
		return nil, &url.Error{Op: "POST", URL: config.IssuerURL + "/token", Err: http.ErrNotSupported}
	}

	var token OIDCToken
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

// Helper functions

// extractOIDCBearerToken extracts Bearer token from Authorization header.
func extractOIDCBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func generateOIDCState() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func getOIDCScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return []string{"openid"}
	}
	// Ensure "openid" is included
	hasOpenID := false
	for _, scope := range scopes {
		if scope == "openid" {
			hasOpenID = true
			break
		}
	}
	if !hasOpenID {
		return append(scopes, "openid")
	}
	return scopes
}

func unauthorizedOIDC(w http.ResponseWriter, handler http.Handler) {
	if handler != nil {
		handler.ServeHTTP(w, nil)
		return
	}

	w.Header().Set("WWW-Authenticate", `Bearer`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("Unauthorized"))
}

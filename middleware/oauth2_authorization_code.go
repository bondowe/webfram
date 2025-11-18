package middleware

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PKCE challenge methods
type PKCEChallengeMethod string

const (
	PKCES256  PKCEChallengeMethod = "S256"
	PKCEPlain PKCEChallengeMethod = "plain"
)

// OAuth2AuthorizationCodeConfig holds configuration for Authorization Code flow.
type OAuth2AuthorizationCodeConfig struct {
	OAuth2BaseConfig
	// ClientSecret is the OAuth2 client secret
	ClientSecret string
	// AuthorizationURL is the OAuth2 authorization endpoint
	AuthorizationURL string
	// RedirectURL is the OAuth2 redirect URI
	RedirectURL string
	// StateStore stores/retrieves OAuth2 state parameters
	StateStore func(state string) (redirectURL string, ok bool)
	// TokenStore stores/retrieves OAuth2 tokens
	TokenStore func(sessionID string) (*OAuth2Token, bool)
	// SessionIDExtractor extracts session ID from request
	SessionIDExtractor func(r *http.Request) string
	// PKCE configuration (optional)
	PKCE *OAuth2PKCEConfig
}

// OAuth2PKCEConfig holds PKCE (Proof Key for Code Exchange) configuration.
type OAuth2PKCEConfig struct {
	// CodeVerifierStore stores/retrieves PKCE code verifiers by state
	CodeVerifierStore func(state string) (codeVerifier string, ok bool)
	// ChallengeMethod specifies the PKCE challenge method ("S256" or "plain")
	ChallengeMethod PKCEChallengeMethod
}

// OAuth2AuthorizationCodeAuth returns middleware for OAuth2 Authorization Code flow.
func OAuth2AuthorizationCodeAuth(config OAuth2AuthorizationCodeConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if this is a callback from the authorization server
			if r.URL.Query().Get("code") != "" && r.URL.Query().Get("state") != "" {
				handleAuthorizationCodeCallback(w, r, config, next)
				return
			}

			// Check for valid access token in header
			if token := extractBearerToken(r); token != "" && config.TokenValidator(token) {
				next.ServeHTTP(w, r)
				return
			}

			// Try to get and refresh token from store
			if config.TokenStore != nil && config.SessionIDExtractor != nil {
				if token, err := validateAndRefreshToken(r, config.OAuth2BaseConfig, config.ClientID, config.ClientSecret, config.TokenStore, config.SessionIDExtractor); err == nil && token != nil {
					if config.TokenValidator(token.AccessToken) {
						ctx := context.WithValue(r.Context(), OAuth2TokenKey{}, token)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}

			// Redirect to authorization server
			redirectToAuthorizationServer(w, r, config)
		})
	}
}

// Helper functions for Authorization Code flow

func redirectToAuthorizationServer(w http.ResponseWriter, r *http.Request, config OAuth2AuthorizationCodeConfig) {
	state := generateState()

	// Store state and original URL
	if config.StateStore != nil {
		config.StateStore(state) // In real implementation, store with TTL
	}

	authURL, _ := url.Parse(config.AuthorizationURL)
	q := authURL.Query()
	q.Set("response_type", "code")
	q.Set("client_id", config.ClientID)
	q.Set("redirect_uri", config.RedirectURL)
	q.Set("scope", strings.Join(config.Scopes, " "))
	q.Set("state", state)

	// Add PKCE parameters if configured
	if config.PKCE != nil {
		codeVerifier := generateCodeVerifier()
		codeChallenge := generateCodeChallenge(codeVerifier, config.PKCE.ChallengeMethod)

		// Store code verifier with state
		if config.PKCE.CodeVerifierStore != nil {
			// In real implementation, this would store the codeVerifier with the state
			// For now, we'll just call it to indicate storage
			config.PKCE.CodeVerifierStore(state)
		}

		q.Set("code_challenge", codeChallenge)
		q.Set("code_challenge_method", getChallengeMethod(config.PKCE.ChallengeMethod))
	}

	authURL.RawQuery = q.Encode()

	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

// PKCE helper functions

func generateCodeVerifier() string {
	// Generate a random 32-byte string
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(err) // In production, handle error properly
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}

func generateCodeChallenge(verifier string, method PKCEChallengeMethod) string {
	switch method {
	case PKCES256:
		hash := sha256.Sum256([]byte(verifier))
		return base64.RawURLEncoding.EncodeToString(hash[:])
	case PKCEPlain:
		return verifier
	default:
		return verifier // Default to plain
	}
}

func getChallengeMethod(method PKCEChallengeMethod) string {
	switch method {
	case PKCES256:
		return "S256"
	case PKCEPlain:
		return "plain"
	default:
		return "plain"
	}
}

func handleAuthorizationCodeCallback(
	w http.ResponseWriter,
	r *http.Request,
	config OAuth2AuthorizationCodeConfig,
	next http.Handler,
) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	// Verify state
	if config.StateStore != nil {
		if _, ok := config.StateStore(state); !ok {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}
	}

	// Exchange code for token
	token, err := exchangeCodeForToken(config, code, state)
	if err != nil {
		unauthorizedOAuth2(w, config.UnauthorizedHandler)
		return
	}

	// Store token
	if config.TokenStore != nil && config.SessionIDExtractor != nil {
		sessionID := config.SessionIDExtractor(r)
		// In real implementation, this would be a setter function
		// For now, we'll just call it to indicate storage
		config.TokenStore(sessionID)
	}

	// Add token to context and proceed
	ctx := context.WithValue(r.Context(), OAuth2TokenKey{}, token)
	next.ServeHTTP(w, r.WithContext(ctx))
}

func exchangeCodeForToken(config OAuth2AuthorizationCodeConfig, code string, state string) (*OAuth2Token, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {config.RedirectURL},
		"client_id":     {config.ClientID},
		"client_secret": {config.ClientSecret},
	}

	// Add code_verifier if PKCE is configured
	if config.PKCE != nil && config.PKCE.CodeVerifierStore != nil {
		if codeVerifier, ok := config.PKCE.CodeVerifierStore(state); ok {
			data.Set("code_verifier", codeVerifier)
		}
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

	var token OAuth2Token
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

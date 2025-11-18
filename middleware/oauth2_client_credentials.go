package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuth2ClientCredentialsConfig holds configuration for Client Credentials flow.
type OAuth2ClientCredentialsConfig struct {
	OAuth2BaseConfig
	// ClientSecret is the OAuth2 client secret
	ClientSecret string
	// TokenStore stores/retrieves OAuth2 tokens (optional for caching)
	TokenStore func(sessionID string) (*OAuth2Token, bool)
	// SessionIDExtractor extracts session ID from request (optional)
	SessionIDExtractor func(r *http.Request) string
}

// OAuth2ClientCredentialsAuth returns middleware for OAuth2 Client Credentials flow.
func OAuth2ClientCredentialsAuth(config OAuth2ClientCredentialsConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for valid access token first
			if token := extractBearerToken(r); token != "" && config.TokenValidator(token) {
				next.ServeHTTP(w, r)
				return
			}

			// Try to get cached token and refresh if needed
			if config.TokenStore != nil && config.SessionIDExtractor != nil {
				sessionID := config.SessionIDExtractor(r)
				if cachedToken, ok := config.TokenStore(sessionID); ok {
					buffer := config.RefreshBuffer
					if buffer == 0 {
						buffer = 5 * time.Minute
					}

					if !cachedToken.NeedsRefresh(buffer) && config.TokenValidator(cachedToken.AccessToken) {
						ctx := context.WithValue(r.Context(), OAuth2TokenKey{}, cachedToken)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}

					// Try to refresh the token
					if cachedToken.RefreshToken != "" {
						if newToken, err := refreshOAuth2Token(config.OAuth2BaseConfig, config.ClientID, config.ClientSecret, cachedToken.RefreshToken); err == nil {
							ctx := context.WithValue(r.Context(), OAuth2TokenKey{}, newToken)
							next.ServeHTTP(w, r.WithContext(ctx))
							return
						}
					}
				}
			}

			// Exchange client credentials for new token
			token, err := exchangeClientCredentialsForToken(config)
			if err != nil {
				unauthorizedOAuth2(w, config.UnauthorizedHandler)
				return
			}

			// Add token to request context
			ctx := context.WithValue(r.Context(), OAuth2TokenKey{}, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Helper function for Client Credentials flow

func exchangeClientCredentialsForToken(config OAuth2ClientCredentialsConfig) (*OAuth2Token, error) {
	data := url.Values{
		"grant_type": {"client_credentials"},
		"scope":      {strings.Join(config.Scopes, " ")},
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
	req.SetBasicAuth(config.ClientID, config.ClientSecret)

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

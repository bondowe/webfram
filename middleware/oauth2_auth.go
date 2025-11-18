package middleware

import (
	"net/http"
	"strings"
)

// OAuth2TokenConfig holds configuration for simple token validation.
type OAuth2TokenConfig struct {
	// TokenValidator validates access tokens
	TokenValidator func(token string) bool
	// UnauthorizedHandler is called when authentication fails
	UnauthorizedHandler http.Handler
}

// OAuth2TokenAuth returns middleware that validates OAuth2 Bearer tokens.
func OAuth2TokenAuth(config OAuth2TokenConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				unauthorizedOAuth2(w, config.UnauthorizedHandler)
				return
			}

			if !strings.HasPrefix(auth, "Bearer ") {
				unauthorizedOAuth2(w, config.UnauthorizedHandler)
				return
			}

			token := strings.TrimPrefix(auth, "Bearer ")
			if !config.TokenValidator(token) {
				unauthorizedOAuth2(w, config.UnauthorizedHandler)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

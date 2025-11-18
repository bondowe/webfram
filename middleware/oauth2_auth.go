package middleware

import (
	"net/http"
	"strings"

	"github.com/bondowe/webfram"
)

// OAuth2AuthConfig holds configuration for OAuth2 authentication middleware
type OAuth2AuthConfig struct {
	// TokenValidator is called with the access token, should return true if valid
	TokenValidator func(token string) bool
	// Scopes are the required scopes (optional)
	Scopes []string
	// UnauthorizedHandler is called when authentication fails (optional)
	UnauthorizedHandler webfram.Handler
}

// OAuth2Auth returns a middleware that enforces OAuth2 Bearer Token Authentication
func OAuth2Auth(config OAuth2AuthConfig) func(webfram.Handler) webfram.Handler {
	return func(next webfram.Handler) webfram.Handler {
		return webfram.HandlerFunc(func(w webfram.ResponseWriter, r *webfram.Request) {
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

			// TODO: validate scopes if provided

			next.ServeHTTP(w, r)
		})
	}
}

func unauthorizedOAuth2(w webfram.ResponseWriter, handler webfram.Handler) {
	if handler != nil {
		handler.ServeHTTP(w, nil)
		return
	}

	w.Header().Set("WWW-Authenticate", `Bearer`)
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Unauthorized"))
}

package middleware

import (
	"net/http"
	"strings"

	"github.com/bondowe/webfram"
)

// OpenIDConnectAuthConfig holds configuration for OpenID Connect authentication middleware
type OpenIDConnectAuthConfig struct {
	// TokenValidator is called with the ID token, should return true if valid
	TokenValidator func(token string) bool
	// UnauthorizedHandler is called when authentication fails (optional)
	UnauthorizedHandler webfram.Handler
}

// OpenIDConnectAuth returns a middleware that enforces OpenID Connect Authentication
func OpenIDConnectAuth(config OpenIDConnectAuthConfig) func(webfram.Handler) webfram.Handler {
	return func(next webfram.Handler) webfram.Handler {
		return webfram.HandlerFunc(func(w webfram.ResponseWriter, r *webfram.Request) {
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

func unauthorizedOIDC(w webfram.ResponseWriter, handler webfram.Handler) {
	if handler != nil {
		handler.ServeHTTP(w, nil)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Unauthorized"))
}

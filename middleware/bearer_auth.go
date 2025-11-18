package middleware

import (
	"net/http"
	"strings"

	"github.com/bondowe/webfram"
)

// BearerAuthConfig holds configuration for bearer token authentication middleware
type BearerAuthConfig struct {
	// Authenticator is called with the token, should return true if valid
	Authenticator func(token string) bool
	// UnauthorizedHandler is called when authentication fails (optional)
	UnauthorizedHandler webfram.Handler
}

// BearerAuth returns a middleware that enforces HTTP Bearer Token Authentication
func BearerAuth(config BearerAuthConfig) func(webfram.Handler) webfram.Handler {
	return func(next webfram.Handler) webfram.Handler {
		return webfram.HandlerFunc(func(w webfram.ResponseWriter, r *webfram.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				unauthorizedBearer(w, config.UnauthorizedHandler)
				return
			}

			if !strings.HasPrefix(auth, "Bearer ") {
				unauthorizedBearer(w, config.UnauthorizedHandler)
				return
			}

			token := strings.TrimPrefix(auth, "Bearer ")
			if !config.Authenticator(token) {
				unauthorizedBearer(w, config.UnauthorizedHandler)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func unauthorizedBearer(w webfram.ResponseWriter, handler webfram.Handler) {
	if handler != nil {
		handler.ServeHTTP(w, nil)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Unauthorized"))
}

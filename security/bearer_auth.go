package security

import (
	"net/http"
	"strings"
)

// BearerAuthConfig holds configuration for bearer token authentication middleware.
type BearerAuthConfig struct {
	// TokenValidator is called with the bearer token, should return true if valid
	TokenValidator func(token string) bool
	// UnauthorizedHandler is called when authentication fails (optional)
	UnauthorizedHandler http.Handler
}

// BearerAuth returns a middleware that enforces HTTP Bearer Token Authentication.
func BearerAuth(config BearerAuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			if !config.TokenValidator(token) {
				unauthorizedBearer(w, config.UnauthorizedHandler)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func unauthorizedBearer(w http.ResponseWriter, handler http.Handler) {
	if handler != nil {
		handler.ServeHTTP(w, nil)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("Unauthorized"))
}

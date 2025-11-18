package middleware

import (
	"encoding/base64"
	"net/http"
	"strings"
)

// BasicAuthConfig holds configuration for basic authentication middleware.
type BasicAuthConfig struct {
	// Authenticator is called with username and password, should return true if valid
	Authenticator func(username, password string) bool
	// Realm is the authentication realm (default: "Restricted")
	Realm string
	// UnauthorizedHandler is called when authentication fails (optional)
	UnauthorizedHandler http.Handler
}

// BasicAuth returns a middleware that enforces HTTP Basic Authentication.
func BasicAuth(config BasicAuthConfig) func(http.Handler) http.Handler {
	if config.Realm == "" {
		config.Realm = "Restricted"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				unauthorized(w, config.Realm, config.UnauthorizedHandler)
				return
			}

			if !strings.HasPrefix(auth, "Basic ") {
				unauthorized(w, config.Realm, config.UnauthorizedHandler)
				return
			}

			encoded := strings.TrimPrefix(auth, "Basic ")
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				unauthorized(w, config.Realm, config.UnauthorizedHandler)
				return
			}

			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) != 2 {
				unauthorized(w, config.Realm, config.UnauthorizedHandler)
				return
			}

			username, password := parts[0], parts[1]
			if !config.Authenticator(username, password) {
				unauthorized(w, config.Realm, config.UnauthorizedHandler)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func unauthorized(w http.ResponseWriter, realm string, handler http.Handler) {
	if handler != nil {
		handler.ServeHTTP(w, nil) // TODO: pass request?
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("Unauthorized"))
}

package middleware

import (
	"net/http"

	"github.com/bondowe/webfram"
)

// APIKeyAuthConfig holds configuration for API key authentication middleware
type APIKeyAuthConfig struct {
	// KeyName is the name of the API key (e.g., "X-API-Key")
	KeyName string
	// In specifies where to look for the key: "header", "query", or "cookie"
	In string
	// Authenticator is called with the key value, should return true if valid
	Authenticator func(key string) bool
	// UnauthorizedHandler is called when authentication fails (optional)
	UnauthorizedHandler webfram.Handler
}

// APIKeyAuth returns a middleware that enforces API Key Authentication
func APIKeyAuth(config APIKeyAuthConfig) func(webfram.Handler) webfram.Handler {
	return func(next webfram.Handler) webfram.Handler {
		return webfram.HandlerFunc(func(w webfram.ResponseWriter, r *webfram.Request) {
			var key string
			var found bool

			switch config.In {
			case "header":
				key = r.Header.Get(config.KeyName)
				found = key != ""
			case "query":
				key = r.URL.Query().Get(config.KeyName)
				found = key != ""
			case "cookie":
				cookie, err := r.Cookie(config.KeyName)
				if err == nil {
					key = cookie.Value
					found = true
				}
			default:
				unauthorizedAPIKey(w, config.UnauthorizedHandler)
				return
			}

			if !found || !config.Authenticator(key) {
				unauthorizedAPIKey(w, config.UnauthorizedHandler)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func unauthorizedAPIKey(w webfram.ResponseWriter, handler webfram.Handler) {
	if handler != nil {
		handler.ServeHTTP(w, nil)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Unauthorized"))
}

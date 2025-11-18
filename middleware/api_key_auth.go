package middleware

import (
	"net/http"
)

// APIKeyAuthConfig holds configuration for API key authentication middleware.
type APIKeyAuthConfig struct {
	// KeyValidator is called with the API key, should return true if valid
	KeyValidator func(key string) bool
	// KeyName is the name of the API key parameter (default: "api_key")
	KeyName string
	// KeyLocation specifies where to look for the API key: "header", "query", "cookie"
	KeyLocation string
	// UnauthorizedHandler is called when authentication fails (optional)
	UnauthorizedHandler http.Handler
}

// APIKeyAuth returns a middleware that enforces API Key Authentication.
func APIKeyAuth(config APIKeyAuthConfig) func(http.Handler) http.Handler {
	if config.KeyName == "" {
		config.KeyName = "api_key"
	}
	if config.KeyLocation == "" {
		config.KeyLocation = "header"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var key string
			var found bool

			switch config.KeyLocation {
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

			if !found || !config.KeyValidator(key) {
				unauthorizedAPIKey(w, config.UnauthorizedHandler)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func unauthorizedAPIKey(w http.ResponseWriter, handler http.Handler) {
	if handler != nil {
		handler.ServeHTTP(w, nil)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("Unauthorized"))
}

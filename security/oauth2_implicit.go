package security

import (
	"net/http"
	"net/url"
	"strings"
)

// OAuth2ImplicitConfig holds configuration for Implicit flow.
type OAuth2ImplicitConfig struct {
	OAuth2BaseConfig
	// AuthorizationURL is the OAuth2 authorization endpoint
	AuthorizationURL string
	// RedirectURL is the OAuth2 redirect URI
	RedirectURL string
	// StateStore stores/retrieves OAuth2 state parameters
	StateStore func(state string) (redirectURL string, ok bool)
}

// OAuth2ImplicitAuth returns middleware for OAuth2 Implicit flow.
func OAuth2ImplicitAuth(config OAuth2ImplicitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for access token in URL fragment (handled by frontend)
			if token := r.URL.Query().Get("access_token"); token != "" &&
				config.TokenValidator(token) {
				// Remove token from URL and proceed
				q := r.URL.Query()
				q.Del("access_token")
				r.URL.RawQuery = q.Encode()
				next.ServeHTTP(w, r)
				return
			}

			// Check for Bearer token in header
			if token := extractBearerToken(r); token != "" && config.TokenValidator(token) {
				next.ServeHTTP(w, r)
				return
			}

			// Redirect to authorization server with implicit flow
			redirectToAuthorizationServerImplicit(w, r, config)
		})
	}
}

// Helper function for Implicit flow

func redirectToAuthorizationServerImplicit(
	w http.ResponseWriter,
	r *http.Request,
	config OAuth2ImplicitConfig,
) {
	state := generateState()

	authURL, _ := url.Parse(config.AuthorizationURL)
	q := authURL.Query()
	q.Set("response_type", "token")
	q.Set("client_id", config.ClientID)
	q.Set("redirect_uri", config.RedirectURL)
	q.Set("scope", strings.Join(config.Scopes, " "))
	q.Set("state", state)
	authURL.RawQuery = q.Encode()

	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

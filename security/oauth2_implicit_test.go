package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOAuth2ImplicitAuth_Redirect(t *testing.T) {
	config := OAuth2ImplicitConfig{
		OAuth2BaseConfig: OAuth2BaseConfig{
			ClientID: "test-client",
			TokenURL: "https://auth.example.com/oauth/token",
			Scopes:   []string{"read"},
			TokenValidator: func(token string) bool {
				return token == "valid-token"
			},
			UnauthorizedHandler: nil,
		},
		AuthorizationURL: "https://auth.example.com/oauth/authorize",
		RedirectURL:      "https://app.example.com/callback",
		StateStore: func(state string) (redirectURL string, ok bool) {
			return "/", true
		},
	}

	middleware := OAuth2ImplicitAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Expected status 302 (redirect), got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if location == "" {
		t.Error("Expected Location header for redirect")
	}
	if !contains(location, "auth.example.com/oauth/authorize") {
		t.Errorf("Expected redirect to authorization endpoint, got %q", location)
	}
	if !contains(location, "response_type=token") {
		t.Errorf("Expected implicit flow response_type=token, got %q", location)
	}
}

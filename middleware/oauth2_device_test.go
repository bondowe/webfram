package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOAuth2DeviceAuth_DeviceCodeRequest(t *testing.T) {
	config := OAuth2DeviceConfig{
		OAuth2BaseConfig: OAuth2BaseConfig{
			ClientID: "test-client",
			TokenURL: "https://auth.example.com/oauth/token",
			Scopes:   []string{"read"},
			TokenValidator: func(token string) bool {
				return token == "valid-token"
			},
			UnauthorizedHandler: nil,
		},
	}

	middleware := OAuth2DeviceAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("POST", "/?request_device_code=true", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", contentType)
	}
}

func TestOAuth2DeviceAuth_NoToken(t *testing.T) {
	config := OAuth2DeviceConfig{
		OAuth2BaseConfig: OAuth2BaseConfig{
			ClientID: "test-client",
			TokenURL: "https://auth.example.com/oauth/token",
			Scopes:   []string{"read"},
			TokenValidator: func(token string) bool {
				return token == "valid-token"
			},
			UnauthorizedHandler: nil,
		},
	}

	middleware := OAuth2DeviceAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

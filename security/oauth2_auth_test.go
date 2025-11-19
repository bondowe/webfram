package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOAuth2TokenAuth_ValidToken(t *testing.T) {
	config := OAuth2TokenConfig{
		TokenValidator: func(token string) bool {
			return token == "valid-token"
		},
		UnauthorizedHandler: nil,
	}

	middleware := OAuth2TokenAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body != "success" {
		t.Errorf("Expected body 'success', got %q", body)
	}
}

func TestOAuth2TokenAuth_InvalidToken(t *testing.T) {
	config := OAuth2TokenConfig{
		TokenValidator: func(token string) bool {
			return token == "valid-token"
		},
		UnauthorizedHandler: nil,
	}

	middleware := OAuth2TokenAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestOAuth2TokenAuth_NoAuthorizationHeader(t *testing.T) {
	config := OAuth2TokenConfig{
		TokenValidator: func(token string) bool {
			return token == "valid-token"
		},
		UnauthorizedHandler: nil,
	}

	middleware := OAuth2TokenAuth(config)
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

func TestOAuth2TokenAuth_WrongPrefix(t *testing.T) {
	config := OAuth2TokenConfig{
		TokenValidator: func(token string) bool {
			return token == "valid-token"
		},
		UnauthorizedHandler: nil,
	}

	middleware := OAuth2TokenAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestOAuth2TokenAuth_CustomUnauthorizedHandler(t *testing.T) {
	customHandlerCalled := false
	config := OAuth2TokenConfig{
		TokenValidator: func(token string) bool {
			return false
		},
		UnauthorizedHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			customHandlerCalled = true
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("custom unauthorized"))
		}),
	}

	middleware := OAuth2TokenAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	if !customHandlerCalled {
		t.Error("Expected custom unauthorized handler to be called")
	}

	body := w.Body.String()
	if body != "custom unauthorized" {
		t.Errorf("Expected body 'custom unauthorized', got %q", body)
	}
}

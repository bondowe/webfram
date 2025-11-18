package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyAuth_Header_Success(t *testing.T) {
	config := APIKeyAuthConfig{
		KeyName:     "X-API-Key",
		KeyLocation: "header",
		KeyValidator: func(key string) bool {
			return key == "valid-key"
		},
	}

	middleware := APIKeyAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "valid-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("Expected 'success', got %q", w.Body.String())
	}
}

func TestAPIKeyAuth_Query_Success(t *testing.T) {
	config := APIKeyAuthConfig{
		KeyName:     "api_key",
		KeyLocation: "query",
		KeyValidator: func(key string) bool {
			return key == "valid-key"
		},
	}

	middleware := APIKeyAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/?api_key=valid-key", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("Expected 'success', got %q", w.Body.String())
	}
}

func TestAPIKeyAuth_Cookie_Success(t *testing.T) {
	config := APIKeyAuthConfig{
		KeyName:     "api_key",
		KeyLocation: "cookie",
		KeyValidator: func(key string) bool {
			return key == "valid-key"
		},
	}

	middleware := APIKeyAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "api_key", Value: "valid-key"})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("Expected 'success', got %q", w.Body.String())
	}
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	config := APIKeyAuthConfig{
		KeyName:     "X-API-Key",
		KeyLocation: "header",
		KeyValidator: func(key string) bool {
			return key == "valid-key"
		},
	}

	middleware := APIKeyAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "invalid-key")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAPIKeyAuth_NoKey(t *testing.T) {
	config := APIKeyAuthConfig{
		KeyName:     "X-API-Key",
		KeyLocation: "header",
		KeyValidator: func(key string) bool {
			return key == "valid-key"
		},
	}

	middleware := APIKeyAuth(config)
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

func TestAPIKeyAuth_Defaults(t *testing.T) {
	config := APIKeyAuthConfig{
		KeyValidator: func(key string) bool {
			return key == "valid-key"
		},
	}

	middleware := APIKeyAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("api_key", "valid-key") // Default key name
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

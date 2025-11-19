package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAllScopes_Success(t *testing.T) {
	// Token with all required scopes
	token := &OAuth2Token{
		AccessToken: "test-token",
		Scope:       "read write profile",
	}

	handler := RequireAllScopes("read", "write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), OAuth2TokenKey{}, token))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "success" {
		t.Errorf("Expected body 'success', got %q", w.Body.String())
	}
}

func TestRequireAllScopes_InsufficientScopes(t *testing.T) {
	// Token missing one required scope
	token := &OAuth2Token{
		AccessToken: "test-token",
		Scope:       "read profile", // missing "write"
	}

	handler := RequireAllScopes("read", "write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), OAuth2TokenKey{}, token))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	expectedBody := "Insufficient scopes\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
	}
}

func TestRequireAllScopes_NoToken(t *testing.T) {
	handler := RequireAllScopes("read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	expectedBody := "No OAuth2 token in context\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
	}
}

func TestRequireAnyScopes_Success(t *testing.T) {
	// Token with at least one required scope
	token := &OAuth2Token{
		AccessToken: "test-token",
		Scope:       "read profile", // has "read"
	}

	handler := RequireAnyScopes("read", "write", "admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), OAuth2TokenKey{}, token))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "success" {
		t.Errorf("Expected body 'success', got %q", w.Body.String())
	}
}

func TestRequireAnyScopes_InsufficientScopes(t *testing.T) {
	// Token with none of the required scopes
	token := &OAuth2Token{
		AccessToken: "test-token",
		Scope:       "profile email", // has neither "read", "write", nor "admin"
	}

	handler := RequireAnyScopes("read", "write", "admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), OAuth2TokenKey{}, token))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	expectedBody := "Insufficient scopes\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
	}
}

func TestRequireAnyScopes_NoToken(t *testing.T) {
	handler := RequireAnyScopes("read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	expectedBody := "No OAuth2 token in context\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
	}
}

func TestRequireAnyScopes_MultipleMatchingScopes(t *testing.T) {
	// Token with multiple matching scopes
	token := &OAuth2Token{
		AccessToken: "test-token",
		Scope:       "read write admin", // has "read" and "write"
	}

	handler := RequireAnyScopes("read", "write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), OAuth2TokenKey{}, token))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHasAllScopes(t *testing.T) {
	tests := []struct {
		name           string
		tokenScopes    []string
		requiredScopes []string
		expected       bool
	}{
		{
			name:           "has all scopes",
			tokenScopes:    []string{"read", "write", "profile"},
			requiredScopes: []string{"read", "write"},
			expected:       true,
		},
		{
			name:           "missing one scope",
			tokenScopes:    []string{"read", "profile"},
			requiredScopes: []string{"read", "write"},
			expected:       false,
		},
		{
			name:           "empty required scopes",
			tokenScopes:    []string{"read", "write"},
			requiredScopes: []string{},
			expected:       true,
		},
		{
			name:           "empty token scopes",
			tokenScopes:    []string{},
			requiredScopes: []string{"read"},
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasAllScopes(tt.tokenScopes, tt.requiredScopes)
			if result != tt.expected {
				t.Errorf("hasAllScopes() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestHasAnyScopes(t *testing.T) {
	tests := []struct {
		name           string
		tokenScopes    []string
		requiredScopes []string
		expected       bool
	}{
		{
			name:           "has one scope",
			tokenScopes:    []string{"read", "profile"},
			requiredScopes: []string{"read", "write"},
			expected:       true,
		},
		{
			name:           "has multiple scopes",
			tokenScopes:    []string{"read", "write", "profile"},
			requiredScopes: []string{"read", "write"},
			expected:       true,
		},
		{
			name:           "has none of the scopes",
			tokenScopes:    []string{"profile", "email"},
			requiredScopes: []string{"read", "write"},
			expected:       false,
		},
		{
			name:           "empty required scopes",
			tokenScopes:    []string{"read", "write"},
			requiredScopes: []string{},
			expected:       false,
		},
		{
			name:           "empty token scopes",
			tokenScopes:    []string{},
			requiredScopes: []string{"read"},
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasAnyScopes(tt.tokenScopes, tt.requiredScopes)
			if result != tt.expected {
				t.Errorf("hasAnyScopes() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

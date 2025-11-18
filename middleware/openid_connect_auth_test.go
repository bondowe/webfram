package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOIDCToken_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		token    OIDCToken
		buffer   time.Duration
		expected bool
	}{
		{
			name: "token not expired",
			token: OIDCToken{
				ExpiresAt: now.Add(time.Hour),
			},
			buffer:   time.Minute,
			expected: false,
		},
		{
			name: "token expired",
			token: OIDCToken{
				ExpiresAt: now.Add(-time.Hour),
			},
			buffer:   time.Minute,
			expected: true,
		},
		{
			name: "token expires within buffer",
			token: OIDCToken{
				ExpiresAt: now.Add(30 * time.Second),
			},
			buffer:   time.Minute,
			expected: true,
		},
		{
			name: "no expiration time",
			token: OIDCToken{
				ExpiresAt: time.Time{},
			},
			buffer:   time.Minute,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.IsExpired(tt.buffer)
			if result != tt.expected {
				t.Errorf("IsExpired() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestOIDCToken_NeedsRefresh(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		token    OIDCToken
		buffer   time.Duration
		expected bool
	}{
		{
			name: "token needs refresh - expired",
			token: OIDCToken{
				ExpiresAt:    now.Add(-time.Hour),
				RefreshToken: "refresh-token",
			},
			buffer:   time.Minute,
			expected: true,
		},
		{
			name: "token needs refresh - expires soon",
			token: OIDCToken{
				ExpiresAt:    now.Add(30 * time.Second),
				RefreshToken: "refresh-token",
			},
			buffer:   time.Minute,
			expected: true,
		},
		{
			name: "token doesn't need refresh - valid",
			token: OIDCToken{
				ExpiresAt:    now.Add(time.Hour),
				RefreshToken: "refresh-token",
			},
			buffer:   time.Minute,
			expected: false,
		},
		{
			name: "token doesn't need refresh - no refresh token",
			token: OIDCToken{
				ExpiresAt: now.Add(-time.Hour),
			},
			buffer:   time.Minute,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.NeedsRefresh(tt.buffer)
			if result != tt.expected {
				t.Errorf("NeedsRefresh() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestOpenIDConnectAuth_ValidStoredToken(t *testing.T) {
	// Mock token store with valid stored token
	var storedToken *OIDCToken
	tokenStore := func(sessionID string) (*OIDCToken, bool) {
		if storedToken != nil {
			return storedToken, true
		}
		return nil, false
	}

	config := OpenIDConnectAuthConfig{
		IssuerURL:    "https://accounts.example.com",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "https://app.example.com/oidc/callback",
		Scopes:       []string{"openid", "profile"},
		TokenValidator: func(token string) bool {
			return token == "valid-stored-id-token"
		},
		StateStore: func(state string) (redirectURL string, ok bool) {
			return "/", true
		},
		TokenStore: tokenStore,
		SessionIDExtractor: func(r *http.Request) string {
			return "test-session"
		},
		RefreshBuffer: 5 * time.Minute,
	}

	// Set up valid stored token
	storedToken = &OIDCToken{
		AccessToken: "valid-access-token",
		IDToken:     "valid-stored-id-token",
		ExpiresAt:   time.Now().Add(time.Hour), // Valid
	}

	middleware := OpenIDConnectAuth(config)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	handler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
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

func TestOpenIDConnectAuth_ExpiredTokenWithRefresh(t *testing.T) {
	// Mock token store with expired token that has refresh token
	var storedToken *OIDCToken
	tokenStore := func(sessionID string) (*OIDCToken, bool) {
		if storedToken != nil {
			return storedToken, true
		}
		return nil, false
	}

	config := OpenIDConnectAuthConfig{
		IssuerURL:    "https://accounts.example.com",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "https://app.example.com/oidc/callback",
		Scopes:       []string{"openid", "profile"},
		TokenValidator: func(token string) bool {
			return token == "refreshed-id-token"
		},
		StateStore: func(state string) (redirectURL string, ok bool) {
			return "/", true
		},
		TokenStore: tokenStore,
		SessionIDExtractor: func(r *http.Request) string {
			return "test-session"
		},
		RefreshBuffer: 5 * time.Minute,
	}

	// Set up expired token with refresh token
	storedToken = &OIDCToken{
		AccessToken:  "expired-access-token",
		IDToken:      "expired-id-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(-time.Hour), // Expired
	}

	middleware := OpenIDConnectAuth(config)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	handler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should redirect since refresh will fail (no real HTTP call)
	if w.Code != http.StatusFound {
		t.Errorf("Expected status 302 (redirect), got %d", w.Code)
	}
}

func TestOpenIDConnectAuth_TokenExpiringSoon(t *testing.T) {
	// Mock token store with token expiring soon
	var storedToken *OIDCToken
	tokenStore := func(sessionID string) (*OIDCToken, bool) {
		if storedToken != nil {
			return storedToken, true
		}
		return nil, false
	}

	config := OpenIDConnectAuthConfig{
		IssuerURL:    "https://accounts.example.com",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "https://app.example.com/oidc/callback",
		Scopes:       []string{"openid", "profile"},
		TokenValidator: func(token string) bool {
			return token == "expiring-id-token"
		},
		StateStore: func(state string) (redirectURL string, ok bool) {
			return "/", true
		},
		TokenStore: tokenStore,
		SessionIDExtractor: func(r *http.Request) string {
			return "test-session"
		},
		RefreshBuffer: 10 * time.Minute, // Large buffer
	}

	// Set up token expiring in 2 minutes (within buffer)
	storedToken = &OIDCToken{
		AccessToken:  "expiring-access-token",
		IDToken:      "expiring-id-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(2 * time.Minute),
	}

	middleware := OpenIDConnectAuth(config)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	handler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should redirect since refresh will fail (no real HTTP call)
	if w.Code != http.StatusFound {
		t.Errorf("Expected status 302 (redirect), got %d", w.Code)
	}
}

func TestOpenIDConnectAuth_CustomRefreshBuffer(t *testing.T) {
	// Mock token store
	var storedToken *OIDCToken
	tokenStore := func(sessionID string) (*OIDCToken, bool) {
		if storedToken != nil {
			return storedToken, true
		}
		return nil, false
	}

	config := OpenIDConnectAuthConfig{
		IssuerURL:    "https://accounts.example.com",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "https://app.example.com/oidc/callback",
		Scopes:       []string{"openid", "profile"},
		TokenValidator: func(token string) bool {
			return token == "valid-id-token"
		},
		StateStore: func(state string) (redirectURL string, ok bool) {
			return "/", true
		},
		TokenStore: tokenStore,
		SessionIDExtractor: func(r *http.Request) string {
			return "test-session"
		},
		RefreshBuffer: 30 * time.Minute, // Very large buffer
	}

	// Set up token expiring in 45 minutes (outside buffer)
	storedToken = &OIDCToken{
		AccessToken:  "valid-access-token",
		IDToken:      "valid-id-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(45 * time.Minute),
	}

	middleware := OpenIDConnectAuth(config)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	handler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
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

func TestOpenIDConnectAuth_ExpiredTokenNoRefreshToken(t *testing.T) {
	// Mock token store with expired token but no refresh token
	var storedToken *OIDCToken
	tokenStore := func(sessionID string) (*OIDCToken, bool) {
		if storedToken != nil {
			return storedToken, true
		}
		return nil, false
	}

	config := OpenIDConnectAuthConfig{
		IssuerURL:    "https://accounts.example.com",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "https://app.example.com/oidc/callback",
		Scopes:       []string{"openid", "profile"},
		TokenValidator: func(token string) bool {
			return token == "expired-no-refresh"
		},
		StateStore: func(state string) (redirectURL string, ok bool) {
			return "/", true
		},
		TokenStore: tokenStore,
		SessionIDExtractor: func(r *http.Request) string {
			return "test-session"
		},
		RefreshBuffer: 5 * time.Minute,
	}

	// Set up expired token without refresh token
	storedToken = &OIDCToken{
		AccessToken: "expired-access-token",
		IDToken:     "expired-no-refresh",
		ExpiresAt:   time.Now().Add(-time.Hour), // Expired
		// No RefreshToken
	}

	middleware := OpenIDConnectAuth(config)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	handler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should redirect since token is expired and no refresh token
	if w.Code != http.StatusFound {
		t.Errorf("Expected status 302 (redirect), got %d", w.Code)
	}
}

func TestOpenIDConnectAuth_TokenValidation(t *testing.T) {
	config := OpenIDConnectAuthConfig{
		TokenValidator: func(token string) bool {
			return token == "valid-token"
		},
	}

	middleware := OpenIDConnectAuth(config)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	handler := middleware(testHandler)

	// Test valid token
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("Expected 'success', got %q", w.Body.String())
	}

	// Test invalid token
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("Authorization", "Bearer invalid-token")
	w2 := httptest.NewRecorder()

	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w2.Code)
	}

	// Test no token
	req3 := httptest.NewRequest("GET", "/", nil)
	w3 := httptest.NewRecorder()

	handler.ServeHTTP(w3, req3)

	if w3.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w3.Code)
	}
}

func TestOpenIDConnectAuth_RedirectFlow(t *testing.T) {
	config := OpenIDConnectAuthConfig{
		IssuerURL:    "https://accounts.example.com",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "https://app.example.com/oidc/callback",
		Scopes:       []string{"openid", "profile"},
		TokenValidator: func(token string) bool {
			return token == "valid-token"
		},
		StateStore: func(state string) (redirectURL string, ok bool) {
			return "/", true
		},
	}

	middleware := OpenIDConnectAuth(config)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	handler := middleware(testHandler)

	// Test redirect when no token
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
	if !contains(location, "accounts.example.com/authorize") {
		t.Errorf("Expected redirect to authorization endpoint, got %q", location)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	if start+len(substr) > len(s) {
		return false
	}
	for i := 0; i < len(substr); i++ {
		if s[start+i] != substr[i] {
			return containsAt(s, substr, start+1)
		}
	}
	return true
}

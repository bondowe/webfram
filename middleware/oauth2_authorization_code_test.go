package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOAuth2AuthorizationCodeAuth_Redirect(t *testing.T) {
	config := OAuth2AuthorizationCodeConfig{
		OAuth2BaseConfig: OAuth2BaseConfig{
			ClientID: "test-client",
			TokenURL: "https://auth.example.com/oauth/token",
			Scopes:   []string{"read", "write"},
			TokenValidator: func(token string) bool {
				return token == "valid-token"
			},
			UnauthorizedHandler: nil,
		},
		ClientSecret:     "test-secret",
		AuthorizationURL: "https://auth.example.com/oauth/authorize",
		RedirectURL:      "https://app.example.com/callback",
		StateStore: func(state string) (redirectURL string, ok bool) {
			return "/", true
		},
		TokenStore: func(sessionID string) (*OAuth2Token, bool) {
			return nil, false
		},
		SessionIDExtractor: func(r *http.Request) string {
			return "test-session"
		},
	}

	middleware := OAuth2AuthorizationCodeAuth(config)
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
}

func TestOAuth2Token_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		token    OAuth2Token
		buffer   time.Duration
		expected bool
	}{
		{
			name: "token not expired",
			token: OAuth2Token{
				ExpiresAt: now.Add(time.Hour),
			},
			buffer:   time.Minute,
			expected: false,
		},
		{
			name: "token expired",
			token: OAuth2Token{
				ExpiresAt: now.Add(-time.Hour),
			},
			buffer:   time.Minute,
			expected: true,
		},
		{
			name: "token expires within buffer",
			token: OAuth2Token{
				ExpiresAt: now.Add(30 * time.Second),
			},
			buffer:   time.Minute,
			expected: true,
		},
		{
			name: "no expiration time",
			token: OAuth2Token{
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

func TestOAuth2Token_NeedsRefresh(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		token    OAuth2Token
		buffer   time.Duration
		expected bool
	}{
		{
			name: "token needs refresh - expired",
			token: OAuth2Token{
				ExpiresAt:    now.Add(-time.Hour),
				RefreshToken: "refresh-token",
			},
			buffer:   time.Minute,
			expected: true,
		},
		{
			name: "token needs refresh - expires soon",
			token: OAuth2Token{
				ExpiresAt:    now.Add(30 * time.Second),
				RefreshToken: "refresh-token",
			},
			buffer:   time.Minute,
			expected: true,
		},
		{
			name: "token doesn't need refresh - valid",
			token: OAuth2Token{
				ExpiresAt:    now.Add(time.Hour),
				RefreshToken: "refresh-token",
			},
			buffer:   time.Minute,
			expected: false,
		},
		{
			name: "token doesn't need refresh - no refresh token",
			token: OAuth2Token{
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

func TestOAuth2AuthorizationCodeAuth_TokenRefresh(t *testing.T) {
	// Mock token store with expired token
	var storedToken *OAuth2Token
	tokenStore := func(sessionID string) (*OAuth2Token, bool) {
		if storedToken != nil {
			return storedToken, true
		}
		return nil, false
	}

	config := OAuth2AuthorizationCodeConfig{
		OAuth2BaseConfig: OAuth2BaseConfig{
			ClientID: "test-client",
			TokenURL: "https://auth.example.com/oauth/token",
			Scopes:   []string{"read"},
			TokenValidator: func(token string) bool {
				return token == "refreshed-token"
			},
			UnauthorizedHandler: nil,
			RefreshBuffer:       5 * time.Minute,
		},
		ClientSecret:     "test-secret",
		AuthorizationURL: "https://auth.example.com/oauth/authorize",
		RedirectURL:      "https://app.example.com/callback",
		StateStore: func(state string) (redirectURL string, ok bool) {
			return "/", true
		},
		TokenStore: tokenStore,
		SessionIDExtractor: func(r *http.Request) string {
			return "test-session"
		},
	}

	// Set up expired token with refresh token
	storedToken = &OAuth2Token{
		AccessToken:  "expired-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(-time.Hour), // Expired
	}

	middleware := OAuth2AuthorizationCodeAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should redirect to auth server since refresh failed (no real HTTP call)
	if w.Code != http.StatusFound {
		t.Errorf("Expected status 302 (redirect), got %d", w.Code)
	}
}

func TestOAuth2AuthorizationCodeAuth_ValidStoredToken(t *testing.T) {
	// Mock token store with valid token
	var storedToken *OAuth2Token
	tokenStore := func(sessionID string) (*OAuth2Token, bool) {
		if storedToken != nil {
			return storedToken, true
		}
		return nil, false
	}

	config := OAuth2AuthorizationCodeConfig{
		OAuth2BaseConfig: OAuth2BaseConfig{
			ClientID: "test-client",
			TokenURL: "https://auth.example.com/oauth/token",
			Scopes:   []string{"read"},
			TokenValidator: func(token string) bool {
				return token == "valid-stored-token"
			},
			UnauthorizedHandler: nil,
			RefreshBuffer:       5 * time.Minute,
		},
		ClientSecret:     "test-secret",
		AuthorizationURL: "https://auth.example.com/oauth/authorize",
		RedirectURL:      "https://app.example.com/callback",
		StateStore: func(state string) (redirectURL string, ok bool) {
			return "/", true
		},
		TokenStore: tokenStore,
		SessionIDExtractor: func(r *http.Request) string {
			return "test-session"
		},
	}

	// Set up valid token
	storedToken = &OAuth2Token{
		AccessToken: "valid-stored-token",
		ExpiresAt:   time.Now().Add(time.Hour), // Valid
	}

	middleware := OAuth2AuthorizationCodeAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

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

func TestOAuth2AuthorizationCodeAuth_CustomRefreshBuffer(t *testing.T) {
	// Mock token store
	var storedToken *OAuth2Token
	tokenStore := func(sessionID string) (*OAuth2Token, bool) {
		if storedToken != nil {
			return storedToken, true
		}
		return nil, false
	}

	config := OAuth2AuthorizationCodeConfig{
		OAuth2BaseConfig: OAuth2BaseConfig{
			ClientID: "test-client",
			TokenURL: "https://auth.example.com/oauth/token",
			Scopes:   []string{"read"},
			TokenValidator: func(token string) bool {
				return token == "valid-token"
			},
			UnauthorizedHandler: nil,
			RefreshBuffer:       10 * time.Minute, // Custom buffer
		},
		ClientSecret:     "test-secret",
		AuthorizationURL: "https://auth.example.com/oauth/authorize",
		RedirectURL:      "https://app.example.com/callback",
		StateStore: func(state string) (redirectURL string, ok bool) {
			return "/", true
		},
		TokenStore: tokenStore,
		SessionIDExtractor: func(r *http.Request) string {
			return "test-session"
		},
	}

	// Set up token that expires in 5 minutes (within custom buffer)
	storedToken = &OAuth2Token{
		AccessToken:  "expired-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(5 * time.Minute),
	}

	middleware := OAuth2AuthorizationCodeAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should redirect since token needs refresh (within buffer)
	if w.Code != http.StatusFound {
		t.Errorf("Expected status 302 (redirect), got %d", w.Code)
	}
}

func TestGenerateCodeVerifier(t *testing.T) {
	verifier1 := generateCodeVerifier()
	verifier2 := generateCodeVerifier()

	if verifier1 == "" {
		t.Error("Code verifier should not be empty")
	}

	if verifier1 == verifier2 {
		t.Error("Code verifiers should be unique")
	}

	// Should be base64url encoded (no padding, URL-safe chars)
	if len(verifier1) != 43 { // 32 bytes -> base64url = 43 chars
		t.Errorf("Expected verifier length 43, got %d", len(verifier1))
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := "test-verifier"

	// Test S256 method
	challengeS256 := generateCodeChallenge(verifier, PKCES256)
	expectedS256 := "JBbiqONGWPaAmwXk_8bT6UnlPfrn65D32eZlJS-zGG0" // SHA256 of "test-verifier" base64url
	if challengeS256 != expectedS256 {
		t.Errorf("S256 challenge = %q, expected %q", challengeS256, expectedS256)
	}

	// Test plain method
	challengePlain := generateCodeChallenge(verifier, PKCEPlain)
	if challengePlain != verifier {
		t.Errorf("Plain challenge = %q, expected %q", challengePlain, verifier)
	}
}

func TestGetChallengeMethod(t *testing.T) {
	if getChallengeMethod(PKCES256) != "S256" {
		t.Error("Expected S256 for PKCES256")
	}

	if getChallengeMethod(PKCEPlain) != "plain" {
		t.Error("Expected plain for PKCEPlain")
	}
}

func TestOAuth2AuthorizationCodeAuth_PKCE_Redirect(t *testing.T) {
	var storedVerifier string
	config := OAuth2AuthorizationCodeConfig{
		OAuth2BaseConfig: OAuth2BaseConfig{
			ClientID: "test-client",
			TokenURL: "https://auth.example.com/oauth/token",
			Scopes:   []string{"read", "write"},
			TokenValidator: func(token string) bool {
				return token == "valid-token"
			},
			UnauthorizedHandler: nil,
		},
		ClientSecret:     "test-secret",
		AuthorizationURL: "https://auth.example.com/oauth/authorize",
		RedirectURL:      "https://app.example.com/callback",
		StateStore: func(state string) (redirectURL string, ok bool) {
			return "/", true
		},
		TokenStore: func(sessionID string) (*OAuth2Token, bool) {
			return nil, false
		},
		SessionIDExtractor: func(r *http.Request) string {
			return "test-session"
		},
		PKCE: &OAuth2PKCEConfig{
			CodeVerifierStore: func(state string) (codeVerifier string, ok bool) {
				return storedVerifier, storedVerifier != ""
			},
			ChallengeMethod: PKCES256,
		},
	}

	middleware := OAuth2AuthorizationCodeAuth(config)
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

	// Check that PKCE parameters are included
	if !contains(location, "code_challenge=") {
		t.Error("Expected code_challenge parameter in redirect URL")
	}
	if !contains(location, "code_challenge_method=S256") {
		t.Error("Expected code_challenge_method=S256 in redirect URL")
	}
}

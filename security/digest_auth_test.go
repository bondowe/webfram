package security

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDigestAuth_Success(t *testing.T) {
	config := DigestAuthConfig{
		Realm: "TestRealm",
		PasswordGetter: func(username, realm string) (string, bool) {
			if username == "testuser" && realm == "TestRealm" {
				return "testpass", true
			}
			return "", false
		},
		NonceTTL: 30 * time.Minute,
	}

	middleware := DigestAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	// Create a proper digest auth header
	username := "testuser"
	realm := "TestRealm"
	password := "testpass"
	method := "GET"
	uri := "/test"
	nonce := "abc123"

	// Calculate HA1 = MD5(username:realm:password)
	ha1 := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", username, realm, password)))
	ha1Hex := hex.EncodeToString(ha1[:])

	// Calculate HA2 = MD5(method:uri)
	ha2 := md5.Sum([]byte(fmt.Sprintf("%s:%s", method, uri)))
	ha2Hex := hex.EncodeToString(ha2[:])

	// Calculate response = MD5(HA1:nonce:HA2)
	response := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", ha1Hex, nonce, ha2Hex)))
	responseHex := hex.EncodeToString(response[:])

	authHeader := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
		username, realm, nonce, uri, responseHex)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", authHeader)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("Expected 'success', got %q", w.Body.String())
	}
}

func TestDigestAuth_InvalidCredentials(t *testing.T) {
	config := DigestAuthConfig{
		Realm: "TestRealm",
		PasswordGetter: func(username, realm string) (string, bool) {
			if username == "testuser" && realm == "TestRealm" {
				return "testpass", true
			}
			return "", false
		},
		NonceTTL: 30 * time.Minute,
	}

	middleware := DigestAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	// Create digest auth header with wrong password
	authHeader := `Digest username="testuser", realm="TestRealm", nonce="abc123", uri="/test", response="wrongresponse"`

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", authHeader)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestDigestAuth_NoAuth(t *testing.T) {
	config := DigestAuthConfig{
		Realm: "TestRealm",
		PasswordGetter: func(username, realm string) (string, bool) {
			return "", false
		},
		NonceTTL: 30 * time.Minute,
	}

	middleware := DigestAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	// Check that WWW-Authenticate header is set
	authHeader := w.Header().Get("WWW-Authenticate")
	if !strings.Contains(authHeader, `Digest realm="TestRealm"`) {
		t.Errorf("Expected WWW-Authenticate header with Digest realm, got %q", authHeader)
	}
}

func TestDigestAuth_WrongPrefix(t *testing.T) {
	config := DigestAuthConfig{
		Realm: "TestRealm",
		PasswordGetter: func(username, realm string) (string, bool) {
			return "", false
		},
		NonceTTL: 30 * time.Minute,
	}

	middleware := DigestAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

package middleware

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bondowe/webfram"
)

func TestBasicAuth_Success(t *testing.T) {
	config := BasicAuthConfig{
		Authenticator: func(username, password string) bool {
			return username == "user" && password == "pass"
		},
		Realm: "Test",
	}

	middleware := BasicAuth(config)

	handler := middleware(webfram.HandlerFunc(func(w webfram.ResponseWriter, r *webfram.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("user:pass")))

	w := httptest.NewRecorder()
	statusCode := 0
	rw := webfram.ResponseWriter{ResponseWriter: w, statusCode: &statusCode}
	handler.ServeHTTP(rw, &webfram.Request{Request: req})

	if statusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", statusCode)
	}
}

func TestBasicAuth_InvalidCredentials(t *testing.T) {
	config := BasicAuthConfig{
		Authenticator: func(username, password string) bool {
			return username == "user" && password == "pass"
		},
		Realm: "Test",
	}

	middleware := BasicAuth(config)

	handler := middleware(webfram.HandlerFunc(func(w webfram.ResponseWriter, r *webfram.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("user:wrong")))

	w := &mockResponseWriter{}
	handler.ServeHTTP(w, &webfram.Request{Request: req})

	if w.statusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.statusCode)
	}
}

func TestBasicAuth_NoAuth(t *testing.T) {
	config := BasicAuthConfig{
		Authenticator: func(username, password string) bool {
			return false
		},
		Realm: "Test",
	}

	middleware := BasicAuth(config)

	handler := middleware(webfram.HandlerFunc(func(w webfram.ResponseWriter, r *webfram.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/", nil)

	w := &mockResponseWriter{}
	handler.ServeHTTP(w, &webfram.Request{Request: req})

	if w.statusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.statusCode)
	}
}

type mockResponseWriter struct {
	statusCode int
	header     http.Header
	body       []byte
}

func (m *mockResponseWriter) Header() http.Header {
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	m.body = append(m.body, data...)
	return len(data), nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

func (m *mockResponseWriter) StatusCode() (int, bool) {
	return m.statusCode, true
}

func (m *mockResponseWriter) ResponseWriter() http.ResponseWriter {
	return nil
}

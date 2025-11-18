package middleware

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMutualTLSAuth_Success(t *testing.T) {
	config := MutualTLSAuthConfig{
		CertificateValidator: func(cert *x509.Certificate) bool {
			return cert.Subject.CommonName == "test-client"
		},
	}

	middleware := MutualTLSAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	// Create a mock certificate
	cert := &x509.Certificate{
		Subject: pkix.Name{
			CommonName: "test-client",
		},
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	// Create a request with mock TLS info
	req := httptest.NewRequest("GET", "/", nil)
	req.TLS = &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{cert},
	}
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("Expected 'success', got %q", w.Body.String())
	}
}

func TestMutualTLSAuth_InvalidCertificate(t *testing.T) {
	config := MutualTLSAuthConfig{
		CertificateValidator: func(cert *x509.Certificate) bool {
			return cert.Subject.CommonName == "test-client"
		},
	}

	middleware := MutualTLSAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	// Create a mock certificate with wrong CN
	cert := &x509.Certificate{
		Subject: pkix.Name{
			CommonName: "wrong-client",
		},
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.TLS = &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{cert},
	}
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestMutualTLSAuth_NoCertificate(t *testing.T) {
	config := MutualTLSAuthConfig{
		CertificateValidator: func(cert *x509.Certificate) bool {
			return true // Accept any cert
		},
	}

	middleware := MutualTLSAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	// No TLS or certificates
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestMutualTLSAuth_NoTLS(t *testing.T) {
	config := MutualTLSAuthConfig{
		CertificateValidator: func(cert *x509.Certificate) bool {
			return true // Accept any cert
		},
	}

	middleware := MutualTLSAuth(config)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.TLS = &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{}, // Empty certificates
	}
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

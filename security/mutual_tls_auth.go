package security

import (
	"crypto/x509"
	"net/http"
)

// MutualTLSAuthConfig holds configuration for mutual TLS authentication middleware.
type MutualTLSAuthConfig struct {
	// CertificateValidator is called with the client certificate, should return true if valid
	CertificateValidator func(cert *x509.Certificate) bool
	// UnauthorizedHandler is called when authentication fails (optional)
	UnauthorizedHandler http.Handler
}

// MutualTLSAuth returns a middleware that enforces Mutual TLS Authentication.
func MutualTLSAuth(config MutualTLSAuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
				unauthorizedMutualTLS(w, config.UnauthorizedHandler)
				return
			}

			clientCert := r.TLS.PeerCertificates[0]
			if !config.CertificateValidator(clientCert) {
				unauthorizedMutualTLS(w, config.UnauthorizedHandler)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func unauthorizedMutualTLS(w http.ResponseWriter, handler http.Handler) {
	if handler != nil {
		handler.ServeHTTP(w, nil)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("Unauthorized"))
}

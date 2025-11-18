package middleware

import (
	"crypto/x509"
	"net/http"

	"github.com/bondowe/webfram"
)

// MutualTLSAuthConfig holds configuration for mutual TLS authentication middleware
type MutualTLSAuthConfig struct {
	// CertificateValidator is called with the client certificate, should return true if valid
	CertificateValidator func(cert *x509.Certificate) bool
	// UnauthorizedHandler is called when authentication fails (optional)
	UnauthorizedHandler webfram.Handler
}

// MutualTLSAuth returns a middleware that enforces Mutual TLS Authentication
func MutualTLSAuth(config MutualTLSAuthConfig) func(webfram.Handler) webfram.Handler {
	return func(next webfram.Handler) webfram.Handler {
		return webfram.HandlerFunc(func(w webfram.ResponseWriter, r *webfram.Request) {
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

func unauthorizedMutualTLS(w webfram.ResponseWriter, handler webfram.Handler) {
	if handler != nil {
		handler.ServeHTTP(w, nil)
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Unauthorized"))
}

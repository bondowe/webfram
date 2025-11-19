package security

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DigestAuthConfig holds configuration for digest authentication middleware.
type DigestAuthConfig struct {
	// Realm is the authentication realm
	Realm string
	// PasswordGetter is called with username and realm, should return the password and true if user exists
	PasswordGetter func(username, realm string) (password string, ok bool)
	// NonceTTL is the time-to-live for nonces (default 30 minutes)
	NonceTTL time.Duration
	// UnauthorizedHandler is called when authentication fails (optional)
	UnauthorizedHandler http.Handler
}

var (
	nonceStore = sync.Map{} // map[[16]byte]time.Time
)

// DigestAuth returns a middleware that enforces HTTP Digest Authentication.
func DigestAuth(config DigestAuthConfig) func(http.Handler) http.Handler {
	if config.Realm == "" {
		config.Realm = "Restricted"
	}
	if config.NonceTTL == 0 {
		config.NonceTTL = 30 * time.Minute
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				unauthorizedDigest(w, config.Realm, config.UnauthorizedHandler)
				return
			}

			if !strings.HasPrefix(auth, "Digest ") {
				unauthorizedDigest(w, config.Realm, config.UnauthorizedHandler)
				return
			}

			params := parseDigestParams(strings.TrimPrefix(auth, "Digest "))
			if !validateDigest(params, r.Method, r.URL.Path, config) {
				unauthorizedDigest(w, config.Realm, config.UnauthorizedHandler)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func parseDigestParams(auth string) map[string]string {
	params := make(map[string]string)
	parts := strings.Split(auth, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if kv := strings.SplitN(part, "=", 2); len(kv) == 2 {
			key := kv[0]
			value := strings.Trim(kv[1], `"`)
			params[key] = value
		}
	}
	return params
}

func validateDigest(params map[string]string, method, uri string, config DigestAuthConfig) bool {
	username := params["username"]
	realm := params["realm"]
	nonce := params["nonce"]
	qop := params["qop"]
	nc := params["nc"]
	cnonce := params["cnonce"]
	uriParam := params["uri"]
	response := params["response"]

	if username == "" || realm != config.Realm || nonce == "" || uriParam != uri || response == "" {
		return false
	}

	password, ok := config.PasswordGetter(username, realm)
	if !ok {
		return false
	}

	// Check nonce (simple check, in production should validate timestamp)
	if !isValidNonce(nonce, config.NonceTTL) {
		return false
	}

	// Calculate expected response
	ha1 := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", username, realm, password)))
	ha2 := md5.Sum([]byte(fmt.Sprintf("%s:%s", method, uri)))

	var expected [16]byte

	if qop != "" && nc != "" && cnonce != "" {
		// RFC 2617 (modern) formula
		expected = md5.Sum([]byte(fmt.Sprintf("%s:%s:%s:%s:%s:%s",
			hex.EncodeToString(ha1[:]), nonce, nc, cnonce, qop, hex.EncodeToString(ha2[:]))))
	} else {
		// Legacy formula (no qop)
		expected = md5.Sum([]byte(fmt.Sprintf("%s:%s:%s",
			hex.EncodeToString(ha1[:]), nonce, hex.EncodeToString(ha2[:]))))
	}
	return hex.EncodeToString(expected[:]) == response
}

func isValidNonce(nonce string, nonceTTL time.Duration) bool {
	// TODO: improve nonce validation (e.g., central store, housekeeping, timestamp, uniqueness)
	t, ok := nonceStore.Load(nonce)

	if !ok {
		return false
	}

	createdAt, ok := t.(time.Time)
	if !ok {
		return false
	}

	if time.Since(createdAt) > nonceTTL {
		nonceStore.Delete(nonce)
		return false
	}

	return true
}

func unauthorizedDigest(w http.ResponseWriter, realm string, handler http.Handler) {
	if handler != nil {
		handler.ServeHTTP(w, nil)
		return
	}

	nonce := generateNonce()
	nonceStore.Store(nonce, time.Now())

	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Digest realm="%s", nonce="%s", algorithm=MD5, qop="auth"`,
		realm, nonce))
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("Unauthorized"))
}

func generateNonce() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

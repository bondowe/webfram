package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// OAuth2DeviceConfig holds configuration for Device Authorization Grant flow.
type OAuth2DeviceConfig struct {
	OAuth2BaseConfig
}

// OAuth2DeviceAuth returns middleware for OAuth2 Device Authorization Grant flow.
func OAuth2DeviceAuth(config OAuth2DeviceConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for valid access token first
			if token := extractBearerToken(r); token != "" && config.TokenValidator(token) {
				next.ServeHTTP(w, r)
				return
			}

			// Check if this is a device code request
			if r.Method == http.MethodPost && r.URL.Query().Get("request_device_code") == "true" {
				handleDeviceCodeRequest(w, r, config)
				return
			}

			// Check if this is a token polling request
			if r.Method == http.MethodPost && r.URL.Query().Get("device_code") != "" {
				handleDeviceTokenPolling(w, r, config, next)
				return
			}

			// For regular requests, require valid token
			unauthorizedOAuth2(w, config.UnauthorizedHandler)
		})
	}
}

// Helper functions for Device flow

func handleDeviceCodeRequest(w http.ResponseWriter, _ *http.Request, config OAuth2DeviceConfig) {
	// Request device code from authorization server
	deviceCode := requestDeviceCode(config)

	// Return device code information to client
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(deviceCode)
}

func handleDeviceTokenPolling(
	w http.ResponseWriter,
	r *http.Request,
	config OAuth2DeviceConfig,
	next http.Handler,
) {
	deviceCode := r.URL.Query().Get("device_code")

	// Poll for token
	token, err := pollForDeviceToken(config, deviceCode)
	if err != nil {
		// Return pending status for polling
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "authorization_pending",
		})
		return
	}

	// Token received, add to context and proceed
	ctx := context.WithValue(r.Context(), OAuth2TokenKey{}, token)
	next.ServeHTTP(w, r.WithContext(ctx))
}

func requestDeviceCode(_ OAuth2DeviceConfig) *OAuth2DeviceCode {
	// In a real implementation, this would make a request to the device authorization endpoint
	// For now, return a mock response
	return &OAuth2DeviceCode{
		DeviceCode:              generateState(),
		UserCode:                "ABCD-1234",
		VerificationURI:         "https://auth.example.com/device",
		VerificationURIComplete: "https://auth.example.com/device?user_code=ABCD-1234",
		ExpiresIn:               1800,
		Interval:                5,
	}
}

func pollForDeviceToken(_ OAuth2DeviceConfig, _ string) (*OAuth2Token, error) {
	// In a real implementation, this would poll the token endpoint
	// For now, simulate pending status
	return nil, errors.New("authorization_pending")
}

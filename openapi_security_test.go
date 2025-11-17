package webfram

import (
	"testing"
)

// =============================================================================
// mapSecurityScheme Tests
// =============================================================================

func TestMapSecurityScheme_HTTPBearer(t *testing.T) {
	scheme := NewHTTPBearerSecurityScheme(&HTTPBearerSecuritySchemeOptions{
		Description:  "JWT Bearer Auth",
		BearerFormat: "JWT",
		Extensions: map[string]interface{}{
			"x-custom": "value",
		},
		Deprecated: true,
	})

	mapped := mapSecurityScheme(scheme)

	if mapped.Type != "http" {
		t.Errorf("Expected Type 'http', got %q", mapped.Type)
	}
	if mapped.Scheme != "bearer" {
		t.Errorf("Expected Scheme 'bearer', got %q", mapped.Scheme)
	}
	if mapped.BearerFormat != "JWT" {
		t.Errorf("Expected BearerFormat 'JWT', got %q", mapped.BearerFormat)
	}
	if mapped.Description != "JWT Bearer Auth" {
		t.Errorf("Expected Description 'JWT Bearer Auth', got %q", mapped.Description)
	}
	if !mapped.Deprecated {
		t.Error("Expected Deprecated to be true")
	}
	if mapped.Extensions["x-custom"] != "value" {
		t.Error("Expected custom extension to be preserved")
	}
}

func TestMapSecurityScheme_HTTPBasic(t *testing.T) {
	scheme := NewHTTPBasicSecurityScheme(&HTTPBasicSecuritySchemeOptions{
		Description: "Basic Auth",
	})

	mapped := mapSecurityScheme(scheme)

	if mapped.Type != "http" {
		t.Errorf("Expected Type 'http', got %q", mapped.Type)
	}
	if mapped.Scheme != "basic" {
		t.Errorf("Expected Scheme 'basic', got %q", mapped.Scheme)
	}
	if mapped.Description != "Basic Auth" {
		t.Errorf("Expected Description 'Basic Auth', got %q", mapped.Description)
	}
}

func TestMapSecurityScheme_HTTPDigest(t *testing.T) {
	scheme := NewHTTPDigestSecurityScheme(&HTTPDigestSecuritySchemeOptions{
		Description: "Digest Auth",
	})

	mapped := mapSecurityScheme(scheme)

	if mapped.Type != "http" {
		t.Errorf("Expected Type 'http', got %q", mapped.Type)
	}
	if mapped.Scheme != "digest" {
		t.Errorf("Expected Scheme 'digest', got %q", mapped.Scheme)
	}
}

func TestMapSecurityScheme_APIKey_Header(t *testing.T) {
	scheme := NewAPIKeySecurityScheme(&APIKeySecuritySchemeOptions{
		Name:        "X-API-Key",
		In:          "header",
		Description: "API Key Auth",
	})

	mapped := mapSecurityScheme(scheme)

	if mapped.Type != "apiKey" {
		t.Errorf("Expected Type 'apiKey', got %q", mapped.Type)
	}
	if mapped.Name != "X-API-Key" {
		t.Errorf("Expected Name 'X-API-Key', got %q", mapped.Name)
	}
	if mapped.In != "header" {
		t.Errorf("Expected In 'header', got %q", mapped.In)
	}
}

func TestMapSecurityScheme_APIKey_Query(t *testing.T) {
	scheme := NewAPIKeySecurityScheme(&APIKeySecuritySchemeOptions{
		Name: "api_key",
		In:   "query",
	})

	mapped := mapSecurityScheme(scheme)

	if mapped.In != "query" {
		t.Errorf("Expected In 'query', got %q", mapped.In)
	}
}

func TestMapSecurityScheme_APIKey_Cookie(t *testing.T) {
	scheme := NewAPIKeySecurityScheme(&APIKeySecuritySchemeOptions{
		Name: "session",
		In:   "cookie",
	})

	mapped := mapSecurityScheme(scheme)

	if mapped.In != "cookie" {
		t.Errorf("Expected In 'cookie', got %q", mapped.In)
	}
}

func TestMapSecurityScheme_MutualTLS(t *testing.T) {
	scheme := NewMutualTLSSecurityScheme(&MutualTLSSecuritySchemeOptions{
		Description: "mTLS Auth",
	})

	mapped := mapSecurityScheme(scheme)

	if mapped.Type != "mutualTLS" {
		t.Errorf("Expected Type 'mutualTLS', got %q", mapped.Type)
	}
	if mapped.Description != "mTLS Auth" {
		t.Errorf("Expected Description 'mTLS Auth', got %q", mapped.Description)
	}
}

func TestMapSecurityScheme_OpenIdConnect(t *testing.T) {
	scheme := NewOpenIdConnectSecurityScheme(&OpenIdConnectSecuritySchemeOptions{
		OpenIdConnectURL: "https://example.com/.well-known/openid-configuration",
		Description:      "OIDC Auth",
	})

	mapped := mapSecurityScheme(scheme)

	if mapped.Type != "openIdConnect" {
		t.Errorf("Expected Type 'openIdConnect', got %q", mapped.Type)
	}
	if mapped.OpenIdConnectURL != "https://example.com/.well-known/openid-configuration" {
		t.Errorf("Expected OpenIdConnectURL to be set, got %q", mapped.OpenIdConnectURL)
	}
}

func TestMapSecurityScheme_OAuth2(t *testing.T) {
	scheme := NewOAuth2SecurityScheme(&OAuth2SecuritySchemeOptions{
		Description: "OAuth2 Auth",
		Flows: []OAuthFlow{
			NewAuthorizationCodeOAuthFlow(&AuthorizationCodeOAuthFlowOptions{
				AuthorizationURL: "https://example.com/oauth/authorize",
				TokenURL:         "https://example.com/oauth/token",
				Scopes: map[string]string{
					"read": "Read access",
				},
			}),
		},
	})

	mapped := mapSecurityScheme(scheme)

	if mapped.Type != "oauth2" {
		t.Errorf("Expected Type 'oauth2', got %q", mapped.Type)
	}
	if mapped.Flows == nil {
		t.Fatal("Expected Flows to be set")
	}
	if mapped.Flows.AuthorizationCode == nil {
		t.Error("Expected AuthorizationCode flow to be set")
	}
}

// =============================================================================
// mapOAuthFlows Tests
// =============================================================================

func TestMapOAuthFlows_Implicit(t *testing.T) {
	flows := []OAuthFlow{
		NewImplicitOAuthFlow(&ImplicitOAuthFlowOptions{
			AuthorizationURL: "https://example.com/oauth/authorize",
			Scopes: map[string]string{
				"read": "Read access",
			},
			RefreshURL: "https://example.com/oauth/refresh",
			Extensions: map[string]interface{}{
				"x-custom": "value",
			},
		}),
	}

	mapped := mapOAuthFlows(flows)

	if mapped == nil {
		t.Fatal("Expected mapped flows to not be nil")
	}
	if mapped.Implicit == nil {
		t.Fatal("Expected Implicit flow to be set")
	}
	if mapped.Implicit.AuthorizationURL != "https://example.com/oauth/authorize" {
		t.Errorf("Expected AuthorizationURL to be set, got %q", mapped.Implicit.AuthorizationURL)
	}
	if mapped.Implicit.RefreshURL != "https://example.com/oauth/refresh" {
		t.Errorf("Expected RefreshURL to be set, got %q", mapped.Implicit.RefreshURL)
	}
	if mapped.Implicit.Scopes["read"] != "Read access" {
		t.Error("Expected read scope to be set")
	}
	if mapped.Implicit.Extensions["x-custom"] != "value" {
		t.Error("Expected custom extension to be preserved")
	}
}

func TestMapOAuthFlows_ClientCredentials(t *testing.T) {
	flows := []OAuthFlow{
		NewClientCredentialsOAuthFlow(&ClientCredentialsOAuthFlowOptions{
			TokenURL: "https://example.com/oauth/token",
			Scopes: map[string]string{
				"admin": "Admin access",
			},
		}),
	}

	mapped := mapOAuthFlows(flows)

	if mapped == nil {
		t.Fatal("Expected mapped flows to not be nil")
	}
	if mapped.ClientCredentials == nil {
		t.Fatal("Expected ClientCredentials flow to be set")
	}
	if mapped.ClientCredentials.TokenURL != "https://example.com/oauth/token" {
		t.Errorf("Expected TokenURL to be set, got %q", mapped.ClientCredentials.TokenURL)
	}
	if mapped.ClientCredentials.Scopes["admin"] != "Admin access" {
		t.Error("Expected admin scope to be set")
	}
}

func TestMapOAuthFlows_AuthorizationCode(t *testing.T) {
	flows := []OAuthFlow{
		NewAuthorizationCodeOAuthFlow(&AuthorizationCodeOAuthFlowOptions{
			AuthorizationURL: "https://example.com/oauth/authorize",
			TokenURL:         "https://example.com/oauth/token",
			Scopes: map[string]string{
				"read":  "Read access",
				"write": "Write access",
			},
			RefreshURL: "https://example.com/oauth/refresh",
		}),
	}

	mapped := mapOAuthFlows(flows)

	if mapped == nil {
		t.Fatal("Expected mapped flows to not be nil")
	}
	if mapped.AuthorizationCode == nil {
		t.Fatal("Expected AuthorizationCode flow to be set")
	}
	if mapped.AuthorizationCode.AuthorizationURL != "https://example.com/oauth/authorize" {
		t.Errorf("Expected AuthorizationURL to be set, got %q", mapped.AuthorizationCode.AuthorizationURL)
	}
	if mapped.AuthorizationCode.TokenURL != "https://example.com/oauth/token" {
		t.Errorf("Expected TokenURL to be set, got %q", mapped.AuthorizationCode.TokenURL)
	}
	if len(mapped.AuthorizationCode.Scopes) != 2 {
		t.Errorf("Expected 2 scopes, got %d", len(mapped.AuthorizationCode.Scopes))
	}
}

func TestMapOAuthFlows_DeviceAuthorization(t *testing.T) {
	flows := []OAuthFlow{
		NewDeviceAuthorizationOAuthFlow(&DeviceAuthorizationOAuthFlowOptions{
			DeviceAuthorizationURL: "https://example.com/oauth/device_authorize",
			TokenURL:               "https://example.com/oauth/token",
			Scopes: map[string]string{
				"device": "Device access",
			},
		}),
	}

	mapped := mapOAuthFlows(flows)

	if mapped == nil {
		t.Fatal("Expected mapped flows to not be nil")
	}
	if mapped.DeviceAuthorization == nil {
		t.Fatal("Expected DeviceAuthorization flow to be set")
	}
	if mapped.DeviceAuthorization.DeviceAuthorizationURL != "https://example.com/oauth/device_authorize" {
		t.Errorf("Expected DeviceAuthorizationURL to be set, got %q", mapped.DeviceAuthorization.DeviceAuthorizationURL)
	}
	if mapped.DeviceAuthorization.TokenURL != "https://example.com/oauth/token" {
		t.Errorf("Expected TokenURL to be set, got %q", mapped.DeviceAuthorization.TokenURL)
	}
}

func TestMapOAuthFlows_Multiple(t *testing.T) {
	flows := []OAuthFlow{
		NewImplicitOAuthFlow(&ImplicitOAuthFlowOptions{
			AuthorizationURL: "https://example.com/oauth/authorize",
			Scopes: map[string]string{
				"read": "Read access",
			},
		}),
		NewClientCredentialsOAuthFlow(&ClientCredentialsOAuthFlowOptions{
			TokenURL: "https://example.com/oauth/token",
			Scopes: map[string]string{
				"admin": "Admin access",
			},
		}),
		NewAuthorizationCodeOAuthFlow(&AuthorizationCodeOAuthFlowOptions{
			AuthorizationURL: "https://example.com/oauth/authorize",
			TokenURL:         "https://example.com/oauth/token",
			Scopes: map[string]string{
				"write": "Write access",
			},
		}),
		NewDeviceAuthorizationOAuthFlow(&DeviceAuthorizationOAuthFlowOptions{
			DeviceAuthorizationURL: "https://example.com/oauth/device_authorize",
			TokenURL:               "https://example.com/oauth/token",
			Scopes: map[string]string{
				"device": "Device access",
			},
		}),
	}

	mapped := mapOAuthFlows(flows)

	if mapped == nil {
		t.Fatal("Expected mapped flows to not be nil")
	}
	if mapped.Implicit == nil {
		t.Error("Expected Implicit flow to be set")
	}
	if mapped.ClientCredentials == nil {
		t.Error("Expected ClientCredentials flow to be set")
	}
	if mapped.AuthorizationCode == nil {
		t.Error("Expected AuthorizationCode flow to be set")
	}
	if mapped.DeviceAuthorization == nil {
		t.Error("Expected DeviceAuthorization flow to be set")
	}
}

func TestMapOAuthFlows_EmptyList(t *testing.T) {
	flows := []OAuthFlow{}

	mapped := mapOAuthFlows(flows)

	if mapped != nil {
		t.Error("Expected nil for empty flows list")
	}
}

func TestMapOAuthFlows_NilList(t *testing.T) {
	var flows []OAuthFlow

	mapped := mapOAuthFlows(flows)

	if mapped != nil {
		t.Error("Expected nil for nil flows list")
	}
}

// =============================================================================
// configureOpenAPI with SecuritySchemes Tests
// =============================================================================

func TestConfigureOpenAPI_WithSecuritySchemes(t *testing.T) {
	resetAppConfig()
	openAPIConfig = &OpenAPI{Enabled: true}

	cfg := &Config{
		OpenAPI: &OpenAPI{
			Enabled: true,
			Config: &OpenAPIConfig{
				Components: &Components{
					SecuritySchemes: map[string]SecurityScheme{
						"BearerAuth": NewHTTPBearerSecurityScheme(&HTTPBearerSecuritySchemeOptions{
							Description: "JWT Authentication",
						}),
						"ApiKeyAuth": NewAPIKeySecurityScheme(&APIKeySecuritySchemeOptions{
							Name: "X-API-Key",
							In:   "header",
						}),
					},
				},
			},
		},
	}

	configureOpenAPI(cfg)

	if openAPIConfig.internalConfig.Components.SecuritySchemes == nil {
		t.Fatal("Expected SecuritySchemes to be initialized")
	}

	if len(openAPIConfig.internalConfig.Components.SecuritySchemes) != 2 {
		t.Errorf("Expected 2 security schemes, got %d", len(openAPIConfig.internalConfig.Components.SecuritySchemes))
	}

	bearerScheme := openAPIConfig.internalConfig.Components.SecuritySchemes["BearerAuth"]
	if bearerScheme.SecurityScheme == nil {
		t.Error("Expected BearerAuth scheme to be set")
	}
	if bearerScheme.SecurityScheme.Type != "http" {
		t.Errorf("Expected BearerAuth type 'http', got %q", bearerScheme.SecurityScheme.Type)
	}

	apiKeyScheme := openAPIConfig.internalConfig.Components.SecuritySchemes["ApiKeyAuth"]
	if apiKeyScheme.SecurityScheme == nil {
		t.Error("Expected ApiKeyAuth scheme to be set")
	}
	if apiKeyScheme.SecurityScheme.Type != "apiKey" {
		t.Errorf("Expected ApiKeyAuth type 'apiKey', got %q", apiKeyScheme.SecurityScheme.Type)
	}
}

func TestConfigureOpenAPI_WithoutSecuritySchemes(t *testing.T) {
	resetAppConfig()
	openAPIConfig = &OpenAPI{Enabled: true}

	cfg := &Config{
		OpenAPI: &OpenAPI{
			Enabled: true,
			Config: &OpenAPIConfig{
				Components: &Components{
					SecuritySchemes: map[string]SecurityScheme{},
				},
			},
		},
	}

	configureOpenAPI(cfg)

	// Empty map should not create SecuritySchemes in internal config
	if openAPIConfig.internalConfig.Components.SecuritySchemes != nil {
		t.Error("Expected SecuritySchemes to not be initialized for empty map")
	}
}

func TestConfigureOpenAPI_NilComponents(t *testing.T) {
	resetAppConfig()
	openAPIConfig = &OpenAPI{Enabled: true}

	cfg := &Config{
		OpenAPI: &OpenAPI{
			Enabled: true,
			Config: &OpenAPIConfig{
				Components: nil,
			},
		},
	}

	configureOpenAPI(cfg)

	// Should not panic with nil Components
	if openAPIConfig.internalConfig.Components.SecuritySchemes != nil {
		t.Error("Expected SecuritySchemes to be nil when Components is nil")
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestSecuritySchemes_EndToEnd(t *testing.T) {
	resetAppConfig()

	// Configure app with security schemes
	Configure(&Config{
		OpenAPI: &OpenAPI{
			Enabled: true,
			Config: &OpenAPIConfig{
				Info: &Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
				Components: &Components{
					SecuritySchemes: map[string]SecurityScheme{
						"BasicAuth": NewHTTPBasicSecurityScheme(&HTTPBasicSecuritySchemeOptions{
							Description: "Basic Authentication",
						}),
						"BearerAuth": NewHTTPBearerSecurityScheme(&HTTPBearerSecuritySchemeOptions{
							Description:  "JWT Authentication",
							BearerFormat: "JWT",
						}),
						"ApiKeyAuth": NewAPIKeySecurityScheme(&APIKeySecuritySchemeOptions{
							Name:        "X-API-Key",
							In:          "header",
							Description: "API Key",
						}),
						"OAuth2Auth": NewOAuth2SecurityScheme(&OAuth2SecuritySchemeOptions{
							Description: "OAuth2",
							Flows: []OAuthFlow{
								NewAuthorizationCodeOAuthFlow(&AuthorizationCodeOAuthFlowOptions{
									AuthorizationURL: "https://example.com/oauth/authorize",
									TokenURL:         "https://example.com/oauth/token",
									Scopes: map[string]string{
										"read":  "Read access",
										"write": "Write access",
									},
								}),
							},
						}),
					},
				},
			},
		},
	})

	// Verify configuration
	if openAPIConfig == nil {
		t.Fatal("Expected openAPIConfig to be set")
	}

	if openAPIConfig.internalConfig.Components.SecuritySchemes == nil {
		t.Fatal("Expected SecuritySchemes to be initialized")
	}

	schemes := openAPIConfig.internalConfig.Components.SecuritySchemes
	if len(schemes) != 4 {
		t.Errorf("Expected 4 security schemes, got %d", len(schemes))
	}

	// Verify each scheme type
	schemeTypes := make(map[string]string)
	for name, scheme := range schemes {
		if scheme.SecurityScheme != nil {
			schemeTypes[name] = scheme.SecurityScheme.Type
		}
	}

	expectedTypes := map[string]string{
		"BasicAuth":  "http",
		"BearerAuth": "http",
		"ApiKeyAuth": "apiKey",
		"OAuth2Auth": "oauth2",
	}

	for name, expectedType := range expectedTypes {
		if actualType, ok := schemeTypes[name]; !ok {
			t.Errorf("Expected %s scheme to be present", name)
		} else if actualType != expectedType {
			t.Errorf("Expected %s type %q, got %q", name, expectedType, actualType)
		}
	}
}

func TestSecuritySchemes_WithAllOAuthFlows(t *testing.T) {
	resetAppConfig()

	Configure(&Config{
		OpenAPI: &OpenAPI{
			Enabled: true,
			Config: &OpenAPIConfig{
				Components: &Components{
					SecuritySchemes: map[string]SecurityScheme{
						"OAuth2Auth": NewOAuth2SecurityScheme(&OAuth2SecuritySchemeOptions{
							Flows: []OAuthFlow{
								NewImplicitOAuthFlow(&ImplicitOAuthFlowOptions{
									AuthorizationURL: "https://example.com/oauth/authorize",
									Scopes: map[string]string{
										"public": "Public access",
									},
								}),
								NewClientCredentialsOAuthFlow(&ClientCredentialsOAuthFlowOptions{
									TokenURL: "https://example.com/oauth/token",
									Scopes: map[string]string{
										"admin": "Admin access",
									},
								}),
								NewAuthorizationCodeOAuthFlow(&AuthorizationCodeOAuthFlowOptions{
									AuthorizationURL: "https://example.com/oauth/authorize",
									TokenURL:         "https://example.com/oauth/token",
									Scopes: map[string]string{
										"read":  "Read access",
										"write": "Write access",
									},
								}),
								NewDeviceAuthorizationOAuthFlow(&DeviceAuthorizationOAuthFlowOptions{
									DeviceAuthorizationURL: "https://example.com/oauth/device_authorize",
									TokenURL:               "https://example.com/oauth/token",
									Scopes: map[string]string{
										"device": "Device access",
									},
								}),
							},
						}),
					},
				},
			},
		},
	})

	scheme := openAPIConfig.internalConfig.Components.SecuritySchemes["OAuth2Auth"]
	if scheme.SecurityScheme == nil {
		t.Fatal("Expected OAuth2Auth scheme to be set")
	}

	flows := scheme.SecurityScheme.Flows
	if flows == nil {
		t.Fatal("Expected flows to be set")
	}

	// Verify all flow types are present
	if flows.Implicit == nil {
		t.Error("Expected Implicit flow to be set")
	}
	if flows.ClientCredentials == nil {
		t.Error("Expected ClientCredentials flow to be set")
	}
	if flows.AuthorizationCode == nil {
		t.Error("Expected AuthorizationCode flow to be set")
	}
	if flows.DeviceAuthorization == nil {
		t.Error("Expected DeviceAuthorization flow to be set")
	}
}

package security

type (
	Config struct {
		// AllowAnonymousAuth indicates whether anonymous (unauthenticated) access is allowed.
		AllowAnonymousAuth bool
		// APIKeyAuth configures API Key authentication settings.
		APIKeyAuth *APIKeyAuthConfig
		// BasicAuth configures Basic authentication settings.
		BasicAuth *BasicAuthConfig
		// BearerAuth configures Bearer authentication settings.
		BearerAuth *BearerAuthConfig
		// DigestAuth configures Digest authentication settings.
		DigestAuth *DigestAuthConfig
		// MutualTLSAuthConfig configures Mutual TLS authentication settings.
		MutualTLSAuth *MutualTLSAuthConfig
		// OAuth2AuthorizationCode configures OAuth2 Authorization Code flow settings.
		OAuth2AuthorizationCode *OAuth2AuthorizationCodeConfig
		// OAuth2ClientCredentials configures OAuth2 Client Credentials flow settings.
		OAuth2ClientCredentials *OAuth2ClientCredentialsConfig
		// OAuth2Device configures OAuth2 Device Code flow settings.
		OAuth2Device *OAuth2DeviceConfig
		// OAuth2Implicit configures OAuth2 Implicit flow settings.
		OAuth2Implicit *OAuth2ImplicitConfig
		// OpenIDConnectAuth configures OpenID Connect authentication settings.
		OpenIDConnectAuth *OpenIDConnectAuthConfig
	}
)

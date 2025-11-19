package main

import (
	"embed"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	app "github.com/bondowe/webfram"
	"github.com/bondowe/webfram/security"
	"github.com/google/uuid"
	"golang.org/x/text/language"
)

const (
	sseUpdateInterval = 5 * time.Second
	defaultClapCount  = 5
)

//go:generate go get -tool github.com/bondowe/webfram/cmd/webfram-i18n
//go:generate go tool webfram-i18n -languages "en,fr,es" -templates assets/templates -locales assets/locales

//go:embed all:assets
var assetsFS embed.FS

//nolint:golines // struct tags require specific formatting
type User struct {
	XMLName   xml.Name  `json:"-" xml:"user"` // Root element name for XML serialization
	ID        uuid.UUID `json:"id" xml:"id" form:"id"`
	Name      string    `json:"name" xml:"name" form:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"` //nolint:lll // struct tags must be on one line
	Email     string    `json:"email" xml:"email" form:"email" validate:"required,format=email" errmsg:"required=Email is required;format=Invalid email"`                    //nolint:lll // struct tags must be on one line
	Role      string    `json:"role" xml:"role" form:"role" validate:"enum=admin|user|guest" errmsg:"enum=Must be admin, user, or guest"`                                    //nolint:lll // struct tags must be on one line
	Birthdate time.Time `form:"birthdate" validate:"required" format:"2006-01-02"`
}

func loggingMiddleware(next app.Handler) app.Handler {
	return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		log.Printf("%s %s - %v", r.Method, r.URL.Path, duration)
	})
}

func main() {
	// Configure the application
	app.Configure(&app.Config{
		Telemetry: &app.Telemetry{
			Enabled:            true,
			UseDefaultRegistry: true,
			Addr:               ":8081", // Separate telemetry server. Defaults to main server if empty
		},
		Assets: &app.Assets{
			FS: assetsFS,
			Templates: &app.Templates{
				Dir: "assets/templates",
			},
			I18nMessages: &app.I18nMessages{
				Dir:                "assets/locales",
				SupportedLanguages: []string{"en-GB", "fr-FR", "es-ES"},
			},
		},
		JSONPCallbackParamName: "callback", // Enable JSONP support
		OpenAPI: &app.OpenAPI{
			Enabled: true,
			Config:  getOpenAPIConfig(),
		},
	})

	// Global middleware
	app.Use(loggingMiddleware)

	// Create mux
	mux := app.NewServeMux()

	// Routes
	mux.HandleFunc("GET /", func(w app.ResponseWriter, r *app.Request) {
		user := User{Name: "John Doe", Email: "john@example.com"}
		err := w.HTML(r.Context(), "home/index", &user)
		if err != nil {
			w.Error(http.StatusInternalServerError, err.Error())
		}
	})

	// JSON endpoint with JSONP support
	mux.HandleFunc("GET /users", func(w app.ResponseWriter, r *app.Request) {
		users := []User{
			{ID: uuid.New(), Name: "John Doe", Email: "john@example.com"},
			{ID: uuid.New(), Name: "Jane Smith", Email: "jane@example.com"},
		}
		if err := w.JSON(r.Context(), users); err != nil {
			w.Error(http.StatusInternalServerError, err.Error())
		}
	}).UseSecurity(security.Config{
		BasicAuth: &security.BasicAuthConfig{
			Authenticator: func(username, password string) bool {
				return username == "admin" && password == "password"
			},
		},
	}).OpenAPIOperation(app.OperationConfig{
		OperationID: "listUsers",
		Summary:     "List all users",
		Tags:        []string{"User Service"},
		Security: []map[string][]string{
			{"ApiKeyAuth": {}},
		},
		Responses: map[string]app.Response{
			"200": {
				Description: "List of users",
				Content: map[string]app.TypeInfo{
					"application/json": {TypeHint: &[]User{}},
				},
			},
		},
	})

	// JSON Sequence endpoint
	mux.HandleFunc("GET /users/json-seq", func(w app.ResponseWriter, r *app.Request) {
		users := []User{
			{ID: uuid.New(), Name: "John Doe", Email: "john@example.com"},
			{ID: uuid.New(), Name: "Jane Smith", Email: "jane@example.com"},
			{ID: uuid.New(), Name: "Alice Johnson", Email: "alice@example.com"},
			{ID: uuid.New(), Name: "Bob Brown", Email: "bob@example.com"},
			{ID: uuid.New(), Name: "Charlie Davis", Email: "charlie@example.com"},
			{ID: uuid.New(), Name: "Diana Evans", Email: "diana@example.com"},
		}
		if err := w.JSONSeq(r.Context(), users); err != nil {
			w.Error(http.StatusInternalServerError, err.Error())
		}
	}).OpenAPIOperation(app.OperationConfig{
		OperationID: "listUsersSeq",
		Summary:     "List all users in JSON Sequence format",
		Tags:        []string{"User Service"},
		Responses: map[string]app.Response{
			"200": {
				Description: "List of users in JSON Sequence format",
				Content: map[string]app.TypeInfo{
					"application/json-seq": {TypeHint: &User{}},
				},
			},
		},
	})

	// Create user with JSON
	mux.HandleFunc("POST /api/users/json", func(w app.ResponseWriter, r *app.Request) {
		user, valErrors, err := app.BindJSON[User](r, true)

		if err != nil {
			w.Error(http.StatusBadRequest, err.Error())
			return
		}

		if valErrors.Any() {
			w.WriteHeader(http.StatusBadRequest)
			if jsonErr := w.JSON(r.Context(), valErrors); jsonErr != nil {
				w.Error(http.StatusInternalServerError, jsonErr.Error())
			}
			return
		}

		user.ID = uuid.New()
		w.WriteHeader(http.StatusCreated)
		if jsonErr := w.JSON(r.Context(), user); jsonErr != nil {
			w.Error(http.StatusInternalServerError, jsonErr.Error())
		}
	}).OpenAPIOperation(app.OperationConfig{
		OperationID: "createUser",
		Summary:     "Create a new user",
		Tags:        []string{"User Service"},
		RequestBody: &app.RequestBody{
			Required: true,
			Content: map[string]app.TypeInfo{
				"application/json": {
					TypeHint: &User{},
					Examples: map[string]app.Example{
						"Standard": {
							Summary: "Standard user creation",
							DataValue: User{
								ID:        uuid.New(),
								Name:      "Alice Johnson",
								Role:      "guest",
								Email:     "alice@example.com",
								Birthdate: time.Date(1985, time.May, 15, 0, 0, 0, 0, time.UTC),
							},
						},
					},
				},
			},
		},
		Responses: map[string]app.Response{
			"201": {
				Description: "User created",
				Content: map[string]app.TypeInfo{
					"application/json": {TypeHint: &User{}},
				},
			},
			"400": {Description: "Validation error"},
		},
	})

	// Update user with JSON Patch
	mux.HandleFunc("PATCH /users/{id}", func(w app.ResponseWriter, r *app.Request) {
		id := r.PathValue("id")

		// Fetch existing user
		user := User{
			ID:    uuid.MustParse(id),
			Name:  "John Doe",
			Email: "john@example.com",
			Role:  "user",
		}

		// Apply JSON Patch with validation
		valErrors, err := app.PatchJSON(r, &user, true)
		if err != nil {
			if errors.Is(err, app.ErrMethodNotAllowed) {
				w.Error(http.StatusMethodNotAllowed, err.Error())
				return
			}
			w.Error(http.StatusBadRequest, err.Error())
			return
		}

		if len(valErrors) > 0 {
			w.WriteHeader(http.StatusBadRequest)
			if jsonErr := w.JSON(r.Context(), app.ValidationErrors{Errors: valErrors}); jsonErr != nil {
				w.Error(http.StatusInternalServerError, jsonErr.Error())
			}
			return
		}

		if jsonErr := w.JSON(r.Context(), user); jsonErr != nil {
			w.Error(http.StatusInternalServerError, jsonErr.Error())
		}
	})

	// XML endpoint demonstrating XML schema generation with custom tags
	mux.HandleFunc("GET /users/xml", func(w app.ResponseWriter, r *app.Request) {
		users := []User{
			{ID: uuid.New(), Name: "John Doe", Email: "john@example.com", Role: "admin"},
			{ID: uuid.New(), Name: "Jane Smith", Email: "jane@example.com", Role: "user"},
		}
		// Use XMLArray to wrap the slice with a root element for valid XML
		if err := w.XMLArray(users, "users"); err != nil {
			w.Error(http.StatusInternalServerError, err.Error())
		}
	}).OpenAPIOperation(app.OperationConfig{
		OperationID: "listUsersXML",
		Summary:     "List all users in XML format",
		Description: "Returns a list of users with XML serialization that respects custom xml tags (attributes vs elements). " +
			"Uses XMLArray() to wrap the slice in a <users> root element. Each item uses its XMLName.",
		Tags: []string{"User Service"},
		Responses: map[string]app.Response{
			"200": {
				Description: "List of users in XML format. " +
					"Note: id, email, and role are serialized as XML attributes, " +
					"while name and birthdate are serialized as XML elements.",
				Content: map[string]app.TypeInfo{
					"application/xml": {
						TypeHint:    []User{},
						XMLRootName: "users",
					},
				},
			},
		},
	})

	// SSE endpoint for real-time updates
	mux.Handle("GET /events", app.SSE(
		func() app.SSEPayload {
			return app.SSEPayload{
				ID:       uuid.New().String(),
				Event:    "TIME_UPDATE",
				Comments: []string{"Server time update"},
				Data:     fmt.Sprintf("Current server time: %s", time.Now().Format(time.RFC3339)),
			}
		},
		func() {
			log.Println("Client disconnected from events stream")
		},
		func(err error) {
			log.Printf("SSE error: %v\n", err)
		},
		sseUpdateInterval,
		nil,
	)).OpenAPIOperation(app.OperationConfig{
		OperationID: "timeEvents",
		Summary:     "Stream server time updates via SSE",
		Tags:        []string{"Time Service"},
		Responses: map[string]app.Response{
			"200": {
				Description: "SSE stream of time updates",
				Content: map[string]app.TypeInfo{
					// No need to set TypeHint for SSE, this defaults to &app.SSEPayload{}
					"text/event-stream": {},
				},
			},
		},
	})

	// i18n example
	mux.HandleFunc("GET /greeting", func(w app.ResponseWriter, r *app.Request) {
		printer := app.GetI18nPrinter(language.Spanish)
		msg := printer.Sprintf("Welcome to %s! Clap %d times.", "WebFram", defaultClapCount)
		if err := w.JSON(r.Context(), map[string]string{"message": msg}); err != nil {
			w.Error(http.StatusInternalServerError, err.Error())
		}
	})

	mux.HandleFunc("GET /xml", func(w app.ResponseWriter, r *app.Request) {
		w.ServeFile(r, "public/sample.xml", &app.ServeFileOptions{
			Inline:   false,
			Filename: "custom-name.xml",
		})
	})

	mux.HandleFunc("GET /js", func(w app.ResponseWriter, r *app.Request) {
		w.ServeFileFS(r, assetsFS, "assets/js/main.js", &app.ServeFileOptions{
			Inline:   true,
			Filename: "main-01.js",
		})
	})

	// Start server
	app.ListenAndServe(":8080", mux, nil)
}

func getOpenAPIConfig() *app.OpenAPIConfig {
	return &app.OpenAPIConfig{
		Info: &app.Info{
			Title:       "WebFram Example API",
			Summary:     "An example API demonstrating WebFram features.",
			Description: "This is an example API documentation generated by WebFram.",
			Version:     "1.0.0",
			Contact: &app.Contact{
				Name:  "WebFram Support",
				Email: "support@webfram.io",
				URL:   "https://webfram.io",
			},
			License: &app.License{
				Name:       "MIT License",
				Identifier: "MIT",
			},
		},
		Tags: []app.Tag{
			{
				Name:        "User Service",
				Summary:     "User Service",
				Description: "Operations related to users",
			},
			{
				Name:        "ProductService",
				Summary:     "Product Service",
				Description: "Operations related to products",
			},
			{
				Name:        "Time Service",
				Summary:     "Time Service",
				Description: "Operations related to time updates",
			},
		},
		// Security: []map[string][]string{
		// 	{
		// 		"BasicAuth": {},
		// 	},
		// },
		Components: &app.Components{
			SecuritySchemes: map[string]app.SecurityScheme{
				"BasicAuth": app.NewHTTPBasicSecurityScheme(&app.HTTPBasicSecuritySchemeOptions{
					Description: "HTTP Basic Authentication",
				}),
				"DigestAuth": app.NewHTTPDigestSecurityScheme(&app.HTTPDigestSecuritySchemeOptions{
					Description: "HTTP Digest Authentication",
				}),
				"ApiKeyAuth": app.NewAPIKeySecurityScheme(&app.APIKeySecuritySchemeOptions{
					Name:        "X-API-Key",
					In:          "header",
					Description: "API Key Authentication",
				}),
				"BearerAuth": app.NewHTTPBearerSecurityScheme(&app.HTTPBearerSecuritySchemeOptions{
					Description:  "HTTP Bearer Authentication",
					BearerFormat: "JWT",
				}),
				"OAuth2Auth": app.NewOAuth2SecurityScheme(&app.OAuth2SecuritySchemeOptions{
					Description: "OAuth2 Authentication",
					Flows: []app.OAuthFlow{
						app.NewAuthorizationCodeOAuthFlow(&app.AuthorizationCodeOAuthFlowOptions{
							AuthorizationURL: "https://example.com/oauth/authorize",
							TokenURL:         "https://example.com/oauth/token",
							RefreshURL:       "https://example.com/oauth/refresh",
							Scopes: map[string]string{
								"read":  "Read access",
								"write": "Write access",
							},
						}),
						app.NewClientCredentialsOAuthFlow(&app.ClientCredentialsOAuthFlowOptions{
							TokenURL:   "https://example.com/oauth/token",
							RefreshURL: "https://example.com/oauth/refresh",
							Scopes: map[string]string{
								"admin": "Admin access",
							},
						}),
						app.NewImplicitOAuthFlow(&app.ImplicitOAuthFlowOptions{
							AuthorizationURL: "https://example.com/oauth/authorize",
							RefreshURL:       "https://example.com/oauth/refresh",
							Scopes: map[string]string{
								"public": "Public access",
							},
						}),
						app.NewDeviceAuthorizationOAuthFlow(&app.DeviceAuthorizationOAuthFlowOptions{
							DeviceAuthorizationURL: "https://example.com/oauth/device_authorize",
							TokenURL:               "https://example.com/oauth/token",
							RefreshURL:             "https://example.com/oauth/refresh",
							Scopes: map[string]string{
								"device": "Device access",
							},
						}),
					},
				}),
				"OpenIDConnect": app.NewOpenIdConnectSecurityScheme(&app.OpenIdConnectSecuritySchemeOptions{
					Description:      "OpenID Connect Authentication",
					OpenIdConnectURL: "https://example.com/.well-known/openid-configuration",
				}),
			},
		},
		Servers: []app.Server{
			{
				URL:         "http://localhost:8080",
				Description: "Local development server",
			},
			{
				URL:         "http://prod-site.com/api/v1",
				Description: "Production server",
			},
		},
	}
}

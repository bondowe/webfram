package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"time"

	app "github.com/bondowe/webfram"
	"github.com/bondowe/webfram/openapi"
	"github.com/google/uuid"
	"golang.org/x/text/language"
)

//go:embed assets
var assetsFS embed.FS

type User struct {
	ID        uuid.UUID `json:"id" xml:"id" form:"id"`
	Name      string    `json:"name" xml:"name" form:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"`
	Email     string    `json:"email" xml:"email" form:"email" validate:"required,format=email" errmsg:"required=Email is required;format=Invalid email"`
	Role      string    `json:"role" xml:"role" form:"role" validate:"enum=admin|user|guest" errmsg:"enum=Must be admin, user, or guest"`
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
		Assets: &app.Assets{
			FS: assetsFS,
			Templates: &app.Templates{
				Dir: "assets/templates",
			},
			I18nMessages: &app.I18nMessages{
				Dir: "assets/locales",
			},
		},
		JSONPCallbackParamName: "callback", // Enable JSONP support
		OpenAPI: &app.OpenAPI{
			EndpointEnabled: true,
			Config:          getOpenAPIConfig(),
		},
	})

	// Global middleware
	app.Use(loggingMiddleware)

	// Create mux
	mux := app.NewServeMux()

	// Routes
	mux.HandleFunc("GET /", func(w app.ResponseWriter, r *app.Request) {
		user := User{Name: "John Doe", Email: "john@example.com"}
		err := w.HTML("home/index", &user)
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
		w.JSON(users)
	}).WithAPIConfig(&app.APIConfig{
		OperationID: "listUsers",
		Summary:     "List all users",
		Tags:        []string{"Users"},
		Responses: map[string]app.Response{
			"200": {
				Description: "List of users",
				Content: map[string]app.TypeInfo{
					"application/json": {TypeHint: &[]User{}},
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
			w.JSON(valErrors)
			return
		}

		user.ID = uuid.New()
		w.WriteHeader(http.StatusCreated)
		w.JSON(user)
	}).WithAPIConfig(&app.APIConfig{
		OperationID: "createUser",
		Summary:     "Create a new user",
		Tags:        []string{"Users"},
		RequestBody: &app.RequestBody{
			Required: true,
			Content: map[string]app.TypeInfo{
				"application/json": {TypeHint: &User{}},
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
			if err == app.ErrMethodNotAllowed {
				w.Error(http.StatusMethodNotAllowed, err.Error())
				return
			}
			w.Error(http.StatusBadRequest, err.Error())
			return
		}

		if len(valErrors) > 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.JSON(app.ValidationErrors{Errors: valErrors})
			return
		}

		w.JSON(user)
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
		5*time.Second,
		nil,
	))

	// i18n example
	mux.HandleFunc("GET /greeting", func(w app.ResponseWriter, r *app.Request) {
		printer := app.GetI18nPrinter(language.Spanish)
		msg := printer.Sprintf("Welcome to %s! Clap %d times.", "WebFram", 5)
		w.JSON(map[string]string{"message": msg})
	})

	// Start server
	log.Println("Server starting on :8080")
	log.Println("OpenAPI docs: http://localhost:8080/openapi.json")
	app.ListenAndServe(":8080", mux, nil)
}

func getOpenAPIConfig() *openapi.Config {
	return &openapi.Config{
		Info: &openapi.Info{
			Title:       "WebFram Example API",
			Summary:     "An example API demonstrating WebFram features.",
			Description: "This is an example API documentation generated by WebFram.",
			Version:     "1.0.0",
		},
		Servers: []openapi.Server{
			{
				URL:         "http://localhost:8080",
				Description: "Local development server",
			},
		},
	}
}

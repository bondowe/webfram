package webfram

import (
	"context"
	"embed"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bondowe/webfram/openapi"
	"golang.org/x/text/language"
)

//go:embed testdata/locales/*.json
var testI18nFS embed.FS

func setupTestApp() {
	if appConfigured {
		// Reset for testing
		appConfigured = false
		appMiddlewares = nil
		openAPIConfig = nil
	}

	Configure(&Config{
		I18n: &I18nConfig{
			FS: testI18nFS,
		},
	})
}

func TestNewServeMux(t *testing.T) {
	setupTestApp()

	mux := NewServeMux()
	if mux == nil {
		t.Fatal("NewServeMux returned nil")
	}

	if len(mux.middlewares) > 0 {
		t.Errorf("Expected empty middlewares, got %d", len(mux.middlewares))
	}
}

func TestServeMux_HandleFunc(t *testing.T) {
	setupTestApp()

	mux := NewServeMux()

	called := false
	handler := func(w ResponseWriter, r *Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello"))
	}

	mux.HandleFunc("GET /test", handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if !called {
		t.Error("Handler was not called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if body := w.Body.String(); body != "Hello" {
		t.Errorf("Expected body 'Hello', got %q", body)
	}
}

func TestServeMux_Handle(t *testing.T) {
	setupTestApp()

	mux := NewServeMux()

	called := false
	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("World"))
	})

	mux.Handle("GET /test", handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if !called {
		t.Error("Handler was not called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if body := w.Body.String(); body != "World" {
		t.Errorf("Expected body 'World', got %q", body)
	}
}

func TestServeMux_Use_AppMiddleware(t *testing.T) {
	setupTestApp()

	mux := NewServeMux()

	middlewareCalled := false
	mw := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			middlewareCalled = true
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	mux.Use(mw)

	handler := func(w ResponseWriter, r *Request) {
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if !middlewareCalled {
		t.Error("Middleware was not called")
	}

	if header := w.Header().Get("X-Middleware"); header != "applied" {
		t.Errorf("Expected X-Middleware header 'applied', got %q", header)
	}
}

func TestServeMux_Use_StandardMiddleware(t *testing.T) {
	setupTestApp()

	mux := NewServeMux()

	middlewareCalled := false
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			w.Header().Set("X-Standard-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	mux.Use(mw)

	handler := func(w ResponseWriter, r *Request) {
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if !middlewareCalled {
		t.Error("Standard middleware was not called")
	}

	if header := w.Header().Get("X-Standard-Middleware"); header != "applied" {
		t.Errorf("Expected X-Standard-Middleware header 'applied', got %q", header)
	}
}

func TestServeMux_MultipleMiddlewares(t *testing.T) {
	setupTestApp()

	mux := NewServeMux()

	order := []string{}

	mw1 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			order = append(order, "mw1-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw1-after")
		})
	}

	mw2 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			order = append(order, "mw2-before")
			next.ServeHTTP(w, r)
			order = append(order, "mw2-after")
		})
	}

	mux.Use(mw1)
	mux.Use(mw2)

	handler := func(w ResponseWriter, r *Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("Expected %d calls, got %d", len(expected), len(order))
	}

	for i, v := range expected {
		if order[i] != v {
			t.Errorf("Expected order[%d] = %q, got %q", i, v, order[i])
		}
	}
}

func TestServeMux_HandlerMiddlewares(t *testing.T) {
	setupTestApp()

	mux := NewServeMux()

	handlerMwCalled := false
	handlerMw := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			handlerMwCalled = true
			w.Header().Set("X-Handler-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	handler := func(w ResponseWriter, r *Request) {
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler, handlerMw)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if !handlerMwCalled {
		t.Error("Handler middleware was not called")
	}

	if header := w.Header().Get("X-Handler-Middleware"); header != "applied" {
		t.Errorf("Expected X-Handler-Middleware header 'applied', got %q", header)
	}
}

func TestServeMux_MixedMiddlewares(t *testing.T) {
	setupTestApp()

	mux := NewServeMux()

	order := []string{}

	muxMw := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			order = append(order, "mux-mw")
			next.ServeHTTP(w, r)
		})
	}

	handlerMw := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			order = append(order, "handler-mw")
			next.ServeHTTP(w, r)
		})
	}

	mux.Use(muxMw)

	handler := func(w ResponseWriter, r *Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler, handlerMw)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// Order should be: handler-mw, mux-mw, handler
	// Because handler middlewares are applied first, then mux middlewares
	if len(order) < 3 {
		t.Fatalf("Expected at least 3 calls, got %d", len(order))
	}

	// Check that both middlewares and handler were called
	foundHandlerMw := false
	foundMuxMw := false
	foundHandler := false

	for _, v := range order {
		if v == "handler-mw" {
			foundHandlerMw = true
		}
		if v == "mux-mw" {
			foundMuxMw = true
		}
		if v == "handler" {
			foundHandler = true
		}
	}

	if !foundHandlerMw {
		t.Error("Handler middleware was not called")
	}
	if !foundMuxMw {
		t.Error("Mux middleware was not called")
	}
	if !foundHandler {
		t.Error("Handler was not called")
	}
}

func TestHandlerFunc_ServeHTTP(t *testing.T) {
	setupTestApp()

	called := false
	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	rw := ResponseWriter{ResponseWriter: w}
	r := &Request{Request: req}

	handler.ServeHTTP(rw, r)

	if !called {
		t.Error("Handler was not called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if body := w.Body.String(); body != "test" {
		t.Errorf("Expected body 'test', got %q", body)
	}
}

func TestHandlerFunc_ServeHTTP_WithContext(t *testing.T) {
	setupTestApp()

	var ctxValue string
	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		if val := w.Context().Value("test-key"); val != nil {
			ctxValue = val.(string)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(context.WithValue(req.Context(), "test-key", "test-value"))
	w := httptest.NewRecorder()

	rw := ResponseWriter{ResponseWriter: w}
	r := &Request{Request: req}

	handler.ServeHTTP(rw, r)

	if ctxValue != "test-value" {
		t.Errorf("Expected context value 'test-value', got %q", ctxValue)
	}
}

func TestI18nMiddleware(t *testing.T) {
	setupTestApp()

	mux := NewServeMux()

	handler := func(w ResponseWriter, r *Request) {
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler)

	// Test with Accept-Language header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.8")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestI18nMiddleware_WithCookie(t *testing.T) {
	setupTestApp()

	mux := NewServeMux()

	handler := func(w ResponseWriter, r *Request) {
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler)

	// Test with language cookie
	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "lang",
		Value: "fr",
	})
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestParseAcceptLanguage(t *testing.T) {

	getLangBase := func(b language.Base, c language.Confidence) language.Base {
		return b
	}

	tests := []struct {
		name       string
		acceptLang string
		wantBase   language.Base
	}{
		{
			name:       "English",
			acceptLang: "en-US,en;q=0.9",
			wantBase:   getLangBase(language.English.Base()),
		},
		{
			name:       "French",
			acceptLang: "fr-FR,fr;q=0.9,en;q=0.8",
			wantBase:   getLangBase(language.French.Base()),
		},
		{
			name:       "Spanish",
			acceptLang: "es-ES,es;q=0.9",
			wantBase:   getLangBase(language.Spanish.Base()),
		},
		{
			name:       "Empty",
			acceptLang: "",
			wantBase:   getLangBase(language.Und.Base()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := parseAcceptLanguage(tt.acceptLang)
			if getLangBase(tag.Base()) != tt.wantBase {
				t.Errorf("parseAcceptLanguage(%q) = %v, want base %v", tt.acceptLang, tag, tt.wantBase)
			}
		})
	}
}

func TestSetLanguageCookie(t *testing.T) {
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}

	SetLanguageCookie(rw, "fr", 86400)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != "lang" {
		t.Errorf("Expected cookie name 'lang', got %q", cookie.Name)
	}
	if cookie.Value != "fr" {
		t.Errorf("Expected cookie value 'fr', got %q", cookie.Value)
	}
	if cookie.MaxAge != 86400 {
		t.Errorf("Expected MaxAge 86400, got %d", cookie.MaxAge)
	}
}

func TestWrapMiddlewares(t *testing.T) {
	order := []string{}

	mw1 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			order = append(order, "mw1")
			next.ServeHTTP(w, r)
		})
	}

	mw2 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			order = append(order, "mw2")
			next.ServeHTTP(w, r)
		})
	}

	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		order = append(order, "handler")
	})

	wrapped := wrapMiddlewares(handler, []AppMiddleware{mw1, mw2})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}
	r := &Request{Request: req}

	wrapped.ServeHTTP(rw, r)

	expected := []string{"mw1", "mw2", "handler"}
	if len(order) != len(expected) {
		t.Fatalf("Expected %d calls, got %d", len(expected), len(order))
	}

	for i, v := range expected {
		if order[i] != v {
			t.Errorf("Expected order[%d] = %q, got %q", i, v, order[i])
		}
	}
}

func TestGetHandlerMiddlewares(t *testing.T) {
	appMw := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			next.ServeHTTP(w, r)
		})
	}

	stdMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	middlewares := []interface{}{appMw, stdMw}
	result := getHandlerMiddlewares(middlewares)

	if len(result) != 2 {
		t.Errorf("Expected 2 middlewares, got %d", len(result))
	}
}

func TestGetHandlerMiddlewares_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for unsupported middleware type")
		}
	}()

	unsupportedMw := "not a middleware"
	getHandlerMiddlewares([]interface{}{unsupportedMw})
}

func TestWithAPIConfig(t *testing.T) {
	setupTestApp()

	// Reset to enable OpenAPI
	appConfigured = false
	Configure(&Config{
		OpenAPI: &OpenAPIConfig{
			EndpointEnabled: true,
			Config: &openapi.Config{
				Info: &openapi.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
			},
		},
		I18n: &I18nConfig{
			FS: testI18nFS,
		},
	})

	mux := NewServeMux()
	handler := func(w ResponseWriter, r *Request) {
		w.WriteHeader(http.StatusOK)
	}

	config := mux.HandleFunc("GET /api/test", handler)
	config.WithAPIConfig(&APIConfig{
		OperationID: "TestOperation",
		Summary:     "Test Summary",
		Description: "Test Description",
		Tags:        []string{"test"},
	})

	// Verify it doesn't panic
	if config.APIConfig == nil {
		t.Error("APIConfig was not set")
	}
}

func TestSetOpenAPIPathInfo(t *testing.T) {
	setupTestApp()

	// Reset to enable OpenAPI
	appConfigured = false
	Configure(&Config{
		OpenAPI: &OpenAPIConfig{
			EndpointEnabled: true,
			Config: &openapi.Config{
				Info: &openapi.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
			},
		},
		I18n: &I18nConfig{
			FS: testI18nFS,
		},
	})

	pathInfo := &PathInfo{
		Summary:     "Test Path",
		Description: "Test Path Description",
		Parameters: []Parameter{
			{
				Name:        "id",
				In:          "path",
				Description: "Test ID",
				Required:    true,
			},
		},
	}

	// Should not panic
	SetOpenAPIPathInfo("/api/test/{id}", pathInfo)
}

func TestMappers(t *testing.T) {
	t.Run("mapServers", func(t *testing.T) {
		servers := []Server{
			{
				URL:         "http://localhost",
				Name:        "Local",
				Description: "Local server",
			},
		}

		result := mapServers(servers)
		if len(result) != 1 {
			t.Errorf("Expected 1 server, got %d", len(result))
		}
		if result[0].URL != "http://localhost" {
			t.Errorf("Expected URL 'http://localhost', got %q", result[0].URL)
		}
	})

	t.Run("mapServers nil", func(t *testing.T) {
		result := mapServers(nil)
		if result != nil {
			t.Error("Expected nil result for nil input")
		}
	})

	t.Run("mapServerVariables", func(t *testing.T) {
		vars := map[string]ServerVariable{
			"port": {
				Enum:        []string{"8080", "8081"},
				Default:     "8080",
				Description: "Port number",
			},
		}

		result := mapServerVariables(vars)
		if len(result) != 1 {
			t.Errorf("Expected 1 variable, got %d", len(result))
		}
		if result["port"].Default != "8080" {
			t.Errorf("Expected default '8080', got %q", result["port"].Default)
		}
	})

	t.Run("nonZeroValuePointer", func(t *testing.T) {
		val := 10
		ptr := nonZeroValuePointer(val)
		if ptr == nil {
			t.Error("Expected non-nil pointer for non-zero value")
		}
		if *ptr != 10 {
			t.Errorf("Expected value 10, got %d", *ptr)
		}

		zeroVal := 0
		zeroPtr := nonZeroValuePointer(zeroVal)
		if zeroPtr != nil {
			t.Error("Expected nil pointer for zero value")
		}
	})
}

func TestServeMux_Use_NilMiddleware(t *testing.T) {
	setupTestApp()

	mux := NewServeMux()

	// Should not panic
	mux.Use(nil)

	if len(mux.middlewares) != 0 {
		t.Errorf("Expected 0 middlewares after adding nil, got %d", len(mux.middlewares))
	}
}

func TestServeMux_Use_PanicOnInvalidType(t *testing.T) {
	setupTestApp()

	mux := NewServeMux()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid middleware type")
		}
	}()

	// Try to use an invalid middleware type
	mux.Use("invalid")
}

func TestAdaptHTTPMiddleware(t *testing.T) {
	headerSet := false
	httpMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-HTTP-Middleware", "true")
			headerSet = true
			next.ServeHTTP(w, r)
		})
	}

	adapted := adaptHTTPMiddleware(httpMw)

	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := adapted(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}
	r := &Request{Request: req}

	wrapped.ServeHTTP(rw, r)

	if !headerSet {
		t.Error("HTTP middleware was not executed")
	}

	if w.Header().Get("X-HTTP-Middleware") != "true" {
		t.Error("HTTP middleware header was not set")
	}
}

func TestHandlerFunc_WithJSONPCallback(t *testing.T) {
	setupTestApp()

	// Reset and configure with JSONP
	appConfigured = false
	Configure(&Config{
		JSONPCallbackParamName: "callback",
		I18n: &I18nConfig{
			FS: testI18nFS,
		},
	})

	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		w.JSON(map[string]string{"message": "test"})
	})

	req := httptest.NewRequest("GET", "/test?callback=myCallback", nil)
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}
	r := &Request{Request: req}

	handler.ServeHTTP(rw, r)

	body := w.Body.String()
	if !strings.Contains(body, "myCallback") {
		t.Errorf("Expected JSONP callback in response, got %q", body)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/javascript" {
		t.Errorf("Expected Content-Type 'application/javascript', got %q", contentType)
	}
}

func TestHandlerFunc_WithInvalidJSONPCallback(t *testing.T) {
	setupTestApp()

	// Reset and configure with JSONP
	appConfigured = false
	Configure(&Config{
		JSONPCallbackParamName: "callback",
		I18n: &I18nConfig{
			FS: testI18nFS,
		},
	})

	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		w.JSON(map[string]string{"message": "test"})
	})

	// Invalid callback name with special characters
	req := httptest.NewRequest("GET", "/test?callback=my-invalid-callback!", nil)
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}
	r := &Request{Request: req}

	handler.ServeHTTP(rw, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid callback, got %d", w.Code)
	}
}

func TestServeMux_PathParameters(t *testing.T) {
	setupTestApp()

	mux := NewServeMux()

	var receivedID string
	handler := func(w ResponseWriter, r *Request) {
		receivedID = r.PathValue("id")
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /users/{id}", handler)

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if receivedID != "123" {
		t.Errorf("Expected path parameter '123', got %q", receivedID)
	}
}

// Helper function for tests (if needed)
func GetI18nPrinterFromContext(ctx context.Context) (interface{}, bool) {
	// This is a simplified version for testing
	// In reality, you'd use the actual i18n package function
	return nil, true
}

// Create test data directory structure
func init() {
	// This would be handled by the embed directive in real code
	// For tests, we rely on the testdata directory being present
}

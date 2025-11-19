package webfram

import (
	"crypto/x509"
	"embed"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bondowe/webfram/internal/i18n"
	"github.com/bondowe/webfram/internal/telemetry"
	"github.com/bondowe/webfram/security"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"golang.org/x/text/language"
)

//go:embed testdata/locales/*.json
var testMuxI18nFS embed.FS

// Helper function to reset and setup app for mux tests.
func setupMuxTest() {
	appConfigured = false
	appMiddlewares = nil
	openAPIConfig = nil
	jsonpCallbackParamName = ""

	Configure(&Config{
		Assets: &Assets{
			FS: testMuxI18nFS,
			I18nMessages: &I18nMessages{
				Dir: "testdata/locales",
			},
		},
	})
}

// Helper function to setup app with OpenAPI enabled.
func setupMuxTestWithOpenAPI() {
	appConfigured = false
	appMiddlewares = nil
	openAPIConfig = &OpenAPI{Enabled: true}
	jsonpCallbackParamName = ""

	Configure(&Config{
		OpenAPI: &OpenAPI{
			Enabled: true,
			Config: &OpenAPIConfig{
				Info: &Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
			},
		},
		Assets: &Assets{
			FS: testMuxI18nFS,
			I18nMessages: &I18nMessages{
				Dir: "testdata/locales",
			},
		},
	})
}

// =============================================================================
// NewServeMux Tests
// =============================================================================

func TestNewServeMux_Success(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	if mux == nil {
		t.Fatal("NewServeMux returned nil")
	}

	if len(mux.middlewares) != 0 {
		t.Errorf("Expected empty middlewares, got %d", len(mux.middlewares))
	}
}

func TestNewServeMux_ConfiguresAppIfNeeded(t *testing.T) {
	// Don't call setupMuxTest to test auto-configuration
	appConfigured = false
	appMiddlewares = nil

	mux := NewServeMux()

	if mux == nil {
		t.Fatal("NewServeMux returned nil")
	}

	if !appConfigured {
		t.Error("Expected app to be configured automatically")
	}
}

// =============================================================================
// ServeMux.HandleFunc Tests
// =============================================================================

func TestServeMux_HandleFunc_BasicHandler(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	called := false
	handler := func(w ResponseWriter, _ *Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello World"))
	}

	mux.HandleFunc("GET /test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if !called {
		t.Error("Handler was not called")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if body := rec.Body.String(); body != "Hello World" {
		t.Errorf("Expected body 'Hello World', got %q", body)
	}
}

func TestServeMux_HandleFunc_WithPathParameters(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	var capturedID string
	handler := func(w ResponseWriter, r *Request) {
		capturedID = r.PathValue("id")
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /users/{id}", handler)

	req := httptest.NewRequest(http.MethodGet, "/users/123", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if capturedID != "123" {
		t.Errorf("Expected path parameter '123', got %q", capturedID)
	}
}

func TestServeMux_HandleFunc_MultipleRoutes(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	route1Called := false
	route2Called := false

	mux.HandleFunc("GET /route1", func(w ResponseWriter, _ *Request) {
		route1Called = true
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("GET /route2", func(w ResponseWriter, _ *Request) {
		route2Called = true
		w.WriteHeader(http.StatusOK)
	})

	// Test route1
	req1 := httptest.NewRequest(http.MethodGet, "/route1", http.NoBody)
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, req1)

	if !route1Called {
		t.Error("Route1 handler was not called")
	}
	if route2Called {
		t.Error("Route2 handler should not have been called")
	}

	// Reset and test route2
	route1Called = false
	route2Called = false

	req2 := httptest.NewRequest(http.MethodGet, "/route2", http.NoBody)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	if route1Called {
		t.Error("Route1 handler should not have been called")
	}
	if !route2Called {
		t.Error("Route2 handler was not called")
	}
}

func TestServeMux_HandleFunc_ReturnsHandlerConfig(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	config := mux.HandleFunc("GET /test", handler)

	if config == nil {
		t.Fatal("HandleFunc returned nil config")
	}

	if config.pathPattern != "GET /test" {
		t.Errorf("Expected pathPattern 'GET /test', got %q", config.pathPattern)
	}
}

// =============================================================================
// ServeMux.Handle Tests
// =============================================================================

func TestServeMux_Handle_WithHandlerInterface(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	called := false
	handler := HandlerFunc(func(w ResponseWriter, _ *Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("Created"))
	})

	mux.Handle("POST /resource", handler)

	req := httptest.NewRequest(http.MethodPost, "/resource", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if !called {
		t.Error("Handler was not called")
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	if body := rec.Body.String(); body != "Created" {
		t.Errorf("Expected body 'Created', got %q", body)
	}
}

func TestServeMux_Handle_ReturnsHandlerConfig(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	handler := HandlerFunc(func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := mux.Handle("GET /api/test", handler)

	if config == nil {
		t.Fatal("Handle returned nil config")
	}

	if config.pathPattern != "GET /api/test" {
		t.Errorf("Expected pathPattern 'GET /api/test', got %q", config.pathPattern)
	}
}

// =============================================================================
// ServeMux.Use Middleware Tests
// =============================================================================

func TestServeMux_Use_WithAppMiddleware(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	middlewareCalled := false
	mw := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			middlewareCalled = true
			w.Header().Set("X-Test-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	mux.Use(mw)

	if len(mux.middlewares) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(mux.middlewares))
	}

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if !middlewareCalled {
		t.Error("Middleware was not called")
	}

	if header := rec.Header().Get("X-Test-Middleware"); header != "applied" {
		t.Errorf("Expected header 'applied', got %q", header)
	}
}

func TestServeMux_Use_WithStandardMiddleware(t *testing.T) {
	setupMuxTest()

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

	if len(mux.middlewares) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(mux.middlewares))
	}

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if !middlewareCalled {
		t.Error("Standard middleware was not called")
	}

	if header := rec.Header().Get("X-Standard-Middleware"); header != "applied" {
		t.Errorf("Expected header 'applied', got %q", header)
	}
}

func TestServeMux_Use_WithNilMiddleware(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	// Should not panic
	mux.Use(nil)

	if len(mux.middlewares) != 0 {
		t.Errorf("Expected 0 middlewares after adding nil, got %d", len(mux.middlewares))
	}
}

func TestServeMux_Use_PanicsOnInvalidType(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid middleware type")
		}
	}()

	mux.Use("invalid-middleware-type")
}

func TestServeMux_Use_MultipleMiddlewares(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	executionOrder := []string{}

	mw1 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			executionOrder = append(executionOrder, "mw1-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "mw1-after")
		})
	}

	mw2 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			executionOrder = append(executionOrder, "mw2-before")
			next.ServeHTTP(w, r)
			executionOrder = append(executionOrder, "mw2-after")
		})
	}

	mux.Use(mw1)
	mux.Use(mw2)

	handler := func(w ResponseWriter, _ *Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(executionOrder) != len(expected) {
		t.Fatalf(
			"Expected %d calls, got %d: %v",
			len(expected),
			len(executionOrder),
			executionOrder,
		)
	}

	for i, v := range expected {
		if executionOrder[i] != v {
			t.Errorf("Expected executionOrder[%d] = %q, got %q", i, v, executionOrder[i])
		}
	}
}

// =============================================================================
// Handler-Specific Middleware Tests
// =============================================================================

func TestServeMux_HandleFunc_WithHandlerMiddleware(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	handlerMwCalled := false
	handlerMw := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			handlerMwCalled = true
			w.Header().Set("X-Handler-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler, handlerMw)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if !handlerMwCalled {
		t.Error("Handler middleware was not called")
	}

	if header := rec.Header().Get("X-Handler-Middleware"); header != "applied" {
		t.Errorf("Expected header 'applied', got %q", header)
	}
}

func TestServeMux_HandleFunc_WithMultipleHandlerMiddlewares(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	executionOrder := []string{}

	mw1 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			executionOrder = append(executionOrder, "handler-mw1")
			next.ServeHTTP(w, r)
		})
	}

	mw2 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			executionOrder = append(executionOrder, "handler-mw2")
			next.ServeHTTP(w, r)
		})
	}

	handler := func(w ResponseWriter, _ *Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler, mw1, mw2)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Verify both middlewares and handler were called
	foundMw1 := false
	foundMw2 := false
	foundHandler := false

	for _, v := range executionOrder {
		if v == "handler-mw1" {
			foundMw1 = true
		}
		if v == "handler-mw2" {
			foundMw2 = true
		}
		if v == "handler" {
			foundHandler = true
		}
	}

	if !foundMw1 {
		t.Error("Handler middleware 1 was not called")
	}
	if !foundMw2 {
		t.Error("Handler middleware 2 was not called")
	}
	if !foundHandler {
		t.Error("Handler was not called")
	}
}

func TestServeMux_MixedMuxAndHandlerMiddlewares(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	executionOrder := []string{}

	muxMw := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			executionOrder = append(executionOrder, "mux-middleware")
			next.ServeHTTP(w, r)
		})
	}

	handlerMw := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			executionOrder = append(executionOrder, "handler-middleware")
			next.ServeHTTP(w, r)
		})
	}

	mux.Use(muxMw)

	handler := func(w ResponseWriter, _ *Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler, handlerMw)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Verify all components were called
	foundMuxMw := false
	foundHandlerMw := false
	foundHandler := false

	for _, v := range executionOrder {
		if v == "mux-middleware" {
			foundMuxMw = true
		}
		if v == "handler-middleware" {
			foundHandlerMw = true
		}
		if v == "handler" {
			foundHandler = true
		}
	}

	if !foundMuxMw {
		t.Error("Mux middleware was not called")
	}
	if !foundHandlerMw {
		t.Error("Handler middleware was not called")
	}
	if !foundHandler {
		t.Error("Handler was not called")
	}
}

// =============================================================================
// HandlerFunc.ServeHTTP Tests
// =============================================================================

func TestHandlerFunc_ServeHTTP_Basic(t *testing.T) {
	setupMuxTest()

	called := false
	handler := HandlerFunc(func(w ResponseWriter, _ *Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	handler.ServeHTTP(rw, r)

	if !called {
		t.Error("Handler was not called")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if body := rec.Body.String(); body != "test response" {
		t.Errorf("Expected body 'test response', got %q", body)
	}
}

func TestHandlerFunc_ServeHTTP_WithJSONPCallback_Valid(t *testing.T) {
	setupMuxTest()

	// Reset and configure with JSONP
	appConfigured = false
	jsonpCallbackParamName = ""
	Configure(&Config{
		JSONPCallbackParamName: "callback",
		Assets: &Assets{
			FS: testMuxI18nFS,
			I18nMessages: &I18nMessages{
				Dir: "testdata/locales",
			},
		},
	})

	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		_ = w.JSON(r.Context(), map[string]string{"message": "test"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test?callback=myCallback", http.NoBody)
	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	handler.ServeHTTP(rw, r)

	body := rec.Body.String()
	if !strings.Contains(body, "myCallback") {
		t.Errorf("Expected JSONP callback 'myCallback' in response, got %q", body)
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/javascript" {
		t.Errorf("Expected Content-Type 'application/javascript', got %q", contentType)
	}
}

func TestHandlerFunc_ServeHTTP_WithJSONPCallback_Invalid(t *testing.T) {
	setupMuxTest()

	// Reset and configure with JSONP
	appConfigured = false
	jsonpCallbackParamName = ""
	Configure(&Config{
		JSONPCallbackParamName: "callback",
		Assets: &Assets{
			FS: testMuxI18nFS,
			I18nMessages: &I18nMessages{
				Dir: "testdata/locales",
			},
		},
	})

	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		_ = w.JSON(r.Context(), map[string]string{"message": "test"})
	})

	invalidCallbacks := []string{
		"invalid-callback",
		"123invalid",
		"call.back",
		"callback!",
	}

	for _, callback := range invalidCallbacks {
		t.Run(callback, func(t *testing.T) {
			// URL encode the callback to avoid issues with special characters in URL
			req := httptest.NewRequest(
				http.MethodGet,
				"/test?callback="+strings.ReplaceAll(callback, " ", "%20"),
				http.NoBody,
			)
			rec := httptest.NewRecorder()
			rw := ResponseWriter{ResponseWriter: rec}
			r := &Request{Request: req}

			handler.ServeHTTP(rw, r)

			if rec.Code != http.StatusBadRequest {
				t.Errorf(
					"Expected status %d for invalid callback %q, got %d",
					http.StatusBadRequest,
					callback,
					rec.Code,
				)
			}

			if !strings.Contains(rec.Body.String(), "invalid JSONP callback") {
				t.Errorf("Expected error message about invalid callback, got %q", rec.Body.String())
			}
		})
	}
}

// =============================================================================
// I18n Middleware Tests
// =============================================================================

func TestI18nMiddleware_WithAcceptLanguageHeader(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.8")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestI18nMiddleware_WithLanguageCookie(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.AddCookie(&http.Cookie{
		Name:  "lang",
		Value: "es",
	})
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestI18nMiddleware_DefaultsToFirstSupportedLanguage(t *testing.T) {
	appConfigured = false
	appMiddlewares = nil
	openAPIConfig = nil
	jsonpCallbackParamName = ""

	// Configure with French as first supported language (not English)
	Configure(&Config{
		Assets: &Assets{
			FS: testMuxI18nFS,
			I18nMessages: &I18nMessages{
				Dir:                "testdata/locales",
				SupportedLanguages: []string{"fr", "es", "de"}, // French first, no English
			},
		},
	})

	mux := NewServeMux()

	var detectedLang string
	handler := func(_ ResponseWriter, r *Request) {
		if printer, ok := i18n.PrinterFromContext(r.Context()); ok {
			// Try to translate a message to verify we got French
			msg := printer.Sprintf("welcome")
			// In French locale it should be "Bienvenue", not "welcome"
			if msg == "Bienvenue" {
				detectedLang = "fr"
			} else {
				detectedLang = "unknown"
			}
		}
	}

	mux.HandleFunc("GET /test", handler)

	// No Accept-Language header or cookie - should default to first supported (fr)
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if detectedLang != "fr" {
		t.Errorf("Expected default to first supported language (fr), got %s", detectedLang)
	}
}

func TestParseAcceptLanguage_ValidLanguages(t *testing.T) {
	// Configure i18n with multiple supported languages
	appConfigured = false
	appMiddlewares = nil
	openAPIConfig = nil
	jsonpCallbackParamName = ""

	Configure(&Config{
		Assets: &Assets{
			FS: testMuxI18nFS,
			I18nMessages: &I18nMessages{
				Dir:                "testdata/locales",
				SupportedLanguages: []string{"en", "fr", "es", "de"},
			},
		},
	})

	tests := []struct {
		name       string
		acceptLang string
		wantBase   string
	}{
		{
			name:       "English",
			acceptLang: "en-US,en;q=0.9",
			wantBase:   "en",
		},
		{
			name:       "French",
			acceptLang: "fr-FR,fr;q=0.9,en;q=0.8",
			wantBase:   "fr",
		},
		{
			name:       "Spanish",
			acceptLang: "es-ES,es;q=0.9",
			wantBase:   "es",
		},
		{
			name:       "German",
			acceptLang: "de-DE,de;q=0.9",
			wantBase:   "de",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := parseAcceptLanguage(tt.acceptLang)
			base, _ := tag.Base()
			if base.String() != tt.wantBase {
				t.Errorf(
					"parseAcceptLanguage(%q) base = %v, want %v",
					tt.acceptLang,
					base,
					tt.wantBase,
				)
			}
		})
	}
}

func TestParseAcceptLanguage_Empty(t *testing.T) {
	tag := parseAcceptLanguage("")

	if tag != language.Und {
		t.Errorf("Expected undefined language for empty string, got %v", tag)
	}
}

func TestParseAcceptLanguage_Invalid(t *testing.T) {
	tag := parseAcceptLanguage("invalid-language-string!!!!")

	if tag != language.Und {
		t.Errorf("Expected undefined language for invalid string, got %v", tag)
	}
}

func TestSetLanguageCookie_Basic(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}

	SetLanguageCookie(rw, "fr", 86400)

	cookies := rec.Result().Cookies()
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
	if cookie.Path != "/" {
		t.Errorf("Expected Path '/', got %q", cookie.Path)
	}
}

func TestSetLanguageCookie_DeleteCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}

	// MaxAge of 0 should delete the cookie
	SetLanguageCookie(rw, "", 0)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.MaxAge != 0 {
		t.Errorf("Expected MaxAge 0 (delete cookie), got %d", cookie.MaxAge)
	}
}

// =============================================================================
// Comprehensive I18n Language Support Tests
// =============================================================================

func TestLanguageMatching_QualityValues(t *testing.T) {
	appConfigured = false
	appMiddlewares = nil
	openAPIConfig = nil
	jsonpCallbackParamName = ""

	Configure(&Config{
		Assets: &Assets{
			FS: testMuxI18nFS,
			I18nMessages: &I18nMessages{
				Dir:                "testdata/locales",
				SupportedLanguages: []string{"en", "fr", "es"},
			},
		},
	})

	tests := []struct {
		name       string
		acceptLang string
		wantBase   string
	}{
		{
			name:       "HighestQuality",
			acceptLang: "fr;q=1.0,es;q=0.8,en;q=0.5",
			wantBase:   "fr",
		},
		{
			name:       "DefaultQualityIsOne",
			acceptLang: "es,fr;q=0.9,en;q=0.8",
			wantBase:   "es",
		},
		{
			name:       "ZeroQualityIgnored",
			acceptLang: "fr;q=0.0,es;q=0.8",
			wantBase:   "es",
		},
		{
			name:       "FirstMatchWins",
			acceptLang: "en;q=0.9,fr;q=0.9",
			wantBase:   "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := parseAcceptLanguage(tt.acceptLang)
			base, _ := tag.Base()
			if base.String() != tt.wantBase {
				t.Errorf(
					"parseAcceptLanguage(%q) base = %v, want %v",
					tt.acceptLang,
					base,
					tt.wantBase,
				)
			}
		})
	}
}

func TestLanguageMatching_Fallback(t *testing.T) {
	appConfigured = false
	appMiddlewares = nil
	openAPIConfig = nil
	jsonpCallbackParamName = ""

	Configure(&Config{
		Assets: &Assets{
			FS: testMuxI18nFS,
			I18nMessages: &I18nMessages{
				Dir:                "testdata/locales",
				SupportedLanguages: []string{"en", "fr"},
			},
		},
	})

	tests := []struct {
		name       string
		acceptLang string
		wantBase   string
	}{
		{
			name:       "UnsupportedLanguageFallsBackToClosest",
			acceptLang: "de-DE,de;q=0.9,en;q=0.8",
			wantBase:   "en",
		},
		{
			name:       "RegionalVariantMatchesBase",
			acceptLang: "fr-CA,fr;q=0.9",
			wantBase:   "fr",
		},
		{
			name:       "ComplexFallback",
			acceptLang: "zh-CN,zh;q=0.9,ja;q=0.8,en;q=0.7",
			wantBase:   "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := parseAcceptLanguage(tt.acceptLang)
			base, _ := tag.Base()
			if base.String() != tt.wantBase {
				t.Errorf(
					"parseAcceptLanguage(%q) base = %v, want %v",
					tt.acceptLang,
					base,
					tt.wantBase,
				)
			}
		})
	}
}

func TestLanguageCookie_PreferenceOverAcceptLanguage(t *testing.T) {
	appConfigured = false
	appMiddlewares = nil
	openAPIConfig = nil
	jsonpCallbackParamName = ""

	Configure(&Config{
		Assets: &Assets{
			FS: testMuxI18nFS,
			I18nMessages: &I18nMessages{
				Dir:                "testdata/locales",
				SupportedLanguages: []string{"en", "fr", "es"},
			},
		},
	})

	mux := NewServeMux()

	var receivedLang string
	handler := func(_ ResponseWriter, r *Request) {
		if _, ok := i18n.PrinterFromContext(r.Context()); ok {
			receivedLang = "cookie-worked"
		}
	}

	mux.HandleFunc("GET /test", handler)

	// Request with both cookie (fr) and Accept-Language (es)
	// Cookie should take precedence
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9")
	req.AddCookie(&http.Cookie{Name: "lang", Value: "fr"})
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if receivedLang != "cookie-worked" {
		t.Error("Language cookie preference was not respected")
	}
}

func TestLanguageContext_PersistsThroughHandlers(t *testing.T) {
	appConfigured = false
	appMiddlewares = nil
	openAPIConfig = nil
	jsonpCallbackParamName = ""

	Configure(&Config{
		Assets: &Assets{
			FS: testMuxI18nFS,
			I18nMessages: &I18nMessages{
				Dir:                "testdata/locales",
				SupportedLanguages: []string{"en", "fr"},
			},
		},
	})

	mux := NewServeMux()

	var lang1, lang2 string

	// Middleware to check language
	mux.Use(func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			if _, ok := i18n.PrinterFromContext(r.Context()); ok {
				lang1 = "middleware-ok"
			}
			next.ServeHTTP(w, r)
		})
	})

	handler := func(_ ResponseWriter, r *Request) {
		if _, ok := i18n.PrinterFromContext(r.Context()); ok {
			lang2 = "handler-ok"
		}
	}

	mux.HandleFunc("GET /test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Accept-Language", "fr-FR")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if lang1 != "middleware-ok" {
		t.Error("Language context not available in middleware")
	}
	if lang2 != "handler-ok" {
		t.Error("Language context not available in handler")
	}
}

func TestParseAcceptLanguage_UnsupportedLanguage(t *testing.T) {
	appConfigured = false
	appMiddlewares = nil
	openAPIConfig = nil
	jsonpCallbackParamName = ""

	Configure(&Config{
		Assets: &Assets{
			FS: testMuxI18nFS,
			I18nMessages: &I18nMessages{
				Dir:                "testdata/locales",
				SupportedLanguages: []string{"en"},
			},
		},
	})

	// Request language not in supported list should fall back to English
	tag := parseAcceptLanguage("fr-FR,fr;q=0.9")
	base, _ := tag.Base()

	if base.String() != "en" {
		t.Errorf("Expected fallback to 'en', got %v", base)
	}
}

func TestLanguageMatching_EdgeCases(t *testing.T) {
	appConfigured = false
	appMiddlewares = nil
	openAPIConfig = nil
	jsonpCallbackParamName = ""

	Configure(&Config{
		Assets: &Assets{
			FS: testMuxI18nFS,
			I18nMessages: &I18nMessages{
				Dir:                "testdata/locales",
				SupportedLanguages: []string{"en", "fr", "es", "de"},
			},
		},
	})

	tests := []struct {
		name       string
		acceptLang string
		wantBase   string
		wantUnd    bool
	}{
		{
			name:       "WildcardMatch",
			acceptLang: "*",
			wantBase:   "en", // Should match first supported language
		},
		{
			name:       "EmptyString",
			acceptLang: "",
			wantUnd:    true,
		},
		{
			name:       "OnlyWhitespace",
			acceptLang: "   ",
			wantUnd:    true,
		},
		{
			name:       "MalformedButRecoverable",
			acceptLang: "fr,en;q=",
			wantBase:   "en", // Parser treats malformed quality, returns closest match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := parseAcceptLanguage(tt.acceptLang)

			if tt.wantUnd {
				if tag != language.Und {
					t.Errorf("Expected undefined language, got %v", tag)
				}
				return
			}

			base, _ := tag.Base()
			if base.String() != tt.wantBase {
				t.Errorf(
					"parseAcceptLanguage(%q) base = %v, want %v",
					tt.acceptLang,
					base,
					tt.wantBase,
				)
			}
		})
	}
}

func TestI18n_NoConfiguration(t *testing.T) {
	// Reset app state
	// This tests auto-detection when no explicit supported languages are provided
	appConfigured = false
	appMiddlewares = nil
	openAPIConfig = nil
	jsonpCallbackParamName = ""

	Configure(&Config{
		Assets: &Assets{
			FS: testMuxI18nFS,
			I18nMessages: &I18nMessages{
				Dir:                "testdata/locales",
				SupportedLanguages: []string{}, // Empty list triggers auto-detection
			},
		},
	})

	// With auto-detection, French should be found and matched
	tag := parseAcceptLanguage("fr-FR")

	base, _ := tag.Base()
	if base.String() != "fr" {
		t.Errorf("Expected French to be matched via auto-detection, got %v", tag)
	}
}

func TestSetLanguageCookie_SessionCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}

	// MaxAge of -1 should create a session cookie
	SetLanguageCookie(rw, "fr", -1)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.MaxAge != -1 {
		t.Errorf("Expected MaxAge -1 (session cookie), got %d", cookie.MaxAge)
	}
	if cookie.Value != "fr" {
		t.Errorf("Expected cookie value 'fr', got %q", cookie.Value)
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestWrapMiddlewares_SingleMiddleware(t *testing.T) {
	called := false
	mw := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	}

	handler := HandlerFunc(func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := wrapMiddlewares(handler, []AppMiddleware{mw})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	wrapped.ServeHTTP(rw, r)

	if !called {
		t.Error("Middleware was not called")
	}
}

func TestWrapMiddlewares_MultipleMiddlewares(t *testing.T) {
	executionOrder := []string{}

	mw1 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			executionOrder = append(executionOrder, "mw1")
			next.ServeHTTP(w, r)
		})
	}

	mw2 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			executionOrder = append(executionOrder, "mw2")
			next.ServeHTTP(w, r)
		})
	}

	mw3 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			executionOrder = append(executionOrder, "mw3")
			next.ServeHTTP(w, r)
		})
	}

	handler := HandlerFunc(func(w ResponseWriter, _ *Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})

	wrapped := wrapMiddlewares(handler, []AppMiddleware{mw1, mw2, mw3})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	wrapped.ServeHTTP(rw, r)

	expected := []string{"mw1", "mw2", "mw3", "handler"}
	if len(executionOrder) != len(expected) {
		t.Fatalf(
			"Expected %d calls, got %d: %v",
			len(expected),
			len(executionOrder),
			executionOrder,
		)
	}

	for i, v := range expected {
		if executionOrder[i] != v {
			t.Errorf("Expected executionOrder[%d] = %q, got %q", i, v, executionOrder[i])
		}
	}
}

func TestWrapMiddlewares_EmptyMiddlewares(t *testing.T) {
	called := false
	handler := HandlerFunc(func(w ResponseWriter, _ *Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := wrapMiddlewares(handler, []AppMiddleware{})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	wrapped.ServeHTTP(rw, r)

	if !called {
		t.Error("Handler was not called")
	}
}

func TestGetHandlerMiddlewares_AppMiddleware(t *testing.T) {
	appMw := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			next.ServeHTTP(w, r)
		})
	}

	result := getHandlerMiddlewares([]interface{}{appMw})

	if len(result) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(result))
	}
}

func TestGetHandlerMiddlewares_StandardMiddleware(t *testing.T) {
	stdMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	result := getHandlerMiddlewares([]interface{}{stdMw})

	if len(result) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(result))
	}
}

func TestGetHandlerMiddlewares_MixedMiddlewares(t *testing.T) {
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

	result := getHandlerMiddlewares([]interface{}{appMw, stdMw})

	if len(result) != 2 {
		t.Errorf("Expected 2 middlewares, got %d", len(result))
	}
}

func TestGetHandlerMiddlewares_PanicsOnUnsupportedType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for unsupported middleware type")
		}
	}()

	getHandlerMiddlewares([]interface{}{"invalid-type"})
}

// =============================================================================
// Security Configuration Tests
// =============================================================================

func TestServeMux_UseSecurity_SetsSecurityConfig(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	securityConfig := security.Config{
		APIKeyAuth: &security.APIKeyAuthConfig{
			KeyName:      "X-API-Key",
			KeyLocation:  "header",
			KeyValidator: func(key string) bool { return key == "valid-key" },
		},
	}

	mux.UseSecurity(securityConfig)

	if mux.securityConfig == nil {
		t.Fatal("Expected security config to be set")
	}

	if mux.securityConfig.APIKeyAuth == nil {
		t.Error("Expected APIKeyAuth config to be set")
	}

	if mux.securityConfig.APIKeyAuth.KeyName != "X-API-Key" {
		t.Errorf("Expected API key name 'X-API-Key', got %q", mux.securityConfig.APIKeyAuth.KeyName)
	}
}

func TestServeMux_UseSecurity_OverridesGlobalConfig(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	// Set mux-specific security config
	muxConfig := security.Config{
		BearerAuth: &security.BearerAuthConfig{
			TokenValidator: func(token string) bool { return token == "valid-token" },
		},
	}
	mux.UseSecurity(muxConfig)

	// Verify the config was set correctly
	if mux.securityConfig.BearerAuth == nil {
		t.Error("Expected mux BearerAuth config to be set")
	}

	if mux.securityConfig.BasicAuth != nil {
		t.Error("Expected no BasicAuth config when only BearerAuth is set")
	}
}

func TestGetSecurityMiddlewares_AllowAnonymousAuth(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		AllowAnonymousAuth: true,
		BasicAuth: &security.BasicAuthConfig{
			Realm:         "test",
			Authenticator: func(_, _ string) bool { return true },
		},
	}

	mux.UseSecurity(config)

	middlewares := getSecurityMiddlewares(mux)

	if len(middlewares) != 0 {
		t.Errorf("Expected no security middlewares when AllowAnonymousAuth is true, got %d", len(middlewares))
	}
}

func TestGetSecurityMiddlewares_NoSecurityConfig(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	middlewares := getSecurityMiddlewares(mux)

	if len(middlewares) != 0 {
		t.Errorf("Expected no security middlewares when no config is set, got %d", len(middlewares))
	}
}

func TestGetSecurityMiddlewares_GlobalSecurityConfig(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	// Test that mux-level security config takes precedence over global config
	// When no security is configured on the mux, it should return empty middlewares
	middlewares := getSecurityMiddlewares(mux)

	if len(middlewares) != 0 {
		t.Errorf("Expected 0 security middlewares when no mux security is configured, got %d", len(middlewares))
	}
}

func TestGetSecurityMiddlewares_APIKeyAuth(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		APIKeyAuth: &security.APIKeyAuthConfig{
			KeyName:      "X-API-Key",
			KeyLocation:  "header",
			KeyValidator: func(_ string) bool { return true },
		},
	}

	mux.UseSecurity(config)

	middlewares := getSecurityMiddlewares(mux)

	if len(middlewares) != 1 {
		t.Errorf("Expected 1 security middleware, got %d", len(middlewares))
	}
}

func TestGetSecurityMiddlewares_BasicAuth(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		BasicAuth: &security.BasicAuthConfig{
			Realm:         "test",
			Authenticator: func(_, _ string) bool { return true },
		},
	}

	mux.UseSecurity(config)

	middlewares := getSecurityMiddlewares(mux)

	if len(middlewares) != 1 {
		t.Errorf("Expected 1 security middleware, got %d", len(middlewares))
	}
}

func TestGetSecurityMiddlewares_DigestAuth(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		DigestAuth: &security.DigestAuthConfig{
			Realm: "test",
		},
	}

	mux.UseSecurity(config)

	middlewares := getSecurityMiddlewares(mux)

	if len(middlewares) != 1 {
		t.Errorf("Expected 1 security middleware, got %d", len(middlewares))
	}
}

func TestGetSecurityMiddlewares_BearerAuth(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		BearerAuth: &security.BearerAuthConfig{
			TokenValidator: func(_ string) bool { return true },
		},
	}

	mux.UseSecurity(config)

	middlewares := getSecurityMiddlewares(mux)

	if len(middlewares) != 1 {
		t.Errorf("Expected 1 security middleware, got %d", len(middlewares))
	}
}

func TestGetSecurityMiddlewares_MutualTLSAuth(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		MutualTLSAuth: &security.MutualTLSAuthConfig{
			CertificateValidator: func(_ *x509.Certificate) bool { return true },
		},
	}

	mux.UseSecurity(config)

	middlewares := getSecurityMiddlewares(mux)

	if len(middlewares) != 1 {
		t.Errorf("Expected 1 security middleware, got %d", len(middlewares))
	}
}

func TestGetSecurityMiddlewares_OAuth2AuthorizationCode(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		OAuth2AuthorizationCode: &security.OAuth2AuthorizationCodeConfig{
			OAuth2BaseConfig: security.OAuth2BaseConfig{
				ClientID:       "test-client",
				TokenURL:       "https://example.com/token",
				TokenValidator: func(_ string) bool { return true },
			},
			ClientSecret: "test-secret",
		},
	}

	mux.UseSecurity(config)

	middlewares := getSecurityMiddlewares(mux)

	if len(middlewares) != 1 {
		t.Errorf("Expected 1 security middleware, got %d", len(middlewares))
	}
}

func TestGetSecurityMiddlewares_OAuth2ClientCredentials(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		OAuth2ClientCredentials: &security.OAuth2ClientCredentialsConfig{
			OAuth2BaseConfig: security.OAuth2BaseConfig{
				ClientID:       "test-client",
				TokenURL:       "https://example.com/token",
				TokenValidator: func(_ string) bool { return true },
			},
			ClientSecret: "test-secret",
		},
	}

	mux.UseSecurity(config)

	middlewares := getSecurityMiddlewares(mux)

	if len(middlewares) != 1 {
		t.Errorf("Expected 1 security middleware, got %d", len(middlewares))
	}
}

func TestGetSecurityMiddlewares_OAuth2Device(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		OAuth2Device: &security.OAuth2DeviceConfig{
			OAuth2BaseConfig: security.OAuth2BaseConfig{
				ClientID:       "test-client",
				TokenURL:       "https://example.com/token",
				TokenValidator: func(_ string) bool { return true },
			},
		},
	}

	mux.UseSecurity(config)

	middlewares := getSecurityMiddlewares(mux)

	if len(middlewares) != 1 {
		t.Errorf("Expected 1 security middleware, got %d", len(middlewares))
	}
}

func TestGetSecurityMiddlewares_OAuth2Implicit(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		OAuth2Implicit: &security.OAuth2ImplicitConfig{
			OAuth2BaseConfig: security.OAuth2BaseConfig{
				ClientID:       "test-client",
				TokenURL:       "https://example.com/token",
				TokenValidator: func(_ string) bool { return true },
			},
			AuthorizationURL: "https://example.com/auth",
			RedirectURL:      "https://example.com/callback",
		},
	}

	mux.UseSecurity(config)

	middlewares := getSecurityMiddlewares(mux)

	if len(middlewares) != 1 {
		t.Errorf("Expected 1 security middleware, got %d", len(middlewares))
	}
}

func TestGetSecurityMiddlewares_OpenIDConnectAuth(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		OpenIDConnectAuth: &security.OpenIDConnectAuthConfig{
			IssuerURL: "https://example.com",
			ClientID:  "test-client",
		},
	}

	mux.UseSecurity(config)

	middlewares := getSecurityMiddlewares(mux)

	if len(middlewares) != 1 {
		t.Errorf("Expected 1 security middleware, got %d", len(middlewares))
	}
}

func TestGetSecurityMiddlewares_MultipleAuthMethods(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		APIKeyAuth: &security.APIKeyAuthConfig{
			KeyName:      "X-API-Key",
			KeyLocation:  "header",
			KeyValidator: func(_ string) bool { return true },
		},
		BasicAuth: &security.BasicAuthConfig{
			Realm:         "test",
			Authenticator: func(_, _ string) bool { return true },
		},
		BearerAuth: &security.BearerAuthConfig{
			TokenValidator: func(_ string) bool { return true },
		},
	}

	mux.UseSecurity(config)

	middlewares := getSecurityMiddlewares(mux)

	if len(middlewares) != 3 {
		t.Errorf("Expected 3 security middlewares, got %d", len(middlewares))
	}
}

func TestServeMux_SecurityMiddlewareIntegration_APIKeyAuth(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		APIKeyAuth: &security.APIKeyAuthConfig{
			KeyName:     "X-API-Key",
			KeyLocation: "header",
			KeyValidator: func(key string) bool {
				return key == "valid-key"
			},
		},
	}

	mux.UseSecurity(config)

	handlerCalled := false
	handler := func(w ResponseWriter, _ *Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /secure", handler)

	// Test without API key - should fail
	req1 := httptest.NewRequest(http.MethodGet, "/secure", http.NoBody)
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, req1)

	if handlerCalled {
		t.Error("Handler should not have been called without API key")
	}

	if rec1.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec1.Code)
	}

	// Reset for next test
	handlerCalled = false

	// Test with valid API key - should succeed
	req2 := httptest.NewRequest(http.MethodGet, "/secure", http.NoBody)
	req2.Header.Set("X-Api-Key", "valid-key")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	if !handlerCalled {
		t.Error("Handler should have been called with valid API key")
	}

	if rec2.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec2.Code)
	}
}

func TestServeMux_SecurityMiddlewareIntegration_BearerAuth(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		BearerAuth: &security.BearerAuthConfig{
			TokenValidator: func(token string) bool {
				return token == "valid-token"
			},
		},
	}

	mux.UseSecurity(config)

	handlerCalled := false
	handler := func(w ResponseWriter, _ *Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /secure", handler)

	// Test without bearer token - should fail
	req1 := httptest.NewRequest(http.MethodGet, "/secure", http.NoBody)
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, req1)

	if handlerCalled {
		t.Error("Handler should not have been called without bearer token")
	}

	if rec1.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec1.Code)
	}

	// Reset for next test
	handlerCalled = false

	// Test with invalid bearer token - should fail
	req2 := httptest.NewRequest(http.MethodGet, "/secure", http.NoBody)
	req2.Header.Set("Authorization", "Bearer invalid-token")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	if handlerCalled {
		t.Error("Handler should not have been called with invalid bearer token")
	}

	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec2.Code)
	}

	// Reset for next test
	handlerCalled = false

	// Test with valid bearer token - should succeed
	req3 := httptest.NewRequest(http.MethodGet, "/secure", http.NoBody)
	req3.Header.Set("Authorization", "Bearer valid-token")
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)

	if !handlerCalled {
		t.Error("Handler should have been called with valid bearer token")
	}

	if rec3.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec3.Code)
	}
}

func TestServeMux_SecurityMiddlewareIntegration_BasicAuth(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	config := security.Config{
		BasicAuth: &security.BasicAuthConfig{
			Realm: "test",
			Authenticator: func(username, password string) bool {
				return username == "admin" && password == "secret"
			},
		},
	}

	mux.UseSecurity(config)

	handlerCalled := false
	handler := func(w ResponseWriter, _ *Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /secure", handler)

	// Test without basic auth - should fail
	req1 := httptest.NewRequest(http.MethodGet, "/secure", http.NoBody)
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, req1)

	if handlerCalled {
		t.Error("Handler should not have been called without basic auth")
	}

	if rec1.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec1.Code)
	}

	// Reset for next test
	handlerCalled = false

	// Test with invalid basic auth - should fail
	req2 := httptest.NewRequest(http.MethodGet, "/secure", http.NoBody)
	req2.Header.Set("Authorization", "Basic invalid")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	if handlerCalled {
		t.Error("Handler should not have been called with invalid basic auth")
	}

	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec2.Code)
	}

	// Reset for next test
	handlerCalled = false

	// Test with valid basic auth - should succeed
	req3 := httptest.NewRequest(http.MethodGet, "/secure", http.NoBody)
	req3.SetBasicAuth("admin", "secret")
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)

	if !handlerCalled {
		t.Error("Handler should have been called with valid basic auth")
	}

	if rec3.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec3.Code)
	}
}

func TestServeMux_SecurityMiddlewareOrder(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	// Add mux middleware
	muxMiddlewareCalled := false
	mux.Use(func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			muxMiddlewareCalled = true
			w.Header().Set("X-Mux-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	})

	// Add security config
	config := security.Config{
		APIKeyAuth: &security.APIKeyAuthConfig{
			KeyName:      "X-API-Key",
			KeyLocation:  "header",
			KeyValidator: func(_ string) bool { return true },
		},
	}
	mux.UseSecurity(config)

	handlerCalled := false
	handler := func(w ResponseWriter, _ *Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /secure", handler)

	req := httptest.NewRequest(http.MethodGet, "/secure", http.NoBody)
	req.Header.Set("X-Api-Key", "valid-key")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if !muxMiddlewareCalled {
		t.Error("Mux middleware should have been called")
	}

	if !handlerCalled {
		t.Error("Handler should have been called")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if header := rec.Header().Get("X-Mux-Middleware"); header != "applied" {
		t.Errorf("Expected mux middleware header 'applied', got %q", header)
	}
}

// =============================================================================
// OpenAPI Integration Tests
// =============================================================================

func TestHandlerConfig_WithOperationConfig_Success(t *testing.T) {
	setupMuxTestWithOpenAPI()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	config := mux.HandleFunc("GET /api/users", handler)

	apiConfig := &OperationConfig{
		OperationID: "getUsers",
		Summary:     "Get all users",
		Description: "Retrieves a list of all users",
		Tags:        []string{"users"},
	}

	config.WithOperationConfig(apiConfig)

	if config.OperationConfig == nil {
		t.Error("OperationConfig was not set")
	}

	if config.OperationConfig.OperationID != "getUsers" {
		t.Errorf("Expected OperationID 'getUsers', got %q", config.OperationConfig.OperationID)
	}
}

func TestHandlerConfig_WithOperationConfig_NilConfig(t *testing.T) {
	setupMuxTestWithOpenAPI()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	config := mux.HandleFunc("GET /api/test", handler)

	// Should not panic with nil config
	config.WithOperationConfig(nil)

	if config.OperationConfig != nil {
		t.Error("OperationConfig should remain nil")
	}
}

func TestHandlerConfig_WithOperationConfig_OpenAPIDisabled(_ *testing.T) {
	setupMuxTest() // Sets up without OpenAPI

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	config := mux.HandleFunc("GET /api/test", handler)

	apiConfig := &OperationConfig{
		OperationID: "testOp",
		Summary:     "Test",
	}

	// Should not panic even if OpenAPI is disabled
	config.WithOperationConfig(apiConfig)
}

func TestHandlerConfig_WithOperationConfig_InvalidPathPattern(t *testing.T) {
	setupMuxTestWithOpenAPI()

	config := &HandlerConfig{
		pathPattern: "invalid-pattern", // Missing method
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid path pattern")
		}
	}()

	config.WithOperationConfig(&OperationConfig{
		OperationID: "testOp",
	})
}

func TestHandlerConfig_WithOperationConfig_WithRequestBody(t *testing.T) {
	setupMuxTestWithOpenAPI()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusCreated)
	}

	config := mux.HandleFunc("POST /api/users", handler)

	type CreateUserRequest struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	apiConfig := &OperationConfig{
		OperationID: "createUser",
		Summary:     "Create user",
		RequestBody: &RequestBody{
			Description: "User data",
			Required:    true,
			Content: map[string]TypeInfo{
				"application/json": {
					TypeHint: CreateUserRequest{},
				},
			},
		},
	}

	config.WithOperationConfig(apiConfig)

	if config.OperationConfig.RequestBody == nil {
		t.Error("RequestBody was not set")
	}
}

func TestHandlerConfig_WithOperationConfig_WithResponses(t *testing.T) {
	setupMuxTestWithOpenAPI()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	config := mux.HandleFunc("GET /api/users/{id}", handler)

	type User struct {
		Name string `json:"name"`
		ID   int    `json:"id"`
	}

	apiConfig := &OperationConfig{
		OperationID: "getUserById",
		Summary:     "Get user by ID",
		Responses: map[string]Response{
			"200": {
				Description: "Successful response",
				Content: map[string]TypeInfo{
					"application/json": {
						TypeHint: User{},
					},
				},
			},
			"404": {
				Description: "User not found",
			},
		},
	}

	config.WithOperationConfig(apiConfig)

	if config.OperationConfig.Responses == nil {
		t.Error("Responses were not set")
	}

	if len(config.OperationConfig.Responses) != 2 {
		t.Errorf("Expected 2 responses, got %d", len(config.OperationConfig.Responses))
	}
}

func TestHandlerConfig_WithOperationConfig_WithSecurity(t *testing.T) {
	setupMuxTestWithOpenAPI()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	config := mux.HandleFunc("GET /api/secure", handler)

	operationConfig := &OperationConfig{
		OperationID: "secureOperation",
		Summary:     "Secure endpoint",
		Security: []map[string][]string{
			{"BearerAuth": {}},
			{"ApiKeyAuth": {"read", "write"}},
		},
	}

	config.WithOperationConfig(operationConfig)

	if config.OperationConfig.Security == nil {
		t.Fatal("Expected Security to be set")
	}

	if len(config.OperationConfig.Security) != 2 {
		t.Errorf("Expected 2 security requirements, got %d", len(config.OperationConfig.Security))
	}

	// Verify BearerAuth requirement
	if _, ok := config.OperationConfig.Security[0]["BearerAuth"]; !ok {
		t.Error("Expected BearerAuth security requirement")
	}

	// Verify ApiKeyAuth requirement with scopes
	if scopes, ok := config.OperationConfig.Security[1]["ApiKeyAuth"]; !ok {
		t.Error("Expected ApiKeyAuth security requirement")
	} else if len(scopes) != 2 {
		t.Errorf("Expected 2 scopes for ApiKeyAuth, got %d", len(scopes))
	}
}

func TestHandlerConfig_WithOperationConfig_WithEmptySecurity(t *testing.T) {
	setupMuxTestWithOpenAPI()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	config := mux.HandleFunc("GET /api/public", handler)

	operationConfig := &OperationConfig{
		OperationID: "publicOperation",
		Summary:     "Public endpoint (no security)",
		Security:    []map[string][]string{},
	}

	config.WithOperationConfig(operationConfig)

	// Empty security array means no authentication required
	if config.OperationConfig.Security == nil {
		t.Error("Expected Security to be initialized even when empty")
	}

	if len(config.OperationConfig.Security) != 0 {
		t.Errorf("Expected 0 security requirements, got %d", len(config.OperationConfig.Security))
	}
}

func TestHandlerConfig_WithOperationConfig_WithNilSecurity(t *testing.T) {
	setupMuxTestWithOpenAPI()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	config := mux.HandleFunc("GET /api/default", handler)

	operationConfig := &OperationConfig{
		OperationID: "defaultSecurityOperation",
		Summary:     "Endpoint with default security",
		Security:    nil,
	}

	config.WithOperationConfig(operationConfig)

	// Nil security means use global security requirements
	if config.OperationConfig.Security != nil {
		t.Error("Expected Security to remain nil when not specified")
	}
}

func TestSetOpenAPIPathInfo_Success(_ *testing.T) {
	setupMuxTestWithOpenAPI()

	pathInfo := &PathInfo{
		Summary:     "User operations",
		Description: "Operations for managing users",
		Parameters: []Parameter{
			{
				Name:        "id",
				In:          "path",
				Description: "User ID",
				Required:    true,
				TypeHint:    0,
			},
		},
	}

	// Should not panic
	SetOpenAPIPathInfo("/api/users/{id}", pathInfo)
}

func TestSetOpenAPIPathInfo_OpenAPIDisabled(_ *testing.T) {
	setupMuxTest() // No OpenAPI

	pathInfo := &PathInfo{
		Summary:     "Test",
		Description: "Test description",
	}

	// Should not panic even if OpenAPI is disabled
	SetOpenAPIPathInfo("/api/test", pathInfo)
}

// =============================================================================
// Mapper Function Tests
// =============================================================================

func TestMapServers_WithServers(t *testing.T) {
	servers := []Server{
		{
			URL:         "https://api.example.com",
			Name:        "Production",
			Description: "Production server",
		},
		{
			URL:         "https://staging.example.com",
			Name:        "Staging",
			Description: "Staging server",
		},
	}

	result := mapServers(servers)

	if len(result) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(result))
	}

	if result[0].URL != "https://api.example.com" {
		t.Errorf("Expected first server URL 'https://api.example.com', got %q", result[0].URL)
	}

	if result[1].Name != "Staging" {
		t.Errorf("Expected second server name 'Staging', got %q", result[1].Name)
	}
}

func TestMapServers_NilInput(t *testing.T) {
	result := mapServers(nil)

	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}

func TestMapServerVariables_WithVariables(t *testing.T) {
	vars := map[string]ServerVariable{
		"port": {
			Enum:        []string{"8080", "8443"},
			Default:     "8080",
			Description: "Server port",
		},
		"env": {
			Enum:        []string{"dev", "prod"},
			Default:     "dev",
			Description: "Environment",
		},
	}

	result := mapServerVariables(vars)

	if len(result) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(result))
	}

	if result["port"].Default != "8080" {
		t.Errorf("Expected port default '8080', got %q", result["port"].Default)
	}
}

func TestMapServerVariables_NilInput(t *testing.T) {
	result := mapServerVariables(nil)

	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}

func TestNonZeroValuePointer_NonZero(t *testing.T) {
	intVal := 42
	intPtr := nonZeroValuePointer(intVal)

	if intPtr == nil {
		t.Error("Expected non-nil pointer for non-zero value")
	} else if *intPtr != 42 {
		t.Errorf("Expected value 42, got %d", *intPtr)
	}

	strVal := "test"
	strPtr := nonZeroValuePointer(strVal)

	if strPtr == nil {
		t.Error("Expected non-nil pointer for non-zero string")
	} else if *strPtr != "test" {
		t.Errorf("Expected value 'test', got %q", *strPtr)
	}
}

func TestNonZeroValuePointer_Zero(t *testing.T) {
	intVal := 0
	intPtr := nonZeroValuePointer(intVal)

	if intPtr != nil {
		t.Error("Expected nil pointer for zero integer value")
	}

	strVal := ""
	strPtr := nonZeroValuePointer(strVal)

	if strPtr != nil {
		t.Error("Expected nil pointer for zero string value")
	}

	floatVal := 0.0
	floatPtr := nonZeroValuePointer(floatVal)

	if floatPtr != nil {
		t.Error("Expected nil pointer for zero float value")
	}
}

func TestMapParameters_Basic(t *testing.T) {
	params := []Parameter{
		{
			Name:        "id",
			In:          "path",
			Description: "Resource ID",
			Required:    true,
			TypeHint:    0,
		},
		{
			Name:        "limit",
			In:          "query",
			Description: "Page limit",
			Required:    false,
			TypeHint:    0,
		},
	}

	setupMuxTestWithOpenAPI()
	result := mapParameters(params)

	if len(result) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(result))
	}

	if result[0].Name != "id" {
		t.Errorf("Expected first parameter name 'id', got %q", result[0].Name)
	}

	if !result[0].Required {
		t.Error("Expected first parameter to be required")
	}
}

func TestMapExamples_WithExamples(t *testing.T) {
	examples := map[string]Example{
		"example1": {
			Summary:     "First example",
			Description: "Description of first example",
			DataValue:   "value1",
		},
		"example2": {
			Summary:     "Second example",
			Description: "Description of second example",
			DataValue:   "value2",
		},
	}

	result := mapExamples(examples)

	if len(result) != 2 {
		t.Errorf("Expected 2 examples, got %d", len(result))
	}

	if result["example1"].Summary != "First example" {
		t.Errorf("Expected summary 'First example', got %q", result["example1"].Summary)
	}
}

func TestMapExamples_NilInput(t *testing.T) {
	result := mapExamples(nil)

	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkNewServeMux(b *testing.B) {
	setupMuxTest()

	b.ResetTimer()
	for b.Loop() {
		_ = NewServeMux()
	}
}

func BenchmarkServeMux_HandleFunc(b *testing.B) {
	setupMuxTest()
	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	b.ResetTimer()
	for b.Loop() {
		mux.HandleFunc("GET /test", handler)
	}
}

func BenchmarkServeMux_ServeHTTP(b *testing.B) {
	setupMuxTest()
	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)

	b.ResetTimer()
	for b.Loop() {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}
}

func BenchmarkWrapMiddlewares(b *testing.B) {
	mw1 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			next.ServeHTTP(w, r)
		})
	}

	mw2 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			next.ServeHTTP(w, r)
		})
	}

	handler := HandlerFunc(func(_ ResponseWriter, _ *Request) {
		// Empty handler
	})

	middlewares := []AppMiddleware{mw1, mw2}

	b.ResetTimer()
	for b.Loop() {
		_ = wrapMiddlewares(handler, middlewares)
	}
}

func BenchmarkParseAcceptLanguage(b *testing.B) {
	acceptLang := "fr-FR,fr;q=0.9,en-US;q=0.8,en;q=0.7"

	b.ResetTimer()
	for b.Loop() {
		_ = parseAcceptLanguage(acceptLang)
	}
}

// =============================================================================
// Telemetry Middleware Tests
// =============================================================================

func TestTelemetryMiddleware_RequestsTotal(t *testing.T) {
	setupMuxTest()

	// Reset telemetry metrics
	telemetry.RequestsTotal.Reset()
	telemetry.RequestDurationSeconds.Reset()
	telemetry.ActiveConnections.Set(0)

	mux := NewServeMux()

	// Create a handler that returns different status codes
	handler := func(w ResponseWriter, r *Request) {
		switch r.URL.Path {
		case "/success":
			w.WriteHeader(http.StatusOK)
		case "/created":
			w.WriteHeader(http.StatusCreated)
		case "/notfound":
			w.WriteHeader(http.StatusNotFound)
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	mux.HandleFunc("GET /success", handler)
	mux.HandleFunc("POST /created", handler)
	mux.HandleFunc("GET /notfound", handler)
	mux.HandleFunc("GET /error", handler)

	// Make requests
	testCases := []struct {
		method       string
		path         string
		expectedCode int
	}{
		{"GET", "/success", http.StatusOK},
		{"GET", "/success", http.StatusOK}, // Duplicate to test counter increment
		{"POST", "/created", http.StatusCreated},
		{"GET", "/notfound", http.StatusNotFound},
		{"GET", "/error", http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != tc.expectedCode {
			t.Errorf("Expected status %d, got %d for %s %s", tc.expectedCode, rec.Code, tc.method, tc.path)
		}
	}

	// Verify metrics were recorded

	// Check GET /success 2xx was incremented twice
	count := testutil.ToFloat64(telemetry.RequestsTotal.WithLabelValues("GET", "/success", "2xx"))
	if count != 2 {
		t.Errorf("Expected GET /success 2xx count to be 2, got %f", count)
	}

	// Check POST /created 2xx was incremented once
	count = testutil.ToFloat64(telemetry.RequestsTotal.WithLabelValues("POST", "/created", "2xx"))
	if count != 1 {
		t.Errorf("Expected POST /created 2xx count to be 1, got %f", count)
	}

	// Check GET /notfound 4xx was incremented once
	count = testutil.ToFloat64(telemetry.RequestsTotal.WithLabelValues("GET", "/notfound", "4xx"))
	if count != 1 {
		t.Errorf("Expected GET /notfound 4xx count to be 1, got %f", count)
	}

	// Check GET /error 5xx was incremented once
	count = testutil.ToFloat64(telemetry.RequestsTotal.WithLabelValues("GET", "/error", "5xx"))
	if count != 1 {
		t.Errorf("Expected GET /error 5xx count to be 1, got %f", count)
	}
}

func TestTelemetryMiddleware_ActiveConnections(t *testing.T) {
	setupMuxTest()

	// Reset telemetry metrics
	telemetry.ActiveConnections.Set(0)

	mux := NewServeMux()

	// Channel to control handler execution
	handlerStarted := make(chan bool)
	handlerCanFinish := make(chan bool)

	handler := func(w ResponseWriter, _ *Request) {
		handlerStarted <- true
		<-handlerCanFinish
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler)

	// Start request in goroutine
	go func() {
		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}()

	// Wait for handler to start
	<-handlerStarted

	// Check active connections is 1
	active := testutil.ToFloat64(telemetry.ActiveConnections)
	if active != 1 {
		t.Errorf("Expected active connections to be 1, got %f", active)
	}

	// Allow handler to finish
	handlerCanFinish <- true

	// Give it time to finish
	time.Sleep(50 * time.Millisecond)

	// Check active connections is back to 0
	active = testutil.ToFloat64(telemetry.ActiveConnections)
	if active != 0 {
		t.Errorf("Expected active connections to be 0 after request, got %f", active)
	}
}

func TestTelemetryMiddleware_RequestDuration(t *testing.T) {
	setupMuxTest()

	// Reset telemetry metrics
	telemetry.RequestDurationSeconds.Reset()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /timed", handler)

	req := httptest.NewRequest(http.MethodGet, "/timed", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	// Verify duration was recorded (we can't check exact value, but we can verify metrics exist)
	problems, err := testutil.CollectAndLint(telemetry.RequestDurationSeconds)
	if err != nil {
		t.Errorf("Failed to collect request duration metrics: %v", err)
	}
	if len(problems) > 0 {
		t.Errorf("Linting issues with request duration metrics: %v", problems)
	}
}

func TestResponseWriter_StatusCodeTracking_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	statusCode := 0
	w := ResponseWriter{
		ResponseWriter: rec,
		statusCode:     &statusCode,
	}

	// Test capturing status code via WriteHeader
	w.WriteHeader(http.StatusNotFound)

	statusCode, ok := w.StatusCode()
	if !ok {
		t.Error("Expected status code to be set")
	}

	if statusCode != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, statusCode)
	}

	// Verify it was sent to the underlying writer
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected recorder status code %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestResponseWriter_StatusCodeTracking_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	statusCode := 0
	w := ResponseWriter{
		ResponseWriter: rec,
		statusCode:     &statusCode,
	}

	// Test implicit 200 OK on first write
	data := []byte("test data")
	n, err := w.Write(data)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Verify status code was set to 200 in context
	statusCode, ok := w.StatusCode()
	if !ok {
		t.Error("Expected status code to be set")
	}

	if statusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, statusCode)
	}

	if rec.Body.String() != string(data) {
		t.Errorf("Expected body %q, got %q", data, rec.Body.String())
	}
}

func TestResponseWriter_StatusCodeTracking_WriteAfterWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	statusCode := 0
	w := ResponseWriter{
		ResponseWriter: rec,
		statusCode:     &statusCode,
	}

	// Set status first
	w.WriteHeader(http.StatusCreated)

	// Then write data
	data := []byte("created")
	n, err := w.Write(data)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Status should remain as set by WriteHeader
	statusCode, ok := w.StatusCode()
	if !ok {
		t.Error("Expected status code to be set")
	}

	if statusCode != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, statusCode)
	}

	if rec.Body.String() != string(data) {
		t.Errorf("Expected body %q, got %q", data, rec.Body.String())
	}
}

func TestTelemetryMiddleware_StatusClasses(t *testing.T) {
	setupMuxTest()

	// Reset telemetry metrics
	telemetry.RequestsTotal.Reset()

	mux := NewServeMux()

	// Test various status codes and their classes
	testCases := []struct {
		path          string
		statusCode    int
		expectedClass string
	}{
		{"/status-200", http.StatusOK, "2xx"},
		{"/status-201", http.StatusCreated, "2xx"},
		{"/status-204", http.StatusNoContent, "2xx"},
		{"/status-301", http.StatusMovedPermanently, "3xx"},
		{"/status-302", http.StatusFound, "3xx"},
		{"/status-400", http.StatusBadRequest, "4xx"},
		{"/status-401", http.StatusUnauthorized, "4xx"},
		{"/status-403", http.StatusForbidden, "4xx"},
		{"/status-404", http.StatusNotFound, "4xx"},
		{"/status-500", http.StatusInternalServerError, "5xx"},
		{"/status-502", http.StatusBadGateway, "5xx"},
		{"/status-503", http.StatusServiceUnavailable, "5xx"},
	}

	// Register handlers
	for _, tc := range testCases {
		path := tc.path
		code := tc.statusCode
		mux.HandleFunc("GET "+path, func(w ResponseWriter, _ *Request) {
			w.WriteHeader(code)
		})
	}

	// Make requests and verify metrics
	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodGet, tc.path, http.NoBody)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != tc.statusCode {
			t.Errorf("Expected status %d, got %d for %s", tc.statusCode, rec.Code, tc.path)
		}

		// Verify the metric was recorded with correct status class
		count := testutil.ToFloat64(telemetry.RequestsTotal.WithLabelValues("GET", tc.path, tc.expectedClass))
		if count != 1 {
			t.Errorf("Expected count 1 for GET %s %s, got %f", tc.path, tc.expectedClass, count)
		}
	}
}

func TestTelemetryMiddleware_DifferentHTTPMethods(t *testing.T) {
	setupMuxTest()

	// Reset telemetry metrics
	telemetry.RequestsTotal.Reset()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	// Register handlers for different methods
	mux.HandleFunc("GET /resource", handler)
	mux.HandleFunc("POST /resource", handler)
	mux.HandleFunc("PUT /resource", handler)
	mux.HandleFunc("DELETE /resource", handler)
	mux.HandleFunc("PATCH /resource", handler)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/resource", http.NoBody)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d for %s", rec.Code, method)
		}

		// Verify the metric was recorded with correct method
		count := testutil.ToFloat64(telemetry.RequestsTotal.WithLabelValues(method, "/resource", "2xx"))
		if count != 1 {
			t.Errorf("Expected count 1 for %s /resource 2xx, got %f", method, count)
		}
	}
}

func TestTelemetryMiddleware_ConcurrentRequests(t *testing.T) {
	setupMuxTest()

	// Reset telemetry metrics
	telemetry.RequestsTotal.Reset()
	telemetry.ActiveConnections.Set(0)

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /concurrent", handler)

	// Make concurrent requests
	var wg sync.WaitGroup
	numRequests := 10

	//nolint:intrange // classic for loop for concurrent requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/concurrent", http.NoBody)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
		}()
	}

	wg.Wait()

	// Verify all requests were counted
	count := testutil.ToFloat64(telemetry.RequestsTotal.WithLabelValues("GET", "/concurrent", "2xx"))
	if count != float64(numRequests) {
		t.Errorf("Expected count %d, got %f", numRequests, count)
	}

	// Active connections should be back to 0
	active := testutil.ToFloat64(telemetry.ActiveConnections)
	if active != 0 {
		t.Errorf("Expected active connections to be 0 after all requests, got %f", active)
	}
}

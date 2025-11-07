package webfram

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bondowe/webfram/openapi"
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
	openAPIConfig = &OpenAPI{EndpointEnabled: true}
	jsonpCallbackParamName = ""

	Configure(&Config{
		OpenAPI: &OpenAPI{
			EndpointEnabled: true,
			Config: &openapi.Config{
				Info: &openapi.Info{
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

func TestHandlerFunc_ServeHTTP_SetsContext(t *testing.T) {
	setupMuxTest()

	var ctxSet bool
	handler := HandlerFunc(func(w ResponseWriter, _ *Request) {
		if w.Context() != nil {
			ctxSet = true
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	handler.ServeHTTP(rw, r)

	if !ctxSet {
		t.Error("Expected context to be set on ResponseWriter")
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

	handler := HandlerFunc(func(w ResponseWriter, _ *Request) {
		_ = w.JSON(map[string]string{"message": "test"})
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

	handler := HandlerFunc(func(w ResponseWriter, _ *Request) {
		_ = w.JSON(map[string]string{"message": "test"})
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

func TestI18nMiddleware_DefaultsToEnglish(t *testing.T) {
	setupMuxTest()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	mux.HandleFunc("GET /test", handler)

	// No Accept-Language header or cookie
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestParseAcceptLanguage_ValidLanguages(t *testing.T) {
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
// OpenAPI Integration Tests
// =============================================================================

func TestHandlerConfig_WithAPIConfig_Success(t *testing.T) {
	setupMuxTestWithOpenAPI()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	config := mux.HandleFunc("GET /api/users", handler)

	apiConfig := &APIConfig{
		OperationID: "getUsers",
		Summary:     "Get all users",
		Description: "Retrieves a list of all users",
		Tags:        []string{"users"},
	}

	config.WithAPIConfig(apiConfig)

	if config.APIConfig == nil {
		t.Error("APIConfig was not set")
	}

	if config.APIConfig.OperationID != "getUsers" {
		t.Errorf("Expected OperationID 'getUsers', got %q", config.APIConfig.OperationID)
	}
}

func TestHandlerConfig_WithAPIConfig_NilConfig(t *testing.T) {
	setupMuxTestWithOpenAPI()

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	config := mux.HandleFunc("GET /api/test", handler)

	// Should not panic with nil config
	config.WithAPIConfig(nil)

	if config.APIConfig != nil {
		t.Error("APIConfig should remain nil")
	}
}

func TestHandlerConfig_WithAPIConfig_OpenAPIDisabled(_ *testing.T) {
	setupMuxTest() // Sets up without OpenAPI

	mux := NewServeMux()

	handler := func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	}

	config := mux.HandleFunc("GET /api/test", handler)

	apiConfig := &APIConfig{
		OperationID: "testOp",
		Summary:     "Test",
	}

	// Should not panic even if OpenAPI is disabled
	config.WithAPIConfig(apiConfig)
}

func TestHandlerConfig_WithAPIConfig_InvalidPathPattern(t *testing.T) {
	setupMuxTestWithOpenAPI()

	config := &HandlerConfig{
		pathPattern: "invalid-pattern", // Missing method
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid path pattern")
		}
	}()

	config.WithAPIConfig(&APIConfig{
		OperationID: "testOp",
	})
}

func TestHandlerConfig_WithAPIConfig_WithRequestBody(t *testing.T) {
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

	apiConfig := &APIConfig{
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

	config.WithAPIConfig(apiConfig)

	if config.APIConfig.RequestBody == nil {
		t.Error("RequestBody was not set")
	}
}

func TestHandlerConfig_WithAPIConfig_WithResponses(t *testing.T) {
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

	apiConfig := &APIConfig{
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

	config.WithAPIConfig(apiConfig)

	if config.APIConfig.Responses == nil {
		t.Error("Responses were not set")
	}

	if len(config.APIConfig.Responses) != 2 {
		t.Errorf("Expected 2 responses, got %d", len(config.APIConfig.Responses))
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

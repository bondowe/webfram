package webfram

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bondowe/webfram/openapi"
	"golang.org/x/text/language"
)

//go:embed testdata/locales/*.json
var testI18nFS2 embed.FS

//go:embed testdata/templates/*.go.html
var testTemplatesFS2 embed.FS

// Test helper structs
type testUser struct {
	Name  string `json:"name" xml:"name" form:"name" validate:"required,minlength=2"`
	Email string `json:"email" xml:"email" form:"email" validate:"required,email"`
	Age   int    `json:"age" xml:"age" form:"age" validate:"min=0,max=150"`
}

// resetAppConfig resets all global app configuration to initial state
func resetAppConfig() {
	appConfigured = false
	appMiddlewares = nil
	openAPIConfig = nil
	jsonpCallbackParamName = ""
}

// =============================================================================
// Configure Tests
// =============================================================================

func TestConfigure_Success(t *testing.T) {
	resetAppConfig()

	cfg := &Config{
		JSONPCallbackParamName: "callback",
		Assets: &Assets{
			FS: testI18nFS2,
			I18nMessages: &I18nMessages{
				Dir: "testdata/locales",
			},
			Templates: &Templates{
				Dir: "testdata/templates",
			},
		},
	}

	Configure(cfg)

	if !appConfigured {
		t.Error("Expected appConfigured to be true")
	}

	if jsonpCallbackParamName != "callback" {
		t.Errorf("Expected jsonpCallbackParamName to be 'callback', got %q", jsonpCallbackParamName)
	}
}

func TestConfigure_WithNilConfig(t *testing.T) {
	resetAppConfig()

	// Should not panic with nil config
	Configure(nil)

	if !appConfigured {
		t.Error("Expected appConfigured to be true even with nil config")
	}
}

func TestConfigure_WithMinimalConfig(t *testing.T) {
	resetAppConfig()

	cfg := &Config{}
	Configure(cfg)

	if !appConfigured {
		t.Error("Expected appConfigured to be true")
	}
}

func TestConfigure_PanicsWhenCalledTwice(t *testing.T) {
	resetAppConfig()

	cfg := &Config{}
	Configure(cfg)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when configuring app twice")
		}
	}()

	Configure(cfg)
}

func TestConfigure_InvalidJSONPCallbackName(t *testing.T) {
	tests := []struct {
		name         string
		callbackName string
	}{
		{"starts with number", "123callback"},
		{"contains dash", "call-back"},
		{"contains dot", "call.back"},
		{"contains space", "call back"},
		{"contains special chars", "callback!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAppConfig()

			cfg := &Config{
				JSONPCallbackParamName: tt.callbackName,
			}

			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic for invalid JSONP callback name %q", tt.callbackName)
				}
			}()

			Configure(cfg)
		})
	}
}

func TestConfigure_ValidJSONPCallbackNames(t *testing.T) {
	tests := []struct {
		name         string
		callbackName string
	}{
		{"simple name", "callback"},
		{"with underscore", "my_callback"},
		{"with numbers", "callback123"},
		{"starts with underscore", "_callback"},
		{"mixed case", "myCallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetAppConfig()

			cfg := &Config{
				JSONPCallbackParamName: tt.callbackName,
			}

			Configure(cfg)

			if jsonpCallbackParamName != tt.callbackName {
				t.Errorf("Expected %q, got %q", tt.callbackName, jsonpCallbackParamName)
			}
		})
	}
}

// =============================================================================
// configureOpenAPI Tests
// =============================================================================

func TestConfigureOpenAPI_NilConfig(t *testing.T) {
	openAPIConfig = nil
	configureOpenAPI(nil)

	if openAPIConfig != nil {
		t.Error("Expected openAPIConfig to remain nil")
	}
}

func TestConfigureOpenAPI_NilOpenAPIConfig(t *testing.T) {
	openAPIConfig = nil
	cfg := &Config{}
	configureOpenAPI(cfg)

	if openAPIConfig != nil {
		t.Error("Expected openAPIConfig to remain nil")
	}
}

func TestConfigureOpenAPI_DisabledEndpoint(t *testing.T) {
	openAPIConfig = &OpenAPI{EndpointEnabled: false}
	cfg := &Config{
		OpenAPI: &OpenAPI{
			EndpointEnabled: false,
			Config:          &openapi.Config{},
		},
	}
	configureOpenAPI(cfg)

	// Should not override when disabled
	if openAPIConfig.EndpointEnabled {
		t.Error("Expected endpoint to remain disabled")
	}
}

func TestConfigureOpenAPI_WithDefaultURL(t *testing.T) {
	// Set up initial state - the function checks openAPIConfig.EndpointEnabled
	// so we need to initialize it first
	openAPIConfig = &OpenAPI{EndpointEnabled: true}

	cfg := &Config{
		OpenAPI: &OpenAPI{
			EndpointEnabled: true,
			Config: &openapi.Config{
				Info: &openapi.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
			},
		},
	}
	configureOpenAPI(cfg)

	if openAPIConfig == nil {
		t.Fatal("Expected openAPIConfig to be set")
	}

	if openAPIConfig.URLPath != defaultOpenAPIURLPath {
		t.Errorf("Expected URLPath %q, got %q", defaultOpenAPIURLPath, openAPIConfig.URLPath)
	}

	if openAPIConfig.Config.Components == nil {
		t.Error("Expected Components to be initialized")
	}
}

func TestConfigureOpenAPI_WithCustomURL(t *testing.T) {
	openAPIConfig = &OpenAPI{EndpointEnabled: true}

	cfg := &Config{
		OpenAPI: &OpenAPI{
			EndpointEnabled: true,
			URLPath:         "/api/spec.json",
			Config:          &openapi.Config{},
		},
	}
	configureOpenAPI(cfg)

	expectedPath := "GET /api/spec.json"
	if openAPIConfig.URLPath != expectedPath {
		t.Errorf("Expected URLPath %q, got %q", expectedPath, openAPIConfig.URLPath)
	}
}

func TestConfigureOpenAPI_URLWithExistingGETPrefix(t *testing.T) {
	openAPIConfig = &OpenAPI{EndpointEnabled: true}

	cfg := &Config{
		OpenAPI: &OpenAPI{
			EndpointEnabled: true,
			URLPath:         "GET /custom.json",
			Config:          &openapi.Config{},
		},
	}
	configureOpenAPI(cfg)

	if openAPIConfig.URLPath != "GET /custom.json" {
		t.Errorf("Expected URLPath to remain unchanged, got %q", openAPIConfig.URLPath)
	}
}

// =============================================================================
// configureTemplate Tests
// =============================================================================

func TestConfigureTemplate_NilConfig(t *testing.T) {
	configureTemplate(nil)
	// Should not panic
}

func TestConfigureTemplate_NilTemplateConfig(t *testing.T) {
	cfg := &Config{}
	configureTemplate(cfg)
	// Should not panic
}

func TestConfigureTemplate_NilFS(t *testing.T) {
	cfg := &Config{
		Assets: &Assets{
			Templates: &Templates{},
		},
	}
	configureTemplate(cfg)
	// Should not panic
}

func TestConfigureTemplate_WithDefaults(t *testing.T) {
	cfg := &Config{
		Assets: &Assets{
			FS: testTemplatesFS2,
			Templates: &Templates{
				Dir: "testdata/templates",
			},
		},
	}
	configureTemplate(cfg)
	// Should use default values without panicking
}

func TestConfigureTemplate_WithCustomValues(t *testing.T) {
	cfg := &Config{
		Assets: &Assets{
			FS: testTemplatesFS2,
			Templates: &Templates{
				Dir:                   "testdata/templates",
				LayoutBaseName:        "customLayout",
				HTMLTemplateExtension: ".html",
				TextTemplateExtension: ".txt",
			},
		},
	}
	configureTemplate(cfg)
	// Should accept custom values without panicking
}

func TestConfigureTemplate_NonExistentDirectory(t *testing.T) {
	cfg := &Config{
		Assets: &Assets{
			FS: testTemplatesFS2,
			Templates: &Templates{
				Dir: "nonexistent/path",
			},
		},
	}
	// Should not panic, just return early when directory doesn't exist
	configureTemplate(cfg)
}

func TestConfigureTemplate_DirectoryIsFile(t *testing.T) {
	cfg := &Config{
		Assets: &Assets{
			FS: testTemplatesFS2,
			Templates: &Templates{
				Dir: "testdata/locales/messages.en.json", // This is a file, not a directory
			},
		},
	}
	// Should not panic, just return early when path is not a directory
	configureTemplate(cfg)
}

// =============================================================================
// configureI18n Tests
// =============================================================================

func TestConfigureI18n_NilConfig(t *testing.T) {
	configureI18n(nil)
	// Should not panic
}

func TestConfigureI18n_NilI18nConfig(t *testing.T) {
	cfg := &Config{}
	configureI18n(cfg)
	// Should not panic
}

func TestConfigureI18n_NilFS(t *testing.T) {
	cfg := &Config{
		Assets: &Assets{
			I18nMessages: &I18nMessages{},
		},
	}
	configureI18n(cfg)
	// Should not panic
}

func TestConfigureI18n_WithFS(t *testing.T) {
	cfg := &Config{
		Assets: &Assets{
			FS: testI18nFS2,
			I18nMessages: &I18nMessages{
				Dir: "testdata/locales",
			},
		},
	}
	configureI18n(cfg)
	// Should configure without panicking
}

func TestConfigureI18n_NonExistentDirectory(t *testing.T) {
	cfg := &Config{
		Assets: &Assets{
			FS: testI18nFS2,
			I18nMessages: &I18nMessages{
				Dir: "nonexistent/path",
			},
		},
	}
	// Should not panic, just return early when directory doesn't exist
	configureI18n(cfg)
}

func TestConfigureI18n_WithCustomDirectory(t *testing.T) {
	cfg := &Config{
		Assets: &Assets{
			FS: testI18nFS2,
			I18nMessages: &I18nMessages{
				Dir: "testdata/locales",
			},
		},
	}
	configureI18n(cfg)
	// Should configure with custom directory without panicking
}

// =============================================================================
// Use Middleware Tests
// =============================================================================

func TestUse_WithAppMiddleware(t *testing.T) {
	resetAppConfig()

	called := false
	mw := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	}

	Use(mw)

	if len(appMiddlewares) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(appMiddlewares))
	}

	// Test that middleware is functional
	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := appMiddlewares[0](handler)

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	wrapped.ServeHTTP(rw, r)

	if !called {
		t.Error("Middleware was not called")
	}
}

func TestUse_WithStandardMiddleware(t *testing.T) {
	resetAppConfig()

	called := false
	headerSet := false

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.Header().Set("X-Custom", "test-value")
			next.ServeHTTP(w, r)
		})
	}

	Use(mw)

	if len(appMiddlewares) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(appMiddlewares))
	}

	// Test that adapted middleware works
	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		if w.Header().Get("X-Custom") == "test-value" {
			headerSet = true
		}
		w.WriteHeader(http.StatusOK)
	})
	wrapped := appMiddlewares[0](handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	wrapped.ServeHTTP(rw, r)

	if !called {
		t.Error("Standard middleware was not called")
	}

	if !headerSet {
		t.Error("Standard middleware did not set header correctly")
	}
}

func TestUse_WithNilMiddleware(t *testing.T) {
	resetAppConfig()

	Use[AppMiddleware](nil)

	if len(appMiddlewares) != 0 {
		t.Errorf("Expected 0 middlewares after adding nil, got %d", len(appMiddlewares))
	}
}

func TestUse_MultipleMiddlewares(t *testing.T) {
	resetAppConfig()

	callOrder := []int{}

	mw1 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			callOrder = append(callOrder, 1)
			next.ServeHTTP(w, r)
		})
	}

	mw2 := func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			callOrder = append(callOrder, 2)
			next.ServeHTTP(w, r)
		})
	}

	Use(mw1)
	Use(mw2)

	if len(appMiddlewares) != 2 {
		t.Errorf("Expected 2 middlewares, got %d", len(appMiddlewares))
	}

	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		callOrder = append(callOrder, 3)
	})

	var wrapped Handler = handler
	for i := len(appMiddlewares) - 1; i >= 0; i-- {
		wrapped = appMiddlewares[i](wrapped)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	wrapped.ServeHTTP(rw, r)

	expected := []int{1, 2, 3}
	if len(callOrder) != len(expected) {
		t.Errorf("Expected call order %v, got %v", expected, callOrder)
	}
	for i, v := range expected {
		if i >= len(callOrder) || callOrder[i] != v {
			t.Errorf("Expected call order %v, got %v", expected, callOrder)
			break
		}
	}
}

// =============================================================================
// SSE Tests
// =============================================================================

func TestSSE_Success(t *testing.T) {
	payloadFunc := func() SSEPayload {
		return SSEPayload{
			Id:    "1",
			Event: "message",
			Data:  "test data",
		}
	}

	disconnectCalled := false
	disconnectFunc := func() {
		disconnectCalled = true
	}

	errorCalled := false
	errorFunc := func(err error) {
		errorCalled = true
	}

	handler := SSE(payloadFunc, disconnectFunc, errorFunc, 100*time.Millisecond, map[string]string{
		"X-Custom-Header": "custom-value",
	})

	if handler == nil {
		t.Fatal("SSE returned nil handler")
	}

	if handler.interval != 100*time.Millisecond {
		t.Errorf("Expected interval 100ms, got %v", handler.interval)
	}

	if handler.payloadFunc == nil {
		t.Error("Expected payloadFunc to be set")
	}

	if handler.disconnectFunc == nil {
		t.Error("Expected disconnectFunc to be set")
	}

	if handler.errorFunc == nil {
		t.Error("Expected errorFunc to be set")
	}

	if handler.headers["X-Custom-Header"] != "custom-value" {
		t.Error("Custom headers not set correctly")
	}

	_ = disconnectCalled
	_ = errorCalled
}

func TestSSE_PanicsOnZeroInterval(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for zero interval")
		}
	}()

	SSE(
		func() SSEPayload { return SSEPayload{} },
		nil,
		nil,
		0,
		nil,
	)
}

func TestSSE_PanicsOnNegativeInterval(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for negative interval")
		}
	}()

	SSE(
		func() SSEPayload { return SSEPayload{} },
		nil,
		nil,
		-1*time.Second,
		nil,
	)
}

func TestSSE_PanicsOnNilPayloadFunc(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil payload function")
		}
	}()

	SSE(nil, nil, nil, 1*time.Second, nil)
}

func TestSSE_DefaultDisconnectFunc(t *testing.T) {
	handler := SSE(
		func() SSEPayload { return SSEPayload{} },
		nil,
		nil,
		1*time.Second,
		nil,
	)

	if handler.disconnectFunc == nil {
		t.Fatal("Expected default disconnect function to be set")
	}

	// Should not panic when called
	handler.disconnectFunc()
}

func TestSSE_DefaultErrorFunc(t *testing.T) {
	handler := SSE(
		func() SSEPayload { return SSEPayload{} },
		nil,
		nil,
		1*time.Second,
		nil,
	)

	if handler.errorFunc == nil {
		t.Fatal("Expected default error function to be set")
	}

	// Should not panic when called
	handler.errorFunc(errors.New("test error"))
}

func TestSSE_ServeHTTP_MethodNotAllowed(t *testing.T) {
	handler := SSE(
		func() SSEPayload { return SSEPayload{Data: "test"} },
		nil,
		nil,
		1*time.Second,
		nil,
	)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/sse", nil)
			rec := httptest.NewRecorder()
			rw := ResponseWriter{ResponseWriter: rec}
			r := &Request{Request: req}

			handler.ServeHTTP(rw, r)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
			}
		})
	}
}

func TestSSE_ServeHTTP_SetsCorrectHeaders(t *testing.T) {
	handler := SSE(
		func() SSEPayload { return SSEPayload{} },
		func() {},
		nil,
		10*time.Millisecond,
		map[string]string{
			"X-Custom": "value",
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 50*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	go handler.ServeHTTP(rw, r)

	time.Sleep(20 * time.Millisecond)
	cancel()

	if rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Expected Content-Type 'text/event-stream', got %q", rec.Header().Get("Content-Type"))
	}

	if rec.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("Expected Cache-Control 'no-cache', got %q", rec.Header().Get("Cache-Control"))
	}

	if rec.Header().Get("Connection") != "keep-alive" {
		t.Errorf("Expected Connection 'keep-alive', got %q", rec.Header().Get("Connection"))
	}

	if rec.Header().Get("X-Custom") != "value" {
		t.Errorf("Expected X-Custom 'value', got %q", rec.Header().Get("X-Custom"))
	}
}

func TestSSE_ServeHTTP_CallsDisconnectOnContext(t *testing.T) {
	var disconnectCalled atomic.Bool
	handler := SSE(
		func() SSEPayload { return SSEPayload{Data: "test"} },
		func() { disconnectCalled.Store(true) },
		nil,
		10*time.Millisecond,
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	go handler.ServeHTTP(rw, r)

	time.Sleep(20 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)

	if !disconnectCalled.Load() {
		t.Error("Expected disconnectFunc to be called")
	}
}

// Mock SSE writer for testing error scenarios
type mockSSEWriter struct {
	http.ResponseWriter
	writeError error
	flushError error
	mu         sync.Mutex
	writeCalls []string
}

func (m *mockSSEWriter) Write(b []byte) (int, error) {
	if m.writeError != nil {
		return 0, m.writeError
	}
	m.mu.Lock()
	m.writeCalls = append(m.writeCalls, string(b))
	m.mu.Unlock()
	return m.ResponseWriter.Write(b)
}

func (m *mockSSEWriter) Flush() error {
	return m.flushError
}

func (m *mockSSEWriter) getCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]string, len(m.writeCalls))
	copy(calls, m.writeCalls)
	return calls
}

// sseTestHelper sets up and runs an SSE test, returning the mock writer's calls
func sseTestHelper(
	t *testing.T,
	payloadFunc SSEPayloadFunc,
	errorFunc SSEErrorFunc,
	writeErr, flushErr error,
) (*mockSSEWriter, context.CancelFunc) {
	t.Helper()
	handler := SSE(payloadFunc, nil, errorFunc, 10*time.Millisecond, nil)

	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 50*time.Millisecond)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	mockWriter := &mockSSEWriter{
		ResponseWriter: rec,
		writeError:     writeErr,
		flushError:     flushErr,
	}

	handler.writerFactory = func(w http.ResponseWriter) sseWriter {
		return mockWriter
	}

	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	go handler.ServeHTTP(rw, r)
	return mockWriter, cancel
}

func TestSSE_ServeHTTP_PayloadIdNotEmpty(t *testing.T) {
	payloadFunc := func() SSEPayload {
		return SSEPayload{
			Id: "message-123",
		}
	}

	mockWriter, cancel := sseTestHelper(t, payloadFunc, nil, nil, nil)
	defer cancel()

	time.Sleep(20 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	calls := mockWriter.getCalls()
	found := false
	for _, call := range calls {
		if strings.Contains(call, "id: message-123\n") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'id: message-123\\n' to be written, got calls: %v", calls)
	}
}

func TestSSE_ServeHTTP_PayloadEventNotEmpty(t *testing.T) {
	payloadFunc := func() SSEPayload {
		return SSEPayload{
			Event: "user-connected",
		}
	}

	mockWriter, cancel := sseTestHelper(t, payloadFunc, nil, nil, nil)
	defer cancel()

	time.Sleep(20 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	calls := mockWriter.getCalls()
	found := false
	for _, call := range calls {
		if strings.Contains(call, "event: user-connected\n") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'event: user-connected\\n' to be written, got calls: %v", calls)
	}
}

func TestSSE_ServeHTTP_PayloadCommentsExist(t *testing.T) {
	payloadFunc := func() SSEPayload {
		return SSEPayload{
			Comments: []string{"comment1", "comment2", "comment3"},
		}
	}

	handler := SSE(payloadFunc, nil, nil, 10*time.Millisecond, nil)

	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 25*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	mockWriter := &mockSSEWriter{ResponseWriter: rec}

	handler.writerFactory = func(w http.ResponseWriter) sseWriter {
		return mockWriter
	}

	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	go handler.ServeHTTP(rw, r)
	time.Sleep(20 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	calls := mockWriter.getCalls()
	expectedComments := []string{": comment1\n", ": comment2\n", ": comment3\n"}
	for _, expected := range expectedComments {
		found := false
		for _, call := range calls {
			if strings.Contains(call, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected '%s' to be written, got calls: %v", expected, calls)
		}
	}
}

// sseErrorTestHelper tests SSE error callback functionality
func sseErrorTestHelper(t *testing.T, expectedErr error, writeErr, flushErr error) {
	t.Helper()
	payloadFunc := func() SSEPayload {
		return SSEPayload{
			Data: "test data",
		}
	}

	var errorCalled atomic.Bool
	var capturedError atomic.Value
	errorFunc := func(err error) {
		errorCalled.Store(true)
		capturedError.Store(err)
	}

	_, cancel := sseTestHelper(t, payloadFunc, errorFunc, writeErr, flushErr)
	defer cancel()

	time.Sleep(30 * time.Millisecond)

	if !errorCalled.Load() {
		t.Error("Expected errorFunc to be called")
	}
	if err := capturedError.Load(); err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestSSE_ServeHTTP_PayloadDataWriteError(t *testing.T) {
	writeErr := errors.New("write failed")
	sseErrorTestHelper(t, writeErr, writeErr, nil)
}

func TestSSE_ServeHTTP_PayloadRetrySuccess(t *testing.T) {
	payloadFunc := func() SSEPayload {
		return SSEPayload{
			Retry: 5000 * time.Millisecond,
		}
	}

	handler := SSE(payloadFunc, nil, nil, 10*time.Millisecond, nil)

	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 25*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	mockWriter := &mockSSEWriter{ResponseWriter: rec}

	handler.writerFactory = func(w http.ResponseWriter) sseWriter {
		return mockWriter
	}

	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	go handler.ServeHTTP(rw, r)
	time.Sleep(20 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	calls := mockWriter.getCalls()
	found := false
	for _, call := range calls {
		if strings.Contains(call, "retry: 5000\n") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'retry: 5000\\n' to be written, got calls: %v", calls)
	}
}

func TestSSE_ServeHTTP_PayloadRetryWriteError(t *testing.T) {
	callCount := 0
	payloadFunc := func() SSEPayload {
		callCount++
		return SSEPayload{
			Retry: 3000 * time.Millisecond,
		}
	}

	var errorCalled atomic.Bool
	var capturedError atomic.Value
	errorFunc := func(err error) {
		errorCalled.Store(true)
		capturedError.Store(err)
	}

	handler := SSE(payloadFunc, nil, errorFunc, 10*time.Millisecond, nil)

	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 50*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	writeErr := errors.New("retry write failed")
	mockWriter := &mockSSEWriter{
		ResponseWriter: rec,
		writeError:     writeErr,
	}

	handler.writerFactory = func(w http.ResponseWriter) sseWriter {
		return mockWriter
	}

	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	go handler.ServeHTTP(rw, r)
	time.Sleep(30 * time.Millisecond)

	if !errorCalled.Load() {
		t.Error("Expected errorFunc to be called when retry write fails")
	}
	if err := capturedError.Load(); err != writeErr {
		t.Errorf("Expected error %v, got %v", writeErr, err)
	}
}

func TestSSE_ServeHTTP_FlushError(t *testing.T) {
	flushErr := errors.New("flush failed")
	sseErrorTestHelper(t, flushErr, nil, flushErr)
}

func TestSSE_ServeHTTP_AllPayloadFieldsSet(t *testing.T) {
	payloadFunc := func() SSEPayload {
		return SSEPayload{
			Id:       "msg-456",
			Event:    "update",
			Comments: []string{"status update"},
			Data:     "complete",
			Retry:    2000 * time.Millisecond,
		}
	}

	handler := SSE(payloadFunc, nil, nil, 10*time.Millisecond, nil)

	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 25*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	mockWriter := &mockSSEWriter{ResponseWriter: rec}

	handler.writerFactory = func(w http.ResponseWriter) sseWriter {
		return mockWriter
	}

	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	go handler.ServeHTTP(rw, r)
	time.Sleep(20 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	calls := mockWriter.getCalls()
	expectedStrings := []string{
		"id: msg-456\n",
		"event: update\n",
		": status update\n",
		"data: complete\n",
		"retry: 2000\n",
	}

	for _, expected := range expectedStrings {
		found := false
		for _, call := range calls {
			if strings.Contains(call, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected '%s' to be written, got calls: %v", expected, calls)
		}
	}
}

// =============================================================================
// ValidationErrors Tests
// =============================================================================

func TestValidationErrors_Any_Empty(t *testing.T) {
	errs := &ValidationErrors{}

	if errs.Any() {
		t.Error("Expected Any() to return false for empty errors")
	}
}

func TestValidationErrors_Any_WithErrors(t *testing.T) {
	errs := &ValidationErrors{
		Errors: []ValidationError{
			{Field: "name", Error: "required"},
		},
	}

	if !errs.Any() {
		t.Error("Expected Any() to return true when errors exist")
	}
}

func TestValidationErrors_Any_MultipleErrors(t *testing.T) {
	errs := &ValidationErrors{
		Errors: []ValidationError{
			{Field: "name", Error: "required"},
			{Field: "email", Error: "invalid format"},
			{Field: "age", Error: "must be positive"},
		},
	}

	if !errs.Any() {
		t.Error("Expected Any() to return true with multiple errors")
	}
}

func TestValidationError_JSONMarshaling(t *testing.T) {
	ve := ValidationError{
		Field: "email",
		Error: "invalid format",
	}

	data, err := json.Marshal(ve)
	if err != nil {
		t.Fatalf("Failed to marshal ValidationError: %v", err)
	}

	var unmarshaled ValidationError
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal ValidationError: %v", err)
	}

	if unmarshaled.Field != ve.Field {
		t.Errorf("Expected Field %q, got %q", ve.Field, unmarshaled.Field)
	}

	if unmarshaled.Error != ve.Error {
		t.Errorf("Expected Error %q, got %q", ve.Error, unmarshaled.Error)
	}
}

func TestValidationError_XMLMarshaling(t *testing.T) {
	ve := ValidationError{
		Field: "age",
		Error: "must be positive",
	}

	data, err := xml.Marshal(ve)
	if err != nil {
		t.Fatalf("Failed to marshal ValidationError to XML: %v", err)
	}

	var unmarshaled ValidationError
	if err := xml.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal ValidationError from XML: %v", err)
	}

	if unmarshaled.Field != ve.Field {
		t.Errorf("Expected Field %q, got %q", ve.Field, unmarshaled.Field)
	}

	if unmarshaled.Error != ve.Error {
		t.Errorf("Expected Error %q, got %q", ve.Error, unmarshaled.Error)
	}
}

func TestValidationErrors_JSONMarshaling(t *testing.T) {
	ves := ValidationErrors{
		Errors: []ValidationError{
			{Field: "name", Error: "required"},
			{Field: "age", Error: "must be positive"},
		},
	}

	data, err := json.Marshal(ves)
	if err != nil {
		t.Fatalf("Failed to marshal ValidationErrors: %v", err)
	}

	var unmarshaled ValidationErrors
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal ValidationErrors: %v", err)
	}

	if len(unmarshaled.Errors) != len(ves.Errors) {
		t.Errorf("Expected %d errors, got %d", len(ves.Errors), len(unmarshaled.Errors))
	}
}

// =============================================================================
// BindJSON Tests
// =============================================================================

func TestBindJSON_Success(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	body := `{"name":"John Doe","email":"john@example.com","age":30}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r := &Request{Request: req}

	result, valErrs, err := BindJSON[testUser](r, false)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if valErrs.Any() {
		t.Errorf("Unexpected validation errors: %+v", valErrs)
	}

	if result.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got %q", result.Name)
	}

	if result.Email != "john@example.com" {
		t.Errorf("Expected Email 'john@example.com', got %q", result.Email)
	}

	if result.Age != 30 {
		t.Errorf("Expected Age 30, got %d", result.Age)
	}
}

func TestBindJSON_WithValidation_Valid(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	body := `{"name":"John","email":"john@example.com","age":25}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r := &Request{Request: req}

	result, valErrs, err := BindJSON[testUser](r, true)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if valErrs.Any() {
		t.Errorf("Unexpected validation errors: %+v", valErrs)
	}

	if result.Name != "John" {
		t.Errorf("Expected Name 'John', got %q", result.Name)
	}
}

func TestBindJSON_WithValidation_Invalid(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	body := `{"name":"J","email":"invalid","age":-5}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r := &Request{Request: req}

	_, valErrs, err := BindJSON[testUser](r, true)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !valErrs.Any() {
		t.Error("Expected validation errors but got none")
	}

	if len(valErrs.Errors) == 0 {
		t.Error("Expected at least one validation error")
	}
}

func TestBindJSON_MalformedJSON(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r := &Request{Request: req}

	_, _, err := BindJSON[testUser](r, false)

	if err == nil {
		t.Error("Expected error for malformed JSON")
	}
}

func TestBindJSON_EmptyBody(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	r := &Request{Request: req}

	_, _, err := BindJSON[testUser](r, false)

	if err == nil {
		t.Error("Expected error for empty body")
	}
}

// =============================================================================
// BindXML Tests
// =============================================================================

func TestBindXML_Success(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	body := `<testUser><name>John Doe</name><email>john@example.com</email><age>30</age></testUser>`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/xml")
	r := &Request{Request: req}

	result, valErrs, err := BindXML[testUser](r, false)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if valErrs.Any() {
		t.Errorf("Unexpected validation errors: %+v", valErrs)
	}

	if result.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got %q", result.Name)
	}
}

func TestBindXML_WithValidation_Valid(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	body := `<testUser><name>John</name><email>john@example.com</email><age>25</age></testUser>`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/xml")
	r := &Request{Request: req}

	result, valErrs, err := BindXML[testUser](r, true)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if valErrs.Any() {
		t.Errorf("Unexpected validation errors: %+v", valErrs)
	}

	if result.Name != "John" {
		t.Errorf("Expected Name 'John', got %q", result.Name)
	}
}

func TestBindXML_WithValidation_Invalid(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	body := `<testUser><name>J</name><email>invalid</email><age>200</age></testUser>`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/xml")
	r := &Request{Request: req}

	_, valErrs, err := BindXML[testUser](r, true)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !valErrs.Any() {
		t.Error("Expected validation errors but got none")
	}
}

func TestBindXML_MalformedXML(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	body := `<invalid><unclosed>`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/xml")
	r := &Request{Request: req}

	_, _, err := BindXML[testUser](r, false)

	if err == nil {
		t.Error("Expected error for malformed XML")
	}
}

// =============================================================================
// BindForm Tests
// =============================================================================

func TestBindForm_Success(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	body := "name=John+Doe&email=john%40example.com&age=30"
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r := &Request{Request: req}

	result, valErrs, err := BindForm[testUser](r)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if valErrs.Any() {
		t.Errorf("Unexpected validation errors: %+v", valErrs)
	}

	if result.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got %q", result.Name)
	}
}

// bindFormValidationHelper tests BindForm validation errors
func bindFormValidationHelper(t *testing.T, body, expectedField, expectedErrSubstr string) {
	t.Helper()
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r := &Request{Request: req}

	_, valErrs, err := BindForm[testUser](r)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !valErrs.Any() {
		t.Fatal("Expected validation errors but got none")
	}

	foundError := false
	for _, e := range valErrs.Errors {
		if e.Field == expectedField {
			foundError = true
			if !strings.Contains(e.Error, expectedErrSubstr) {
				t.Errorf("Expected error containing %q for %s field, got: %s", expectedErrSubstr, expectedField, e.Error)
			}
		}
	}

	if !foundError {
		t.Errorf("Expected validation error for %s field, got errors: %+v", expectedField, valErrs.Errors)
	}
}

func TestBindForm_ValidationError_MissingRequiredField(t *testing.T) {
	bindFormValidationHelper(t, "email=john%40example.com&age=30", "Name", "required")
}

func TestBindForm_ValidationError_MinLength(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	// Name is too short (minlength=2)
	body := "name=A&email=john%40example.com&age=30"
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r := &Request{Request: req}

	result, valErrs, err := BindForm[testUser](r)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !valErrs.Any() {
		t.Fatal("Expected validation errors but got none")
	}

	// Check that we have validation error for 'name' field
	foundNameError := false
	for _, e := range valErrs.Errors {
		if e.Field == "Name" {
			foundNameError = true
			if !strings.Contains(e.Error, "at least") && !strings.Contains(e.Error, "characters") {
				t.Errorf("Expected minlength error message for Name field, got: %s", e.Error)
			}
		}
	}

	if !foundNameError {
		t.Errorf("Expected validation error for Name field, got errors: %+v", valErrs.Errors)
	}

	if result.Name != "A" {
		t.Errorf("Expected Name 'A', got %q", result.Name)
	}
}

func TestBindForm_ValidationError_MinValue(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	// Age is below minimum (min=0)
	body := "name=John+Doe&email=john%40example.com&age=-5"
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r := &Request{Request: req}

	result, valErrs, err := BindForm[testUser](r)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !valErrs.Any() {
		t.Fatal("Expected validation errors but got none")
	}

	// Check that we have validation error for 'age' field
	foundAgeError := false
	for _, e := range valErrs.Errors {
		if e.Field == "Age" {
			foundAgeError = true
			if !strings.Contains(e.Error, "at least") {
				t.Errorf("Expected min value error message for Age field, got: %s", e.Error)
			}
		}
	}

	if !foundAgeError {
		t.Errorf("Expected validation error for Age field, got errors: %+v", valErrs.Errors)
	}

	if result.Age != -5 {
		t.Errorf("Expected Age -5, got %d", result.Age)
	}
}

func TestBindForm_ValidationError_MaxValue(t *testing.T) {
	bindFormValidationHelper(t, "name=John+Doe&email=john%40example.com&age=200", "Age", "at most")
}

func TestBindForm_ValidationError_MultipleFields(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	// Multiple validation errors: missing name, missing email, age too high
	body := "age=200"
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r := &Request{Request: req}

	result, valErrs, err := BindForm[testUser](r)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !valErrs.Any() {
		t.Fatal("Expected validation errors but got none")
	}

	// Should have at least 3 validation errors
	if len(valErrs.Errors) < 3 {
		t.Errorf("Expected at least 3 validation errors, got %d: %+v", len(valErrs.Errors), valErrs.Errors)
	}

	// Check for specific field errors
	fieldErrors := make(map[string]bool)
	for _, e := range valErrs.Errors {
		fieldErrors[e.Field] = true
	}

	if !fieldErrors["Name"] {
		t.Error("Expected validation error for Name field")
	}

	if !fieldErrors["Email"] {
		t.Error("Expected validation error for Email field")
	}

	if !fieldErrors["Age"] {
		t.Error("Expected validation error for Age field")
	}

	// Result should still be returned with provided/default values
	if result.Name != "" {
		t.Errorf("Expected empty Name, got %q", result.Name)
	}
	if result.Email != "" {
		t.Errorf("Expected empty Email, got %q", result.Email)
	}
	if result.Age != 200 {
		t.Errorf("Expected Age 200, got %d", result.Age)
	}
}

func TestBindForm_ValidationError_EmptyForm(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	// Empty form body
	body := ""
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r := &Request{Request: req}

	result, valErrs, err := BindForm[testUser](r)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !valErrs.Any() {
		t.Fatal("Expected validation errors but got none")
	}

	// Should have errors for all required fields
	if len(valErrs.Errors) < 2 {
		t.Errorf("Expected at least 2 validation errors for required fields, got %d: %+v", len(valErrs.Errors), valErrs.Errors)
	}

	// Result should have zero values
	var zeroUser testUser
	if result.Name != zeroUser.Name || result.Email != zeroUser.Email || result.Age != zeroUser.Age {
		t.Errorf("Expected zero values for result, got: %+v", result)
	}
}

func TestBindForm_ValidationErrors_ReturnsValidationErrorsStruct(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	// Invalid data
	body := "name=A&age=300"
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r := &Request{Request: req}

	_, valErrs, err := BindForm[testUser](r)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check ValidationErrors struct methods
	if !valErrs.Any() {
		t.Error("Expected valErrs.Any() to return true")
	}

	// Check that each error has Field and Error properties
	for _, e := range valErrs.Errors {
		if e.Field == "" {
			t.Error("Expected Field to be set in ValidationError")
		}
		if e.Error == "" {
			t.Error("Expected Error message to be set in ValidationError")
		}
	}
}

// =============================================================================
// PatchJSON Tests
// =============================================================================

func TestPatchJSON_Success(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	target := testUser{
		Name:  "Old Name",
		Email: "old@example.com",
		Age:   25,
	}

	patch := `[{"op":"replace","path":"/name","value":"New Name"}]`
	req := httptest.NewRequest(http.MethodPatch, "/test", strings.NewReader(patch))
	req.Header.Set("Content-Type", "application/json-patch+json")
	r := &Request{Request: req}

	valErrs, err := PatchJSON(r, &target, false)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(valErrs) > 0 {
		t.Errorf("Unexpected validation errors: %+v", valErrs)
	}

	if target.Name != "New Name" {
		t.Errorf("Expected Name 'New Name', got %q", target.Name)
	}

	if target.Email != "old@example.com" {
		t.Errorf("Email should remain unchanged, got %q", target.Email)
	}
}

func TestPatchJSON_WithValidation_Valid(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	target := testUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
	}

	patch := `[{"op":"replace","path":"/age","value":30}]`
	req := httptest.NewRequest(http.MethodPatch, "/test", strings.NewReader(patch))
	req.Header.Set("Content-Type", "application/json-patch+json")
	r := &Request{Request: req}

	valErrs, err := PatchJSON(r, &target, true)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(valErrs) > 0 {
		t.Errorf("Unexpected validation errors: %+v", valErrs)
	}

	if target.Age != 30 {
		t.Errorf("Expected Age 30, got %d", target.Age)
	}
}

func TestPatchJSON_WithValidation_Invalid(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	target := testUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
	}

	patch := `[{"op":"replace","path":"/age","value":200}]`
	req := httptest.NewRequest(http.MethodPatch, "/test", strings.NewReader(patch))
	req.Header.Set("Content-Type", "application/json-patch+json")
	r := &Request{Request: req}

	valErrs, err := PatchJSON(r, &target, true)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(valErrs) == 0 {
		t.Error("Expected validation errors but got none")
	}
}

func TestPatchJSON_MethodNotAllowed(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	target := testUser{}
	patch := `[]`

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", strings.NewReader(patch))
			req.Header.Set("Content-Type", "application/json-patch+json")
			r := &Request{Request: req}

			_, err := PatchJSON(r, &target, false)

			if !errors.Is(err, ErrMethodNotAllowed) {
				t.Errorf("Expected ErrMethodNotAllowed, got %v", err)
			}
		})
	}
}

func TestPatchJSON_InvalidContentType(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	target := testUser{}
	patch := `[]`

	req := httptest.NewRequest(http.MethodPatch, "/test", strings.NewReader(patch))
	req.Header.Set("Content-Type", "application/json")
	r := &Request{Request: req}

	_, err := PatchJSON(r, &target, false)

	if err == nil {
		t.Error("Expected error for invalid Content-Type")
	}

	if !strings.Contains(err.Error(), "Content-Type") {
		t.Errorf("Expected error to mention Content-Type, got %v", err)
	}
}

func TestPatchJSON_InvalidPatchFormat(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	target := testUser{}
	patch := `[{"invalid":"patch"}]`

	req := httptest.NewRequest(http.MethodPatch, "/test", strings.NewReader(patch))
	req.Header.Set("Content-Type", "application/json-patch+json")
	r := &Request{Request: req}

	_, err := PatchJSON(r, &target, false)

	if err == nil {
		t.Error("Expected error for invalid patch format")
	}
}

func TestPatchJSON_MultipleOperations(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	target := testUser{
		Name:  "John",
		Email: "john@example.com",
		Age:   25,
	}

	patch := `[
		{"op":"replace","path":"/name","value":"Jane"},
		{"op":"replace","path":"/age","value":30}
	]`
	req := httptest.NewRequest(http.MethodPatch, "/test", strings.NewReader(patch))
	req.Header.Set("Content-Type", "application/json-patch+json")
	r := &Request{Request: req}

	valErrs, err := PatchJSON(r, &target, false)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(valErrs) > 0 {
		t.Errorf("Unexpected validation errors: %+v", valErrs)
	}

	if target.Name != "Jane" {
		t.Errorf("Expected Name 'Jane', got %q", target.Name)
	}

	if target.Age != 30 {
		t.Errorf("Expected Age 30, got %d", target.Age)
	}
}

// =============================================================================
// GetI18nPrinter Tests
// =============================================================================

func TestGetI18nPrinter_English(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	printer := GetI18nPrinter(language.English)

	if printer == nil {
		t.Fatal("GetI18nPrinter returned nil")
	}

	result := printer.Sprintf("Test %s", "message")
	if result == "" {
		t.Error("Printer returned empty string")
	}
}

func TestGetI18nPrinter_MultipleLanguages(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	languages := []language.Tag{
		language.English,
		language.Spanish,
		language.French,
	}

	for _, lang := range languages {
		t.Run(lang.String(), func(t *testing.T) {
			printer := GetI18nPrinter(lang)

			if printer == nil {
				t.Fatalf("GetI18nPrinter returned nil for %s", lang)
			}
		})
	}
}

// =============================================================================
// Adapter Tests
// =============================================================================

func TestAdaptToHTTPHandler(t *testing.T) {
	handlerCalled := false
	var receivedStatus int

	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusCreated)
	})

	httpHandler := adaptToHTTPHandler(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	httpHandler.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("Handler was not called")
	}

	receivedStatus = rec.Code
	if receivedStatus != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, receivedStatus)
	}
}

func TestAdaptHTTPHandler(t *testing.T) {
	httpHandlerCalled := false

	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpHandlerCalled = true
		w.Header().Set("X-Test", "value")
		w.WriteHeader(http.StatusOK)
	})

	handler := adaptHTTPHandler(httpHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	handler.ServeHTTP(rw, r)

	if !httpHandlerCalled {
		t.Error("HTTP handler was not called")
	}

	if rec.Header().Get("X-Test") != "value" {
		t.Error("Header not set correctly")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAdaptHTTPMiddleware2(t *testing.T) {
	middlewareCalled := false

	httpMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			w.Header().Set("X-Middleware", "applied")
			next.ServeHTTP(w, r)
		})
	}

	appMw := adaptHTTPMiddleware(httpMw)

	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := appMw(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: rec}
	r := &Request{Request: req}

	wrapped.ServeHTTP(rw, r)

	if !middlewareCalled {
		t.Error("Middleware was not called")
	}

	if rec.Header().Get("X-Middleware") != "applied" {
		t.Error("Middleware did not apply header")
	}
}

// =============================================================================
// Constants and Error Tests
// =============================================================================

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant interface{}
		expected interface{}
	}{
		{"defaultOpenAPIURLPath", defaultOpenAPIURLPath, "GET /openapi.json"},
		{"defaultLayoutBaseName", defaultLayoutBaseName, "layout"},
		{"defaultHTMLTemplateExtension", defaultHTMLTemplateExtension, ".go.html"},
		{"defaultTextTemplateExtension", defaultTextTemplateExtension, ".go.txt"},
		{"defaultI18nMessagesDir", defaultI18nMessagesDir, "i18n"},
		{"defaultI18nFuncName", defaultI18nFuncName, "T"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %v to be %v, got %v", tt.name, tt.expected, tt.constant)
			}
		})
	}
}

func TestErrMethodNotAllowed(t *testing.T) {
	if ErrMethodNotAllowed == nil {
		t.Fatal("ErrMethodNotAllowed is nil")
	}

	expectedMsg := "method not allowed"
	if ErrMethodNotAllowed.Error() != expectedMsg {
		t.Errorf("Expected error message %q, got %q", expectedMsg, ErrMethodNotAllowed.Error())
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestGetValueOrDefault_WithZeroValue(t *testing.T) {
	result := getValueOrDefault("", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got %q", result)
	}
}

func TestGetValueOrDefault_WithProvidedValue(t *testing.T) {
	result := getValueOrDefault("custom", "default")
	if result != "custom" {
		t.Errorf("Expected 'custom', got %q", result)
	}
}

func TestGetValueOrDefault_IntType(t *testing.T) {
	result := getValueOrDefault(0, 42)
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}

	result = getValueOrDefault(10, 42)
	if result != 10 {
		t.Errorf("Expected 10, got %d", result)
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkConfigure(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resetAppConfig()
		Configure(&Config{
			Assets: &Assets{
				FS:           testI18nFS2,
				I18nMessages: &I18nMessages{Dir: "testdata/locales"},
			},
		})
	}
}

func BenchmarkBindJSON(b *testing.B) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	body := `{"name":"John Doe","email":"john@example.com","age":30}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		r := &Request{Request: req}

		_, _, _ = BindJSON[testUser](r, false)
	}
}

func BenchmarkBindJSON_WithValidation(b *testing.B) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	body := `{"name":"John Doe","email":"john@example.com","age":30}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		r := &Request{Request: req}

		_, _, _ = BindJSON[testUser](r, true)
	}
}

func BenchmarkPatchJSON(b *testing.B) {
	resetAppConfig()
	Configure(&Config{
		Assets: &Assets{
			FS:           testI18nFS2,
			I18nMessages: &I18nMessages{Dir: "testdata/locales"},
		},
	})

	patch := `[{"op":"replace","path":"/name","value":"New Name"}]`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		target := testUser{Name: "Old Name", Email: "old@example.com", Age: 25}
		req := httptest.NewRequest(http.MethodPatch, "/test", bytes.NewReader([]byte(patch)))
		req.Header.Set("Content-Type", "application/json-patch+json")
		r := &Request{Request: req}

		_, _ = PatchJSON(r, &target, false)
	}
}

func BenchmarkAdaptHTTPMiddleware(b *testing.B) {
	httpMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adaptHTTPMiddleware(httpMw)
	}
}

func BenchmarkSSE_PayloadGeneration(b *testing.B) {
	handler := SSE(
		func() SSEPayload {
			return SSEPayload{
				Id:    "1",
				Event: "message",
				Data:  "test data",
			}
		},
		nil,
		nil,
		1*time.Second,
		nil,
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handler.payloadFunc()
	}
}

package webfram

import (
	"bytes"
	"embed"
	"encoding/json"
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bondowe/webfram/openapi"
	"github.com/google/uuid"
	"golang.org/x/text/language"
)

//go:embed testdata/locales/*.json
var testI18nFS2 embed.FS

//go:embed testdata/templates/*.go.html
var testTemplateFS embed.FS

func resetAppConfig() {
	appConfigured = false
	appMiddlewares = nil
	openAPIConfig = nil
	jsonpCallbackParamName = ""
}

func TestConfigure(t *testing.T) {
	resetAppConfig()

	cfg := &Config{
		JSONPCallbackParamName: "callback",
		I18n: &I18nConfig{
			FS: testI18nFS2,
		},
		Templates: &TemplateConfig{
			FS:            testTemplateFS,
			TemplatesPath: "testdata/templates",
		},
	}

	Configure(cfg)

	if !appConfigured {
		t.Error("Expected appConfigured to be true")
	}

	if jsonpCallbackParamName != "callback" {
		t.Errorf("Expected jsonpCallbackParamName 'callback', got %q", jsonpCallbackParamName)
	}
}

func TestConfigure_Panic_AlreadyConfigured(t *testing.T) {
	resetAppConfig()

	cfg := &Config{
		I18n: &I18nConfig{
			FS: testI18nFS2,
		},
	}

	Configure(cfg)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when configuring app twice")
		}
	}()

	Configure(cfg)
}

func TestConfigure_InvalidJSONPCallbackName(t *testing.T) {
	resetAppConfig()

	cfg := &Config{
		JSONPCallbackParamName: "invalid-name!", // Invalid characters
		I18n: &I18nConfig{
			FS: testI18nFS2,
		},
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid JSONP callback name")
		}
	}()

	Configure(cfg)
}

func TestConfigure_NilConfig(t *testing.T) {
	resetAppConfig()

	// Should not panic
	Configure(nil)

	if !appConfigured {
		t.Error("Expected appConfigured to be true even with nil config")
	}
}

func TestConfigureOpenAPI(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectNil   bool
		expectedURL string
	}{
		{
			name:      "nil config",
			config:    nil,
			expectNil: true,
		},
		{
			name: "enabled with default URL",
			config: &Config{
				OpenAPI: &OpenAPIConfig{
					EndpointEnabled: true,
					Config: &openapi.Config{
						Info: &openapi.Info{
							Title:   "Test",
							Version: "1.0",
						},
					},
				},
			},
			expectNil:   false,
			expectedURL: defaultOpenAPIURLPath,
		},
		{
			name: "enabled with custom URL",
			config: &Config{
				OpenAPI: &OpenAPIConfig{
					EndpointEnabled: true,
					URLPath:         "/custom.json",
					Config: &openapi.Config{
						Info: &openapi.Info{
							Title:   "Test",
							Version: "1.0",
						},
					},
				},
			},
			expectNil:   false,
			expectedURL: "GET /custom.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openAPIConfig = nil
			configureOpenAPI(tt.config)

			if tt.expectNil && openAPIConfig != nil {
				t.Error("Expected openAPIConfig to be nil")
			}

			if !tt.expectNil && openAPIConfig == nil {
				t.Error("Expected openAPIConfig to not be nil")
			}

			if !tt.expectNil && openAPIConfig.URLPath != tt.expectedURL {
				t.Errorf("Expected URLPath %q, got %q", tt.expectedURL, openAPIConfig.URLPath)
			}
		})
	}
}

func TestUse_AppMiddleware(t *testing.T) {
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

	// Test the middleware works
	handler := HandlerFunc(func(w ResponseWriter, r *Request) {})
	wrapped := appMiddlewares[0](handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}
	r := &Request{Request: req}

	wrapped.ServeHTTP(rw, r)

	if !called {
		t.Error("Middleware was not called")
	}
}

func TestUse_StandardMiddleware(t *testing.T) {
	resetAppConfig()

	called := false
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	}

	Use(mw)

	if len(appMiddlewares) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(appMiddlewares))
	}

	// Test the middleware works
	handler := HandlerFunc(func(w ResponseWriter, r *Request) {})
	wrapped := appMiddlewares[0](handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}
	r := &Request{Request: req}

	wrapped.ServeHTTP(rw, r)

	if !called {
		t.Error("Standard middleware was not called")
	}
}

func TestUse_NilMiddleware(t *testing.T) {
	resetAppConfig()

	Use[AppMiddleware](nil)

	if len(appMiddlewares) != 0 {
		t.Errorf("Expected 0 middlewares after adding nil, got %d", len(appMiddlewares))
	}
}

func TestSSE(t *testing.T) {
	payloadFunc := func() SSEPayload {
		return SSEPayload{
			Id:    "1",
			Event: "message",
			Data:  "test",
		}
	}

	disconnectFunc := func() {
	}

	errorFunc := func(err error) {
	}

	handler := SSE(payloadFunc, disconnectFunc, errorFunc, 1*time.Second, map[string]string{
		"X-Custom": "value",
	})

	if handler == nil {
		t.Fatal("SSE returned nil handler")
	}

	if handler.interval != 1*time.Second {
		t.Errorf("Expected interval 1s, got %v", handler.interval)
	}

	if handler.headers["X-Custom"] != "value" {
		t.Error("Custom headers not set")
	}
}

func TestSSE_PanicOnZeroInterval(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for zero interval")
		}
	}()

	SSE(func() SSEPayload { return SSEPayload{} }, nil, nil, 0, nil)
}

func TestSSE_PanicOnNilPayloadFunc(t *testing.T) {
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
		t.Error("Expected default disconnect function")
	}

	// Should not panic
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
		t.Error("Expected default error function")
	}

	// Should not panic
	handler.errorFunc(errors.New("test error"))
}

func TestSSEHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	handler := SSE(
		func() SSEPayload { return SSEPayload{Data: "test"} },
		nil,
		nil,
		1*time.Second,
		nil,
	)

	req := httptest.NewRequest("POST", "/sse", nil)
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}
	r := &Request{Request: req}

	handler.ServeHTTP(rw, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestValidationErrors_Any(t *testing.T) {
	tests := []struct {
		name     string
		errors   ValidationErrors
		expected bool
	}{
		{
			name:     "no errors",
			errors:   ValidationErrors{},
			expected: false,
		},
		{
			name: "has errors",
			errors: ValidationErrors{
				Errors: []ValidationError{
					{Field: "name", Error: "required"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.errors.Any()
			if result != tt.expected {
				t.Errorf("Expected Any() = %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBindJSON(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		I18n: &I18nConfig{FS: testI18nFS2},
	})

	type TestStruct struct {
		Name  string `json:"name" validate:"required"`
		Value int    `json:"value" validate:"min=1,max=100"`
	}

	tests := []struct {
		name          string
		body          string
		validate      bool
		expectError   bool
		expectValErrs bool
	}{
		{
			name:          "valid data without validation",
			body:          `{"name":"test","value":50}`,
			validate:      false,
			expectError:   false,
			expectValErrs: false,
		},
		{
			name:          "valid data with validation",
			body:          `{"name":"test","value":50}`,
			validate:      true,
			expectError:   false,
			expectValErrs: false,
		},
		{
			name:          "invalid data with validation",
			body:          `{"name":"","value":200}`,
			validate:      true,
			expectError:   false,
			expectValErrs: true,
		},
		{
			name:          "malformed JSON",
			body:          `{invalid}`,
			validate:      false,
			expectError:   true,
			expectValErrs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			r := &Request{Request: req}

			result, valErrs, err := BindJSON[TestStruct](r, tt.validate)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectValErrs && !valErrs.Any() {
				t.Error("Expected validation errors but got none")
			}

			if !tt.expectValErrs && valErrs.Any() {
				t.Errorf("Unexpected validation errors: %+v", valErrs)
			}

			if !tt.expectError && !tt.expectValErrs {
				if result.Name == "" {
					t.Error("Expected non-empty result")
				}
			}
		})
	}
}

func TestBindXML(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		I18n: &I18nConfig{FS: testI18nFS2},
	})

	type TestStruct struct {
		XMLName xml.Name `xml:"test"`
		Name    string   `xml:"name" validate:"required"`
		Value   int      `xml:"value" validate:"min=1,max=100"`
	}

	tests := []struct {
		name          string
		body          string
		validate      bool
		expectError   bool
		expectValErrs bool
	}{
		{
			name:          "valid XML without validation",
			body:          `<test><name>test</name><value>50</value></test>`,
			validate:      false,
			expectError:   false,
			expectValErrs: false,
		},
		{
			name:          "valid XML with validation",
			body:          `<test><name>test</name><value>50</value></test>`,
			validate:      true,
			expectError:   false,
			expectValErrs: false,
		},
		{
			name:          "invalid XML data with validation",
			body:          `<test><name></name><value>200</value></test>`,
			validate:      true,
			expectError:   false,
			expectValErrs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/xml")
			r := &Request{Request: req}

			result, valErrs, err := BindXML[TestStruct](r, tt.validate)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectValErrs && !valErrs.Any() {
				t.Error("Expected validation errors but got none")
			}

			if !tt.expectError && !tt.expectValErrs {
				if result.Name == "" {
					t.Error("Expected non-empty result")
				}
			}
		})
	}
}

func TestPatchJSON(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		I18n: &I18nConfig{FS: testI18nFS2},
	})

	type TestStruct struct {
		Name  string `json:"name" validate:"required,minlength=3"`
		Value int    `json:"value" validate:"min=1,max=100"`
	}

	tests := []struct {
		name          string
		method        string
		contentType   string
		body          string
		target        TestStruct
		validate      bool
		expectError   bool
		expectValErrs bool
		errorContains string
	}{
		{
			name:        "valid patch",
			method:      "PATCH",
			contentType: "application/json-patch+json",
			body:        `[{"op":"replace","path":"/name","value":"newname"}]`,
			target:      TestStruct{Name: "oldname", Value: 50},
			validate:    false,
			expectError: false,
		},
		{
			name:          "method not allowed",
			method:        "POST",
			contentType:   "application/json-patch+json",
			body:          `[]`,
			target:        TestStruct{},
			validate:      false,
			expectError:   true,
			errorContains: "method not allowed",
		},
		{
			name:          "invalid content type",
			method:        "PATCH",
			contentType:   "application/json",
			body:          `[]`,
			target:        TestStruct{},
			validate:      false,
			expectError:   true,
			errorContains: "Content-Type",
		},
		{
			name:          "validation errors",
			method:        "PATCH",
			contentType:   "application/json-patch+json",
			body:          `[{"op":"replace","path":"/name","value":"ab"}]`,
			target:        TestStruct{Name: "oldname", Value: 50},
			validate:      true,
			expectError:   false,
			expectValErrs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			r := &Request{Request: req}

			target := tt.target
			valErrs, err := PatchJSON(r, &target, tt.validate)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectValErrs && len(valErrs) == 0 {
				t.Error("Expected validation errors but got none")
			}
		})
	}
}

func TestGetI18nPrinter(t *testing.T) {
	resetAppConfig()
	Configure(&Config{
		I18n: &I18nConfig{FS: testI18nFS2},
	})

	printer := GetI18nPrinter(language.English)
	if printer == nil {
		t.Fatal("GetI18nPrinter returned nil")
	}

	result := printer.Sprintf("Hello %s", "World")
	if result != "Hello World" {
		t.Errorf("Expected 'Hello World', got %q", result)
	}
}

func TestAdaptToHTTPHandler(t *testing.T) {
	called := false
	var receivedW http.ResponseWriter
	var receivedR *http.Request

	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		called = true
	})

	httpHandler := adaptToHTTPHandler(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	httpHandler.ServeHTTP(w, req)

	if !called {
		t.Error("Handler was not called")
	}

	_ = receivedW
	_ = receivedR
}

func TestAdaptHTTPHandler(t *testing.T) {
	called := false
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := adaptHTTPHandler(httpHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}
	r := &Request{Request: req}

	handler.ServeHTTP(rw, r)

	if !called {
		t.Error("HTTP handler was not called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAdaptHTTPMiddleware2(t *testing.T) {
	middlewareCalled := false
	httpMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			w.Header().Set("X-Test", "value")
			next.ServeHTTP(w, r)
		})
	}

	appMw := adaptHTTPMiddleware(httpMw)

	handler := HandlerFunc(func(w ResponseWriter, r *Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := appMw(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}
	r := &Request{Request: req}

	wrapped.ServeHTTP(rw, r)

	if !middlewareCalled {
		t.Error("Middleware was not called")
	}

	if w.Header().Get("X-Test") != "value" {
		t.Error("Middleware did not set header")
	}
}

func TestConstants(t *testing.T) {
	if defaultOpenAPIURLPath != "GET /openapi.json" {
		t.Errorf("Unexpected defaultOpenAPIURLPath: %q", defaultOpenAPIURLPath)
	}

	if defaultLayoutBaseName != "layout" {
		t.Errorf("Unexpected defaultLayoutBaseName: %q", defaultLayoutBaseName)
	}

	if defaultHTMLTemplateExtension != ".go.html" {
		t.Errorf("Unexpected defaultHTMLTemplateExtension: %q", defaultHTMLTemplateExtension)
	}

	if defaultTextTemplateExtension != ".go.txt" {
		t.Errorf("Unexpected defaultTextTemplateExtension: %q", defaultTextTemplateExtension)
	}

	if defaultI18nFuncName != "T" {
		t.Errorf("Unexpected defaultI18nFuncName: %q", defaultI18nFuncName)
	}
}

func TestErrMethodNotAllowed(t *testing.T) {
	if ErrMethodNotAllowed == nil {
		t.Fatal("ErrMethodNotAllowed is nil")
	}

	if !strings.Contains(ErrMethodNotAllowed.Error(), "method not allowed") {
		t.Errorf("Unexpected error message: %q", ErrMethodNotAllowed.Error())
	}
}

func TestValidationError_Marshaling(t *testing.T) {
	ve := ValidationError{
		Field: "name",
		Error: "is required",
	}

	// JSON marshaling
	jsonData, err := json.Marshal(ve)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	var unmarshaledJSON ValidationError
	if err := json.Unmarshal(jsonData, &unmarshaledJSON); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if unmarshaledJSON.Field != ve.Field || unmarshaledJSON.Error != ve.Error {
		t.Error("JSON roundtrip failed")
	}

	// XML marshaling
	xmlData, err := xml.Marshal(ve)
	if err != nil {
		t.Fatalf("Failed to marshal to XML: %v", err)
	}

	var unmarshaledXML ValidationError
	if err := xml.Unmarshal(xmlData, &unmarshaledXML); err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	if unmarshaledXML.Field != ve.Field || unmarshaledXML.Error != ve.Error {
		t.Error("XML roundtrip failed")
	}
}

func TestValidationErrors_Marshaling(t *testing.T) {
	ves := ValidationErrors{
		Errors: []ValidationError{
			{Field: "name", Error: "is required"},
			{Field: "age", Error: "must be positive"},
		},
	}

	// JSON marshaling
	jsonData, err := json.Marshal(ves)
	if err != nil {
		t.Fatalf("Failed to marshal to JSON: %v", err)
	}

	var unmarshaledJSON ValidationErrors
	if err := json.Unmarshal(jsonData, &unmarshaledJSON); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(unmarshaledJSON.Errors) != len(ves.Errors) {
		t.Error("JSON roundtrip failed")
	}
}

func TestUserAndAddressStructs(t *testing.T) {
	// Test that the example structs are properly defined
	user := User{
		Name:      "Test User",
		Role:      "admin",
		Birthdate: time.Now(),
		Email:     "test@example.com",
		UserID:    uuid.New(),
		Address: Address{
			Street: "123 Main St",
			City:   "Test City",
			Zip:    12345,
		},
	}

	if user.Name != "Test User" {
		t.Error("User struct not working correctly")
	}

	if user.Address.City != "Test City" {
		t.Error("Address struct not working correctly")
	}
}

func TestProductStruct(t *testing.T) {
	product := Product{
		Name:        "Test Product",
		SKU:         "ABC-1234",
		Price:       9999,
		Category:    "electronics",
		Description: "A test product",
	}

	if product.Name != "Test Product" {
		t.Error("Product struct not working correctly")
	}
}

func BenchmarkBindJSON(b *testing.B) {
	resetAppConfig()
	Configure(&Config{
		I18n: &I18nConfig{FS: testI18nFS2},
	})

	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	body := `{"name":"test","value":42}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		r := &Request{Request: req}

		BindJSON[TestStruct](r, false)
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

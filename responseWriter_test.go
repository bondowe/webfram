package webfram

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bondowe/webfram/internal/i18n"
	"golang.org/x/text/language"
	yaml "sigs.k8s.io/yaml/goyaml.v2"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey3 string

const testContextKey3 contextKey3 = "test-key"

//go:embed testdata/templates/*.go.html
//go:embed testdata/templates/*.go.txt
var testTemplatesFS embed.FS

func setupResponseWriterTests() {
	if appConfigured {
		appConfigured = false
	}

	Configure(&Config{
		Assets: &Assets{
			FS: testTemplatesFS,
			Templates: &Templates{
				Dir: "testdata/templates",
			},
			I18nMessages: &I18nMessages{
				Dir: "testdata/locales",
			},
		},
	})
}

func TestResponseWriter_Context(t *testing.T) {
	w := httptest.NewRecorder()
	ctx := context.WithValue(context.Background(), testContextKey3, "test-value")
	rw := ResponseWriter{
		ResponseWriter: w,
		context:        ctx,
	}

	result := rw.Context()
	if result == nil {
		t.Fatal("Context() returned nil")
	}

	val := result.Value(testContextKey3)
	if val != "test-value" {
		t.Errorf("Expected context value 'test-value', got %v", val)
	}
}

func TestResponseWriter_Error(t *testing.T) {
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}

	rw.Error(http.StatusBadRequest, "Bad request error")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	body := strings.TrimSpace(w.Body.String())
	if !strings.Contains(body, "Bad request error") {
		t.Errorf("Expected body to contain 'Bad request error', got %q", body)
	}
}

func TestResponseWriter_Header(t *testing.T) {
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}

	rw.Header().Set("X-Custom-Header", "custom-value")

	if val := w.Header().Get("X-Custom-Header"); val != "custom-value" {
		t.Errorf("Expected header 'custom-value', got %q", val)
	}
}

func TestResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}

	data := []byte("Hello, World!")
	n, err := rw.Write(data)

	if err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}

	if n != len(data) {
		t.Errorf("Write() returned %d bytes, want %d", n, len(data))
	}

	if w.Body.String() != "Hello, World!" {
		t.Errorf("Expected body 'Hello, World!', got %q", w.Body.String())
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := ResponseWriter{
		ResponseWriter: w,
		context:        context.Background(),
	}

	rw.WriteHeader(http.StatusCreated)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	// Verify status code is stored in context
	statusCode, ok := rw.StatusCode()
	if !ok {
		t.Error("Expected status code to be set in context")
	}
	if statusCode != http.StatusCreated {
		t.Errorf("Expected status code %d in context, got %d", http.StatusCreated, statusCode)
	}
}

func TestResponseWriter_StatusCode(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*ResponseWriter)
		expectedCode   int
		expectedExists bool
	}{
		{
			name: "status code set via WriteHeader",
			setupFunc: func(rw *ResponseWriter) {
				rw.WriteHeader(http.StatusOK)
			},
			expectedCode:   http.StatusOK,
			expectedExists: true,
		},
		{
			name: "status code set via NoContent",
			setupFunc: func(rw *ResponseWriter) {
				rw.NoContent()
			},
			expectedCode:   http.StatusNoContent,
			expectedExists: true,
		},
		{
			name: "multiple WriteHeader calls - first wins",
			setupFunc: func(rw *ResponseWriter) {
				rw.WriteHeader(http.StatusCreated)
				rw.WriteHeader(http.StatusOK) // This will be ignored by http.ResponseWriter but updates context
			},
			expectedCode:   http.StatusOK, // Context gets updated with latest call
			expectedExists: true,
		},
		{
			name: "status code not set",
			setupFunc: func(_ *ResponseWriter) {
				// Don't write any headers
			},
			expectedCode:   0,
			expectedExists: false,
		},
		{
			name: "various status codes",
			setupFunc: func(rw *ResponseWriter) {
				rw.WriteHeader(http.StatusNotFound)
			},
			expectedCode:   http.StatusNotFound,
			expectedExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			rw := ResponseWriter{
				ResponseWriter: w,
				context:        context.Background(),
			}

			tt.setupFunc(&rw)

			statusCode, ok := rw.StatusCode()
			if ok != tt.expectedExists {
				t.Errorf("StatusCode() exists = %v, want %v", ok, tt.expectedExists)
			}
			if statusCode != tt.expectedCode {
				t.Errorf("StatusCode() = %d, want %d", statusCode, tt.expectedCode)
			}
		})
	}
}

func TestResponseWriter_StatusCode_WithExistingContext(t *testing.T) {
	w := httptest.NewRecorder()

	// Create context with some existing values
	ctx := context.Background()
	ctx = context.WithValue(ctx, testContextKey3, "existing-value")

	rw := ResponseWriter{
		ResponseWriter: w,
		context:        ctx,
	}

	// Write a status code
	rw.WriteHeader(http.StatusAccepted)

	// Verify status code is accessible
	statusCode, ok := rw.StatusCode()
	if !ok {
		t.Error("Expected status code to be set")
	}
	if statusCode != http.StatusAccepted {
		t.Errorf("Expected status code %d, got %d", http.StatusAccepted, statusCode)
	}

	// Verify existing context value is still accessible
	existingVal := rw.Context().Value(testContextKey3)
	if existingVal != "existing-value" {
		t.Errorf("Expected existing context value 'existing-value', got %v", existingVal)
	}
}

func TestResponseWriter_StatusCode_AfterJSONResponse(t *testing.T) {
	w := httptest.NewRecorder()
	rw := ResponseWriter{
		ResponseWriter: w,
		context:        context.Background(),
	}

	// JSON automatically writes status 200 if not explicitly set
	data := map[string]string{"key": "value"}
	err := rw.JSON(data)
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}

	// httptest.ResponseRecorder defaults to 200 after first write
	// But our StatusCode() only tracks explicit WriteHeader calls
	statusCode, ok := rw.StatusCode()
	if ok {
		t.Errorf("Expected status code not to be set (implicit 200), but got %d", statusCode)
	}
}

func TestResponseWriter_StatusCode_ExplicitThenImplicit(t *testing.T) {
	w := httptest.NewRecorder()
	rw := ResponseWriter{
		ResponseWriter: w,
		context:        context.Background(),
	}

	// Explicitly set status before writing
	rw.WriteHeader(http.StatusCreated)

	// Write some data
	data := map[string]string{"key": "value"}
	err := rw.JSON(data)
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}

	// Should still have the explicit status code
	statusCode, ok := rw.StatusCode()
	if !ok {
		t.Error("Expected status code to be set")
	}
	if statusCode != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, statusCode)
	}
}

func TestResponseWriter_Flush(t *testing.T) {
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}

	// Should not panic even if underlying writer doesn't support Flush
	rw.Flush()

	// With a flusher
	flusher := &mockFlusher{ResponseRecorder: w}
	rw.ResponseWriter = flusher

	rw.Flush()

	if !flusher.flushed {
		t.Error("Expected Flush to be called")
	}
}

func TestResponseWriter_Hijack(t *testing.T) {
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}

	// httptest.ResponseRecorder doesn't support Hijack
	conn, buf, err := rw.Hijack()

	if !errors.Is(err, http.ErrNotSupported) {
		t.Errorf("Expected http.ErrNotSupported, got %v", err)
	}

	if conn != nil {
		t.Error("Expected nil conn")
	}

	if buf != nil {
		t.Error("Expected nil buffer")
	}
}

func TestResponseWriter_Push(t *testing.T) {
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}

	// httptest.ResponseRecorder doesn't support Push
	err := rw.Push("/resource", nil)

	if !errors.Is(err, http.ErrNotSupported) {
		t.Errorf("Expected http.ErrNotSupported, got %v", err)
	}
}

func TestResponseWriter_ReadFrom(t *testing.T) {
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}

	reader := strings.NewReader("test data")

	// httptest.ResponseRecorder doesn't support ReadFrom
	n, err := rw.ReadFrom(reader)

	if !errors.Is(err, http.ErrNotSupported) {
		t.Errorf("Expected http.ErrNotSupported, got %v", err)
	}

	if n != 0 {
		t.Errorf("Expected 0 bytes read, got %d", n)
	}
}

func TestResponseWriter_Unwrap(t *testing.T) {
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}

	unwrapped := rw.Unwrap()

	if unwrapped != w {
		t.Error("Unwrap() did not return the underlying ResponseWriter")
	}
}

func TestResponseWriter_JSON(t *testing.T) {
	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name     string
		expected string
		data     TestData
	}{
		{
			name:     "simple object",
			data:     TestData{Name: "test", Value: 42},
			expected: `{"name":"test","value":42}`,
		},
		{
			name:     "empty object",
			data:     TestData{},
			expected: `{"name":"","value":0}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			rw := ResponseWriter{
				ResponseWriter: w,
				context:        context.Background(),
			}

			err := rw.JSON(tt.data)
			if err != nil {
				t.Fatalf("JSON() returned error: %v", err)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
			}

			var result TestData
			if unmarshalErr := json.Unmarshal(w.Body.Bytes(), &result); unmarshalErr != nil {
				t.Fatalf("Failed to unmarshal response: %v", unmarshalErr)
			}

			if result.Name != tt.data.Name || result.Value != tt.data.Value {
				t.Errorf("Expected %+v, got %+v", tt.data, result)
			}
		})
	}
}

func TestResponseWriter_JSON_JSONP(t *testing.T) {
	setupResponseWriterTests()

	type TestData struct {
		Message string `json:"message"`
	}

	w := httptest.NewRecorder()
	ctx := context.WithValue(context.Background(), jsonpCallbackMethodNameKey, "myCallback")
	rw := ResponseWriter{
		ResponseWriter: w,
		context:        ctx,
	}

	data := TestData{Message: "hello"}
	err := rw.JSON(data)

	if err != nil {
		t.Fatalf("JSON() returned error: %v", err)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/javascript" {
		t.Errorf("Expected Content-Type 'application/javascript', got %q", contentType)
	}

	body := w.Body.String()
	if !strings.HasPrefix(body, "myCallback(") {
		t.Errorf("Expected JSONP response to start with 'myCallback(', got %q", body)
	}
	if !strings.HasSuffix(body, ");") {
		t.Errorf("Expected JSONP response to end with ');', got %q", body)
	}
}

func TestResponseWriter_XML(t *testing.T) {
	type TestData struct {
		XMLName xml.Name `xml:"data"`
		Name    string   `xml:"name"`
		Value   int      `xml:"value"`
	}

	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}

	data := TestData{Name: "test", Value: 42}
	err := rw.XML(data)

	if err != nil {
		t.Fatalf("XML() returned error: %v", err)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/xml" {
		t.Errorf("Expected Content-Type 'application/xml', got %q", contentType)
	}

	var result TestData
	if unmarshalErr := xml.Unmarshal(w.Body.Bytes(), &result); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal response: %v", unmarshalErr)
	}

	if result.Name != data.Name || result.Value != data.Value {
		t.Errorf("Expected %+v, got %+v", data, result)
	}
}

func TestResponseWriter_YAML(t *testing.T) {
	type TestData struct {
		Name  string `yaml:"name"`
		Value int    `yaml:"value"`
	}

	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}

	data := TestData{Name: "test", Value: 42}
	err := rw.YAML(data)

	if err != nil {
		t.Fatalf("YAML() returned error: %v", err)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/x-yaml" {
		t.Errorf("Expected Content-Type 'text/x-yaml', got %q", contentType)
	}

	var result TestData
	if unmarshalErr := yaml.Unmarshal(w.Body.Bytes(), &result); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal response: %v", unmarshalErr)
	}

	if result.Name != data.Name || result.Value != data.Value {
		t.Errorf("Expected %+v, got %+v", data, result)
	}
}

func TestResponseWriter_Bytes(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    string
		data        []byte
	}{
		{
			name:        "with explicit content type",
			data:        []byte("Hello"),
			contentType: "text/plain",
			expected:    "text/plain",
		},
		{
			name:        "auto-detect content type",
			data:        []byte("<html>"),
			contentType: "",
			expected:    "text/html; charset=utf-8",
		},
		{
			name:        "json content type",
			data:        []byte(`{"key":"value"}`),
			contentType: "application/json",
			expected:    "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			rw := ResponseWriter{ResponseWriter: w}

			err := rw.Bytes(tt.data, tt.contentType)
			if err != nil {
				t.Fatalf("Bytes() returned error: %v", err)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != tt.expected {
				t.Errorf("Expected Content-Type %q, got %q", tt.expected, contentType)
			}

			if !bytes.Equal(w.Body.Bytes(), tt.data) {
				t.Errorf("Expected body %q, got %q", string(tt.data), w.Body.String())
			}
		})
	}
}

func TestResponseWriter_NoContent(t *testing.T) {
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}

	rw.NoContent()

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestResponseWriter_Redirect(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		code     int
		expected int
	}{
		{
			name:     "permanent redirect",
			url:      "/new-location",
			code:     http.StatusMovedPermanently,
			expected: http.StatusMovedPermanently,
		},
		{
			name:     "temporary redirect",
			url:      "/temp-location",
			code:     http.StatusTemporaryRedirect,
			expected: http.StatusTemporaryRedirect,
		},
		{
			name:     "see other",
			url:      "/other",
			code:     http.StatusSeeOther,
			expected: http.StatusSeeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			rw := ResponseWriter{ResponseWriter: w}

			req := httptest.NewRequest(http.MethodGet, "/original", http.NoBody)
			r := &Request{Request: req}

			rw.Redirect(r, tt.url, tt.code)

			if w.Code != tt.expected {
				t.Errorf("Expected status %d, got %d", tt.expected, w.Code)
			}

			location := w.Header().Get("Location")
			if location != tt.url {
				t.Errorf("Expected Location %q, got %q", tt.url, location)
			}
		})
	}
}

func TestResponseWriter_HTMLString(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]string
		contains string
	}{
		{
			name:     "simple template",
			template: "<h1>{{.Title}}</h1>",
			data:     map[string]string{"Title": "Hello"},
			contains: "<h1>Hello</h1>",
		},
		{
			name:     "template with multiple values",
			template: "<p>{{.Name}} - {{.Value}}</p>",
			data:     map[string]string{"Name": "Test", "Value": "123"},
			contains: "<p>Test - 123</p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			rw := ResponseWriter{ResponseWriter: w}

			err := rw.HTMLString(tt.template, tt.data)
			if err != nil {
				t.Fatalf("HTMLString() returned error: %v", err)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "text/html" {
				t.Errorf("Expected Content-Type 'text/html', got %q", contentType)
			}

			body := w.Body.String()
			if body != tt.contains {
				t.Errorf("Expected body %q, got %q", tt.contains, body)
			}
		})
	}
}

func TestResponseWriter_TextString_InvalidTemplate(t *testing.T) {
	w := httptest.NewRecorder()
	rw := ResponseWriter{ResponseWriter: w}

	err := rw.TextString("{{.Invalid", nil)
	if err == nil {
		t.Error("Expected error for invalid template")
	}
}

func TestResponseWriter_ServeFile(t *testing.T) {
	setupResponseWriterTests()

	tests := []struct {
		name                string
		filename            string
		expectedDisposition string
		inline              bool
	}{
		{
			name:                "inline file",
			filename:            "testdata/templates/test.go.html",
			inline:              true,
			expectedDisposition: "inline",
		},
		{
			name:                "attachment file",
			filename:            "testdata/templates/test.go.html",
			inline:              false,
			expectedDisposition: "attachment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			rw := ResponseWriter{ResponseWriter: w}

			req := httptest.NewRequest(http.MethodGet, "/file", http.NoBody)
			r := &Request{Request: req}

			rw.ServeFile(r, tt.filename, tt.inline)

			disposition := w.Header().Get("Content-Disposition")
			if !strings.HasPrefix(disposition, tt.expectedDisposition) {
				t.Errorf("Expected Content-Disposition to start with %q, got %q",
					tt.expectedDisposition, disposition)
			}
		})
	}
}

func TestI18nPrinterFunc(t *testing.T) {
	setupResponseWriterTests()

	printer := i18n.GetI18nPrinter(language.English)
	fn := i18nPrinterFunc(printer)

	result := fn("Hello %s", "World")
	if result != "Hello World" {
		t.Errorf("Expected 'Hello World', got %q", result)
	}
}

// Mock types for testing interfaces

type mockFlusher struct {
	*httptest.ResponseRecorder

	flushed bool
}

func (m *mockFlusher) Flush() {
	m.flushed = true
}

type mockHijacker struct {
	*httptest.ResponseRecorder

	hijacked bool
}

func (m *mockHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	m.hijacked = true
	return nil, nil, nil
}

type mockPusher struct {
	*httptest.ResponseRecorder

	target string
	pushed bool
}

func (m *mockPusher) Push(target string, _ *http.PushOptions) error {
	m.pushed = true
	m.target = target
	return nil
}

type mockReaderFrom struct {
	*httptest.ResponseRecorder

	readBytes int64
}

func (m *mockReaderFrom) ReadFrom(r io.Reader) (int64, error) {
	n, err := io.Copy(m.ResponseRecorder, r)
	m.readBytes = n
	return n, err
}

func TestResponseWriter_Hijack_Supported(t *testing.T) {
	w := httptest.NewRecorder()
	hijacker := &mockHijacker{ResponseRecorder: w}
	rw := ResponseWriter{ResponseWriter: hijacker}

	_, _, err := rw.Hijack()
	if err != nil {
		t.Errorf("Hijack() returned error: %v", err)
	}

	if !hijacker.hijacked {
		t.Error("Expected Hijack to be called")
	}
}

func TestResponseWriter_Push_Supported(t *testing.T) {
	w := httptest.NewRecorder()
	pusher := &mockPusher{ResponseRecorder: w}
	rw := ResponseWriter{ResponseWriter: pusher}

	err := rw.Push("/resource", nil)
	if err != nil {
		t.Errorf("Push() returned error: %v", err)
	}

	if !pusher.pushed {
		t.Error("Expected Push to be called")
	}

	if pusher.target != "/resource" {
		t.Error("Expected Hijack to be called")
	}
}

func TestResponseWriter_ReadFrom_Supported(t *testing.T) {
	w := httptest.NewRecorder()
	rf := &mockReaderFrom{ResponseRecorder: w}
	rw := ResponseWriter{ResponseWriter: rf}

	data := "test data"
	reader := strings.NewReader(data)

	n, err := rw.ReadFrom(reader)
	if err != nil {
		t.Errorf("ReadFrom() returned error: %v", err)
	}

	if n != int64(len(data)) {
		t.Errorf("Expected %d bytes read, got %d", len(data), n)
	}

	if rf.readBytes != int64(len(data)) {
		t.Errorf("Expected %d bytes in mockReaderFrom, got %d", len(data), rf.readBytes)
	}
}

func BenchmarkResponseWriter_JSON(b *testing.B) {
	type Data struct {
		Name  string
		Value int
	}

	data := Data{Name: "benchmark", Value: 123}

	b.ResetTimer()
	for b.Loop() {
		w := httptest.NewRecorder()
		rw := ResponseWriter{
			ResponseWriter: w,
			context:        context.Background(),
		}
		_ = rw.JSON(data)
	}
}

func BenchmarkResponseWriter_XML(b *testing.B) {
	type Data struct {
		XMLName xml.Name `xml:"data"`
		Name    string   `xml:"name"`
		Value   int      `xml:"value"`
	}

	data := Data{Name: "benchmark", Value: 123}

	b.ResetTimer()
	for b.Loop() {
		w := httptest.NewRecorder()
		rw := ResponseWriter{ResponseWriter: w}
		_ = rw.XML(data)
	}
}

func BenchmarkResponseWriter_Bytes(b *testing.B) {
	data := []byte("benchmark data")

	b.ResetTimer()
	for b.Loop() {
		w := httptest.NewRecorder()
		rw := ResponseWriter{ResponseWriter: w}
		_ = rw.Bytes(data, "text/plain")
	}
}

func TestResponseWriter_HTML(t *testing.T) {
	setupResponseWriterTests()

	tests := []struct {
		data        any
		name        string
		path        string
		wantContain string
		wantError   bool
	}{
		{
			name:        "valid template",
			path:        "test",
			data:        map[string]string{"Title": "Test Page"},
			wantError:   false,
			wantContain: "Test Page",
		},
		{
			name:      "template not found",
			path:      "nonexistent",
			data:      nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			rw := ResponseWriter{
				ResponseWriter: w,
				context:        context.Background(),
			}

			err := rw.HTML(tt.path, tt.data)
			if (err != nil) != tt.wantError {
				t.Errorf("HTML() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				if ct := w.Header().Get("Content-Type"); ct != "text/html" {
					t.Errorf("Expected Content-Type 'text/html', got %q", ct)
				}
			}
		})
	}
}

func TestResponseWriter_HTML_WithI18n(t *testing.T) {
	setupResponseWriterTests()

	w := httptest.NewRecorder()

	// Create context with i18n printer
	printer := i18n.GetI18nPrinter(language.English)
	ctx := i18n.ContextWithI18nPrinter(context.Background(), printer)

	rw := ResponseWriter{
		ResponseWriter: w,
		context:        ctx,
	}

	err := rw.HTML("test", map[string]string{"Title": "I18n Test"})
	if err != nil {
		t.Fatalf("HTML() error = %v", err)
	}

	if ct := w.Header().Get("Content-Type"); ct != "text/html" {
		t.Errorf("Expected Content-Type 'text/html', got %q", ct)
	}
}

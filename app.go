// Package webfram provides a lightweight web framework with built-in support for i18n, templating, and OpenAPI.
package webfram

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bondowe/webfram/internal/bind"
	"github.com/bondowe/webfram/internal/i18n"
	"github.com/bondowe/webfram/internal/telemetry"
	"github.com/bondowe/webfram/internal/template"
	"github.com/bondowe/webfram/openapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	jsonpatch "github.com/evanphx/json-patch"
)

type (
	contextKey string
	// Middleware is a generic middleware function that wraps handlers.
	Middleware[H any] = func(H) H
	// AppMiddleware is a middleware for custom Handler types.
	AppMiddleware = Middleware[Handler]
	// StandardMiddleware is a middleware for standard http.Handler types.
	StandardMiddleware = Middleware[http.Handler]

	// SSEPayload represents a Server-Sent Events message payload.
	SSEPayload struct {
		// Data is the event data.
		Data any
		// ID is the event ID.
		ID string
		// Event is the event type.
		Event string
		// Comments are optional comments for the event.
		Comments []string
		// Retry is the reconnection time in case of connection loss.
		Retry time.Duration
	}
	// SSEPayloadFunc is a function that generates SSE payloads.
	SSEPayloadFunc func() SSEPayload
	// SSEDisconnectFunc is called when an SSE connection is closed.
	SSEDisconnectFunc func()
	// SSEErrorFunc is called when an SSE error occurs.
	SSEErrorFunc func(error)

	// sseWriter interface for testability.
	sseWriter interface {
		http.ResponseWriter
		Flush() error
	}

	// defaultSSEWriter wraps http.ResponseWriter with flush capability.
	defaultSSEWriter struct {
		http.ResponseWriter

		rc *http.ResponseController
	}

	// SSEHandler is the handler returned by SSE function for server-sent events.
	SSEHandler struct {
		headers        map[string]string
		payloadFunc    SSEPayloadFunc
		disconnectFunc SSEDisconnectFunc
		errorFunc      SSEErrorFunc
		writerFactory  func(http.ResponseWriter) sseWriter
		interval       time.Duration
	}

	// ValidationError represents a single field validation error.
	ValidationError struct {
		XMLName xml.Name `json:"-"     xml:"validationError" form:"-"`
		Field   string   `json:"field" xml:"field"           form:"field"`
		Error   string   `json:"error" xml:"error"           form:"error"`
	}

	// ValidationErrors represents a collection of validation errors.
	ValidationErrors struct {
		XMLName xml.Name          `json:"-"      xml:"validationErrors" form:"-"`
		Errors  []ValidationError `json:"errors" xml:"errors"           form:"errors"`
	}

	// Templates configures template settings for the framework.
	Templates struct {
		// Dir is the directory where template files are located.
		Dir string
		// LayoutBaseName is the base name of the layout template.
		LayoutBaseName string
		// HTMLTemplateExtension is the file extension for HTML templates.
		HTMLTemplateExtension string
		// TextTemplateExtension is the file extension for text templates.
		TextTemplateExtension string
	}

	// Telemetry configures telemetry settings for the framework.
	Telemetry struct {
		// UseDefaultRegistry indicates whether to use the default Prometheus registry.
		UseDefaultRegistry bool
		// Collectors are custom Prometheus collectors to register.
		Collectors []prometheus.Collector
		// URLPath is the HTTP path for the metrics endpoint (e.g., "GET /metrics").
		URLPath string
		// Addr is the optional address for a separate telemetry server (e.g., ":9090").
		// If empty or equal to the main server address, telemetry runs on the main server.
		Addr string
		// Enabled indicates whether telemetry is enabled.
		Enabled bool
		// HandlerOpts are options for the Prometheus HTTP handler.
		HandlerOpts promhttp.HandlerOpts
	}

	// I18nMessages configures internationalization message settings.
	I18nMessages struct {
		// Dir is the directory where i18n message files are located.
		Dir string
		// SupportedLanguages is a list of supported language tags.
		SupportedLanguages []string
	}

	// Assets configures static assets and their locations.
	Assets struct {
		// FS is the file system containing the static assets.
		FS fs.FS
		// Templates configures template settings for the framework.
		Templates *Templates
		// I18nMessages configures internationalization message settings.
		I18nMessages *I18nMessages
	}

	// OpenAPI configures OpenAPI documentation settings.
	OpenAPI struct {
		// Config is the OpenAPI configuration.
		Config *openapi.Config
		// URLPath is the HTTP path for the OpenAPI JSON endpoint (e.g., "GET /openapi.json").
		URLPath string
		// Enabled indicates whether OpenAPI documentation is enabled.
		Enabled bool
	}

	// Config represents the framework configuration.
	Config struct {
		// Telemetry configures telemetry settings for the framework.
		Telemetry *Telemetry
		// I18nMessages configures internationalization message settings.
		I18nMessages *I18nMessages
		// Assets configures static assets and their locations.
		Assets *Assets
		// OpenAPI configures OpenAPI documentation settings.
		OpenAPI *OpenAPI
		// JSONPCallbackParamName is the name of the query parameter for JSONP callbacks.
		JSONPCallbackParamName string
	}
)

const (
	jsonpCallbackMethodNameKey   contextKey = "jsonpCallbackMethodName"
	defaultTelemetryURLPath      string     = "GET /metrics"
	defaultOpenAPIURLPath        string     = "GET /openapi.json"
	defaultTemplateDir           string     = "assets/templates"
	defaultLayoutBaseName        string     = "layout"
	defaultHTMLTemplateExtension string     = ".go.html"
	defaultTextTemplateExtension string     = ".go.txt"
	defaultI18nMessagesDir       string     = "assets/locales"
	defaultI18nFuncName          string     = "T"
)

//nolint:gochecknoglobals // Package-level state for framework configuration and middleware
var (
	appConfigured            = false
	appMiddlewares           []AppMiddleware
	telemetryConfig          *Telemetry
	openAPIConfig            *OpenAPI
	jsonpCallbackParamName   string
	jsonpCallbackNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	defaultLanguage          = language.English

	// ErrMethodNotAllowed is returned when an HTTP method is not allowed for a route.
	ErrMethodNotAllowed = errors.New("method not allowed")
)

func (w *defaultSSEWriter) Flush() error {
	return w.rc.Flush()
}

func adaptToHTTPHandler(h Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customW := &ResponseWriter{ResponseWriter: w}
		customR := &Request{Request: r}
		h.ServeHTTP(*customW, customR)
	})
}

func adaptHTTPHandler(h http.Handler) Handler {
	return HandlerFunc(func(w ResponseWriter, r *Request) {
		h.ServeHTTP(w.ResponseWriter, r.Request)
	})
}

func adaptHTTPMiddleware(mw StandardMiddleware) AppMiddleware {
	return func(h Handler) Handler {
		httpHandler := adaptToHTTPHandler(h)
		wrappedHTTPHandler := mw(httpHandler)
		return adaptHTTPHandler(wrappedHTTPHandler)
	}
}

func (m *SSEHandler) ServeHTTP(w ResponseWriter, r *Request) {
	if r.Method != http.MethodGet {
		http.Error(w.ResponseWriter, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	for k, v := range m.headers {
		w.Header().Set(k, v)
	}

	clientDisconnected := r.Context().Done()

	var sseW sseWriter
	if m.writerFactory != nil {
		sseW = m.writerFactory(w.ResponseWriter)
	} else {
		sseW = &defaultSSEWriter{
			ResponseWriter: w.ResponseWriter,
			rc:             http.NewResponseController(w.ResponseWriter),
		}
	}

	t := time.NewTicker(m.interval)
	defer t.Stop()

	msgWritten := false

	for {
		select {
		case <-clientDisconnected:
			m.disconnectFunc()
			return
		case <-t.C:
			payload := m.payloadFunc()

			if payload.ID != "" {
				_, err := fmt.Fprintf(sseW, "id: %s\n", payload.ID)
				if err != nil {
					m.errorFunc(err)
					return
				}
				msgWritten = true
			}
			if payload.Event != "" {
				_, err := fmt.Fprintf(sseW, "event: %s\n", payload.Event)
				if err != nil {
					m.errorFunc(err)
					return
				}
				msgWritten = true
			}
			if len(payload.Comments) > 0 {
				for _, comment := range payload.Comments {
					_, err := fmt.Fprintf(sseW, ": %s\n", comment)
					if err != nil {
						m.errorFunc(err)
						return
					}
				}
				msgWritten = true
			}
			if payload.Data != nil {
				_, err := fmt.Fprintf(sseW, "data: %s\n", payload.Data)
				if err != nil {
					m.errorFunc(err)
					return
				}
				msgWritten = true
			}
			if payload.Retry > 0 {
				_, err := fmt.Fprintf(sseW, "retry: %d\n", int(payload.Retry.Milliseconds()))
				if err != nil {
					m.errorFunc(err)
					return
				}
				msgWritten = true
			}

			if msgWritten {
				_, err := fmt.Fprintf(sseW, "\n")
				if err != nil {
					m.errorFunc(err)
					return
				}

				err = sseW.Flush()
				if err != nil {
					m.errorFunc(err)
					return
				}
			}
		}
	}
}

func configureTelemetry(cfg *Config) {
	if cfg == nil || cfg.Telemetry == nil || !cfg.Telemetry.Enabled {
		return
	}
	telemetryConfig = cfg.Telemetry

	telemetry.ConfigureTelemetry(telemetryConfig.UseDefaultRegistry, telemetryConfig.Collectors...)

	if telemetryConfig.URLPath == "" {
		telemetryConfig.URLPath = defaultTelemetryURLPath
	} else if telemetryConfig.URLPath[0:4] != "GET " {
		telemetryConfig.URLPath = "GET " + telemetryConfig.URLPath
	}
}

func configureOpenAPI(cfg *Config) {
	if cfg == nil || cfg.OpenAPI == nil || !cfg.OpenAPI.Enabled {
		return
	}
	openAPIConfig = cfg.OpenAPI

	openAPIConfig.Config.Components = &openapi.Components{}
	if openAPIConfig.URLPath == "" {
		openAPIConfig.URLPath = defaultOpenAPIURLPath
	} else if openAPIConfig.URLPath[0:4] != "GET " {
		openAPIConfig.URLPath = "GET " + openAPIConfig.URLPath
	}
}

func configureTemplate(cfg *Config) {
	var assetsFS fs.FS
	var dir string
	var layoutBaseName string
	var htmlTemplateExtension string
	var textTemplateExtension string

	// Set defaults if config is nil
	if cfg == nil || cfg.Assets == nil {
		assetsFS = os.DirFS(".")
		dir = defaultTemplateDir
		layoutBaseName = defaultLayoutBaseName
		htmlTemplateExtension = defaultHTMLTemplateExtension
		textTemplateExtension = defaultTextTemplateExtension
	} else {
		assetsFS = getAssetsFS(cfg)
		dir, layoutBaseName, htmlTemplateExtension, textTemplateExtension = getTemplateConfig(cfg)
	}

	stat, err := fs.Stat(assetsFS, dir)
	if err != nil || !stat.IsDir() {
		return
	}
	templateFS, err := fs.Sub(assetsFS, dir)

	if err != nil {
		return
	}

	tmplConfig := &template.Config{
		FS:                    templateFS,
		LayoutBaseName:        layoutBaseName,
		HTMLTemplateExtension: htmlTemplateExtension,
		TextTemplateExtension: textTemplateExtension,
		I18nFuncName:          defaultI18nFuncName,
	}

	template.Configure(tmplConfig)
}

func configureI18n(cfg *Config) {
	var assetsFS fs.FS
	var dir string
	var supportedLanguages []language.Tag

	// Set defaults if config is nil
	if cfg == nil || cfg.Assets == nil {
		assetsFS = os.DirFS(".")
		dir = defaultI18nMessagesDir
	} else {
		assetsFS = getAssetsFS(cfg)
		dir = getI18nMessagesDir(cfg)
	}

	supportedLanguages = getSupportedLanguages(cfg, assetsFS, dir)

	stat, err := fs.Stat(assetsFS, dir)
	if err != nil || !stat.IsDir() {
		return
	}
	i18nMessagesFS, err := fs.Sub(assetsFS, dir)

	if err != nil {
		return
	}

	i18nConfig := &i18n.Config{
		FS:                 i18nMessagesFS,
		SupportedLanguages: supportedLanguages,
	}

	i18n.Configure(i18nConfig)
}

// Configure initializes the webfram application with the provided configuration.
// It sets up templates, i18n messages, OpenAPI documentation, and JSONP callback handling.
// This function must be called only once before using the framework. Calling it multiple times will panic.
// Pass nil to use default configuration values.
func Configure(cfg *Config) {
	if appConfigured {
		panic("app already configured")
	}
	appConfigured = true

	configureTelemetry(cfg)
	configureOpenAPI(cfg)
	configureTemplate(cfg)
	configureI18n(cfg)

	if cfg != nil {
		if cfg.JSONPCallbackParamName != "" {
			matched := jsonpCallbackNamePattern.MatchString(cfg.JSONPCallbackParamName)
			if !matched {
				panic(fmt.Errorf(
					"invalid JSONP callback param name: %q. "+
						"Must start with a letter or underscore and only contain alphanumeric characters and underscores",
					cfg.JSONPCallbackParamName))
			}
		}
		jsonpCallbackParamName = cfg.JSONPCallbackParamName
	}
}

// Use registers a global middleware that will be applied to all handlers.
// Accepts either AppMiddleware (func(Handler) Handler) or StandardMiddleware (func(http.Handler) http.Handler).
// Middlewares are executed in the order they are registered.
func Use[H AppMiddleware | StandardMiddleware](mw H) {
	if mw == nil {
		return
	}

	switch v := any(mw).(type) {
	case AppMiddleware:
		appMiddlewares = append(appMiddlewares, v)
	case StandardMiddleware:
		adaptedMw := adaptHTTPMiddleware(v)
		appMiddlewares = append(appMiddlewares, adaptedMw)
	}
}

// SSE creates a Server-Sent Events handler that sends real-time updates to clients.
// The payloadFunc is called at the specified interval to generate SSE payloads.
// The disconnectFunc is called when the client disconnects (can be nil for no-op).
// The errorFunc is called when an error occurs during streaming (can be nil for no-op).
// The interval must be positive, and custom headers can be added to each response.
// Panics if payloadFunc is nil or interval is non-positive.
func SSE(
	payloadFunc SSEPayloadFunc,
	disconnectFunc SSEDisconnectFunc,
	errorFunc SSEErrorFunc,
	interval time.Duration,
	headers map[string]string,
) *SSEHandler {
	h := &SSEHandler{
		interval:       interval,
		payloadFunc:    payloadFunc,
		headers:        headers,
		disconnectFunc: disconnectFunc,
		errorFunc:      errorFunc,
	}

	if h.interval <= 0 {
		panic(errors.New("SSE interval must be greater than zero"))
	}
	if h.payloadFunc == nil {
		panic(errors.New("SSE payload function must not be nil"))
	}
	if h.disconnectFunc == nil {
		h.disconnectFunc = func() {}
	}
	if h.errorFunc == nil {
		h.errorFunc = func(_ error) {
			// Default error handler - errors are silently ignored
			// Users should provide a custom errorFunc to handle SSE errors
		}
	}

	return h
}

// Any returns true if there are any validation errors in the collection.
func (errs *ValidationErrors) Any() bool {
	return len(errs.Errors) > 0
}

// BindForm parses form data from the request and binds it to the provided type T.
// It validates the data according to struct tags (validate, errmsg) and returns validation errors if any.
// Returns the bound data, validation errors (nil if valid), and a parsing error (nil if successful).
func BindForm[T any](r *Request) (T, *ValidationErrors, error) {
	val, valErrors, err := bind.Form[T](r.Request)

	vErrors := &ValidationErrors{}
	for _, err := range valErrors {
		vErrors.Errors = append(vErrors.Errors, ValidationError{
			Field: err.Field,
			Error: err.Error,
		})
	}

	return val, vErrors, err
}

// BindJSON parses JSON from the request body and binds it to the provided type T.
// If validate is true, validates the data according to struct tags (validate, errmsg).
// Returns the bound data, validation errors (nil if valid or validation disabled), and a parsing error (nil if successful).
func BindJSON[T any](r *Request, validate bool) (T, *ValidationErrors, error) {
	val, valErrors, err := bind.JSON[T](r.Request, validate)

	vErrors := &ValidationErrors{}
	for _, err := range valErrors {
		vErrors.Errors = append(vErrors.Errors, ValidationError{
			Field: err.Field,
			Error: err.Error,
		})
	}

	return val, vErrors, err
}

// BindXML parses XML from the request body and binds it to the provided type T.
// If validate is true, validates the data according to struct tags (validate, errmsg).
// Returns the bound data, validation errors (nil if valid or validation disabled), and a parsing error (nil if successful).
func BindXML[T any](r *Request, validate bool) (T, *ValidationErrors, error) {
	val, valErrors, err := bind.XML[T](r.Request, validate)

	vErrors := &ValidationErrors{}
	for _, err := range valErrors {
		vErrors.Errors = append(vErrors.Errors, ValidationError{
			Field: err.Field,
			Error: err.Error,
		})
	}

	return val, vErrors, err
}

// BindPath parses URL path parameters from the request and binds them to the provided type T.
// Path parameters are extracted using r.PathValue() method (Go 1.22+).
// It validates the data according to struct tags (validate, errmsg) and returns validation errors if any.
// Struct fields should use the "form" tag to specify parameter names.
// Returns the bound data and validation errors (nil if valid).
func BindPath[T any](r *Request) (T, *ValidationErrors) {
	val, valErrors, _ := bind.Path[T](r.Request)

	vErrors := &ValidationErrors{}
	for _, err := range valErrors {
		vErrors.Errors = append(vErrors.Errors, ValidationError{
			Field: err.Field,
			Error: err.Error,
		})
	}

	return val, vErrors
}

// BindQuery parses query parameters from the request URL and binds them to the provided type T.
// It validates the data according to struct tags (validate, errmsg) and returns validation errors if any.
// Struct fields should use the "form" tag to specify parameter names.
// Supports slices for multi-value query parameters.
// Returns the bound data, validation errors (nil if valid), and a parsing error (nil if successful).
func BindQuery[T any](r *Request) (T, *ValidationErrors, error) {
	val, valErrors, err := bind.Query[T](r.Request)

	vErrors := &ValidationErrors{}
	for _, err := range valErrors {
		vErrors.Errors = append(vErrors.Errors, ValidationError{
			Field: err.Field,
			Error: err.Error,
		})
	}

	return val, vErrors, err
}

// BindCookie parses HTTP cookies from the request and binds them to the provided type T.
// It validates the data according to struct tags (validate, errmsg) and returns validation errors if any.
// Struct fields should use the "form" tag to specify cookie names.
// Returns the bound data, validation errors (nil if valid), and a parsing error (nil if successful).
func BindCookie[T any](r *Request) (T, *ValidationErrors, error) {
	val, valErrors, err := bind.Cookie[T](r.Request)

	vErrors := &ValidationErrors{}
	for _, err := range valErrors {
		vErrors.Errors = append(vErrors.Errors, ValidationError{
			Field: err.Field,
			Error: err.Error,
		})
	}

	return val, vErrors, err
}

// BindHeader parses HTTP headers from the request and binds them to the provided type T.
// It validates the data according to struct tags (validate, errmsg) and returns validation errors if any.
// Struct fields should use the "form" tag to specify header names (case-insensitive).
// Supports slices for multi-value headers.
// Returns the bound data, validation errors (nil if valid), and a parsing error (nil if successful).
func BindHeader[T any](r *Request) (T, *ValidationErrors, error) {
	val, valErrors, err := bind.Header[T](r.Request)

	vErrors := &ValidationErrors{}
	for _, err := range valErrors {
		vErrors.Errors = append(vErrors.Errors, ValidationError{
			Field: err.Field,
			Error: err.Error,
		})
	}

	return val, vErrors, err
}

// PatchJSON applies JSON Patch (RFC 6902) operations to the provided data.
// The request must use PATCH method and have Content-Type application/json-patch+json.
// If validate is true, validates the patched data according to struct tags.
// Returns validation errors (empty if valid or validation disabled) and a parsing/application error (nil if successful).
func PatchJSON[T any](r *Request, t *T, validate bool) ([]ValidationError, error) {
	if r.Method != http.MethodPatch {
		return nil, ErrMethodNotAllowed
	}

	if r.Header.Get("Content-Type") != "application/json-patch+json" {
		return nil, errors.New("invalid Content-Type header, expected application/json-patch+json")
	}

	body, err := io.ReadAll(r.Body)

	if err != nil {
		return nil, err
	}

	patch, err := jsonpatch.DecodePatch(body)

	if err != nil {
		return nil, err
	}

	original, err := json.Marshal(*t)

	if err != nil {
		return nil, err
	}

	modified, err := patch.Apply(original)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(modified, t)

	if err != nil {
		return nil, err
	}

	if validate {
		validationErrors := bind.ValidateJSON(t)

		vErrors := []ValidationError{}
		for _, err := range validationErrors {
			vErrors = append(vErrors, ValidationError{
				Field: err.Field,
				Error: err.Error,
			})
		}
		return vErrors, nil
	}

	return nil, nil
}

// GetI18nPrinter creates a message printer for the specified language tag.
// The printer can be used to format messages according to the configured i18n catalogs.
// Returns a printer that will use the best available language match from configured catalogs.
func GetI18nPrinter(tag language.Tag) *message.Printer {
	return i18n.GetI18nPrinter(tag)
}

func getValueOrDefault[T comparable](value, defaultValue T) T {
	var zero T

	if value == zero {
		return defaultValue
	}
	return value
}

func getAssetsFS(cfg *Config) fs.FS {
	if cfg.Assets.FS == nil {
		return os.DirFS(".")
	}
	return cfg.Assets.FS
}

func getTemplateConfig(cfg *Config) (string, string, string, string) {
	if cfg.Assets.Templates == nil {
		return defaultTemplateDir, defaultLayoutBaseName, defaultHTMLTemplateExtension, defaultTextTemplateExtension
	}
	return getValueOrDefault(cfg.Assets.Templates.Dir, defaultTemplateDir),
		getValueOrDefault(cfg.Assets.Templates.LayoutBaseName, defaultLayoutBaseName),
		getValueOrDefault(cfg.Assets.Templates.HTMLTemplateExtension, defaultHTMLTemplateExtension),
		getValueOrDefault(cfg.Assets.Templates.TextTemplateExtension, defaultTextTemplateExtension)
}

func getI18nMessagesDir(cfg *Config) string {
	if cfg.Assets.I18nMessages == nil {
		return defaultI18nMessagesDir
	}
	return getValueOrDefault(cfg.Assets.I18nMessages.Dir, defaultI18nMessagesDir)
}

func getSupportedLanguages(cfg *Config, assetsFS fs.FS, localesDir string) []language.Tag {
	var langs []string
	// TODO: Consider refactoring to reduce complexity (currently ignored for clarity)
	//nolint:nestif // Nested if-else structure is intentional for auto-detection logic
	if cfg == nil ||
		cfg.Assets == nil ||
		cfg.Assets.I18nMessages == nil ||
		len(cfg.Assets.I18nMessages.SupportedLanguages) == 0 {
		entries, err := fs.ReadDir(assetsFS, localesDir)
		if err != nil {
			return []language.Tag{defaultLanguage}
		}

		for _, entry := range entries {
			name := entry.Name()
			// Skip directories, non-JSON files, hidden files, and empty names
			if entry.IsDir() ||
				!strings.HasSuffix(name, ".json") ||
				len(name) == 0 ||
				name[0] == '.' {
				continue
			}
			baseName := strings.TrimSuffix(name, ".json")
			parts := strings.Split(baseName, ".")
			if len(parts) != 2 || parts[0] != "messages" || parts[1] == "" {
				continue
			}
			// Validate that it's a valid language code before adding
			if _, err = language.Parse(parts[1]); err == nil {
				langs = append(langs, parts[1])
			}
		}
	} else {
		langs = cfg.Assets.I18nMessages.SupportedLanguages
	}

	if len(langs) == 0 {
		return []language.Tag{defaultLanguage}
	}

	var supportedLanguages []language.Tag
	for _, lang := range langs {
		supportedLanguages = append(supportedLanguages, language.MustParse(lang))
	}
	return supportedLanguages
}

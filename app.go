package webfram

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/bondowe/webfram/internal/bind"
	"github.com/bondowe/webfram/internal/i18n"
	"github.com/bondowe/webfram/internal/template"
	"github.com/bondowe/webfram/openapi"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	jsonpatch "github.com/evanphx/json-patch"
)

type (
	contextKey         string
	Middleware[H any]  = func(H) H
	AppMiddleware      = Middleware[Handler]
	StandardMiddleware = Middleware[http.Handler]

	SSEPayload struct {
		Id       string
		Event    string
		Comments []string
		Data     any
		Retry    time.Duration
	}
	SSEPayloadFunc    func() SSEPayload
	SSEDisconnectFunc func()
	SSEErrorFunc      func(error)
	sseHandler        struct {
		interval       time.Duration
		headers        map[string]string
		payloadFunc    SSEPayloadFunc
		disconnectFunc SSEDisconnectFunc
		errorFunc      SSEErrorFunc
	}

	ValidationError struct {
		XMLName xml.Name `json:"-" xml:"validationError" form:"-"`
		Field   string   `json:"field" xml:"field" form:"field"`
		Error   string   `json:"error" xml:"error" form:"error"`
	}

	ValidationErrors struct {
		XMLName xml.Name          `json:"-" xml:"validationErrors" form:"-"`
		Errors  []ValidationError `json:"errors" xml:"errors" form:"errors"`
	}

	Templates struct {
		Dir                   string
		LayoutBaseName        string
		HTMLTemplateExtension string
		TextTemplateExtension string
	}

	I18nMessages struct {
		Dir string
	}

	Assets struct {
		FS           fs.FS
		Templates    *Templates
		I18nMessages *I18nMessages
	}

	OpenAPI struct {
		EndpointEnabled bool
		URLPath         string
		Config          *openapi.Config
	}

	Config struct {
		I18nMessages           *I18nMessages
		JSONPCallbackParamName string
		Assets                 *Assets
		OpenAPI                *OpenAPI
	}
)

const (
	jsonpCallbackMethodNameKey   contextKey = "jsonpCallbackMethodName"
	defaultOpenAPIURLPath        string     = "GET /openapi.json"
	defaultTemplateDir           string     = "templates"
	defaultLayoutBaseName        string     = "layout"
	defaultHTMLTemplateExtension string     = ".go.html"
	defaultTextTemplateExtension string     = ".go.txt"
	defaultI18nMessagesDir       string     = "i18n"
	defaultI18nFuncName          string     = "T"
)

var (
	appConfigured            = false
	appMiddlewares           []AppMiddleware
	openAPIConfig            *OpenAPI
	jsonpCallbackParamName   string
	jsonpCallbackNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	ErrMethodNotAllowed = fmt.Errorf("method not allowed")
)

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

func (m *sseHandler) ServeHTTP(w ResponseWriter, r *Request) {
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

	rc := http.NewResponseController(w.ResponseWriter)
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

			if payload.Id != "" {
				_, err := fmt.Fprintf(w.ResponseWriter, "id: %s\n", payload.Id)
				if err != nil {
					m.errorFunc(err)
					return
				}
				msgWritten = true
			}
			if payload.Event != "" {
				_, err := fmt.Fprintf(w.ResponseWriter, "event: %s\n", payload.Event)
				if err != nil {
					m.errorFunc(err)
					return
				}
				msgWritten = true
			}
			if len(payload.Comments) > 0 {
				for _, comment := range payload.Comments {
					_, err := fmt.Fprintf(w.ResponseWriter, ": %s\n", comment)
					if err != nil {
						m.errorFunc(err)
						return
					}
				}
				msgWritten = true
			}
			if payload.Data != nil {
				_, err := fmt.Fprintf(w.ResponseWriter, "data: %s\n", payload.Data)
				if err != nil {
					m.errorFunc(err)
					return
				}
				msgWritten = true
			}
			if payload.Retry > 0 {
				_, err := fmt.Fprintf(w.ResponseWriter, "retry: %d\n", int(payload.Retry.Milliseconds()))
				if err != nil {
					m.errorFunc(err)
					return
				}
				msgWritten = true
			}

			if msgWritten {
				_, err := fmt.Fprintf(w.ResponseWriter, "\n")
				if err != nil {
					m.errorFunc(err)
					return
				}

				err = rc.Flush()
				if err != nil {
					m.errorFunc(err)
					return
				}
			}
		}
	}
}

func configureOpenAPI(cfg *Config) {
	if cfg == nil || cfg.OpenAPI == nil || !cfg.OpenAPI.EndpointEnabled {
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

	if cfg == nil || cfg.Assets == nil {
		assetsFS = os.DirFS(".")
		dir = defaultTemplateDir
		layoutBaseName = defaultLayoutBaseName
		htmlTemplateExtension = defaultHTMLTemplateExtension
		textTemplateExtension = defaultTextTemplateExtension
	} else {
		if cfg.Assets.FS == nil {
			assetsFS = os.DirFS(".")
		} else {
			assetsFS = cfg.Assets.FS
		}

		if cfg.Assets.Templates == nil {
			dir = defaultTemplateDir
			layoutBaseName = defaultLayoutBaseName
			htmlTemplateExtension = defaultHTMLTemplateExtension
			textTemplateExtension = defaultTextTemplateExtension
		} else {
			dir = getValueOrDefault(cfg.Assets.Templates.Dir, defaultTemplateDir)
			layoutBaseName = getValueOrDefault(cfg.Assets.Templates.LayoutBaseName, defaultLayoutBaseName)
			htmlTemplateExtension = getValueOrDefault(cfg.Assets.Templates.HTMLTemplateExtension, defaultHTMLTemplateExtension)
			textTemplateExtension = getValueOrDefault(cfg.Assets.Templates.TextTemplateExtension, defaultTextTemplateExtension)
		}
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

	if cfg == nil || cfg.Assets == nil {
		assetsFS = os.DirFS(".")
		dir = defaultI18nMessagesDir
	} else {
		if cfg.Assets.FS == nil {
			assetsFS = os.DirFS(".")
		} else {
			assetsFS = cfg.Assets.FS
		}

		if cfg.Assets.I18nMessages == nil {
			dir = defaultI18nMessagesDir
		} else {
			dir = getValueOrDefault(cfg.Assets.I18nMessages.Dir, defaultI18nMessagesDir)
		}
	}

	stat, err := fs.Stat(assetsFS, dir)
	if err != nil || !stat.IsDir() {
		return
	}
	i18nMessagesFS, err := fs.Sub(assetsFS, dir)

	if err != nil {
		return
	}

	i18nConfig := &i18n.Config{
		FS: i18nMessagesFS,
	}

	i18n.Configure(i18nConfig)
}

func Configure(cfg *Config) {
	if appConfigured {
		panic("app already configured")
	}
	appConfigured = true

	configureOpenAPI(cfg)
	configureTemplate(cfg)
	configureI18n(cfg)

	if cfg != nil {
		if cfg.JSONPCallbackParamName != "" {
			matched := jsonpCallbackNamePattern.MatchString(cfg.JSONPCallbackParamName)
			if !matched {
				panic(fmt.Errorf("invalid JSONP callback param name: %q. Must start with a letter or underscore and only contain alphanumeric characters and underscores", cfg.JSONPCallbackParamName))
			}
		}
		jsonpCallbackParamName = cfg.JSONPCallbackParamName
	}
}

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

func SSE(payloadFunc SSEPayloadFunc, disconnectFunc SSEDisconnectFunc, errorFunc SSEErrorFunc, interval time.Duration, headers map[string]string) *sseHandler {
	h := &sseHandler{
		interval:       interval,
		payloadFunc:    payloadFunc,
		headers:        headers,
		disconnectFunc: disconnectFunc,
		errorFunc:      errorFunc,
	}

	if h.interval <= 0 {
		panic(fmt.Errorf("SSE interval must be greater than zero"))
	}
	if h.payloadFunc == nil {
		panic(fmt.Errorf("SSE payload function must not be nil"))
	}
	if h.disconnectFunc == nil {
		h.disconnectFunc = func() {}
	}
	if h.errorFunc == nil {
		h.errorFunc = func(err error) {
			fmt.Printf("SSE error: %v\n", err)
		}
	}

	return h
}

func (errs *ValidationErrors) Any() bool {
	return len(errs.Errors) > 0
}

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

func PatchJSON[T any](r *Request, t *T, validate bool) ([]ValidationError, error) {
	if r.Method != http.MethodPatch {
		return nil, ErrMethodNotAllowed
	}

	if r.Header.Get("Content-Type") != "application/json-patch+json" {
		return nil, fmt.Errorf("invalid Content-Type header, expected application/json-patch+json")
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

func GetI18nPrinter(tag language.Tag) *message.Printer {
	return i18n.GetI18nPrinter(tag)
}

func getValueOrDefault[T comparable](value T, defaultValue T) T {
	var zero T

	if value == zero {
		fmt.Printf("Using default value: %v\n", defaultValue)
		return defaultValue
	}
	fmt.Printf("Using provided value: %v\n", value)
	return value
}

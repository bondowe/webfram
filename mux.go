package webfram

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"slices"
	"strings"

	"github.com/bondowe/webfram/internal/bind"
	"github.com/bondowe/webfram/internal/i18n"
	"github.com/bondowe/webfram/openapi"
	"golang.org/x/text/language"
)

type (
	Request struct {
		*http.Request
	}
	ServeMux struct {
		middlewares []AppMiddleware
		http.ServeMux
	}
	Handler interface {
		ServeHTTP(ResponseWriter, *Request)
	}
	HandlerFunc func(ResponseWriter, *Request)
	APIConfig   struct {
		Method      string
		Summary     string
		Description string
		OperationID string
		Tags        []string
		Parameters  []Parameter
		RequestBody *RequestBody
		Responses   map[string]Response
		Servers     []Server
	}
	PathInfo struct {
		Summary     string
		Description string
		// AdditionalOperations map[string]*Operation
		Servers    []Server
		Parameters []Parameter
	}
	Parameter struct {
		Example          any
		Default          any
		Const            any
		TypeHint         any
		Explode          *bool
		Examples         map[string]Example
		Content          map[string]any
		Style            string
		Format           string
		Name             string
		Description      string
		In               string
		Pattern          string
		Enum             []any
		Minimum          float64
		MinItems         int
		MinLength        int
		MaxLength        int
		ExclusiveMaximum float64
		ExclusiveMinimum float64
		Maximum          float64
		MaxItems         int
		MultipleOf       float64
		AllowReserved    bool
		Nullable         bool
		UniqueItems      bool
		Deprecated       bool
		Required         bool
	}
	TypeInfo struct {
		TypeHint any
		Examples map[string]Example
	}
	Example struct {
		DataValue       any
		DefaultValue    any
		SerializedValue any
		Server          *Server
		Summary         string
		Description     string
		ExternalValue   string
	}
	Server struct {
		Variables   map[string]ServerVariable
		URL         string
		Name        string
		Description string
	}
	ServerVariable struct {
		Default     string
		Description string
		Enum        []string
	}
	RequestBody struct {
		Content     map[string]TypeInfo
		Example     *Example
		Examples    map[string]Example
		Description string
		Required    bool
	}
	Response struct {
		Headers     map[string]Header
		Content     map[string]TypeInfo
		Links       map[string]Link
		Summary     string
		Description string
	}
	Header struct {
		Example     any
		TypeHint    any
		Examples    map[string]Example
		Explode     *bool
		Content     map[string]TypeInfo
		Description string
		Style       string
		Required    bool
		Deprecated  bool
	}
	Link struct {
		OperationRef string
		OperationID  string
		Parameters   map[string]any
		RequestBody  any
		Description  string
	}
	handlerConfig struct {
		APIConfig   *APIConfig
		pathPattern string
	}
)

// SetOpenAPIPathInfo adds or updates path-level information in the OpenAPI documentation.
// This should be called before registering handlers to set common parameters and servers for a path.
// Only works if OpenAPI endpoint is enabled in configuration.
func SetOpenAPIPathInfo(path string, info *PathInfo) {
	if openAPIConfig == nil || !openAPIConfig.EndpointEnabled {
		return
	}

	if !appConfigured {
		Configure(nil)
	}

	parameters := mapParameters(info.Parameters)
	servers := mapServers(info.Servers)

	openAPIConfig.Config.Paths.SetPathInfo(path, info.Summary, info.Description, parameters, servers)
}

// WithAPIConfig attaches OpenAPI configuration to a handler.
// This generates OpenAPI documentation for the endpoint with request/response schemas, parameters, etc.
// Only works if OpenAPI endpoint is enabled in configuration.
func (c *handlerConfig) WithAPIConfig(apiConfig *APIConfig) {
	if apiConfig == nil || openAPIConfig == nil || !openAPIConfig.EndpointEnabled {
		return
	}

	c.APIConfig = apiConfig

	var requestBody *openapi.RequestBodyOrRef

	if c.APIConfig.RequestBody != nil {
		requestBody = &openapi.RequestBodyOrRef{
			RequestBody: &openapi.RequestBody{
				Description: c.APIConfig.RequestBody.Description,
				Required:    c.APIConfig.RequestBody.Required,
				Content:     mapContent(c.APIConfig.RequestBody.Content),
			},
		}
	}

	var responses map[string]openapi.ResponseOrRef

	if len(c.APIConfig.Responses) > 0 {
		responses = make(map[string]openapi.ResponseOrRef, len(c.APIConfig.Responses))
		for statusCode, resp := range c.APIConfig.Responses {
			responses[statusCode] = openapi.ResponseOrRef{
				Response: &openapi.Response{
					Summary:     resp.Summary,
					Description: resp.Description,
					Headers:     mapHeaders(resp.Headers),
					Content:     mapContent(resp.Content),
					Links:       mapLinks(resp.Links),
				},
			}
		}
	}

	parameters := mapParameters(c.APIConfig.Parameters)

	parts := strings.Fields(c.pathPattern)

	if len(parts) != 2 {
		panic(fmt.Errorf("invalid path pattern: %q. Must be in format 'METHOD /path'", c.pathPattern))
	}

	method := strings.ToLower(parts[0])
	path := parts[1]

	openAPIConfig.Config.Paths.AddOperation(path, method, openapi.Operation{
		Summary:     c.APIConfig.Summary,
		Description: c.APIConfig.Description,
		OperationID: c.APIConfig.OperationID,
		Tags:        c.APIConfig.Tags,
		RequestBody: requestBody,
		Parameters:  parameters,
		Servers:     mapServers(c.APIConfig.Servers),
		Responses:   responses,
	})
}

func mapLinks(links map[string]Link) map[string]openapi.LinkOrRef {
	if links == nil {
		return nil
	}

	output := make(map[string]openapi.LinkOrRef, len(links))
	for k, v := range links {
		output[k] = openapi.LinkOrRef{
			Link: &openapi.Link{
				OperationRef: v.OperationRef,
				OperationId:  v.OperationID,
				Parameters:   v.Parameters,
				RequestBody:  v.RequestBody,
				Description:  v.Description,
			},
		}
	}
	return output
}

func mapContent(typeInfos map[string]TypeInfo) map[string]openapi.MediaType {
	if typeInfos == nil {
		return nil
	}

	content := make(map[string]openapi.MediaType)
	for mediaType, info := range typeInfos {
		for _, mt := range strings.Split(mediaType, ",") {
			schemaOrRef := bind.GenerateJSONSchema(info.TypeHint, openAPIConfig.Config.Components)

			mediaType := openapi.MediaType{
				Schema:   schemaOrRef,
				Examples: mapExampleOrRefs(info.Examples),
			}

			content[mt] = mediaType
		}
	}
	return content
}

func mapHeaders(header map[string]Header) map[string]openapi.HeaderOrRef {
	if header == nil {
		return nil
	}

	output := make(map[string]openapi.HeaderOrRef, len(header))

	for k, v := range header {
		var schemaOrRef *openapi.SchemaOrRef
		var content map[string]openapi.MediaTypeOrRef

		if v.Content != nil {
			content = make(map[string]openapi.MediaTypeOrRef)
			for mediaType, model := range v.Content {
				for _, mt := range strings.Split(mediaType, ",") {
					schema := bind.GenerateJSONSchema(model, openAPIConfig.Config.Components)
					content[mt] = openapi.MediaTypeOrRef{
						MediaType: &openapi.MediaType{
							Schema: schema,
						},
					}
				}
			}
		} else {
			if v.TypeHint == nil {
				v.TypeHint = ""
			}
			schemaOrRef = bind.GenerateJSONSchema(v.TypeHint, openAPIConfig.Config.Components)

			if schemaOrRef.Ref == "" && schemaOrRef.Schema != nil {
				schema := schemaOrRef.Schema
				schema.Example = v.Example
				schema.Examples = mapExamples(v.Examples)
			}
		}

		output[k] = openapi.HeaderOrRef{
			Header: &openapi.Header{
				Description: v.Description,
				Required:    v.Required,
				Deprecated:  v.Deprecated,
				Example:     v.Example,
				Examples:    mapExampleOrRefs(v.Examples),
				Style:       v.Style,
				Explode:     v.Explode,
				Schema:      schemaOrRef,
				Content:     content,
			},
		}
	}

	return output
}

func mapParameters(params []Parameter) []openapi.ParameterOrRef {
	var parameters []openapi.ParameterOrRef
	var schemaOrRef *openapi.SchemaOrRef
	var content map[string]openapi.MediaType
	for i := range params {
		param := &params[i]
		if param.Content != nil {
			content = make(map[string]openapi.MediaType)
			for mediaType, model := range param.Content {
				for _, mt := range strings.Split(mediaType, ",") {
					schema := bind.GenerateJSONSchema(model, openAPIConfig.Config.Components)
					content[mt] = openapi.MediaType{
						Schema: schema,
					}
				}
			}
		} else {
			if param.TypeHint == nil {
				param.TypeHint = ""
			}
			schemaOrRef = bind.GenerateJSONSchema(param.TypeHint, openAPIConfig.Config.Components)

			if schemaOrRef.Ref == "" && schemaOrRef.Schema != nil {
				schema := schemaOrRef.Schema
				schema.Const = param.Const
				schema.Default = param.Default
				schema.Nullable = param.Nullable
				schema.Example = param.Example
				schema.Examples = mapExamples(param.Examples)

				if schema.Type == "string" || schema.Type == "integer" || schema.Type == "number" {
					schema.Enum = param.Enum
					schema.Format = param.Format
				}
				if schema.Type == "string" {
					schema.MaxLength = nonZeroValuePointer(param.MaxLength)
					schema.MinLength = nonZeroValuePointer(param.MinLength)
					schema.Pattern = param.Pattern
				}
				if schema.Type == "integer" || schema.Type == "number" {
					schema.ExclusiveMaximum = nonZeroValuePointer(param.ExclusiveMaximum)
					schema.ExclusiveMinimum = nonZeroValuePointer(param.ExclusiveMinimum)
					schema.Maximum = nonZeroValuePointer(param.Maximum)
					schema.Minimum = nonZeroValuePointer(param.Minimum)
					schema.MultipleOf = nonZeroValuePointer(param.MultipleOf)
				}
				if schema.Type == "array" {
					schema.MaxItems = nonZeroValuePointer(param.MaxItems)
					schema.MinItems = nonZeroValuePointer(param.MinItems)
					schema.UniqueItems = param.UniqueItems
				}
			}
		}

		parameters = append(parameters, openapi.ParameterOrRef{
			Parameter: &openapi.Parameter{
				Name:          param.Name,
				In:            param.In,
				Description:   param.Description,
				Required:      param.Required,
				Deprecated:    param.Deprecated,
				AllowReserved: param.AllowReserved,
				Schema:        schemaOrRef,
				Content:       content,
				Style:         param.Style,
				Explode:       param.Explode,
			},
		})
	}

	return parameters
}

func mapExample(input *Example) *openapi.Example {
	if input == nil {
		return nil
	}

	return &openapi.Example{
		Summary:         input.Summary,
		Description:     input.Description,
		DataValue:       input.DataValue,
		DefaultValue:    input.DefaultValue,
		SerializedValue: input.SerializedValue,
		ExternalValue:   input.ExternalValue,
	}
}

func mapExamples(input map[string]Example) map[string]openapi.Example {
	if input == nil {
		return nil
	}

	output := make(map[string]openapi.Example, len(input))
	for k, v := range input {
		output[k] = *mapExample(&v)
	}
	return output
}

func mapExampleOrRefs(input map[string]Example) map[string]openapi.ExampleOrRef {
	if input == nil {
		return nil
	}

	output := make(map[string]openapi.ExampleOrRef, len(input))
	for k, v := range input {
		output[k] = openapi.ExampleOrRef{
			Example: &openapi.Example{
				Summary:         v.Summary,
				Description:     v.Description,
				DefaultValue:    v.DefaultValue,
				SerializedValue: v.SerializedValue,
				ExternalValue:   v.ExternalValue,
				DataValue:       v.DataValue,
			},
		}
	}
	return output
}

func mapServer(input *Server) *openapi.Server {
	if input == nil {
		return nil
	}

	return &openapi.Server{
		URL:         input.URL,
		Name:        input.Name,
		Description: input.Description,
		Variables:   mapServerVariables(input.Variables),
	}
}

func mapServers(input []Server) []openapi.Server {
	if input == nil {
		return nil
	}
	output := make([]openapi.Server, len(input))
	for i, v := range input {
		output[i] = *mapServer(&v)
	}
	return output
}

func mapServerVariables(input map[string]ServerVariable) map[string]openapi.ServerVariable {
	if input == nil {
		return nil
	}

	output := make(map[string]openapi.ServerVariable, len(input))
	for k, v := range input {
		output[k] = openapi.ServerVariable{
			Enum:        v.Enum,
			Default:     v.Default,
			Description: v.Description,
		}
	}
	return output
}

func nonZeroValuePointer[T comparable](value T) *T {
	var zero T
	if value != zero {
		return &value
	}
	return nil
}

func wrapMiddlewares(handler Handler, middlewares []AppMiddleware) Handler {
	mdwrs := slices.Clone(middlewares)
	wrappedHandler := handler

	for i := len(mdwrs) - 1; i >= 0; i-- {
		wrappedHandler = mdwrs[i](wrappedHandler)
	}

	return wrappedHandler
}

func getHandlerMiddlewares(middlewares []interface{}) []AppMiddleware {
	var mdwrs []AppMiddleware
	for _, mw := range middlewares {
		switch v := mw.(type) {
		case AppMiddleware:
			mdwrs = append(mdwrs, v)
		case StandardMiddleware:
			adaptedMw := adaptHTTPMiddleware(v)
			mdwrs = append(mdwrs, adaptedMw)
		default:
			panic("unsupported middleware type")
		}
	}

	return mdwrs
}

// I18nMiddleware creates middleware that adds internationalization support to handlers.
// It parses the Accept-Language header and language cookie to determine the user's preferred language,
// then injects an i18n printer into the request context for message translation.
func I18nMiddleware(fsys fs.FS) func(Handler) Handler {
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			var langTag language.Tag
			// Try to get language from cookie first
			cookie, err := r.Cookie("lang")
			if err == nil && cookie.Value != "" {
				if tag, err := language.Parse(cookie.Value); err == nil {
					langTag = tag
				}
			}

			// If no valid language from cookie, try Accept-Language header
			if langTag == language.Und {
				acceptLang := r.Header.Get("Accept-Language")
				if acceptLang != "" {
					langTag = parseAcceptLanguage(acceptLang)
				}
			}

			// Default to English if no language could be determined
			if langTag == language.Und {
				langTag = language.English
			}

			msgPrinter := i18n.GetI18nPrinter(langTag)
			ctx := i18n.ContextWithI18nPrinter(context.Background(), msgPrinter)

			req := Request{r.WithContext(ctx)}

			next.ServeHTTP(w, &req)
		})
	}
}

func parseAcceptLanguage(acceptLang string) language.Tag {
	// Parse Accept-Language header (e.g., "en-US,en;q=0.9,fr;q=0.8")
	tags, _, err := language.ParseAcceptLanguage(acceptLang)
	if err != nil || len(tags) == 0 {
		return language.Und
	}

	// Define supported languages (you can customize this list)
	supportedLanguages := []language.Tag{
		language.English,
		language.French,
		language.CanadianFrench,
		language.Swahili,
		language.Russian,
		language.German,
		language.Spanish,
		language.Italian,
		language.Portuguese,
		language.Japanese,
		language.Chinese,
		language.Korean,
	}

	// Create a matcher for supported languages
	matcher := language.NewMatcher(supportedLanguages)

	// Find the best match
	tag, _, _ := matcher.Match(tags...)
	return tag
}

// SetLanguageCookie sets a language preference cookie for the user.
// The maxAge parameter controls cookie lifetime in seconds (0 = delete cookie, -1 = session cookie).
func SetLanguageCookie(w ResponseWriter, lang string, maxAge int) {
	cookie := &http.Cookie{
		Name:     "lang",
		Value:    lang,
		Path:     "/",
		MaxAge:   maxAge, // seconds (e.g., 86400 for 24 hours, 0 to delete)
		HttpOnly: false,  // Allow JavaScript access for language switchers
		Secure:   false,  // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w.ResponseWriter, cookie)
}

// NewServeMux creates a new HTTP request multiplexer with webfram enhancements.
// Automatically calls Configure(nil) if the application hasn't been configured yet.
// Returns a ServeMux that supports middleware, custom handlers, and OpenAPI documentation.
func NewServeMux() *ServeMux {
	if !appConfigured {
		Configure(nil)
	}

	return &ServeMux{
		middlewares: nil,
		ServeMux:    http.ServeMux{},
	}
}

// Use registers middleware to be applied to all handlers registered on this ServeMux.
// Accepts either AppMiddleware (func(Handler) Handler) or StandardMiddleware (func(http.Handler) http.Handler).
// Panics if an unsupported middleware type is provided.
func (m *ServeMux) Use(mw interface{}) {
	if mw == nil {
		return
	}

	switch v := mw.(type) {
	case AppMiddleware:
		m.middlewares = append(m.middlewares, v)
	case StandardMiddleware:
		adaptedMw := adaptHTTPMiddleware(v)
		m.middlewares = append(m.middlewares, adaptedMw)
	default:
		panic(fmt.Errorf("unsupported middleware type"))
	}
}

// Handle registers a handler for the given pattern.
// The pattern can include HTTP method prefix (e.g., "GET /users").
// Optional per-handler middlewares can be provided and will be applied only to this handler.
// Returns a handlerConfig that can be used to attach OpenAPI documentation via WithAPIConfig.
func (m *ServeMux) Handle(pattern string, handler Handler, mdwrs ...interface{}) *handlerConfig {
	wrappedHandler := wrapMiddlewares(handler, getHandlerMiddlewares(mdwrs))
	wrappedHandler = wrapMiddlewares(wrappedHandler, m.middlewares)
	wrappedHandler = wrapMiddlewares(wrappedHandler, appMiddlewares)

	if i18nConfig, ok := i18n.Configuration(); ok && i18nConfig.FS != nil {
		i18nMdwr := I18nMiddleware(i18nConfig.FS)
		wrappedHandler = i18nMdwr(wrappedHandler)
	}

	m.ServeMux.Handle(pattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrappedHandler.ServeHTTP(ResponseWriter{w, nil}, &Request{r})
	}))

	return &handlerConfig{
		pathPattern: pattern,
	}
}

// HandleFunc registers a handler function for the given pattern.
// Convenience method that wraps a HandlerFunc and calls Handle.
// Returns a handlerConfig that can be used to attach OpenAPI documentation via WithAPIConfig.
func (m *ServeMux) HandleFunc(pattern string, handler HandlerFunc, mdwrs ...interface{}) *handlerConfig {
	wrappedHandler := wrapMiddlewares(handler, getHandlerMiddlewares(mdwrs))
	wrappedHandler = wrapMiddlewares(wrappedHandler, m.middlewares)
	wrappedHandler = wrapMiddlewares(wrappedHandler, appMiddlewares)

	if i18nConfig, ok := i18n.Configuration(); ok && i18nConfig.FS != nil {
		i18nMdwr := I18nMiddleware(i18nConfig.FS)
		wrappedHandler = i18nMdwr(wrappedHandler)
	}

	m.ServeMux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		wrappedHandler.ServeHTTP(ResponseWriter{w, nil}, &Request{r})
	})

	return &handlerConfig{
		pathPattern: pattern,
	}
}

// ServeHTTP implements the http.Handler interface.
// It wraps the request, applies middlewares, and handles JSONP callbacks if configured.
func (m *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.ServeMux.ServeHTTP(w, r)
}

// ServeHTTP implements the Handler interface, allowing HandlerFunc to be used as a Handler.
func (hf HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	ctx := context.Background()

	if i18nPrinter, ok := i18n.I18nPrinterFromContext(r.Context()); ok {
		ctx = i18n.ContextWithI18nPrinter(ctx, i18nPrinter)
	}

	if jsonpCallbackMethodName := r.URL.Query().Get(jsonpCallbackParamName); jsonpCallbackMethodName != "" {
		matched := jsonpCallbackNamePattern.MatchString(jsonpCallbackMethodName)
		if !matched {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(fmt.Errorf(
				"invalid JSONP callback method name: %q. "+
					"Must start with a letter or underscore and only contain alphanumeric characters and underscores",
				jsonpCallbackMethodName).Error()))
			return
		}
		ctx = context.WithValue(ctx, jsonpCallbackMethodNameKey, jsonpCallbackMethodName)
	}

	w.context = ctx

	hf(w, r)
}

package webfram

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"slices"
	"strings"

	"github.com/bondowe/webfram/internal/bind"
	"github.com/bondowe/webfram/internal/i18n"
	"github.com/bondowe/webfram/internal/telemetry"
	"github.com/bondowe/webfram/openapi"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/text/language"
)

const (
	mediaTypeTextEventStream = "text/event-stream"
	mediaTypeJSONSeq         = "application/json-seq"
)

var (
	mediaTypesXML = []string{"application/xml", "text/xml"} //nolint:gochecknoglobals
)

type (
	// Request wraps http.Request with additional framework functionality.
	Request struct {
		*http.Request
	}
	// ServeMux is an HTTP request multiplexer with middleware support.
	ServeMux struct {
		http.ServeMux

		middlewares []AppMiddleware
	}
	// Handler responds to HTTP requests.
	Handler interface {
		ServeHTTP(ResponseWriter, *Request)
	}
	// HandlerFunc is a function that serves HTTP requests.
	HandlerFunc func(ResponseWriter, *Request)

	// OperationConfig configures OpenAPI documentation for a route.
	OperationConfig struct {
		Method      string
		Summary     string
		Description string
		OperationID string
		Tags        []string
		Parameters  []Parameter
		Security    []map[string][]string
		RequestBody *RequestBody
		Responses   map[string]Response
		Servers     []Server
	}
	// PathInfo contains path-level OpenAPI documentation.
	PathInfo struct {
		Summary     string
		Description string
		// AdditionalOperations map[string]*Operation
		Servers    []Server
		Parameters []Parameter
	}
	// Parameter describes an operation parameter in OpenAPI.
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
	// TypeInfo provides type information for OpenAPI content types.
	TypeInfo struct {
		// TypeHint provides a hint about the data type.
		TypeHint any
		// XMLRootName specifies the root element name for XML serialization.
		// Only applicable when using XML content type.
		XMLRootName string
		Example     any
		Examples    map[string]Example
	}
	// Example represents an OpenAPI example value.
	Example struct {
		DataValue       any
		DefaultValue    any
		SerializedValue any
		Server          *Server
		Summary         string
		Description     string
		ExternalValue   string
	}
	// Server describes an OpenAPI server.
	Server struct {
		Variables   map[string]ServerVariable
		URL         string
		Name        string
		Description string
	}
	// ServerVariable represents a variable in a server URL template.
	ServerVariable struct {
		Default     string
		Description string
		Enum        []string
	}
	// RequestBody describes an OpenAPI request body.
	RequestBody struct {
		Content     map[string]TypeInfo
		Example     *Example
		Examples    map[string]Example
		Description string
		Required    bool
	}
	// Response describes an OpenAPI response.
	Response struct {
		Headers     map[string]Header
		Content     map[string]TypeInfo
		Links       map[string]Link
		Summary     string
		Description string
	}
	// Header describes an OpenAPI response header.
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
	// Link represents an OpenAPI link.
	Link struct {
		OperationRef string
		OperationID  string
		Parameters   map[string]any
		RequestBody  any
		Description  string
	}
	// HandlerConfig provides configuration for registered handlers, particularly for OpenAPI documentation.
	HandlerConfig struct {
		OperationConfig *OperationConfig
		pathPattern     string
	}
)

// SetOpenAPIPathInfo adds or updates path-level information in the OpenAPI documentation.
// This should be called before registering handlers to set common parameters and servers for a path.
// Only works if OpenAPI endpoint is enabled in configuration.
func SetOpenAPIPathInfo(path string, info *PathInfo) {
	if openAPIConfig == nil || !openAPIConfig.Enabled {
		return
	}

	if !appConfigured {
		Configure(nil)
	}

	parameters := mapParameters(info.Parameters)
	servers := mapServers(info.Servers)

	openAPIConfig.internalConfig.Paths.SetPathInfo(path, info.Summary, info.Description, parameters, servers)
}

// WithOperationConfig attaches OpenAPI configuration to a handler.
// This generates OpenAPI documentation for the endpoint with request/response schemas, parameters, etc.
// Only works if OpenAPI endpoint is enabled in configuration.
func (c *HandlerConfig) WithOperationConfig(operationConfig *OperationConfig) {
	if operationConfig == nil || openAPIConfig == nil || !openAPIConfig.Enabled {
		return
	}

	c.OperationConfig = operationConfig

	var requestBody *openapi.RequestBodyOrRef

	if c.OperationConfig.RequestBody != nil {
		requestBody = &openapi.RequestBodyOrRef{
			RequestBody: &openapi.RequestBody{
				Description: c.OperationConfig.RequestBody.Description,
				Required:    c.OperationConfig.RequestBody.Required,
				Content:     mapContent(c.OperationConfig.RequestBody.Content),
			},
		}
	}

	var responses map[string]openapi.ResponseOrRef

	if len(c.OperationConfig.Responses) > 0 {
		responses = make(map[string]openapi.ResponseOrRef, len(c.OperationConfig.Responses))
		for statusCode, resp := range c.OperationConfig.Responses {
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

	parameters := mapParameters(c.OperationConfig.Parameters)

	parts := strings.Fields(c.pathPattern)

	if len(parts) != 2 { //nolint:mnd // expect METHOD and path
		panic(fmt.Errorf("invalid path pattern: %q. Must be in format 'METHOD /path'", c.pathPattern))
	}

	method := strings.ToLower(parts[0])
	path := parts[1]

	openAPIConfig.internalConfig.Paths.AddOperation(path, method, openapi.Operation{
		Summary:     c.OperationConfig.Summary,
		Description: c.OperationConfig.Description,
		OperationID: c.OperationConfig.OperationID,
		Tags:        c.OperationConfig.Tags,
		Security:    c.OperationConfig.Security,
		RequestBody: requestBody,
		Parameters:  parameters,
		Servers:     mapServers(c.OperationConfig.Servers),
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
			if mt == mediaTypeTextEventStream {
				info.TypeHint = &SSEPayload{}
			}

			var schemaOrRef *openapi.SchemaOrRef

			if slices.Contains(mediaTypesXML, mt) {
				schemaOrRef = bind.GenerateXMLSchema(
					info.TypeHint,
					info.XMLRootName,
					openAPIConfig.internalConfig.Components,
				)
			} else {
				schemaOrRef = bind.GenerateJSONSchema(info.TypeHint, openAPIConfig.internalConfig.Components)
			}

			mediaType := openapi.MediaType{
				Example:  info.Example,
				Examples: mapExampleOrRefs(info.Examples),
			}

			if mt == mediaTypeJSONSeq || mt == mediaTypeTextEventStream {
				mediaType.ItemSchema = schemaOrRef
			} else {
				mediaType.Schema = schemaOrRef
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
					schema := bind.GenerateJSONSchema(model, openAPIConfig.internalConfig.Components)
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
			schemaOrRef = bind.GenerateJSONSchema(v.TypeHint, openAPIConfig.internalConfig.Components)

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
	for i := range params {
		param := &params[i]
		schemaOrRef, content := processParameterSchema(param)
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

func processParameterSchema(param *Parameter) (*openapi.SchemaOrRef, map[string]openapi.MediaType) {
	if param.Content != nil {
		return nil, buildParameterContent(param.Content)
	}
	return buildParameterSchema(param), nil
}

func buildParameterContent(content map[string]any) map[string]openapi.MediaType {
	result := make(map[string]openapi.MediaType)
	for mediaType, model := range content {
		for _, mt := range strings.Split(mediaType, ",") {
			schema := bind.GenerateJSONSchema(model, openAPIConfig.internalConfig.Components)
			result[mt] = openapi.MediaType{
				Schema: schema,
			}
		}
	}
	return result
}

func buildParameterSchema(param *Parameter) *openapi.SchemaOrRef {
	if param.TypeHint == nil {
		param.TypeHint = ""
	}
	schemaOrRef := bind.GenerateJSONSchema(param.TypeHint, openAPIConfig.internalConfig.Components)

	if schemaOrRef.Ref == "" && schemaOrRef.Schema != nil {
		applySchemaConstraints(schemaOrRef.Schema, param)
	}
	return schemaOrRef
}

func applySchemaConstraints(schema *openapi.Schema, param *Parameter) {
	schema.Const = param.Const
	schema.Default = param.Default
	schema.Nullable = param.Nullable
	schema.Example = param.Example
	schema.Examples = mapExamples(param.Examples)

	switch schema.Type {
	case "string":
		applyStringConstraints(schema, param)
	case "integer", "number":
		applyNumericConstraints(schema, param)
	case "array":
		applyArrayConstraints(schema, param)
	}

	if schema.Type == "string" || schema.Type == "integer" || schema.Type == "number" {
		schema.Enum = param.Enum
		schema.Format = param.Format
	}
}

func applyStringConstraints(schema *openapi.Schema, param *Parameter) {
	schema.MaxLength = nonZeroValuePointer(param.MaxLength)
	schema.MinLength = nonZeroValuePointer(param.MinLength)
	schema.Pattern = param.Pattern
}

func applyNumericConstraints(schema *openapi.Schema, param *Parameter) {
	schema.ExclusiveMaximum = nonZeroValuePointer(param.ExclusiveMaximum)
	schema.ExclusiveMinimum = nonZeroValuePointer(param.ExclusiveMinimum)
	schema.Maximum = nonZeroValuePointer(param.Maximum)
	schema.Minimum = nonZeroValuePointer(param.Minimum)
	schema.MultipleOf = nonZeroValuePointer(param.MultipleOf)
}

func applyArrayConstraints(schema *openapi.Schema, param *Parameter) {
	schema.MaxItems = nonZeroValuePointer(param.MaxItems)
	schema.MinItems = nonZeroValuePointer(param.MinItems)
	schema.UniqueItems = param.UniqueItems
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

// / TelemetryMiddleware creates middleware that collects HTTP request metrics using Prometheus.
// / It tracks total requests, request duration, and active connections per endpoint.
// / It uses the telemetry package's predefined Prometheus metrics.
func telemetryMiddleware(next Handler) Handler {
	return HandlerFunc(func(w ResponseWriter, r *Request) {
		path := r.URL.Path
		method := r.Method

		// Track active connections
		telemetry.ActiveConnections.Inc()
		defer telemetry.ActiveConnections.Dec()

		// Start timer and defer recording metrics
		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
			// Get status code from ResponseWriter's context
			statusCode, ok := w.StatusCode()
			if !ok {
				statusCode = http.StatusOK // Default to 200 if not set
			}
			//nolint:mnd // divide by 100 to get status class
			statusClass := fmt.Sprintf("%dxx", statusCode/100)
			telemetry.RequestDurationSeconds.WithLabelValues(method, path, statusClass).Observe(v)
		}))
		defer timer.ObserveDuration()

		next.ServeHTTP(w, r)

		// Record total requests
		statusCode, ok := w.StatusCode()
		if !ok {
			statusCode = http.StatusOK // Default to 200 if not set
		}
		//nolint:mnd // divide by 100 to get status class
		statusClass := fmt.Sprintf("%dxx", statusCode/100)
		telemetry.RequestsTotal.WithLabelValues(method, path, statusClass).Inc()
	})
}

// I18nMiddleware creates middleware that adds internationalization support to handlers.
// It parses the Accept-Language header and language cookie to determine the user's preferred language,
// then injects an i18n printer into the request context for message translation.
func I18nMiddleware(_ fs.FS) func(Handler) Handler {
	return func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			var langTag language.Tag
			// Try to get language from cookie first
			cookie, err := r.Cookie("lang")
			if err == nil && cookie.Value != "" {
				if tag, parseErr := language.Parse(cookie.Value); parseErr == nil {
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

			// Default to first supported language if no language could be determined
			if langTag == language.Und {
				if i18nConfig, ok := i18n.Configuration(); ok && len(i18nConfig.SupportedLanguages) > 0 {
					langTag = i18nConfig.SupportedLanguages[0]
				} else {
					langTag = language.English
				}
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

	i18nConfig, ok := i18n.Configuration()

	if !ok {
		return language.Und
	}

	supportedLanguages := i18nConfig.SupportedLanguages

	if len(supportedLanguages) == 0 {
		return language.Und
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
		panic(errors.New("unsupported middleware type"))
	}
}

// Handle registers a handler for the given pattern.
// The pattern can include HTTP method prefix (e.g., "GET /users").
// Optional per-handler middlewares can be provided and will be applied only to this handler.
// Returns a handlerConfig that can be used to attach OpenAPI documentation via WithAPIConfig.
func (m *ServeMux) Handle(pattern string, handler Handler, mdwrs ...interface{}) *HandlerConfig {
	wrappedHandler := wrapMiddlewares(handler, getHandlerMiddlewares(mdwrs))
	wrappedHandler = wrapMiddlewares(wrappedHandler, m.middlewares)
	wrappedHandler = wrapMiddlewares(wrappedHandler, appMiddlewares)
	wrappedHandler = telemetryMiddleware(wrappedHandler)

	if i18nConfig, ok := i18n.Configuration(); ok && i18nConfig.FS != nil {
		i18nMdwr := I18nMiddleware(i18nConfig.FS)
		wrappedHandler = i18nMdwr(wrappedHandler)
	}

	m.ServeMux.Handle(pattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		statusCode := 0
		wrappedHandler.ServeHTTP(ResponseWriter{w, &statusCode}, &Request{r})
	}))

	return &HandlerConfig{
		pathPattern: pattern,
	}
}

// HandleFunc registers a handler function for the given pattern.
// Convenience method that wraps a HandlerFunc and calls Handle.
// Returns a handlerConfig that can be used to attach OpenAPI documentation via WithAPIConfig.
func (m *ServeMux) HandleFunc(pattern string, handler HandlerFunc, mdwrs ...interface{}) *HandlerConfig {
	wrappedHandler := wrapMiddlewares(handler, getHandlerMiddlewares(mdwrs))
	wrappedHandler = wrapMiddlewares(wrappedHandler, m.middlewares)
	wrappedHandler = wrapMiddlewares(wrappedHandler, appMiddlewares)
	wrappedHandler = telemetryMiddleware(wrappedHandler)

	if i18nConfig, ok := i18n.Configuration(); ok && i18nConfig.FS != nil {
		i18nMdwr := I18nMiddleware(i18nConfig.FS)
		wrappedHandler = i18nMdwr(wrappedHandler)
	}

	m.ServeMux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		statusCode := 0
		wrappedHandler.ServeHTTP(ResponseWriter{w, &statusCode}, &Request{r})
	})

	return &HandlerConfig{
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

	if i18nPrinter, ok := i18n.PrinterFromContext(r.Context()); ok {
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

	// Update request context if modified (for i18n or JSONP)
	if ctx != r.Context() {
		r.Request = r.WithContext(ctx)
	}

	hf(w, r)
}

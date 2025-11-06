package openapi

import (
	"encoding/json"
	"fmt"

	yaml "sigs.k8s.io/yaml/goyaml.v2"
)

const (
	version           = "3.2.0"
	JSONSchemaDialect = "https://json-schema.org/draft/2020-12/schema"
)

type (
	Config struct {
		Version           string `json:"openapi" yaml:"openapi"`
		Self              string `json:"$self,omitempty" yaml:"$self,omitempty"`
		Info              *Info
		JSONSchemaDialect string `json:"jsonSchemaDialect,omitempty" yaml:"jsonSchemaDialect,omitempty"`
		Servers           []Server
		Tags              []Tag         `json:"tags,omitempty" yaml:"tags,omitempty"`
		ExternalDocs      *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
		Paths             Paths         `json:"paths" yaml:"paths"`
		Components        *Components   `json:"components,omitempty" yaml:"components,omitempty"`
	}
	Info struct {
		Title          string `json:"title" yaml:"title"`
		Summary        string `json:"summary,omitempty" yaml:"summary,omitempty"`
		Description    string `json:"description,omitempty" yaml:"description,omitempty"`
		TermsOfService string `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`
		Contact        *Contact
		License        *License
		Version        string `json:"version" yaml:"version"`
	}
	Contact struct {
		Name  string `json:"name,omitempty" yaml:"name,omitempty"`
		URL   string `json:"url,omitempty" yaml:"url,omitempty"`
		Email string `json:"email,omitempty" yaml:"email,omitempty"`
	}
	License struct {
		Name       string `json:"name" yaml:"name"`
		Identifier string `json:"identifier,omitempty" yaml:"identifier,omitempty"`
		URL        string `json:"url,omitempty" yaml:"url,omitempty"`
	}
	Server struct {
		URL         string                    `json:"url" yaml:"url"`
		Name        string                    `json:"name,omitempty" yaml:"name,omitempty"`
		Description string                    `json:"description,omitempty" yaml:"description,omitempty"`
		Variables   map[string]ServerVariable `json:"variables,omitempty" yaml:"variables,omitempty"`
	}
	ServerVariable struct {
		Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
		Default     string   `json:"default" yaml:"default"`
		Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	}
	Tag struct {
		Name         string        `json:"name" yaml:"name"`
		Summary      string        `json:"summary,omitempty" yaml:"summary,omitempty"`
		Description  string        `json:"description,omitempty" yaml:"description,omitempty"`
		ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
		Parent       string        `json:"parent,omitempty" yaml:"parent,omitempty"`
		Kind         string        `json:"kind,omitempty" yaml:"kind,omitempty"`
	}
	ExternalDocs struct {
		Description string `json:"description,omitempty" yaml:"description,omitempty"`
		URL         string `json:"url" yaml:"url"`
	}
	Paths      map[string]PathItem
	Components struct {
		Schemas         map[string]Schema              `json:"schemas,omitempty" yaml:"schemas,omitempty"`
		Responses       map[string]ResponseOrRef       `json:"responses,omitempty" yaml:"responses,omitempty"`
		Parameters      map[string]ParameterOrRef      `json:"parameters,omitempty" yaml:"parameters,omitempty"`
		Examples        map[string]ExampleOrRef        `json:"examples,omitempty" yaml:"examples,omitempty"`
		RequestBodies   map[string]RequestBodyOrRef    `json:"requestBodies,omitempty" yaml:"requestBodies,omitempty"`
		Headers         map[string]HeaderOrRef         `json:"headers,omitempty" yaml:"headers,omitempty"`
		SecuritySchemes map[string]SecuritySchemeOrRef `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
		Links           map[string]LinkOrRef           `json:"links,omitempty" yaml:"links,omitempty"`
		Callbacks       map[string]CallbackOrRef       `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
		PathItems       map[string]PathItem            `json:"pathItems,omitempty" yaml:"pathItems,omitempty"`
		MediaTypes      map[string]MediaTypeOrRef      `json:"mediaTypes,omitempty" yaml:"mediaTypes,omitempty"`
	}
	Schema struct {
		Schema        string         `json:"$schema,omitempty" yaml:"$schema,omitempty"`
		Discriminator *Discriminator `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`
		XML           *XML           `json:"xml,omitempty" yaml:"xml,omitempty"`
		ExternalDocs  *ExternalDocs  `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`

		// Core JSON Schema fields (2020-12 subset commonly used in OAS 3.x)
		Title            string   `json:"title,omitempty" yaml:"title,omitempty"`
		MultipleOf       *float64 `json:"multipleOf,omitempty,omitzero" yaml:"multipleOf,omitempty,omitzero"`
		Maximum          *float64 `json:"maximum,omitempty,omitzero" yaml:"maximum,omitempty,omitzero"`
		ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty,omitzero" yaml:"exclusiveMaximum,omitempty,omitzero"`
		Minimum          *float64 `json:"minimum,omitempty,omitzero" yaml:"minimum,omitempty,omitzero"`
		ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty,omitzero" yaml:"exclusiveMinimum,omitempty,omitzero"`
		MaxLength        *int     `json:"maxLength,omitempty,omitzero" yaml:"maxLength,omitempty,omitzero"`
		MinLength        *int     `json:"minLength,omitempty,omitzero" yaml:"minLength,omitempty,omitzero"`
		Pattern          string   `json:"pattern,omitempty" yaml:"pattern,omitempty"`

		// Arrays
		Items       *SchemaOrRef `json:"items,omitempty" yaml:"items,omitempty"`
		MaxItems    *int         `json:"maxItems,omitempty,omitzero" yaml:"maxItems,omitempty,omitzero"`
		MinItems    *int         `json:"minItems,omitempty,omitzero" yaml:"minItems,omitempty,omitzero"`
		UniqueItems bool         `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`

		// Objects
		Properties map[string]SchemaOrRef `json:"properties,omitempty" yaml:"properties,omitempty"`
		// AdditionalProperties can be bool or SchemaOrRef
		AdditionalProperties any      `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
		Required             []string `json:"required,omitempty" yaml:"required,omitempty"`
		MaxProperties        *int     `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
		MinProperties        *int     `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`

		// General
		Type        string             `json:"type,omitempty" yaml:"type,omitempty"`
		Format      string             `json:"format,omitempty" yaml:"format,omitempty"`
		Enum        []any              `json:"enum,omitempty" yaml:"enum,omitempty"`
		Const       any                `json:"const,omitempty" yaml:"const,omitempty"`
		Nullable    bool               `json:"nullable,omitempty" yaml:"nullable,omitempty"`
		ReadOnly    bool               `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
		WriteOnly   bool               `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`
		Description string             `json:"description,omitempty" yaml:"description,omitempty"`
		Default     any                `json:"default,omitempty" yaml:"default,omitempty"`
		Example     any                `json:"example,omitempty" yaml:"example,omitempty"`
		Examples    map[string]Example `json:"examples,omitempty" yaml:"examples,omitempty"`

		// Composition
		AllOf []SchemaOrRef `json:"allOf,omitempty" yaml:"allOf,omitempty"`
		OneOf []SchemaOrRef `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
		AnyOf []SchemaOrRef `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
		Not   *SchemaOrRef  `json:"not,omitempty" yaml:"not,omitempty"`

		// Misc OAS-specific
		Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	}
	Discriminator struct {
		PropertyName   string            `json:"propertyName" yaml:"propertyName"`
		Mapping        map[string]string `json:"mapping,omitempty" yaml:"mapping,omitempty"`
		DefaultMapping string            `json:"defaultMapping,omitempty" yaml:"defaultMapping,omitempty"`
	}
	XML struct {
		NodeType  string `json:"nodeType,omitempty" yaml:"nodeType,omitempty"`
		Name      string `json:"name,omitempty" yaml:"name,omitempty"`
		Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
		Prefix    string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
	}
	Operation struct {
		Tags         []string                 `json:"tags,omitempty" yaml:"tags,omitempty"`
		Summary      string                   `json:"summary,omitempty" yaml:"summary,omitempty"`
		Description  string                   `json:"description,omitempty" yaml:"description,omitempty"`
		ExternalDocs *ExternalDocs            `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
		OperationID  string                   `json:"operationId,omitempty" yaml:"operationId,omitempty"`
		Parameters   []ParameterOrRef         `json:"parameters,omitempty" yaml:"parameters,omitempty"`
		RequestBody  *RequestBodyOrRef        `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
		Responses    map[string]ResponseOrRef `json:"responses" yaml:"responses"`
		Callbacks    map[string]CallbackOrRef `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
		Deprecated   bool                     `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
		Security     []map[string][]string    `json:"security,omitempty" yaml:"security,omitempty"`
		Servers      []Server                 `json:"servers,omitempty" yaml:"servers,omitempty"`
	}
	PathItem struct {
		Summary              string                `json:"summary,omitempty" yaml:"summary,omitempty"`
		Description          string                `json:"description,omitempty" yaml:"description,omitempty"`
		Get                  *Operation            `json:"get,omitempty" yaml:"get,omitempty"`
		Put                  *Operation            `json:"put,omitempty" yaml:"put,omitempty"`
		Post                 *Operation            `json:"post,omitempty" yaml:"post,omitempty"`
		Delete               *Operation            `json:"delete,omitempty" yaml:"delete,omitempty"`
		Options              *Operation            `json:"options,omitempty" yaml:"options,omitempty"`
		Head                 *Operation            `json:"head,omitempty" yaml:"head,omitempty"`
		Patch                *Operation            `json:"patch,omitempty" yaml:"patch,omitempty"`
		Trace                *Operation            `json:"trace,omitempty" yaml:"trace,omitempty"`
		Query                *Operation            `json:"query,omitempty" yaml:"query,omitempty"`
		AdditionalOperations map[string]*Operation `json:"additionalOperations,omitempty" yaml:"additionalOperations,omitempty"`
		Servers              []Server              `json:"servers,omitempty" yaml:"servers,omitempty"`
		Parameters           []ParameterOrRef      `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	}
	Reference struct {
		Ref         string `json:"$ref" yaml:"$ref"`
		Summary     string `json:"summary,omitempty" yaml:"summary,omitempty"`
		Description string `json:"description,omitempty" yaml:"description,omitempty"`
	}
	Callback map[string]PathItem
	Link     struct {
		OperationRef string         `json:"operationRef,omitempty" yaml:"operationRef,omitempty"`
		OperationId  string         `json:"operationId,omitempty" yaml:"operationId,omitempty"`
		Parameters   map[string]any `json:"parameters,omitempty" yaml:"parameters,omitempty"`
		RequestBody  any            `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
		Description  string         `json:"description,omitempty" yaml:"description,omitempty"`
	}
	Header struct {
		Description string                    `json:"description,omitempty" yaml:"description,omitempty"`
		Required    bool                      `json:"required,omitempty" yaml:"required,omitempty"`
		Deprecated  bool                      `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
		Example     any                       `json:"example,omitempty" yaml:"example,omitempty"`
		Examples    map[string]ExampleOrRef   `json:"examples,omitempty" yaml:"examples,omitempty"`
		Style       string                    `json:"style,omitempty" yaml:"style,omitempty"`
		Explode     *bool                     `json:"explode,omitempty" yaml:"explode,omitempty"`
		Schema      *SchemaOrRef              `json:"schema,omitempty" yaml:"schema,omitempty"`
		Content     map[string]MediaTypeOrRef `json:"content,omitempty" yaml:"content,omitempty"`
	}
	Example struct {
		Summary         string `json:"summary,omitempty" yaml:"summary,omitempty"`
		Description     string `json:"description,omitempty" yaml:"description,omitempty"`
		DefaultValue    any    `json:"defaultValue,omitempty" yaml:"defaultValue,omitempty"`
		SerializedValue any    `json:"serializedValue,omitempty" yaml:"serializedValue,omitempty"`
		ExternalValue   string `json:"externalValue,omitempty" yaml:"externalValue,omitempty"`
		DataValue       any    `json:"dataValue,omitempty" yaml:"dataValue,omitempty"`
	}
	ParameterOrRef struct {
		*Parameter
		Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	}
	RequestBodyOrRef struct {
		*RequestBody
		Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	}
	CallbackOrRef struct {
		*Callback
		Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	}
	ExampleOrRef struct {
		*Example
		Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	}
	HeaderOrRef struct {
		*Header
		Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	}
	LinkOrRef struct {
		*Link
		Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	}
	SchemaOrRef struct {
		*Schema
		Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	}
	ResponseOrRef struct {
		*Response
		Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	}
	SecuritySchemeOrRef struct {
		*SecurityScheme
		Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	}
	MediaTypeOrRef struct {
		*MediaType
		Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	}
	Parameter struct {
		Name          string                  `json:"name" yaml:"name"`
		In            string                  `json:"in" yaml:"in"` // "query", "header", "path", "cookie"
		Description   string                  `json:"description,omitempty" yaml:"description,omitempty"`
		Required      bool                    `json:"required,omitempty" yaml:"required,omitempty"`
		Deprecated    bool                    `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
		Style         string                  `json:"style,omitempty" yaml:"style,omitempty"`
		Explode       *bool                   `json:"explode,omitempty" yaml:"explode,omitempty"`
		AllowReserved bool                    `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty"`
		Schema        *SchemaOrRef            `json:"schema,omitempty" yaml:"schema,omitempty"`
		Example       any                     `json:"example,omitempty" yaml:"example,omitempty"`
		Examples      map[string]ExampleOrRef `json:"examples,omitempty" yaml:"examples,omitempty"`
		Content       map[string]MediaType    `json:"content,omitempty" yaml:"content,omitempty"`
	}
	RequestBody struct {
		Description string               `json:"description,omitempty" yaml:"description,omitempty"`
		Content     map[string]MediaType `json:"content" yaml:"content"`
		Required    bool                 `json:"required,omitempty" yaml:"required,omitempty"`
	}
	Response struct {
		Summary     string                 `json:"summary,omitempty" yaml:"summary,omitempty"`
		Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
		Headers     map[string]HeaderOrRef `json:"headers,omitempty" yaml:"headers,omitempty"`
		Content     map[string]MediaType   `json:"content,omitempty" yaml:"content,omitempty"`
		Links       map[string]LinkOrRef   `json:"links,omitempty" yaml:"links,omitempty"`
	}
	MediaType struct {
		Schema     *SchemaOrRef            `json:"schema,omitempty" yaml:"schema,omitempty"`
		ItemSchema *SchemaOrRef            `json:"itemSchema,omitempty" yaml:"itemSchema,omitempty"`
		Example    any                     `json:"example,omitempty" yaml:"example,omitempty"`
		Examples   map[string]ExampleOrRef `json:"examples,omitempty" yaml:"examples,omitempty"`
	}
	Encoding struct {
		ContentType  string                 `json:"contentType,omitempty" yaml:"contentType,omitempty"`
		Headers      map[string]HeaderOrRef `json:"headers,omitempty" yaml:"headers,omitempty"`
		Encoding     map[string]Encoding    `json:"encoding,omitempty" yaml:"encoding,omitempty"`
		Prefix       *Encoding              `json:"prefix,omitempty" yaml:"prefix,omitempty"`
		ItemEncoding *Encoding              `json:"itemEncoding,omitempty" yaml:"itemEncoding,omitempty"`
	}
	SecurityScheme struct {
		Type              string     `json:"type" yaml:"type"` // "apiKey", "http", "oauth2", "openIdConnect"
		Description       string     `json:"description,omitempty" yaml:"description,omitempty"`
		Name              string     `json:"name" yaml:"name"`
		In                string     `json:"in" yaml:"in"` // "query", "header", "cookie"
		Scheme            string     `json:"scheme" yaml:"scheme"`
		BearerFormat      string     `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`
		Flows             OAuthFlows `json:"flows" yaml:"flows"`
		OpenIdConnectURL  string     `json:"openIdConnectUrl" yaml:"openIdConnectUrl"`
		OAuth2MetadataUrl string     `json:"oauth2MetadataUrl,omitempty" yaml:"oauth2MetadataUrl,omitempty"`
		Deprecated        bool       `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	}
	OAuthFlows struct {
		Implicit            *OAuthFlow `json:"implicit,omitempty" yaml:"implicit,omitempty"`
		Password            *OAuthFlow `json:"password,omitempty" yaml:"password,omitempty"`
		ClientCredentials   *OAuthFlow `json:"clientCredentials,omitempty" yaml:"clientCredentials,omitempty"`
		AuthorizationCode   *OAuthFlow `json:"authorizationCode,omitempty" yaml:"authorizationCode,omitempty"`
		DeviceAuthorization *OAuthFlow `json:"deviceAuthorization,omitempty" yaml:"deviceAuthorization,omitempty"`
	}
	OAuthFlow struct {
		AuthorizationURL       string            `json:"authorizationUrl" yaml:"authorizationUrl"`
		DeviceAuthorizationURL string            `json:"deviceAuthorizationUrl" yaml:"deviceAuthorizationUrl"`
		TokenURL               string            `json:"tokenUrl" yaml:"tokenUrl"`
		RefreshURL             string            `json:"refreshUrl,omitempty" yaml:"refreshUrl,omitempty"`
		Scopes                 map[string]string `json:"scopes" yaml:"scopes"`
	}
)

// SetDefaults initializes required OpenAPI configuration fields with default values.
// Sets OpenAPI version to 3.2.0, ensures Info is initialized, creates empty paths/components if nil.
func (c *Config) SetDefaults() {
	if c.Version == "" {
		c.Version = version
	} else if c.Version != version {
		panic(fmt.Errorf("wrong version provided. Must be %q", version))
	}

	if c.Info != nil {
		if c.Info.Version == "" {
			c.Info.Version = "1.0.0"
		}
		if c.Info.Title == "" {
			c.Info.Title = "API"
		}
	}

	if c.JSONSchemaDialect == "" {
		c.JSONSchemaDialect = JSONSchemaDialect
	} else if c.JSONSchemaDialect != JSONSchemaDialect {
		panic(fmt.Errorf("wrong JSON Schema Dialect provided. Must be %q", JSONSchemaDialect))
	}
}

// SetPathInfo sets or updates path-level information in the OpenAPI specification.
// Allows setting common parameters, servers, and descriptions that apply to all operations on a path.
// Creates a new PathItem if the path doesn't exist.
func (ps *Paths) SetPathInfo(path string, summary string, description string, parameters []ParameterOrRef, servers []Server) {
	if *ps == nil {
		*ps = make(map[string]PathItem)
	}

	pathItem, exists := (*ps)[path]

	if !exists {
		pathItem = PathItem{}
	}

	pathItem.Summary = summary
	pathItem.Description = description
	pathItem.Parameters = parameters
	pathItem.Servers = servers

	(*ps)[path] = pathItem
}

// AddOperation adds an HTTP operation (GET, POST, PUT, etc.) to a path in the OpenAPI specification.
// The method parameter should be lowercase (get, post, put, delete, patch, options, head, trace).
// Creates a new PathItem if the path doesn't exist.
func (ps *Paths) AddOperation(path string, method string, operation Operation) {
	if *ps == nil {
		*ps = make(map[string]PathItem)
	}

	pathItem, exists := (*ps)[path]

	if !exists {
		pathItem = PathItem{}
	}

	switch method {
	case "get":
		pathItem.Get = &operation
	case "put":
		pathItem.Put = &operation
	case "post":
		pathItem.Post = &operation
	case "delete":
		pathItem.Delete = &operation
	case "options":
		pathItem.Options = &operation
	case "head":
		pathItem.Head = &operation
	case "patch":
		pathItem.Patch = &operation
	case "trace":
		pathItem.Trace = &operation
	case "query":
		pathItem.Query = &operation
	}

	(*ps)[path] = pathItem
}

// Validate checks that the OpenAPI configuration is valid.
// Currently performs no validation and always returns nil.
func (c *Config) Validate() error {
	if c.Info == nil {
		return fmt.Errorf("info is required in OpenAPI configuration")
	}
	return nil
}

// MarshalToJSON marshals a SchemaOrRef to JSON format.
// Handles both schema references ($ref) and inline schema definitions.
func (s SchemaOrRef) MarshalToJSON() ([]byte, error) {
	if s.Ref != "" {
		return json.Marshal(s.Ref)
	}
	return json.Marshal(s.Schema)
}

// MarshalToYAML marshals a SchemaOrRef to YAML format.
// Handles both schema references ($ref) and inline schema definitions.
func (s SchemaOrRef) MarshalToYAML() ([]byte, error) {
	if s.Ref != "" {
		return yaml.Marshal(s.Ref)
	}
	return yaml.Marshal(s.Schema)
}

// ToJSON converts a Schema to formatted JSON string with indentation.
// Returns the JSON string or an error if marshaling fails.
func (s *Schema) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ToJSONCompact converts a Schema to compact JSON string without indentation.
// Returns the JSON string or an error if marshaling fails.
func (s *Schema) ToJSONCompact() (string, error) {
	bytes, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ToYAML converts a Schema to YAML string.
// Returns the YAML string or an error if marshaling fails.
func (s *Schema) ToYAML() (string, error) {
	bytes, err := yaml.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// MarshalYaml converts the entire OpenAPI configuration to YAML format.
// Returns the YAML bytes or an error if marshaling fails.
func (c *Config) MarshalYaml() ([]byte, error) {
	c.SetDefaults()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	// Use type alias to avoid infinite recursion
	type ConfigAlias Config
	return yaml.Marshal((*ConfigAlias)(c))
}

// MarshalJSON converts the entire OpenAPI configuration to formatted JSON.
// Automatically sets defaults before marshaling if not already set.
// Returns the JSON bytes or an error if marshaling fails.
func (c *Config) MarshalJSON() ([]byte, error) {
	c.SetDefaults()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	// Use type alias to avoid infinite recursion
	type ConfigAlias Config
	return json.Marshal((*ConfigAlias)(c))
}

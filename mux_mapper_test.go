package webfram

import (
	"testing"

	"github.com/bondowe/webfram/openapi"
)

// =============================================================================
// mapLinks Tests
// =============================================================================

func TestMapLinks_WithLinks(t *testing.T) {
	setupMuxTestWithOpenAPI()

	links := map[string]Link{
		"getUserAddress": {
			OperationRef: "#/paths/~1users~1{id}~1address/get",
			OperationID:  "getUserAddress",
			Parameters: map[string]any{
				"id": "$response.body#/id",
			},
			Description: "Get user's address",
		},
		"deleteUser": {
			OperationID: "deleteUser",
			Parameters: map[string]any{
				"userId": "$request.path.id",
			},
			RequestBody: map[string]any{
				"reason": "$request.body#/reason",
			},
			Description: "Delete the user",
		},
	}

	result := mapLinks(links)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 links, got %d", len(result))
	}

	getUserLink, ok := result["getUserAddress"]
	if !ok {
		t.Fatal("Expected 'getUserAddress' link to exist")
	}

	if getUserLink.Link == nil {
		t.Fatal("Expected Link to be non-nil")
	}

	if getUserLink.Link.OperationRef != "#/paths/~1users~1{id}~1address/get" {
		t.Errorf("Expected OperationRef '#/paths/~1users~1{id}~1address/get', got %q", getUserLink.Link.OperationRef)
	}

	if getUserLink.Link.OperationId != "getUserAddress" {
		t.Errorf("Expected OperationId 'getUserAddress', got %q", getUserLink.Link.OperationId)
	}

	if getUserLink.Link.Description != "Get user's address" {
		t.Errorf("Expected Description 'Get user's address', got %q", getUserLink.Link.Description)
	}

	deleteLink, ok := result["deleteUser"]
	if !ok {
		t.Fatal("Expected 'deleteUser' link to exist")
	}

	if deleteLink.Link.RequestBody == nil {
		t.Error("Expected RequestBody to be non-nil")
	}
}

func TestMapLinks_NilInput(t *testing.T) {
	result := mapLinks(nil)

	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}

func TestMapLinks_EmptyMap(t *testing.T) {
	result := mapLinks(map[string]Link{})

	if result == nil {
		t.Error("Expected non-nil result for empty map")
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 links, got %d", len(result))
	}
}

// =============================================================================
// mapContent Tests
// =============================================================================

func TestMapContent_WithSingleMediaType(t *testing.T) {
	setupMuxTestWithOpenAPI()

	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	content := map[string]TypeInfo{
		"application/json": {
			TypeHint: User{},
			Example: map[string]any{
				"name":  "John Doe",
				"email": "john@example.com",
			},
		},
	}

	result := mapContent(content)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 media type, got %d", len(result))
	}

	mediaType, ok := result["application/json"]
	if !ok {
		t.Fatal("Expected 'application/json' media type to exist")
	}

	if mediaType.Schema == nil {
		t.Error("Expected Schema to be non-nil")
	}

	if mediaType.Example == nil {
		t.Error("Expected Example to be non-nil")
	}
}

func TestMapContent_WithMultipleMediaTypes(t *testing.T) {
	setupMuxTestWithOpenAPI()

	type Response struct {
		Message string `json:"message" xml:"message"`
	}

	content := map[string]TypeInfo{
		"application/json,application/xml": {
			TypeHint: Response{},
		},
	}

	result := mapContent(content)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Should create separate entries for each media type
	if len(result) != 2 {
		t.Errorf("Expected 2 media types, got %d", len(result))
	}

	_, hasJSON := result["application/json"]
	if !hasJSON {
		t.Error("Expected 'application/json' media type to exist")
	}

	_, hasXML := result["application/xml"]
	if !hasXML {
		t.Error("Expected 'application/xml' media type to exist")
	}
}

func TestMapContent_WithExamples(t *testing.T) {
	setupMuxTestWithOpenAPI()

	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	content := map[string]TypeInfo{
		"application/json": {
			TypeHint: User{},
			Example: map[string]any{
				"id":   1,
				"name": "Default User",
			},
			Examples: map[string]Example{
				"user1": {
					Summary:     "First user",
					Description: "Example of first user",
					DataValue: map[string]any{
						"id":   1,
						"name": "Alice",
					},
				},
				"user2": {
					Summary:     "Second user",
					Description: "Example of second user",
					DataValue: map[string]any{
						"id":   2,
						"name": "Bob",
					},
				},
			},
		},
	}

	result := mapContent(content)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	mediaType := result["application/json"]
	if mediaType.Examples == nil {
		t.Fatal("Expected Examples to be non-nil")
	}

	if len(mediaType.Examples) != 2 {
		t.Errorf("Expected 2 examples, got %d", len(mediaType.Examples))
	}

	example1, ok := mediaType.Examples["user1"]
	if !ok {
		t.Fatal("Expected 'user1' example to exist")
	}

	if example1.Example == nil {
		t.Fatal("Expected Example to be non-nil")
	}

	if example1.Example.Summary != "First user" {
		t.Errorf("Expected Summary 'First user', got %q", example1.Example.Summary)
	}
}

func TestMapContent_NilInput(t *testing.T) {
	result := mapContent(nil)

	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}

func TestMapContent_TextEventStream_AutoSSEPayload(t *testing.T) {
	setupMuxTestWithOpenAPI()

	// Test that text/event-stream automatically uses SSEPayload type
	content := map[string]TypeInfo{
		"text/event-stream": {
			TypeHint: nil, // Should be auto-set to SSEPayload
		},
	}

	result := mapContent(content)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 media type, got %d", len(result))
	}

	mediaType, ok := result["text/event-stream"]
	if !ok {
		t.Fatal("Expected 'text/event-stream' media type to exist")
	}

	// For SSE, ItemSchema should be set, not Schema
	if mediaType.ItemSchema == nil {
		t.Error("Expected ItemSchema to be set for text/event-stream")
	}

	if mediaType.Schema != nil {
		t.Error("Expected Schema to be nil for text/event-stream (should use ItemSchema)")
	}
}

func TestMapContent_TextEventStream_IgnoresCustomTypeHint(t *testing.T) {
	setupMuxTestWithOpenAPI()

	type CustomPayload struct {
		CustomField string `json:"customField"`
	}

	// Even with custom TypeHint, should be overridden to SSEPayload
	content := map[string]TypeInfo{
		"text/event-stream": {
			TypeHint: &CustomPayload{},
		},
	}

	result := mapContent(content)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	mediaType, ok := result["text/event-stream"]
	if !ok {
		t.Fatal("Expected 'text/event-stream' media type to exist")
	}

	// Should still use ItemSchema (indicating SSEPayload was used)
	if mediaType.ItemSchema == nil {
		t.Error("Expected ItemSchema to be set (SSEPayload should be auto-applied)")
	}

	if mediaType.Schema != nil {
		t.Error("Expected Schema to be nil for text/event-stream")
	}
}

func TestMapContent_ApplicationJSONSeq_UsesItemSchema(t *testing.T) {
	setupMuxTestWithOpenAPI()

	type LogEntry struct {
		Timestamp string `json:"timestamp"`
		Level     string `json:"level"`
		Message   string `json:"message"`
	}

	// For JSON-seq, TypeHint should point to the line item type
	content := map[string]TypeInfo{
		"application/json-seq": {
			TypeHint: &LogEntry{},
		},
	}

	result := mapContent(content)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 media type, got %d", len(result))
	}

	mediaType, ok := result["application/json-seq"]
	if !ok {
		t.Fatal("Expected 'application/json-seq' media type to exist")
	}

	// For JSON-seq, ItemSchema should be set, not Schema
	if mediaType.ItemSchema == nil {
		t.Error("Expected ItemSchema to be set for application/json-seq")
	}

	if mediaType.Schema != nil {
		t.Error("Expected Schema to be nil for application/json-seq (should use ItemSchema)")
	}
}

func TestMapContent_MixedStreamingAndRegular(t *testing.T) {
	setupMuxTestWithOpenAPI()

	type DataModel struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	// Mix of regular, SSE, and JSON-seq media types
	content := map[string]TypeInfo{
		"application/json": {
			TypeHint: &DataModel{},
		},
		"text/event-stream": {
			TypeHint: nil, // Will be auto-set to SSEPayload
		},
		"application/json-seq": {
			TypeHint: &DataModel{},
		},
	}

	result := mapContent(content)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 media types, got %d", len(result))
	}

	// Check regular JSON uses Schema
	jsonMedia, ok := result["application/json"]
	if !ok {
		t.Fatal("Expected 'application/json' media type to exist")
	}

	if jsonMedia.Schema == nil {
		t.Error("Expected Schema to be set for application/json")
	}

	if jsonMedia.ItemSchema != nil {
		t.Error("Expected ItemSchema to be nil for application/json")
	}

	// Check SSE uses ItemSchema
	sseMedia, ok := result["text/event-stream"]
	if !ok {
		t.Fatal("Expected 'text/event-stream' media type to exist")
	}

	if sseMedia.ItemSchema == nil {
		t.Error("Expected ItemSchema to be set for text/event-stream")
	}

	if sseMedia.Schema != nil {
		t.Error("Expected Schema to be nil for text/event-stream")
	}

	// Check JSON-seq uses ItemSchema
	jsonSeqMedia, ok := result["application/json-seq"]
	if !ok {
		t.Fatal("Expected 'application/json-seq' media type to exist")
	}

	if jsonSeqMedia.ItemSchema == nil {
		t.Error("Expected ItemSchema to be set for application/json-seq")
	}

	if jsonSeqMedia.Schema != nil {
		t.Error("Expected Schema to be nil for application/json-seq")
	}
}

func TestMapContent_CommaSeparated_WithStreamingTypes(t *testing.T) {
	setupMuxTestWithOpenAPI()

	type Item struct {
		Value string `json:"value"`
	}

	// Comma-separated string with streaming types
	content := map[string]TypeInfo{
		"application/json,text/event-stream,application/json-seq": {
			TypeHint: &Item{},
		},
	}

	result := mapContent(content)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Should split into 3 separate media types
	if len(result) != 3 {
		t.Errorf("Expected 3 media types after split, got %d", len(result))
	}

	// Regular JSON should use Schema
	if jsonMedia, ok := result["application/json"]; ok {
		if jsonMedia.Schema == nil {
			t.Error("Expected Schema for application/json")
		}
		if jsonMedia.ItemSchema != nil {
			t.Error("Expected ItemSchema to be nil for application/json")
		}
	} else {
		t.Error("Expected 'application/json' to exist")
	}

	// SSE should use ItemSchema and auto-convert to SSEPayload
	if sseMedia, ok := result["text/event-stream"]; ok {
		if sseMedia.ItemSchema == nil {
			t.Error("Expected ItemSchema for text/event-stream")
		}
		if sseMedia.Schema != nil {
			t.Error("Expected Schema to be nil for text/event-stream")
		}
	} else {
		t.Error("Expected 'text/event-stream' to exist")
	}

	// JSON-seq should use ItemSchema
	if jsonSeqMedia, ok := result["application/json-seq"]; ok {
		if jsonSeqMedia.ItemSchema == nil {
			t.Error("Expected ItemSchema for application/json-seq")
		}
		if jsonSeqMedia.Schema != nil {
			t.Error("Expected Schema to be nil for application/json-seq")
		}
	} else {
		t.Error("Expected 'application/json-seq' to exist")
	}
}

func TestMapContent_WithExamplesAndStreamingTypes(t *testing.T) {
	setupMuxTestWithOpenAPI()

	type Notification struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	}

	examples := map[string]Example{
		"info": {
			Summary:   "Info notification",
			DataValue: Notification{ID: "1", Message: "Info message"},
		},
		"warning": {
			Summary:   "Warning notification",
			DataValue: Notification{ID: "2", Message: "Warning message"},
		},
	}

	exampleValue := Notification{ID: "0", Message: "Default"}

	content := map[string]TypeInfo{
		"text/event-stream": {
			TypeHint: nil, // Auto SSEPayload
			Example:  exampleValue,
			Examples: examples,
		},
		"application/json-seq": {
			TypeHint: &Notification{},
			Example:  exampleValue,
			Examples: examples,
		},
	}

	result := mapContent(content)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check SSE has examples
	sseMedia := result["text/event-stream"]
	if sseMedia.Example == nil {
		t.Error("Expected Example to be set for text/event-stream")
	}

	if sseMedia.Examples == nil {
		t.Fatal("Expected Examples to be set for text/event-stream")
	}

	if len(sseMedia.Examples) != 2 {
		t.Errorf("Expected 2 examples for text/event-stream, got %d", len(sseMedia.Examples))
	}

	// Check JSON-seq has examples
	jsonSeqMedia := result["application/json-seq"]
	if jsonSeqMedia.Example == nil {
		t.Error("Expected Example to be set for application/json-seq")
	}

	if jsonSeqMedia.Examples == nil {
		t.Fatal("Expected Examples to be set for application/json-seq")
	}

	if len(jsonSeqMedia.Examples) != 2 {
		t.Errorf("Expected 2 examples for application/json-seq, got %d", len(jsonSeqMedia.Examples))
	}
}

// =============================================================================
// mapHeaders Tests
// =============================================================================

func TestMapHeaders_WithBasicHeaders(t *testing.T) {
	setupMuxTestWithOpenAPI()

	headers := map[string]Header{
		"X-Rate-Limit": {
			Description: "Rate limit",
			TypeHint:    0,
			Required:    true,
		},
		"X-Request-ID": {
			Description: "Request ID",
			TypeHint:    "",
			Deprecated:  false,
		},
	}

	result := mapHeaders(headers)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(result))
	}

	rateLimit, ok := result["X-Rate-Limit"]
	if !ok {
		t.Fatal("Expected 'X-Rate-Limit' header to exist")
	}

	if rateLimit.Header == nil {
		t.Fatal("Expected Header to be non-nil")
	}

	if rateLimit.Header.Description != "Rate limit" {
		t.Errorf("Expected Description 'Rate limit', got %q", rateLimit.Header.Description)
	}

	if !rateLimit.Header.Required {
		t.Error("Expected Required to be true")
	}

	if rateLimit.Header.Schema == nil {
		t.Error("Expected Schema to be non-nil")
	}
}

func TestMapHeaders_WithContent(t *testing.T) {
	setupMuxTestWithOpenAPI()

	type ErrorDetail struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	headers := map[string]Header{
		"X-Error-Details": {
			Description: "Error details",
			Content: map[string]TypeInfo{
				"application/json": {
					TypeHint: ErrorDetail{},
				},
			},
		},
	}

	result := mapHeaders(headers)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	errorHeader, ok := result["X-Error-Details"]
	if !ok {
		t.Fatal("Expected 'X-Error-Details' header to exist")
	}

	if errorHeader.Header.Content == nil {
		t.Fatal("Expected Content to be non-nil")
	}

	if len(errorHeader.Header.Content) != 1 {
		t.Errorf("Expected 1 content type, got %d", len(errorHeader.Header.Content))
	}

	// When Content is provided, Schema should be nil
	if errorHeader.Header.Schema != nil {
		t.Error("Expected Schema to be nil when Content is provided")
	}
}

func TestMapHeaders_WithExamples(t *testing.T) {
	setupMuxTestWithOpenAPI()

	headers := map[string]Header{
		"X-Custom-Header": {
			Description: "Custom header",
			TypeHint:    "",
			Example:     "example-value",
			Examples: map[string]Example{
				"example1": {
					Summary:   "First example",
					DataValue: "value1",
				},
				"example2": {
					Summary:   "Second example",
					DataValue: "value2",
				},
			},
		},
	}

	result := mapHeaders(headers)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	customHeader, ok := result["X-Custom-Header"]
	if !ok {
		t.Fatal("Expected 'X-Custom-Header' header to exist")
	}

	if customHeader.Header.Example != "example-value" {
		t.Errorf("Expected Example 'example-value', got %v", customHeader.Header.Example)
	}

	if customHeader.Header.Examples == nil {
		t.Fatal("Expected Examples to be non-nil")
	}

	if len(customHeader.Header.Examples) != 2 {
		t.Errorf("Expected 2 examples, got %d", len(customHeader.Header.Examples))
	}
}

func TestMapHeaders_WithExplodeAndStyle(t *testing.T) {
	setupMuxTestWithOpenAPI()

	explodeTrue := true

	headers := map[string]Header{
		"X-Array-Header": {
			Description: "Array header",
			TypeHint:    []string{},
			Style:       "simple",
			Explode:     &explodeTrue,
		},
	}

	result := mapHeaders(headers)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	arrayHeader, ok := result["X-Array-Header"]
	if !ok {
		t.Fatal("Expected 'X-Array-Header' header to exist")
	}

	if arrayHeader.Header.Style != "simple" {
		t.Errorf("Expected Style 'simple', got %q", arrayHeader.Header.Style)
	}

	if arrayHeader.Header.Explode == nil {
		t.Fatal("Expected Explode to be non-nil")
	}

	if !*arrayHeader.Header.Explode {
		t.Error("Expected Explode to be true")
	}
}

func TestMapHeaders_NilInput(t *testing.T) {
	result := mapHeaders(nil)

	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}

// =============================================================================
// processParameterSchema Tests
// =============================================================================

func TestProcessParameterSchema_WithContent(t *testing.T) {
	setupMuxTestWithOpenAPI()

	type CustomModel struct {
		Value string `json:"value"`
	}

	param := &Parameter{
		Name: "custom",
		In:   "query",
		Content: map[string]any{
			"application/json": CustomModel{},
		},
	}

	schema, content := processParameterSchema(param)

	// When Content is provided, Schema should be nil
	if schema != nil {
		t.Error("Expected Schema to be nil when Content is provided")
	}

	if content == nil {
		t.Fatal("Expected Content to be non-nil")
	}

	if len(content) != 1 {
		t.Errorf("Expected 1 content type, got %d", len(content))
	}

	_, ok := content["application/json"]
	if !ok {
		t.Error("Expected 'application/json' content type to exist")
	}
}

func TestProcessParameterSchema_WithoutContent(t *testing.T) {
	setupMuxTestWithOpenAPI()

	param := &Parameter{
		Name:     "limit",
		In:       "query",
		TypeHint: 0,
	}

	schema, content := processParameterSchema(param)

	// When Content is not provided, Schema should be non-nil
	if schema == nil {
		t.Error("Expected Schema to be non-nil when Content is not provided")
	}

	if content != nil {
		t.Error("Expected Content to be nil when not provided")
	}
}

// =============================================================================
// buildParameterContent Tests
// =============================================================================

func TestBuildParameterContent_SingleMediaType(t *testing.T) {
	setupMuxTestWithOpenAPI()

	type Filter struct {
		Field string `json:"field"`
		Value string `json:"value"`
	}

	content := map[string]any{
		"application/json": Filter{},
	}

	result := buildParameterContent(content)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 media type, got %d", len(result))
	}

	mediaType, ok := result["application/json"]
	if !ok {
		t.Fatal("Expected 'application/json' media type to exist")
	}

	if mediaType.Schema == nil {
		t.Error("Expected Schema to be non-nil")
	}
}

func TestBuildParameterContent_MultipleMediaTypes(t *testing.T) {
	setupMuxTestWithOpenAPI()

	type Data struct {
		Value string `json:"value" xml:"value"`
	}

	content := map[string]any{
		"application/json,application/xml": Data{},
	}

	result := buildParameterContent(content)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Should create separate entries for each media type
	if len(result) != 2 {
		t.Errorf("Expected 2 media types, got %d", len(result))
	}

	_, hasJSON := result["application/json"]
	if !hasJSON {
		t.Error("Expected 'application/json' media type to exist")
	}

	_, hasXML := result["application/xml"]
	if !hasXML {
		t.Error("Expected 'application/xml' media type to exist")
	}
}

// =============================================================================
// buildParameterSchema Tests
// =============================================================================

func TestBuildParameterSchema_DefaultTypeHint(t *testing.T) {
	setupMuxTestWithOpenAPI()

	param := &Parameter{
		Name:     "test",
		In:       "query",
		TypeHint: nil, // Should default to ""
	}

	result := buildParameterSchema(param)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Should have generated schema for empty string (string type)
	if result.Schema == nil {
		t.Error("Expected Schema to be non-nil")
	}
}

func TestBuildParameterSchema_WithConstraints(t *testing.T) {
	setupMuxTestWithOpenAPI()

	param := &Parameter{
		Name:      "username",
		In:        "query",
		TypeHint:  "",
		MinLength: 3,
		MaxLength: 20,
		Pattern:   "^[a-zA-Z0-9_]+$",
		Default:   "guest",
	}

	result := buildParameterSchema(param)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Schema == nil {
		t.Fatal("Expected Schema to be non-nil")
	}

	schema := result.Schema

	if schema.MinLength == nil {
		t.Error("Expected MinLength to be set")
	} else if *schema.MinLength != 3 {
		t.Errorf("Expected MinLength 3, got %d", *schema.MinLength)
	}

	if schema.MaxLength == nil {
		t.Error("Expected MaxLength to be set")
	} else if *schema.MaxLength != 20 {
		t.Errorf("Expected MaxLength 20, got %d", *schema.MaxLength)
	}

	if schema.Pattern != "^[a-zA-Z0-9_]+$" {
		t.Errorf("Expected Pattern '^[a-zA-Z0-9_]+$', got %q", schema.Pattern)
	}

	if schema.Default != "guest" {
		t.Errorf("Expected Default 'guest', got %v", schema.Default)
	}
}

// =============================================================================
// applySchemaConstraints Tests
// =============================================================================

func TestApplySchemaConstraints_StringType(t *testing.T) {
	setupMuxTestWithOpenAPI()

	schema := &openapi.Schema{
		Type: "string",
	}

	param := &Parameter{
		MinLength: 5,
		MaxLength: 50,
		Pattern:   "^[A-Z]",
		Format:    "email",
		Enum:      []any{"admin", "user", "guest"},
		Const:     "constant",
		Default:   "default",
		Nullable:  true,
	}

	applySchemaConstraints(schema, param)

	if schema.MinLength == nil {
		t.Error("Expected MinLength to be set")
	}

	if schema.MaxLength == nil {
		t.Error("Expected MaxLength to be set")
	}

	if schema.Pattern != "^[A-Z]" {
		t.Errorf("Expected Pattern '^[A-Z]', got %q", schema.Pattern)
	}

	if schema.Format != "email" {
		t.Errorf("Expected Format 'email', got %q", schema.Format)
	}

	if len(schema.Enum) != 3 {
		t.Errorf("Expected 3 enum values, got %d", len(schema.Enum))
	}

	if schema.Const != "constant" {
		t.Errorf("Expected Const 'constant', got %v", schema.Const)
	}

	if schema.Default != "default" {
		t.Errorf("Expected Default 'default', got %v", schema.Default)
	}

	if !schema.Nullable {
		t.Error("Expected Nullable to be true")
	}
}

func TestApplySchemaConstraints_IntegerType(t *testing.T) {
	setupMuxTestWithOpenAPI()

	schema := &openapi.Schema{
		Type: "integer",
	}

	param := &Parameter{
		Minimum:          10,
		Maximum:          100,
		ExclusiveMinimum: 5,
		ExclusiveMaximum: 105,
		MultipleOf:       5,
		Format:           "int64",
		Enum:             []any{10, 20, 30},
	}

	applySchemaConstraints(schema, param)

	if schema.Minimum == nil {
		t.Error("Expected Minimum to be set")
	} else if *schema.Minimum != 10 {
		t.Errorf("Expected Minimum 10, got %f", *schema.Minimum)
	}

	if schema.Maximum == nil {
		t.Error("Expected Maximum to be set")
	} else if *schema.Maximum != 100 {
		t.Errorf("Expected Maximum 100, got %f", *schema.Maximum)
	}

	if schema.ExclusiveMinimum == nil {
		t.Error("Expected ExclusiveMinimum to be set")
	} else if *schema.ExclusiveMinimum != 5 {
		t.Errorf("Expected ExclusiveMinimum 5, got %f", *schema.ExclusiveMinimum)
	}

	if schema.ExclusiveMaximum == nil {
		t.Error("Expected ExclusiveMaximum to be set")
	} else if *schema.ExclusiveMaximum != 105 {
		t.Errorf("Expected ExclusiveMaximum 105, got %f", *schema.ExclusiveMaximum)
	}

	if schema.MultipleOf == nil {
		t.Error("Expected MultipleOf to be set")
	} else if *schema.MultipleOf != 5 {
		t.Errorf("Expected MultipleOf 5, got %f", *schema.MultipleOf)
	}

	if schema.Format != "int64" {
		t.Errorf("Expected Format 'int64', got %q", schema.Format)
	}

	if len(schema.Enum) != 3 {
		t.Errorf("Expected 3 enum values, got %d", len(schema.Enum))
	}
}

func TestApplySchemaConstraints_NumberType(t *testing.T) {
	setupMuxTestWithOpenAPI()

	schema := &openapi.Schema{
		Type: "number",
	}

	param := &Parameter{
		Minimum:          0.5,
		Maximum:          99.5,
		ExclusiveMinimum: 0.1,
		ExclusiveMaximum: 100.0,
		MultipleOf:       0.25,
		Format:           "double",
	}

	applySchemaConstraints(schema, param)

	if schema.Minimum == nil {
		t.Error("Expected Minimum to be set")
	}

	if schema.Maximum == nil {
		t.Error("Expected Maximum to be set")
	}

	if schema.ExclusiveMinimum == nil {
		t.Error("Expected ExclusiveMinimum to be set")
	}

	if schema.ExclusiveMaximum == nil {
		t.Error("Expected ExclusiveMaximum to be set")
	}

	if schema.MultipleOf == nil {
		t.Error("Expected MultipleOf to be set")
	}

	if schema.Format != "double" {
		t.Errorf("Expected Format 'double', got %q", schema.Format)
	}
}

func TestApplySchemaConstraints_ArrayType(t *testing.T) {
	setupMuxTestWithOpenAPI()

	schema := &openapi.Schema{
		Type: "array",
	}

	param := &Parameter{
		MinItems:    1,
		MaxItems:    10,
		UniqueItems: true,
	}

	applySchemaConstraints(schema, param)

	if schema.MinItems == nil {
		t.Error("Expected MinItems to be set")
	} else if *schema.MinItems != 1 {
		t.Errorf("Expected MinItems 1, got %d", *schema.MinItems)
	}

	if schema.MaxItems == nil {
		t.Error("Expected MaxItems to be set")
	} else if *schema.MaxItems != 10 {
		t.Errorf("Expected MaxItems 10, got %d", *schema.MaxItems)
	}

	if !schema.UniqueItems {
		t.Error("Expected UniqueItems to be true")
	}
}

func TestApplySchemaConstraints_WithExamples(t *testing.T) {
	setupMuxTestWithOpenAPI()

	schema := &openapi.Schema{
		Type: "string",
	}

	param := &Parameter{
		Example: "example-value",
		Examples: map[string]Example{
			"ex1": {
				Summary:   "Example 1",
				DataValue: "value1",
			},
			"ex2": {
				Summary:   "Example 2",
				DataValue: "value2",
			},
		},
	}

	applySchemaConstraints(schema, param)

	if schema.Example != "example-value" {
		t.Errorf("Expected Example 'example-value', got %v", schema.Example)
	}

	if schema.Examples == nil {
		t.Fatal("Expected Examples to be non-nil")
	}

	if len(schema.Examples) != 2 {
		t.Errorf("Expected 2 examples, got %d", len(schema.Examples))
	}
}

// =============================================================================
// applyStringConstraints Tests
// =============================================================================

func TestApplyStringConstraints_AllConstraints(t *testing.T) {
	schema := &openapi.Schema{
		Type: "string",
	}

	param := &Parameter{
		MinLength: 3,
		MaxLength: 100,
		Pattern:   "^[a-z]+$",
	}

	applyStringConstraints(schema, param)

	if schema.MinLength == nil {
		t.Error("Expected MinLength to be set")
	} else if *schema.MinLength != 3 {
		t.Errorf("Expected MinLength 3, got %d", *schema.MinLength)
	}

	if schema.MaxLength == nil {
		t.Error("Expected MaxLength to be set")
	} else if *schema.MaxLength != 100 {
		t.Errorf("Expected MaxLength 100, got %d", *schema.MaxLength)
	}

	if schema.Pattern != "^[a-z]+$" {
		t.Errorf("Expected Pattern '^[a-z]+$', got %q", schema.Pattern)
	}
}

func TestApplyStringConstraints_ZeroValuesNotSet(t *testing.T) {
	schema := &openapi.Schema{
		Type: "string",
	}

	param := &Parameter{
		MinLength: 0,
		MaxLength: 0,
		Pattern:   "",
	}

	applyStringConstraints(schema, param)

	// Zero values should not create pointers
	if schema.MinLength != nil {
		t.Error("Expected MinLength to be nil for zero value")
	}

	if schema.MaxLength != nil {
		t.Error("Expected MaxLength to be nil for zero value")
	}

	if schema.Pattern != "" {
		t.Error("Expected Pattern to remain empty")
	}
}

// =============================================================================
// applyNumericConstraints Tests
// =============================================================================

func TestApplyNumericConstraints_AllConstraints(t *testing.T) {
	schema := &openapi.Schema{
		Type: "number",
	}

	param := &Parameter{
		Minimum:          10.5,
		Maximum:          99.9,
		ExclusiveMinimum: 10.0,
		ExclusiveMaximum: 100.0,
		MultipleOf:       0.5,
	}

	applyNumericConstraints(schema, param)

	if schema.Minimum == nil {
		t.Error("Expected Minimum to be set")
	} else if *schema.Minimum != 10.5 {
		t.Errorf("Expected Minimum 10.5, got %f", *schema.Minimum)
	}

	if schema.Maximum == nil {
		t.Error("Expected Maximum to be set")
	} else if *schema.Maximum != 99.9 {
		t.Errorf("Expected Maximum 99.9, got %f", *schema.Maximum)
	}

	if schema.ExclusiveMinimum == nil {
		t.Error("Expected ExclusiveMinimum to be set")
	} else if *schema.ExclusiveMinimum != 10.0 {
		t.Errorf("Expected ExclusiveMinimum 10.0, got %f", *schema.ExclusiveMinimum)
	}

	if schema.ExclusiveMaximum == nil {
		t.Error("Expected ExclusiveMaximum to be set")
	} else if *schema.ExclusiveMaximum != 100.0 {
		t.Errorf("Expected ExclusiveMaximum 100.0, got %f", *schema.ExclusiveMaximum)
	}

	if schema.MultipleOf == nil {
		t.Error("Expected MultipleOf to be set")
	} else if *schema.MultipleOf != 0.5 {
		t.Errorf("Expected MultipleOf 0.5, got %f", *schema.MultipleOf)
	}
}

func TestApplyNumericConstraints_ZeroValuesNotSet(t *testing.T) {
	schema := &openapi.Schema{
		Type: "integer",
	}

	param := &Parameter{
		Minimum:          0,
		Maximum:          0,
		ExclusiveMinimum: 0,
		ExclusiveMaximum: 0,
		MultipleOf:       0,
	}

	applyNumericConstraints(schema, param)

	// Zero values should not create pointers
	if schema.Minimum != nil {
		t.Error("Expected Minimum to be nil for zero value")
	}

	if schema.Maximum != nil {
		t.Error("Expected Maximum to be nil for zero value")
	}

	if schema.ExclusiveMinimum != nil {
		t.Error("Expected ExclusiveMinimum to be nil for zero value")
	}

	if schema.ExclusiveMaximum != nil {
		t.Error("Expected ExclusiveMaximum to be nil for zero value")
	}

	if schema.MultipleOf != nil {
		t.Error("Expected MultipleOf to be nil for zero value")
	}
}

// =============================================================================
// applyArrayConstraints Tests
// =============================================================================

func TestApplyArrayConstraints_AllConstraints(t *testing.T) {
	schema := &openapi.Schema{
		Type: "array",
	}

	param := &Parameter{
		MinItems:    1,
		MaxItems:    50,
		UniqueItems: true,
	}

	applyArrayConstraints(schema, param)

	if schema.MinItems == nil {
		t.Error("Expected MinItems to be set")
	} else if *schema.MinItems != 1 {
		t.Errorf("Expected MinItems 1, got %d", *schema.MinItems)
	}

	if schema.MaxItems == nil {
		t.Error("Expected MaxItems to be set")
	} else if *schema.MaxItems != 50 {
		t.Errorf("Expected MaxItems 50, got %d", *schema.MaxItems)
	}

	if !schema.UniqueItems {
		t.Error("Expected UniqueItems to be true")
	}
}

func TestApplyArrayConstraints_ZeroValuesNotSet(t *testing.T) {
	schema := &openapi.Schema{
		Type: "array",
	}

	param := &Parameter{
		MinItems:    0,
		MaxItems:    0,
		UniqueItems: false,
	}

	applyArrayConstraints(schema, param)

	// Zero values should not create pointers
	if schema.MinItems != nil {
		t.Error("Expected MinItems to be nil for zero value")
	}

	if schema.MaxItems != nil {
		t.Error("Expected MaxItems to be nil for zero value")
	}

	if schema.UniqueItems {
		t.Error("Expected UniqueItems to be false")
	}
}

// =============================================================================
// mapExample Tests
// =============================================================================

func TestMapExample_WithAllFields(t *testing.T) {
	input := &Example{
		Summary:         "Example summary",
		Description:     "Example description",
		DataValue:       map[string]any{"key": "value"},
		DefaultValue:    "default",
		SerializedValue: `{"key":"value"}`,
		ExternalValue:   "https://example.com/example.json",
	}

	result := mapExample(input)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Summary != "Example summary" {
		t.Errorf("Expected Summary 'Example summary', got %q", result.Summary)
	}

	if result.Description != "Example description" {
		t.Errorf("Expected Description 'Example description', got %q", result.Description)
	}

	if result.DataValue == nil {
		t.Error("Expected DataValue to be non-nil")
	}

	if result.DefaultValue != "default" {
		t.Errorf("Expected DefaultValue 'default', got %v", result.DefaultValue)
	}

	if result.SerializedValue != `{"key":"value"}` {
		t.Errorf("Expected SerializedValue '{\"key\":\"value\"}', got %v", result.SerializedValue)
	}

	if result.ExternalValue != "https://example.com/example.json" {
		t.Errorf("Expected ExternalValue 'https://example.com/example.json', got %q", result.ExternalValue)
	}
}

func TestMapExample_NilInput(t *testing.T) {
	result := mapExample(nil)

	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}

// =============================================================================
// mapExampleOrRefs Tests
// =============================================================================

func TestMapExampleOrRefs_WithMultipleExamples(t *testing.T) {
	input := map[string]Example{
		"example1": {
			Summary:     "First",
			DataValue:   "value1",
			Description: "First example",
		},
		"example2": {
			Summary:     "Second",
			DataValue:   "value2",
			Description: "Second example",
		},
	}

	result := mapExampleOrRefs(input)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 examples, got %d", len(result))
	}

	ex1, ok := result["example1"]
	if !ok {
		t.Fatal("Expected 'example1' to exist")
	}

	if ex1.Example == nil {
		t.Fatal("Expected Example to be non-nil")
	}

	if ex1.Example.Summary != "First" {
		t.Errorf("Expected Summary 'First', got %q", ex1.Example.Summary)
	}
}

func TestMapExampleOrRefs_NilInput(t *testing.T) {
	result := mapExampleOrRefs(nil)

	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}

// =============================================================================
// mapServer Tests
// =============================================================================

func TestMapServer_WithAllFields(t *testing.T) {
	input := &Server{
		URL:         "https://api.example.com",
		Name:        "Production",
		Description: "Production server",
		Variables: map[string]ServerVariable{
			"version": {
				Default:     "v1",
				Description: "API version",
				Enum:        []string{"v1", "v2"},
			},
		},
	}

	result := mapServer(input)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.URL != "https://api.example.com" {
		t.Errorf("Expected URL 'https://api.example.com', got %q", result.URL)
	}

	if result.Name != "Production" {
		t.Errorf("Expected Name 'Production', got %q", result.Name)
	}

	if result.Description != "Production server" {
		t.Errorf("Expected Description 'Production server', got %q", result.Description)
	}

	if result.Variables == nil {
		t.Fatal("Expected Variables to be non-nil")
	}

	if len(result.Variables) != 1 {
		t.Errorf("Expected 1 variable, got %d", len(result.Variables))
	}

	version, ok := result.Variables["version"]
	if !ok {
		t.Fatal("Expected 'version' variable to exist")
	}

	if version.Default != "v1" {
		t.Errorf("Expected Default 'v1', got %q", version.Default)
	}
}

func TestMapServer_NilInput(t *testing.T) {
	result := mapServer(nil)

	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}

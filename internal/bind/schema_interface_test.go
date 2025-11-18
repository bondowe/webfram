package bind

import (
	"testing"

	"github.com/bondowe/webfram/openapi"
)

type StructWithInterface struct {
	Data     any    `json:"data"               validate:"required"`
	Metadata any    `json:"metadata,omitempty"`
	Name     string `json:"name"`
}

func TestGenerateJSONSchema_InterfaceField(t *testing.T) {
	components := &openapi.Components{}
	var s StructWithInterface

	schemaOrRef := GenerateJSONSchema(s, components)
	if schemaOrRef == nil || schemaOrRef.Ref == "" {
		t.Fatalf("expected a reference schema for StructWithInterface, got %v", schemaOrRef)
	}

	// Get the component schema
	typeName := "bind.StructWithInterface"
	schema, ok := components.Schemas[typeName]
	if !ok {
		t.Fatalf("components does not contain schema for StructWithInterface")
	}

	// Verify Data field exists
	dataSchema, ok := schema.Properties["data"]
	if !ok {
		t.Fatalf("expected 'data' property in schema, but it was not found")
	}

	// Verify Data field has a schema (empty schema accepts any type)
	if dataSchema.Schema == nil {
		t.Fatalf("expected 'data' property to have a schema, got nil")
	}

	// Verify Data field is required
	hasData := false
	for _, req := range schema.Required {
		if req == "data" {
			hasData = true
			break
		}
	}
	if !hasData {
		t.Fatalf("expected 'data' to be in required fields, got %v", schema.Required)
	}

	// Verify Metadata field exists
	metadataSchema, ok := schema.Properties["metadata"]
	if !ok {
		t.Fatalf("expected 'metadata' property in schema, but it was not found")
	}

	// Verify Metadata field has a schema (empty schema accepts any type)
	if metadataSchema.Schema == nil {
		t.Fatalf("expected 'metadata' property to have a schema, got nil")
	}

	// Verify Metadata is not required (has omitempty)
	for _, req := range schema.Required {
		if req == "metadata" {
			t.Fatalf("'metadata' should not be in required fields since it has omitempty")
		}
	}

	// Verify Name field exists and is a string
	nameSchema, ok := schema.Properties["name"]
	if !ok {
		t.Fatalf("expected 'name' property in schema")
	}
	if nameSchema.Schema == nil || nameSchema.Schema.Type != "string" {
		t.Fatalf("expected 'name' to be type string, got %v", nameSchema.Schema)
	}
}

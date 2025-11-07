package bind

import (
	"reflect"
	"testing"
	"time"

	"github.com/bondowe/webfram/openapi"
	"github.com/google/uuid"
)

type Address struct {
	Street string `json:"street" validate:"required"`
	Number int    `json:"number,omitempty"`
}

type Person struct {
	CreatedAt  time.Time `json:"created_at" format:"2006-01-02"`
	NestedPtr  *Address  `json:"nested_ptr,omitempty"`
	PtrField   *string   `json:"ptr_field,omitempty" validate:"minlength=1"`
	Name       string    `json:"name" validate:"required,minlength=2,maxlength=50,regexp=^[A-Za-z]+$,enum=John|Jane"`
	Ignored    string    `json:"-"`
	Addr       Address   `json:"address"`
	Tags       []string  `json:"tags" validate:"minItems=1,maxItems=5,uniqueItems"`
	IntSlice   []int     `json:"ints"`
	FloatSlice []float64 `json:"floats"`
	BoolSlice  []bool    `json:"bools"`
	Score      float64   `json:"score" validate:"min=0.0,max=100.0"`
	Age        int       `json:"age" validate:"min=0,max=120"`
	ID         uuid.UUID `json:"id"`
	Active     bool      `json:"active"`
}

func TestGenerateJSONSchema_Struct(t *testing.T) {
	components := &openapi.Components{}
	var p Person

	schemaOrRef := GenerateJSONSchema(p, components)
	if schemaOrRef == nil || schemaOrRef.Ref == "" {
		t.Fatalf("expected a reference schema for Person, got %v", schemaOrRef)
	}

	expectedRef := "#/components/schemas/" + reflect.TypeOf(p).String()
	if schemaOrRef.Ref != expectedRef {
		t.Fatalf("expected ref %s, got %s", expectedRef, schemaOrRef.Ref)
	}

	// Ensure the component schema exists
	personSchema, ok := components.Schemas[reflect.TypeOf(p).String()]
	if !ok {
		t.Fatalf("components does not contain schema for Person")
	}

	// Check presence/absence of properties
	props := personSchema.Properties
	if _, ok := props["name"]; !ok {
		t.Fatalf("expected property 'name' in Person schema")
	}
	if _, ok := props["age"]; !ok {
		t.Fatalf("expected property 'age' in Person schema")
	}
	if _, ok := props["address"]; !ok {
		t.Fatalf("expected property 'address' in Person schema")
	}
	if _, ok := props["ptr_field"]; !ok {
		t.Fatalf("expected property 'ptr_field' in Person schema")
	}
	if _, ok := props["ignored"]; ok {
		t.Fatalf("property with json:\"-\" should be skipped")
	}

	// Validate 'name' schema rules: enum, pattern, minlength/maxlength and required
	nameSchemaOrRef := props["name"]
	if nameSchemaOrRef.Schema == nil {
		t.Fatalf("expected inline schema for name")
	}
	nameSchema := nameSchemaOrRef.Schema

	// Enum check
	if len(nameSchema.Enum) != 2 {
		t.Fatalf("expected enum of length 2 for name, got %v", nameSchema.Enum)
	}
	foundJohn := false
	foundJane := false
	for _, v := range nameSchema.Enum {
		if v == "John" {
			foundJohn = true
		}
		if v == "Jane" {
			foundJane = true
		}
	}
	if !foundJohn || !foundJane {
		t.Fatalf("enum values for name unexpected: %v", nameSchema.Enum)
	}

	// Pattern check
	if nameSchema.Pattern != "^[A-Za-z]+$" {
		t.Fatalf("expected pattern '^[A-Za-z]+$', got %s", nameSchema.Pattern)
	}

	// Min/Max length checks
	if nameSchema.MinLength == nil || *nameSchema.MinLength != 2 {
		t.Fatalf("expected MinLength=2 for name, got %v", nameSchema.MinLength)
	}
	if nameSchema.MaxLength == nil || *nameSchema.MaxLength != 50 {
		t.Fatalf("expected MaxLength=50 for name, got %v", nameSchema.MaxLength)
	}

	// Required check
	foundRequired := false
	for _, r := range personSchema.Required {
		if r == "name" {
			foundRequired = true
			break
		}
	}
	if !foundRequired {
		t.Fatalf("expected 'name' to be required")
	}

	// CreatedAt format should map to "date"
	createdSchemaOrRef := props["created_at"]
	if createdSchemaOrRef.Schema == nil || createdSchemaOrRef.Format != "date" {
		t.Fatalf("expected created_at format 'date', got %v", createdSchemaOrRef.Schema)
	}

	// Address should be a $ref to components
	addressSchemaOrRef := props["address"]
	if addressSchemaOrRef.Ref == "" {
		t.Fatalf("expected address to be a component reference, got %v", addressSchemaOrRef)
	}
	// Ensure address component exists
	if _, ok := components.Schemas[reflect.TypeOf(Address{}).String()]; !ok {
		t.Fatalf("expected component schema for Address to exist")
	}

	// Tags slice constraints
	tagsSchemaOrRef := props["tags"]
	if tagsSchemaOrRef.Schema == nil || tagsSchemaOrRef.Type != "array" {
		t.Fatalf("expected tags to be an array")
	}
	if tagsSchemaOrRef.MinItems == nil || *tagsSchemaOrRef.MinItems != 1 {
		t.Fatalf("expected tags minItems=1, got %v", tagsSchemaOrRef.MinItems)
	}
	if tagsSchemaOrRef.MaxItems == nil || *tagsSchemaOrRef.MaxItems != 5 {
		t.Fatalf("expected tags maxItems=5, got %v", tagsSchemaOrRef.MaxItems)
	}
	if !tagsSchemaOrRef.UniqueItems {
		t.Fatalf("expected tags uniqueItems=true")
	}

	// Ints slice should have item type integer
	intsSchemaOrRef := props["ints"]
	if intsSchemaOrRef.Schema == nil || intsSchemaOrRef.Type != "array" {
		t.Fatalf("expected ints to be an array")
	}
	if intsSchemaOrRef.Items == nil || intsSchemaOrRef.Items.Schema == nil ||
		intsSchemaOrRef.Items.Type != "integer" {
		t.Fatalf("expected ints items type integer")
	}

	// Floats slice item type number
	floatsSchemaOrRef := props["floats"]
	if floatsSchemaOrRef.Schema == nil || floatsSchemaOrRef.Items == nil ||
		floatsSchemaOrRef.Items.Schema == nil || floatsSchemaOrRef.Items.Type != "number" {
		t.Fatalf("expected floats items type number")
	}

	// Bools slice item type boolean
	boolsSchemaOrRef := props["bools"]
	if boolsSchemaOrRef.Schema == nil || boolsSchemaOrRef.Items == nil ||
		boolsSchemaOrRef.Items.Schema == nil || boolsSchemaOrRef.Items.Type != "boolean" {
		t.Fatalf("expected bools items type boolean")
	}
}

func TestGenerateJSONSchema_TopLevelSlice(t *testing.T) {
	components := &openapi.Components{}
	personSlice := []Person{}

	schemaOrRef := GenerateJSONSchema(personSlice, components)
	if schemaOrRef == nil || schemaOrRef.Schema == nil {
		t.Fatalf("expected array schema for top-level slice")
	}
	if schemaOrRef.Type != "array" {
		t.Fatalf("expected schema type array, got %s", schemaOrRef.Type)
	}

	// Items should reference Person component
	if schemaOrRef.Items == nil || schemaOrRef.Items.Ref == "" {
		t.Fatalf("expected items to be a reference to Person component, got %v", schemaOrRef.Items)
	}

	expectedRef := "#/components/schemas/" + reflect.TypeOf(Person{}).String()
	if schemaOrRef.Items.Ref != expectedRef {
		t.Fatalf("expected items ref %s, got %s", expectedRef, schemaOrRef.Items.Ref)
	}

	// Ensure Person component exists
	if _, ok := components.Schemas[reflect.TypeOf(Person{}).String()]; !ok {
		t.Fatalf("expected person component to be present in components")
	}
}

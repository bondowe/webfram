package bind

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bondowe/webfram/openapi"
	"github.com/google/uuid"
)

type Address struct {
	Street string `json:"street"           validate:"required"`
	Number int    `json:"number,omitempty"`
}

type Person struct {
	CreatedAt  time.Time `json:"created_at"           format:"2006-01-02"`
	NestedPtr  *Address  `json:"nested_ptr,omitempty"`
	PtrField   *string   `json:"ptr_field,omitempty"                      validate:"minlength=1"`
	Name       string    `json:"name"                                     validate:"required,minlength=2,maxlength=50,regexp=^[A-Za-z]+$,enum=John|Jane"`
	Ignored    string    `json:"-"`
	Addr       Address   `json:"address"`
	Tags       []string  `json:"tags"                                     validate:"minItems=1,maxItems=5,uniqueItems"`
	IntSlice   []int     `json:"ints"`
	FloatSlice []float64 `json:"floats"`
	BoolSlice  []bool    `json:"bools"`
	Score      float64   `json:"score"                                    validate:"min=0.0,max=100.0"`
	Age        int       `json:"age"                                      validate:"min=0,max=120"`
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
	if _, nameOk := props["name"]; !nameOk {
		t.Fatalf("expected property 'name' in Person schema")
	}
	if _, ageOk := props["age"]; !ageOk {
		t.Fatalf("expected property 'age' in Person schema")
	}
	if _, addressOk := props["address"]; !addressOk {
		t.Fatalf("expected property 'address' in Person schema")
	}
	if _, ptrFieldOk := props["ptr_field"]; !ptrFieldOk {
		t.Fatalf("expected property 'ptr_field' in Person schema")
	}
	if _, ignoredOk := props["ignored"]; ignoredOk {
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
	if _, addressCompOk := components.Schemas[reflect.TypeOf(Address{}).String()]; !addressCompOk {
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

func TestGenerateJSONSchema_UnsignedIntegers(t *testing.T) {
	type UintFields struct {
		DefaultUint uint     `json:"default_uint"`
		TinyUint    uint8    `json:"tiny_uint"`
		SmallUint   uint16   `json:"small_uint"`
		MediumUint  uint32   `json:"medium_uint"`
		LargeUint   uint64   `json:"large_uint"`
		UintSlice   []uint   `json:"uint_slice"`
		Uint8Slice  []uint8  `json:"uint8_slice"`
		Uint16Slice []uint16 `json:"uint16_slice"`
		Uint32Slice []uint32 `json:"uint32_slice"`
		Uint64Slice []uint64 `json:"uint64_slice"`
	}

	components := &openapi.Components{}
	var uf UintFields

	schemaOrRef := GenerateJSONSchema(uf, components)
	if schemaOrRef == nil || schemaOrRef.Ref == "" {
		t.Fatalf("expected a reference schema for UintFields, got %v", schemaOrRef)
	}

	// Get the component schema
	schema, ok := components.Schemas[reflect.TypeOf(uf).String()]
	if !ok {
		t.Fatalf("components does not contain schema for UintFields")
	}

	props := schema.Properties

	// Test individual unsigned integer fields
	tests := []struct {
		name           string
		expectedFormat string
	}{
		{"default_uint", "uint64"}, // uint maps to uint64 format
		{"tiny_uint", "uint8"},
		{"small_uint", "uint16"},
		{"medium_uint", "uint32"},
		{"large_uint", "uint64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propSchema := props[tt.name]
			if propSchema.Schema == nil {
				t.Fatalf("expected inline schema for %s", tt.name)
			}
			if propSchema.Type != "integer" {
				t.Fatalf("expected type 'integer' for %s, got %s", tt.name, propSchema.Type)
			}
			if propSchema.Format != tt.expectedFormat {
				t.Fatalf("expected format '%s' for %s, got %s", tt.expectedFormat, tt.name, propSchema.Format)
			}
		})
	}

	// Test unsigned integer slice fields
	sliceTests := []struct {
		name           string
		expectedFormat string
	}{
		{"uint_slice", "uint64"},
		{"uint8_slice", "uint8"},
		{"uint16_slice", "uint16"},
		{"uint32_slice", "uint32"},
		{"uint64_slice", "uint64"},
	}

	for _, tt := range sliceTests {
		t.Run(tt.name, func(t *testing.T) {
			propSchema := props[tt.name]
			if propSchema.Schema == nil {
				t.Fatalf("expected inline schema for %s", tt.name)
			}
			if propSchema.Type != "array" {
				t.Fatalf("expected type 'array' for %s, got %s", tt.name, propSchema.Type)
			}
			if propSchema.Items == nil || propSchema.Items.Schema == nil {
				t.Fatalf("expected items schema for %s", tt.name)
			}
			if propSchema.Items.Type != "integer" {
				t.Fatalf("expected items type 'integer' for %s, got %s", tt.name, propSchema.Items.Type)
			}
			if propSchema.Items.Format != tt.expectedFormat {
				t.Fatalf(
					"expected items format '%s' for %s, got %s",
					tt.expectedFormat,
					tt.name,
					propSchema.Items.Format,
				)
			}
		})
	}
}

// TestGenerateXMLSchema_BasicTypes tests XML schema generation for basic types.
func TestGenerateXMLSchema_BasicTypes(t *testing.T) {
	type XMLPerson struct {
		ID       int       `xml:"id,attr"`
		Name     string    `xml:"name"`
		Age      int       `xml:"age"`
		Email    string    `xml:"email,attr"`
		Active   bool      `xml:"active"`
		Score    float64   `xml:"score"`
		Created  time.Time `xml:"created"`
		UniqueID uuid.UUID `xml:"unique_id"`
	}

	components := &openapi.Components{}
	var p XMLPerson

	schemaOrRef := GenerateXMLSchema(p, "", components)
	if schemaOrRef == nil || schemaOrRef.Ref == "" {
		t.Fatalf("expected a reference schema for XMLPerson, got %v", schemaOrRef)
	}

	// Check the component schema exists (should have .XML suffix)
	personSchema, ok := components.Schemas[reflect.TypeOf(p).String()+".XML"]
	if !ok {
		t.Fatalf("components does not contain schema for XMLPerson")
	}

	if personSchema.Type != "object" {
		t.Fatalf("expected type 'object', got %s", personSchema.Type)
	}

	// Test ID field (attribute)
	idProp, ok := personSchema.Properties["id"]
	if !ok {
		t.Fatalf("expected 'id' property")
	}
	if idProp.Schema == nil {
		t.Fatalf("expected inline schema for id")
	}
	if idProp.Schema.Type != "integer" {
		t.Fatalf("expected type 'integer' for id, got %s", idProp.Schema.Type)
	}
	if idProp.Schema.XML == nil {
		t.Fatalf("expected XML metadata for id")
	}
	if idProp.Schema.XML.NodeType != "attribute" {
		t.Fatalf("expected nodeType 'attribute' for id, got %s", idProp.Schema.XML.NodeType)
	}
	if idProp.Schema.XML.Name != "id" {
		t.Fatalf("expected XML name 'id', got %s", idProp.Schema.XML.Name)
	}

	// Test Name field (element)
	nameProp, ok := personSchema.Properties["name"]
	if !ok {
		t.Fatalf("expected 'name' property")
	}
	if nameProp.Schema == nil {
		t.Fatalf("expected inline schema for name")
	}
	if nameProp.Schema.Type != "string" {
		t.Fatalf("expected type 'string' for name, got %s", nameProp.Schema.Type)
	}
	if nameProp.Schema.XML == nil {
		t.Fatalf("expected XML metadata for name")
	}
	if nameProp.Schema.XML.NodeType != "element" {
		t.Fatalf("expected nodeType 'element' for name, got %s", nameProp.Schema.XML.NodeType)
	}

	// Test Email field (attribute)
	emailProp, ok := personSchema.Properties["email"]
	if !ok {
		t.Fatalf("expected 'email' property")
	}
	if emailProp.Schema == nil {
		t.Fatalf("expected inline schema for email")
	}
	if emailProp.Schema.XML == nil {
		t.Fatalf("expected XML metadata for email")
	}
	if emailProp.Schema.XML.NodeType != "attribute" {
		t.Fatalf("expected nodeType 'attribute' for email, got %s", emailProp.Schema.XML.NodeType)
	}
}

// TestGenerateXMLSchema_Arrays tests XML schema generation for array types.
func TestGenerateXMLSchema_Arrays(t *testing.T) {
	type XMLBook struct {
		Title   string   `xml:"title"`
		Authors []string `xml:"author"`
		Ratings []int    `xml:"rating"`
	}

	components := &openapi.Components{}
	var book XMLBook

	schemaOrRef := GenerateXMLSchema(book, "", components)
	if schemaOrRef == nil || schemaOrRef.Ref == "" {
		t.Fatalf("expected a reference schema for XMLBook, got %v", schemaOrRef)
	}

	bookSchema, ok := components.Schemas[reflect.TypeOf(book).String()+".XML"]
	if !ok {
		t.Fatalf("components does not contain schema for XMLBook")
	}

	// Test Authors array
	authorsProp, ok := bookSchema.Properties["author"]
	if !ok {
		t.Fatalf("expected 'author' property")
	}
	if authorsProp.Schema == nil {
		t.Fatalf("expected inline schema for authors")
	}
	if authorsProp.Schema.Type != "array" {
		t.Fatalf("expected type 'array' for authors, got %s", authorsProp.Schema.Type)
	}
	if authorsProp.Schema.Items == nil || authorsProp.Schema.Items.Schema == nil {
		t.Fatalf("expected items schema for authors")
	}
	if authorsProp.Schema.Items.Schema.Type != "string" {
		t.Fatalf("expected items type 'string' for authors, got %s", authorsProp.Schema.Items.Schema.Type)
	}
	if authorsProp.Schema.Items.Schema.XML == nil {
		t.Fatalf("expected XML metadata for author items")
	}
	if authorsProp.Schema.Items.Schema.XML.Name != "author" {
		t.Fatalf("expected XML name 'author' for items, got %s", authorsProp.Schema.Items.Schema.XML.Name)
	}

	// Test Ratings array
	ratingsProp, ok := bookSchema.Properties["rating"]
	if !ok {
		t.Fatalf("expected 'rating' property")
	}
	if ratingsProp.Schema == nil {
		t.Fatalf("expected inline schema for ratings")
	}
	if ratingsProp.Schema.Type != "array" {
		t.Fatalf("expected type 'array' for ratings, got %s", ratingsProp.Schema.Type)
	}
	if ratingsProp.Schema.Items == nil || ratingsProp.Schema.Items.Schema == nil {
		t.Fatalf("expected items schema for ratings")
	}
	if ratingsProp.Schema.Items.Schema.Type != "integer" {
		t.Fatalf("expected items type 'integer' for ratings, got %s", ratingsProp.Schema.Items.Schema.Type)
	}
}

// TestGenerateXMLSchema_NestedStructs tests XML schema generation for nested structures.
func TestGenerateXMLSchema_NestedStructs(t *testing.T) {
	type XMLAddress struct {
		Street string `xml:"street"`
		City   string `xml:"city"`
		Zip    string `xml:"zip,attr"`
	}

	type XMLCompany struct {
		Name    string     `xml:"name,attr"`
		Address XMLAddress `xml:"address"`
	}

	components := &openapi.Components{}
	var company XMLCompany

	schemaOrRef := GenerateXMLSchema(company, "", components)
	if schemaOrRef == nil || schemaOrRef.Ref == "" {
		t.Fatalf("expected a reference schema for XMLCompany, got %v", schemaOrRef)
	}

	// Check XMLCompany schema
	companySchema, ok := components.Schemas[reflect.TypeOf(company).String()+".XML"]
	if !ok {
		t.Fatalf("components does not contain schema for XMLCompany")
	}

	// Test Name attribute
	nameProp, ok := companySchema.Properties["name"]
	if !ok {
		t.Fatalf("expected 'name' property")
	}
	if nameProp.Schema == nil {
		t.Fatalf("expected inline schema for name")
	}
	if nameProp.Schema.XML == nil || nameProp.Schema.XML.NodeType != "attribute" {
		t.Fatalf("expected name to be an attribute")
	}

	// Test nested Address
	addressProp, ok := companySchema.Properties["address"]
	if !ok {
		t.Fatalf("expected 'address' property")
	}
	if addressProp.Ref == "" {
		t.Fatalf("expected reference to XMLAddress schema, got inline schema")
	}

	// Check XMLAddress schema exists (should have .XML suffix)
	addressSchema, ok := components.Schemas[reflect.TypeOf(XMLAddress{}).String()+".XML"]
	if !ok {
		t.Fatalf("components does not contain schema for XMLAddress")
	}

	// Test Zip attribute in nested struct
	zipProp, ok := addressSchema.Properties["zip"]
	if !ok {
		t.Fatalf("expected 'zip' property in XMLAddress")
	}
	if zipProp.Schema == nil {
		t.Fatalf("expected inline schema for zip")
	}
	if zipProp.Schema.XML == nil || zipProp.Schema.XML.NodeType != "attribute" {
		t.Fatalf("expected zip to be an attribute")
	}
}

// TestGenerateXMLSchema_UnsignedIntegers tests XML schema generation with unsigned integer types.
func TestGenerateXMLSchema_UnsignedIntegers(t *testing.T) {
	type XMLMetrics struct {
		Count8  uint8  `xml:"count8"`
		Count16 uint16 `xml:"count16"`
		Count32 uint32 `xml:"count32"`
		Count64 uint64 `xml:"count64"`
		Count   uint   `xml:"count"`
	}

	components := &openapi.Components{}
	var metrics XMLMetrics

	schemaOrRef := GenerateXMLSchema(metrics, "", components)
	if schemaOrRef == nil || schemaOrRef.Ref == "" {
		t.Fatalf("expected a reference schema for XMLMetrics, got %v", schemaOrRef)
	}

	metricsSchema, ok := components.Schemas[reflect.TypeOf(metrics).String()+".XML"]
	if !ok {
		t.Fatalf("components does not contain schema for XMLMetrics")
	}

	tests := []struct {
		name           string
		expectedFormat string
	}{
		{"count8", "uint8"},
		{"count16", "uint16"},
		{"count32", "uint32"},
		{"count64", "uint64"},
		{"count", "uint64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prop, propOk := metricsSchema.Properties[tt.name]
			if !propOk {
				t.Fatalf("expected '%s' property", tt.name)
			}
			if prop.Schema == nil {
				t.Fatalf("expected inline schema for %s", tt.name)
			}
			if prop.Schema.Type != "integer" {
				t.Fatalf("expected type 'integer' for %s, got %s", tt.name, prop.Schema.Type)
			}
			if prop.Schema.Format != tt.expectedFormat {
				t.Fatalf("expected format '%s' for %s, got %s", tt.expectedFormat, tt.name, prop.Schema.Format)
			}
			if prop.Schema.XML == nil {
				t.Fatalf("expected XML metadata for %s", tt.name)
			}
			if prop.Schema.XML.Name != tt.name {
				t.Fatalf("expected XML name '%s', got %s", tt.name, prop.Schema.XML.Name)
			}
		})
	}
}

// TestGenerateXMLSchema_SliceExamples tests that slice XML examples are properly wrapped with xmlRootName.
func TestGenerateXMLSchema_SliceExamples(t *testing.T) {
	type XMLUser struct {
		Name  string `xml:"name"`
		Email string `xml:"email"`
	}

	components := &openapi.Components{}
	var users []XMLUser

	schemaOrRef := GenerateXMLSchema(users, "users", components)
	if schemaOrRef == nil || schemaOrRef.Schema == nil {
		t.Fatalf("expected inline schema for []XMLUser, got %v", schemaOrRef)
	}

	usersSchema := schemaOrRef.Schema

	// Check that the schema has an example
	if usersSchema.Example == nil {
		t.Fatalf("expected example for slice schema")
	}

	exampleStr, isString := usersSchema.Example.(string)
	if !isString {
		t.Fatalf("expected example to be a string, got %T", usersSchema.Example)
	}

	// The example should be wrapped with the xmlRootName "users"
	if !strings.Contains(exampleStr, "<users>") || !strings.Contains(exampleStr, "</users>") {
		t.Fatalf("expected example to be wrapped with <users> root element, got: %s", exampleStr)
	}

	// Should contain user elements inside
	if !strings.Contains(exampleStr, "<XMLUser>") || !strings.Contains(exampleStr, "</XMLUser>") {
		t.Fatalf("expected example to contain XMLUser elements, got: %s", exampleStr)
	}
}

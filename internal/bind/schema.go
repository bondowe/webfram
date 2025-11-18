package bind

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/bondowe/webfram/openapi"
	"github.com/google/uuid"
)

const dateTimeFormat = "date-time"
const xmlNodeTypeElement = "element"
const xmlNodeTypeAttribute = "attribute"

// generateMockData creates mock data for the given type for use in examples.
func generateMockData(typ reflect.Type) any {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// Handle special types first
	if typ == reflect.TypeOf(time.Time{}) {
		return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	}
	if typ == reflect.TypeOf(uuid.UUID{}) {
		return uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	}

	switch typ.Kind() {
	case reflect.Struct:
		return generateMockStruct(typ)
	case reflect.Slice:
		return generateMockSlice(typ)
	case reflect.String:
		return "example"
	case reflect.Int:
		return int(42) //nolint:mnd
	case reflect.Int8:
		return int8(42) //nolint:mnd
	case reflect.Int16:
		return int16(42) //nolint:mnd
	case reflect.Int32:
		return int32(42) //nolint:mnd
	case reflect.Int64:
		return int64(42) //nolint:mnd
	case reflect.Uint:
		return uint(42) //nolint:mnd
	case reflect.Uint8:
		return uint8(42) //nolint:mnd
	case reflect.Uint16:
		return uint16(42) //nolint:mnd
	case reflect.Uint32:
		return uint32(42) //nolint:mnd
	case reflect.Uint64:
		return uint64(42) //nolint:mnd
	case reflect.Float32:
		return float32(3.14) //nolint:mnd
	case reflect.Float64:
		return 3.14 //nolint:mnd
	case reflect.Bool:
		return true
	default:
		return nil
	}
}

func generateMockStruct(typ reflect.Type) any {
	mockValue := reflect.New(typ).Elem()

	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Tag.Get("xml") == "-" {
			continue
		}

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		mockData := generateMockData(fieldType)
		if mockData != nil {
			mockValue.Field(i).Set(reflect.ValueOf(mockData))
		}
	}

	return mockValue.Interface()
}

func generateMockSlice(typ reflect.Type) any {
	elemType := typ.Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	mockElem := generateMockData(elemType)
	if mockElem == nil {
		return nil
	}

	slice := reflect.MakeSlice(typ, 1, 1)
	slice.Index(0).Set(reflect.ValueOf(mockElem))
	return slice.Interface()
}

// generateXMLExample generates an XML example string for the given type.
func generateXMLExample(t any, xmlRootName string) string {
	mockData := generateMockData(reflect.TypeOf(t))
	if mockData == nil {
		return ""
	}

	// For slices, wrap with xmlRootName if provided
	if reflect.TypeOf(t).Kind() == reflect.Slice { //nolint:nestif
		data, err := xml.MarshalIndent(mockData, "", "  ")
		if err != nil {
			return ""
		}
		xmlStr := string(data)

		// If xmlRootName is provided, wrap the slice elements
		if xmlRootName != "" {
			// Indent the existing XML content
			lines := strings.Split(xmlStr, "\n")
			indentedContent := make([]string, len(lines))
			for i, line := range lines {
				if strings.TrimSpace(line) != "" {
					indentedContent[i] = "  " + line
				} else {
					indentedContent[i] = line
				}
			}
			xmlStr = strings.Join(indentedContent, "\n")
			xmlStr = "<" + xmlRootName + ">\n" + xmlStr + "\n</" + xmlRootName + ">"
		}

		return xmlStr
	}

	// For structs, marshal with root name if provided
	if xmlRootName != "" {
		data, err := xml.MarshalIndent(mockData, "", "  ")
		if err != nil {
			return ""
		}
		// Replace the default root tag with the provided name
		xmlStr := string(data)
		typ := reflect.TypeOf(t)
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}
		defaultRoot := "<" + typ.Name() + ">"
		customRoot := "<" + xmlRootName + ">"
		xmlStr = strings.Replace(xmlStr, defaultRoot, customRoot, 1)
		closeDefault := "</" + typ.Name() + ">"
		closeCustom := "</" + xmlRootName + ">"
		xmlStr = strings.Replace(xmlStr, closeDefault, closeCustom, 1)
		return xmlStr
	}

	// Default marshaling
	data, err := xml.MarshalIndent(mockData, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

// registerStructSchema adds a struct type to components if not already registered.
func registerStructSchema(typName string, typ reflect.Type, components *openapi.Components) {
	if _, ok := components.Schemas[typName]; ok {
		return
	}

	structSchema := &openapi.Schema{
		Type:       "object",
		Title:      typName,
		Properties: make(map[string]openapi.SchemaOrRef),
		Required:   []string{},
	}

	// Add to components first to handle circular references
	components.Schemas[typName] = *structSchema

	// Generate properties
	generateSchemaForStruct(typ, structSchema, components)

	// Update the schema in components with generated properties
	components.Schemas[typName] = *structSchema
}

// GenerateJSONSchema generates an OpenAPI JSON Schema for the given type.
// It analyzes struct fields, validation tags, and type information to produce
// a complete schema with properties, types, formats, and validation constraints.
// The components parameter is used to register reusable schema definitions.
// Returns a SchemaOrRef that can be used in OpenAPI documentation.
func GenerateJSONSchema(t any, components *openapi.Components) *openapi.SchemaOrRef {
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if components.Schemas == nil {
		components.Schemas = make(map[string]openapi.Schema)
	}

	var schemaOrRef *openapi.SchemaOrRef

	switch typ.Kind() {
	case reflect.Struct:
		typName := typ.String()
		// Check if schema already exists in components
		if _, ok := components.Schemas[typName]; !ok {
			// Create the schema for the struct and add it to components
			structSchema := &openapi.Schema{
				Type:       "object",
				Title:      typName,
				Properties: make(map[string]openapi.SchemaOrRef),
				Required:   []string{},
			}

			// Add to components first to handle circular references
			components.Schemas[typName] = *structSchema

			// Generate properties
			generateSchemaForStruct(typ, structSchema, components)

			// Update the schema in components with generated properties
			components.Schemas[typName] = *structSchema
		}

		// Return a schema that references the component
		schemaOrRef = &openapi.SchemaOrRef{
			Ref: fmt.Sprintf("#/components/schemas/%s", typName),
		}
	case reflect.Slice:
		schemaOrRef = &openapi.SchemaOrRef{
			Schema: &openapi.Schema{
				Type:  "array",
				Items: generateSchemaForSliceElement(&reflect.StructField{Type: typ}, components),
			},
		}
	default:
		// For other types, create a simple schema
		schemaOrRef = generateSchemaForField(&reflect.StructField{Type: typ}, components)
	}

	return schemaOrRef
}

func GenerateXMLSchema(t any, xmlRootName string, components *openapi.Components) *openapi.SchemaOrRef {
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if components.Schemas == nil {
		components.Schemas = make(map[string]openapi.Schema)
	}

	var schemaOrRef *openapi.SchemaOrRef

	switch typ.Kind() {
	case reflect.Struct:
		typName := typ.String() + ".XML"
		// Check if schema already exists in components
		if _, ok := components.Schemas[typName]; !ok {
			// Create the schema for the struct and add it to components
			structSchema := &openapi.Schema{
				Type:       "object",
				Title:      typName,
				Properties: make(map[string]openapi.SchemaOrRef),
				Required:   []string{},
			}

			// Add to components first to handle circular references
			components.Schemas[typName] = *structSchema

			// Generate properties with XML metadata
			generateXMLSchemaForStruct(typ, structSchema, components)

			// Generate and set XML example
			structSchema.Example = generateXMLExample(t, xmlRootName)

			// Update the schema in components with generated properties
			components.Schemas[typName] = *structSchema
		}

		// Return a schema that references the component
		schemaOrRef = &openapi.SchemaOrRef{
			Ref: fmt.Sprintf("#/components/schemas/%s", typName),
		}
	case reflect.Slice:
		// Per OpenAPI 3.2.0 XML spec section 4.26.2.1:
		// Arrays default to nodeType: "none", but for a wrapped array (with a root element),
		// we need nodeType: "element" on the array schema itself
		itemSchema := generateXMLSchemaForSliceElement(&reflect.StructField{Type: typ}, components)

		// Get the element type to determine the wrapper element name
		elemType := typ.Elem()
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}

		// Use XMLRootName if provided, otherwise use lowercase type name as default
		wrapperName := xmlRootName
		if wrapperName == "" {
			wrapperName = "items"
			if elemType.Kind() == reflect.Struct {
				wrapperName = strings.ToLower(elemType.Name()) + "s" // pluralize
			}
		}

		arraySchema := &openapi.Schema{
			Type: "array",
			XML: &openapi.XML{
				NodeType: xmlNodeTypeElement,
				Name:     wrapperName,
			},
			Items: itemSchema,
		}

		// Generate and set XML example for the array
		arraySchema.Example = generateXMLExample(t, xmlRootName)

		schemaOrRef = &openapi.SchemaOrRef{
			Schema: arraySchema,
		}
	default:
		// For other types, create a simple schema
		schemaOrRef = generateXMLSchemaForField(&reflect.StructField{Type: typ}, components)
		// Set example for primitive types
		if schemaOrRef.Schema != nil {
			schemaOrRef.Schema.Example = generateMockData(typ)
		}
	}

	return schemaOrRef
}

func generateXMLSchemaForStruct(typ reflect.Type, schema *openapi.Schema, components *openapi.Components) {
	for i := range typ.NumField() {
		field := typ.Field(i)

		// Get the XML tag to use as element/attribute name
		xmlTag := field.Tag.Get("xml")
		propertyName := field.Name

		// Skip fields with xml:"-"
		if xmlTag == "-" {
			continue
		}

		// Parse XML tag
		xmlNodeType := xmlNodeTypeElement // default
		xmlName := ""
		xmlNamespace := ""
		xmlPrefix := ""

		if xmlTag != "" {
			parts := strings.Split(xmlTag, ",")
			xmlName = parts[0]

			// Check for attribute or other special directives
			for _, part := range parts[1:] {
				if strings.TrimSpace(part) == "attr" {
					xmlNodeType = xmlNodeTypeAttribute
				}
			}
		}

		// Fallback to field name if no XML name specified
		if xmlName == "" {
			xmlName = propertyName
		}

		// Create schema for this field
		fieldSchema := generateXMLSchemaForField(&field, components)

		// Add XML metadata to the field schema if we have a Schema (not a Ref)
		if fieldSchema.Schema != nil {
			if fieldSchema.Schema.XML == nil {
				fieldSchema.Schema.XML = &openapi.XML{}
			}
			fieldSchema.Schema.XML.NodeType = xmlNodeType
			fieldSchema.Schema.XML.Name = xmlName
			fieldSchema.Schema.XML.Namespace = xmlNamespace
			fieldSchema.Schema.XML.Prefix = xmlPrefix
		}

		if fieldSchema.Ref != "" || fieldSchema.Schema != nil {
			schema.Properties[xmlName] = *fieldSchema

			// Check if field is required
			if isFieldRequired(&field) {
				schema.Required = append(schema.Required, xmlName)
			}
		}
	}
}

func generateXMLSchemaForField(field *reflect.StructField, components *openapi.Components) *openapi.SchemaOrRef {
	fieldType := field.Type

	// Handle pointers
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	// Parse XML tag for this field
	xmlTag := field.Tag.Get("xml")
	xmlNodeType := xmlNodeTypeElement // default
	xmlName := ""

	if xmlTag != "" {
		parts := strings.Split(xmlTag, ",")
		xmlName = parts[0]

		for _, part := range parts[1:] {
			if strings.TrimSpace(part) == "attr" {
				xmlNodeType = xmlNodeTypeAttribute
			}
		}
	}

	// Determine the JSON schema type
	switch {
	case fieldType == reflect.TypeOf(time.Time{}):
		schema := &openapi.Schema{
			Type:   "string",
			Format: getTimeFormat(field),
			XML: &openapi.XML{
				NodeType: xmlNodeType,
				Name:     xmlName,
			},
		}
		applyValidationRules(field, schema, reflect.String)
		return &openapi.SchemaOrRef{Schema: schema}

	case fieldType == reflect.TypeOf(uuid.UUID{}):
		schema := &openapi.Schema{
			Type:   "string",
			Format: "uuid",
			XML: &openapi.XML{
				NodeType: xmlNodeType,
				Name:     xmlName,
			},
		}
		applyValidationRules(field, schema, reflect.String)
		return &openapi.SchemaOrRef{Schema: schema}

	case fieldType.Kind() == reflect.Struct:
		// Handle nested structs by adding them to components
		typName := fieldType.String() + ".XML"

		if components.Schemas == nil {
			components.Schemas = make(map[string]openapi.Schema)
		}

		registerXMLStructSchema(typName, fieldType, components)

		// Return a reference to the component schema with XML metadata
		return &openapi.SchemaOrRef{
			Ref: fmt.Sprintf("#/components/schemas/%s", typName),
		}

	case fieldType.Kind() == reflect.Slice:
		schema := &openapi.Schema{
			Type:  "array",
			Items: generateXMLSchemaForSliceElement(field, components),
			XML: &openapi.XML{
				NodeType: xmlNodeType,
				Name:     xmlName,
			},
		}
		applySliceValidationRules(field, schema)
		return &openapi.SchemaOrRef{Schema: schema}

	case fieldType.Kind() == reflect.String:
		schema := &openapi.Schema{
			Type: "string",
			XML: &openapi.XML{
				NodeType: xmlNodeType,
				Name:     xmlName,
			},
		}
		applyValidationRules(field, schema, reflect.String)
		return &openapi.SchemaOrRef{Schema: schema}

	case fieldType.Kind() == reflect.Int, fieldType.Kind() == reflect.Int8,
		fieldType.Kind() == reflect.Int16, fieldType.Kind() == reflect.Int32,
		fieldType.Kind() == reflect.Int64,
		fieldType.Kind() == reflect.Uint, fieldType.Kind() == reflect.Uint8,
		fieldType.Kind() == reflect.Uint16, fieldType.Kind() == reflect.Uint32,
		fieldType.Kind() == reflect.Uint64:
		schema := &openapi.Schema{
			Type:   "integer",
			Format: getIntegerFormat(field),
			XML: &openapi.XML{
				NodeType: xmlNodeType,
				Name:     xmlName,
			},
		}
		applyValidationRules(field, schema, reflect.Int)
		return &openapi.SchemaOrRef{Schema: schema}

	case fieldType.Kind() == reflect.Float32, fieldType.Kind() == reflect.Float64:
		schema := &openapi.Schema{
			Type:   "number",
			Format: getNumberFormat(field),
			XML: &openapi.XML{
				NodeType: xmlNodeType,
				Name:     xmlName,
			},
		}
		applyValidationRules(field, schema, reflect.Float64)
		return &openapi.SchemaOrRef{Schema: schema}

	case fieldType.Kind() == reflect.Bool:
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{
				Type: "boolean",
				XML: &openapi.XML{
					NodeType: xmlNodeType,
					Name:     xmlName,
				},
			},
		}

	case fieldType.Kind() == reflect.Interface:
		// Handle interface{} / any type - accepts any value
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{
				XML: &openapi.XML{
					NodeType: xmlNodeType,
					Name:     xmlName,
				},
			},
		}

	default:
		// Unsupported type
		return &openapi.SchemaOrRef{}
	}
}

func generateXMLSchemaForSliceElement(field *reflect.StructField, components *openapi.Components) *openapi.SchemaOrRef {
	elemType := field.Type.Elem()

	// Handle pointer elements
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	// Parse XML tag for array items
	xmlTag := field.Tag.Get("xml")
	xmlName := ""

	if xmlTag != "" {
		parts := strings.Split(xmlTag, ",")
		xmlName = parts[0]
	}

	switch {
	case elemType == reflect.TypeOf(time.Time{}):
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{
				Type:   "string",
				Format: getTimeFormat(field),
				XML: &openapi.XML{
					NodeType: xmlNodeTypeElement,
					Name:     xmlName,
				},
			},
		}

	case elemType == reflect.TypeOf(uuid.UUID{}):
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{
				Type:   "string",
				Format: "uuid",
				XML: &openapi.XML{
					NodeType: xmlNodeTypeElement,
					Name:     xmlName,
				},
			},
		}

	case elemType.Kind() == reflect.Struct:
		// Handle nested structs in arrays by adding them to components
		typName := elemType.String() + ".XML"

		if components.Schemas == nil {
			components.Schemas = make(map[string]openapi.Schema)
		}

		registerXMLStructSchema(typName, elemType, components)

		// Return a reference to the component schema
		return &openapi.SchemaOrRef{
			Ref: fmt.Sprintf("#/components/schemas/%s", typName),
		}

	case elemType.Kind() == reflect.String:
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{
				Type: "string",
				XML: &openapi.XML{
					NodeType: xmlNodeTypeElement,
					Name:     xmlName,
				},
			},
		}

	case elemType.Kind() == reflect.Int, elemType.Kind() == reflect.Int8,
		elemType.Kind() == reflect.Int16, elemType.Kind() == reflect.Int32,
		elemType.Kind() == reflect.Int64,
		elemType.Kind() == reflect.Uint, elemType.Kind() == reflect.Uint8,
		elemType.Kind() == reflect.Uint16, elemType.Kind() == reflect.Uint32,
		elemType.Kind() == reflect.Uint64:
		format := getIntegerFormatFromType(elemType)
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{
				Type:   "integer",
				Format: format,
				XML: &openapi.XML{
					NodeType: xmlNodeTypeElement,
					Name:     xmlName,
				},
			},
		}

	case elemType.Kind() == reflect.Float32, elemType.Kind() == reflect.Float64:
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{
				Type: "number",
				XML: &openapi.XML{
					NodeType: xmlNodeTypeElement,
					Name:     xmlName,
				},
			},
		}

	case elemType.Kind() == reflect.Bool:
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{
				Type: "boolean",
				XML: &openapi.XML{
					NodeType: xmlNodeTypeElement,
					Name:     xmlName,
				},
			},
		}

	default:
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{
				Type: "string",
				XML: &openapi.XML{
					NodeType: xmlNodeTypeElement,
					Name:     xmlName,
				},
			},
		}
	}
}

func registerXMLStructSchema(typName string, typ reflect.Type, components *openapi.Components) {
	if _, ok := components.Schemas[typName]; ok {
		return
	}

	structSchema := &openapi.Schema{
		Type:       "object",
		Title:      typName,
		Properties: make(map[string]openapi.SchemaOrRef),
		Required:   []string{},
	}

	// Add to components first to handle circular references
	components.Schemas[typName] = *structSchema

	// Generate properties with XML metadata
	generateXMLSchemaForStruct(typ, structSchema, components)

	// Generate and set XML example for nested structs
	mockInstance := generateMockStruct(typ)
	structSchema.Example = generateXMLExample(mockInstance, "")

	// Update the schema in components with generated properties
	components.Schemas[typName] = *structSchema
}

func generateSchemaForStruct(typ reflect.Type, schema *openapi.Schema, components *openapi.Components) {
	for i := range typ.NumField() {
		field := typ.Field(i)

		// Get the JSON tag to use as property name
		propertyName := field.Tag.Get("json")
		if propertyName == "" {
			propertyName = field.Name
		}

		// Skip fields with json:"-"
		if propertyName == "-" {
			continue
		}

		// Remove omitempty and other options from property name
		if idx := strings.Index(propertyName, ","); idx != -1 {
			propertyName = propertyName[:idx]
		}

		// Create schema for this field
		fieldSchema := generateSchemaForField(&field, components)

		if fieldSchema.Ref != "" || fieldSchema.Schema != nil {
			schema.Properties[propertyName] = *fieldSchema

			// Check if field is required
			if isFieldRequired(&field) {
				schema.Required = append(schema.Required, propertyName)
			}
		}
	}
}

func generateSchemaForField(field *reflect.StructField, components *openapi.Components) *openapi.SchemaOrRef {
	fieldType := field.Type

	// Handle pointers
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	// Determine the JSON schema type
	switch {
	case fieldType == reflect.TypeOf(time.Time{}):
		schema := &openapi.Schema{
			Type:   "string",
			Format: getTimeFormat(field),
		}
		applyValidationRules(field, schema, reflect.String)
		return &openapi.SchemaOrRef{Schema: schema}

	case fieldType == reflect.TypeOf(uuid.UUID{}):
		schema := &openapi.Schema{
			Type:   "string",
			Format: "uuid",
		}
		applyValidationRules(field, schema, reflect.String)
		return &openapi.SchemaOrRef{Schema: schema}

	case fieldType.Kind() == reflect.Struct:
		// Handle nested structs by adding them to components
		typName := fieldType.String()

		if components.Schemas == nil {
			components.Schemas = make(map[string]openapi.Schema)
		}

		registerStructSchema(typName, fieldType, components)

		// Return a reference to the component schema
		return &openapi.SchemaOrRef{
			Ref: fmt.Sprintf("#/components/schemas/%s", typName),
		}

	case fieldType.Kind() == reflect.Slice:
		schema := &openapi.Schema{
			Type:  "array",
			Items: generateSchemaForSliceElement(field, components),
		}
		applySliceValidationRules(field, schema)
		return &openapi.SchemaOrRef{Schema: schema}

	case fieldType.Kind() == reflect.String:
		schema := &openapi.Schema{Type: "string"}
		applyValidationRules(field, schema, reflect.String)
		return &openapi.SchemaOrRef{Schema: schema}

	case fieldType.Kind() == reflect.Int, fieldType.Kind() == reflect.Int8,
		fieldType.Kind() == reflect.Int16, fieldType.Kind() == reflect.Int32,
		fieldType.Kind() == reflect.Int64,
		fieldType.Kind() == reflect.Uint, fieldType.Kind() == reflect.Uint8,
		fieldType.Kind() == reflect.Uint16, fieldType.Kind() == reflect.Uint32,
		fieldType.Kind() == reflect.Uint64:
		schema := &openapi.Schema{
			Type:   "integer",
			Format: getIntegerFormat(field),
		}
		applyValidationRules(field, schema, reflect.Int)
		return &openapi.SchemaOrRef{Schema: schema}

	case fieldType.Kind() == reflect.Float32, fieldType.Kind() == reflect.Float64:
		schema := &openapi.Schema{
			Type:   "number",
			Format: getNumberFormat(field),
		}
		applyValidationRules(field, schema, reflect.Float64)
		return &openapi.SchemaOrRef{Schema: schema}

	case fieldType.Kind() == reflect.Bool:
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{Type: "boolean"},
		}

	case fieldType.Kind() == reflect.Interface:
		// Handle interface{} / any type - accepts any JSON value
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{},
		}

	default:
		// Unsupported type
		return &openapi.SchemaOrRef{}
	}
}

func generateSchemaForSliceElement(field *reflect.StructField, components *openapi.Components) *openapi.SchemaOrRef {
	elemType := field.Type.Elem()

	// Handle pointer elements
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	switch {
	case elemType == reflect.TypeOf(time.Time{}):
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{
				Type:   "string",
				Format: getTimeFormat(field),
			},
		}

	case elemType == reflect.TypeOf(uuid.UUID{}):
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{
				Type:   "string",
				Format: "uuid",
			},
		}

	case elemType.Kind() == reflect.Struct:
		// Handle nested structs in arrays by adding them to components
		typName := elemType.String()

		if components.Schemas == nil {
			components.Schemas = make(map[string]openapi.Schema)
		}

		registerStructSchema(typName, elemType, components)

		// Return a reference to the component schema
		return &openapi.SchemaOrRef{
			Ref: fmt.Sprintf("#/components/schemas/%s", typName),
		}

	case elemType.Kind() == reflect.String:
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{Type: "string"},
		}

	case elemType.Kind() == reflect.Int, elemType.Kind() == reflect.Int8,
		elemType.Kind() == reflect.Int16, elemType.Kind() == reflect.Int32,
		elemType.Kind() == reflect.Int64,
		elemType.Kind() == reflect.Uint, elemType.Kind() == reflect.Uint8,
		elemType.Kind() == reflect.Uint16, elemType.Kind() == reflect.Uint32,
		elemType.Kind() == reflect.Uint64:
		format := getIntegerFormatFromType(elemType)
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{
				Type:   "integer",
				Format: format,
			},
		}

	case elemType.Kind() == reflect.Float32, elemType.Kind() == reflect.Float64:
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{Type: "number"},
		}

	case elemType.Kind() == reflect.Bool:
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{Type: "boolean"},
		}

	default:
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{Type: "string"}, // fallback
		}
	}
}

func getIntegerFormat(field *reflect.StructField) string {
	bitSize := field.Type.Bits()
	kind := field.Type.Kind()

	// Check if it's an unsigned integer
	isUnsigned := kind == reflect.Uint || kind == reflect.Uint8 ||
		kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64

	switch bitSize {
	case 8: //nolint:mnd // int8/uint8 bit size
		if isUnsigned {
			return "uint8"
		}
		return "int8"
	case 16: //nolint:mnd // int16/uint16 bit size
		if isUnsigned {
			return "uint16"
		}
		return "int16"
	case 32: //nolint:mnd // int32/uint32 bit size
		if isUnsigned {
			return "uint32"
		}
		return "int32"
	case 64: //nolint:mnd // int64/uint64 bit size
		if isUnsigned {
			return "uint64"
		}
		return "int64"
	default:
		return ""
	}
}

func getIntegerFormatFromType(t reflect.Type) string {
	bitSize := t.Bits()
	kind := t.Kind()

	// Check if it's an unsigned integer
	isUnsigned := kind == reflect.Uint || kind == reflect.Uint8 ||
		kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64

	switch bitSize {
	case 8: //nolint:mnd // int8/uint8 bit size
		if isUnsigned {
			return "uint8"
		}
		return "int8"
	case 16: //nolint:mnd // int16/uint16 bit size
		if isUnsigned {
			return "uint16"
		}
		return "int16"
	case 32: //nolint:mnd // int32/uint32 bit size
		if isUnsigned {
			return "uint32"
		}
		return "int32"
	case 64: //nolint:mnd // int64/uint64 bit size
		if isUnsigned {
			return "uint64"
		}
		return "int64"
	default:
		return ""
	}
}

func getNumberFormat(field *reflect.StructField) string {
	bitSize := field.Type.Bits()
	switch bitSize {
	case 32: //nolint:mnd // float32 bit size
		return "float"
	case 64: //nolint:mnd // float64 bit size
		return "double"
	default:
		return ""
	}
}

func getTimeFormat(field *reflect.StructField) string {
	format := field.Tag.Get("format")
	if format == "" {
		return dateTimeFormat // RFC3339 is the default, which is JSON Schema date-time format
	}

	// Map common Go formats to JSON Schema formats
	switch format {
	case time.RFC3339, time.RFC3339Nano:
		return dateTimeFormat
	case "2006-01-02":
		return "date"
	case "15:04:05":
		return "time"
	default:
		// For custom formats, we could use pattern, but JSON Schema format is better
		return dateTimeFormat
	}
}

func isFieldRequired(field *reflect.StructField) bool {
	validateTag := field.Tag.Get("validate")
	if validateTag == "" {
		return false
	}

	rules := strings.Split(validateTag, ",")
	for _, rule := range rules {
		if strings.TrimSpace(rule) == ruleRequired {
			return true
		}
	}
	return false
}

func applyValidationRules(field *reflect.StructField, schema *openapi.Schema, kind reflect.Kind) {
	validateTag := field.Tag.Get("validate")
	if validateTag == "" {
		return
	}

	rules := strings.Split(validateTag, ",")

	for _, rule := range rules {
		rule = strings.TrimSpace(rule)

		switch {
		case strings.HasPrefix(rule, "min=") && kind == reflect.Int:
			minVal, _ := strconv.Atoi(strings.TrimPrefix(rule, "min="))
			floatValue := float64(minVal)
			schema.Minimum = &floatValue

		case strings.HasPrefix(rule, "min=") && (kind == reflect.Float64 || kind == reflect.Float32):
			minVal, _ := strconv.ParseFloat(strings.TrimPrefix(rule, "min="), 64)
			schema.Minimum = &minVal

		case strings.HasPrefix(rule, "max=") && kind == reflect.Int:
			maxVal, _ := strconv.Atoi(strings.TrimPrefix(rule, "max="))
			floatValue := float64(maxVal)
			schema.Maximum = &floatValue

		case strings.HasPrefix(rule, "max=") && (kind == reflect.Float64 || kind == reflect.Float32):
			maxVal, _ := strconv.ParseFloat(strings.TrimPrefix(rule, "max="), 64)
			schema.Maximum = &maxVal

		case strings.HasPrefix(rule, "minlength=") && kind == reflect.String:
			minLen, _ := strconv.Atoi(strings.TrimPrefix(rule, "minlength="))
			schema.MinLength = &minLen

		case strings.HasPrefix(rule, "maxlength=") && kind == reflect.String:
			maxLen, _ := strconv.Atoi(strings.TrimPrefix(rule, "maxlength="))
			schema.MaxLength = &maxLen

		case strings.HasPrefix(rule, "regexp=") && kind == reflect.String:
			pattern := strings.TrimPrefix(rule, "regexp=")
			schema.Pattern = pattern

		case strings.HasPrefix(rule, "enum=") && kind == reflect.String:
			enumValues := strings.Split(strings.TrimPrefix(rule, "enum="), "|")
			for _, val := range enumValues {
				schema.Enum = append(schema.Enum, strings.TrimSpace(val))
			}
		}
	}
}

func applySliceValidationRules(field *reflect.StructField, schema *openapi.Schema) {
	validateTag := field.Tag.Get("validate")
	if validateTag == "" {
		return
	}

	rules := strings.Split(validateTag, ",")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)

		switch {
		case strings.HasPrefix(rule, "minItems="):
			minItems, _ := strconv.Atoi(strings.TrimPrefix(rule, "minItems="))
			schema.MinItems = &minItems

		case strings.HasPrefix(rule, "maxItems="):
			maxItems, _ := strconv.Atoi(strings.TrimPrefix(rule, "maxItems="))
			schema.MaxItems = &maxItems

		case rule == "uniqueItems":
			schema.UniqueItems = true
		}
	}
}

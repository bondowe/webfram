package bind

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/bondowe/webfram/openapi"
	"github.com/google/uuid"
)

const dateTimeFormat = "date-time"

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
		fieldType.Kind() == reflect.Int64:
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
		elemType.Kind() == reflect.Int64:
		return &openapi.SchemaOrRef{
			Schema: &openapi.Schema{Type: "integer"},
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
	switch bitSize {
	case 8: //nolint:mnd // int8 bit size
		return "int8"
	case 16: //nolint:mnd // int16 bit size
		return "int16"
	case 32: //nolint:mnd // int32 bit size
		return "int32"
	case 64: //nolint:mnd // int64 bit size
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

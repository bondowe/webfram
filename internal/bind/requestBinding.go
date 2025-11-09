package bind

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BindSource represents the source from which to bind data.
//
//nolint:revive // TODO: Consider renaming to Source in a future major version to avoid package stuttering
type BindSource string

const (
	// BindSourceAuto automatically determines the binding source based on precedence rules.
	BindSourceAuto BindSource = "auto"
	// BindSourcePath binds from URL path parameters.
	BindSourcePath BindSource = "path"
	// BindSourceQuery binds from URL query parameters.
	BindSourceQuery BindSource = "query"
	// BindSourceHeader binds from HTTP headers.
	BindSourceHeader BindSource = "header"
	// BindSourceCookie binds from HTTP cookies.
	BindSourceCookie BindSource = "cookie"
	// BindSourceBody binds from the request body (JSON, XML, or Form).
	BindSourceBody BindSource = "body"
	// BindSourceForm binds from form data (query + body).
	BindSourceForm BindSource = "form"
)

// Bind is a unified method that binds data from multiple sources to a struct of type T.
// It supports binding from path parameters, query parameters, headers, cookies, and request body.
// The bindFrom tag on struct fields determines the source of binding. If no tag is present,
// a precedence rule is applied: path > query > header > cookie > body.
// The validate parameter controls whether validation is performed after binding.
// Returns the populated struct, validation errors (if any), and an error if binding fails.
func Bind[T any](r *http.Request, validate bool) (T, []ValidationError, error) {
	var result T
	val := reflect.ValueOf(&result).Elem()
	typ := val.Type()

	errors := []ValidationError{}

	// Collect data from all sources
	sources := collectBindingSources(r)

	// Bind each field
	for i := range val.NumField() {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get bindFrom tag to determine source
		bindFromTag := fieldType.Tag.Get("bindFrom")
		bindSource := BindSource(strings.TrimSpace(bindFromTag))

		if bindSource == "" {
			bindSource = BindSourceAuto
		}

		// Get the field name for binding
		fieldName := getFieldNameForBinding(&fieldType)
		if fieldName == "-" {
			continue
		}

		// Bind based on source (without validation at this stage)
		if err := bindFieldFromSource(r, field, fieldType, fieldName, bindSource, sources, &errors); err != nil {
			return result, errors, err
		}
	}

	// Validate if requested (only once, after all binding is complete)
	if validate {
		bindValidateRecursive(val, "", &errors)
	}

	return result, errors, nil
} // bindingSources holds data from all binding sources.
type bindingSources struct {
	path   map[string]string
	query  url.Values
	header http.Header
	cookie map[string]string
	form   url.Values // For form data
}

// collectBindingSources gathers data from all possible binding sources.
func collectBindingSources(r *http.Request) *bindingSources {
	sources := &bindingSources{
		path:   make(map[string]string),
		query:  r.URL.Query(),
		header: r.Header,
		cookie: make(map[string]string),
	}

	// Collect path parameters (using PathValue method)
	// Note: We'll collect these dynamically as needed since Go 1.22 doesn't provide
	// a method to list all path parameters

	// Collect cookies
	for _, c := range r.Cookies() {
		sources.cookie[c.Name] = c.Value
	}

	// Parse form if content-type suggests it
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") ||
		strings.Contains(contentType, "multipart/form-data") {
		_ = r.ParseForm()
		sources.form = r.Form
	}

	return sources
}

// bindFieldFromSource binds a field from the specified source.
func bindFieldFromSource(
	r *http.Request,
	field reflect.Value,
	fieldType reflect.StructField,
	fieldName string,
	source BindSource,
	sources *bindingSources,
	errors *[]ValidationError,
) error {
	kind := field.Kind()
	isTimeField := field.Type() == reflect.TypeOf(time.Time{})
	isUUIDField := field.Type() == reflect.TypeOf(uuid.UUID{})

	// Handle struct fields recursively (except time.Time and uuid.UUID)
	if kind == reflect.Struct && !isTimeField && !isUUIDField {
		// For nested structs, we don't bind directly
		// The nested fields will be handled in their own iteration
		return nil
	}

	var value string
	var values []string
	found := false

	// Determine the source and get the value
	switch source {
	case BindSourceAuto:
		// Apply precedence: path > query > header > cookie > body
		value, values, found = getValueWithPrecedence(r, fieldName, sources)

	case BindSourcePath:
		// Get path parameter using the PathValue method
		// This works with Go 1.22+ routing
		if r != nil {
			pathValue := r.PathValue(fieldName)
			if pathValue != "" {
				value = pathValue
				values = []string{pathValue}
				found = true
			}
		}

	case BindSourceQuery:
		if qv := sources.query[fieldName]; len(qv) > 0 {
			values = qv
			value = qv[0]
			found = true
		}

	case BindSourceHeader:
		if hv := sources.header.Values(fieldName); len(hv) > 0 {
			values = hv
			value = hv[0]
			found = true
		}

	case BindSourceCookie:
		if cv, ok := sources.cookie[fieldName]; ok {
			value = cv
			values = []string{cv}
			found = true
		}

	case BindSourceForm:
		if sources.form != nil {
			if fv := sources.form[fieldName]; len(fv) > 0 {
				values = fv
				value = fv[0]
				found = true
			}
		}

	case BindSourceBody:
		// Body binding is handled separately for JSON/XML
		// For now, we'll skip individual field binding from body
		return nil

	default:
		return fmt.Errorf("unknown bind source: %s", source)
	}

	if !found {
		// Field not found in any source, leave as zero value
		return nil
	}

	// Handle slice types
	if kind == reflect.Slice && !isTimeField {
		if len(values) == 0 {
			values = []string{""}
		}

		if err := bindSliceField(field, fieldType, values, errors); err != nil {
			return err
		}
		return nil
	}

	// Bind single value (no validation here, it will be done later if requested)
	if err := bindSingleValueWithoutValidation(field, fieldType, value, errors); err != nil {
		return err
	}

	return nil
}

// getValueWithPrecedence retrieves a value following the precedence rule:
// path > query > header > cookie > body/form.
func getValueWithPrecedence(
	r *http.Request,
	fieldName string,
	sources *bindingSources,
) (string, []string, bool) {
	// 1. Check path (using PathValue)
	if r != nil {
		if pathValue := r.PathValue(fieldName); pathValue != "" {
			return pathValue, []string{pathValue}, true
		}
	}

	// 2. Check query
	if qv := sources.query[fieldName]; len(qv) > 0 {
		return qv[0], qv, true
	}

	// 3. Check header
	if hv := sources.header.Values(fieldName); len(hv) > 0 {
		return hv[0], hv, true
	}

	// 4. Check cookie
	if cv, ok := sources.cookie[fieldName]; ok {
		return cv, []string{cv}, true
	}

	// 5. Check form
	if sources.form != nil {
		if fv := sources.form[fieldName]; len(fv) > 0 {
			return fv[0], fv, true
		}
	}

	return "", nil, false
}

// getFieldNameForBinding returns the field name to use for binding.
// It checks the form tag first, then falls back to json or xml tags, then the field name.
func getFieldNameForBinding(fieldType *reflect.StructField) string {
	// Check form tag
	if tag := fieldType.Tag.Get("form"); tag != "" {
		return tag
	}

	// Check json tag as fallback
	if tag := fieldType.Tag.Get("json"); tag != "" {
		// Remove options like omitempty
		if idx := strings.Index(tag, ","); idx != -1 {
			return tag[:idx]
		}
		return tag
	}

	// Check xml tag as fallback
	if tag := fieldType.Tag.Get("xml"); tag != "" {
		// Remove options like omitempty
		if idx := strings.Index(tag, ","); idx != -1 {
			return tag[:idx]
		}
		return tag
	}

	// Use field name
	return fieldType.Name
}

// bindBasicType binds a string value to basic types (string, int, float, bool, etc.).
func bindBasicType(
	field reflect.Value,
	fieldType reflect.StructField,
	value string,
	errors *[]ValidationError,
) {
	kind := field.Kind()
	//nolint:exhaustive // TODO: Add support for complex types, channels, and other advanced types
	switch kind {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value != "" {
			iv, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				*errors = append(
					*errors,
					ValidationError{Field: fieldType.Name, Error: "invalid integer"},
				)
			} else {
				field.SetInt(iv)
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value != "" {
			uv, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				*errors = append(
					*errors,
					ValidationError{Field: fieldType.Name, Error: "invalid unsigned integer"},
				)
			} else {
				field.SetUint(uv)
			}
		}
	case reflect.Float32, reflect.Float64:
		if value != "" {
			fv, err := strconv.ParseFloat(value, 64)
			if err != nil {
				*errors = append(
					*errors,
					ValidationError{Field: fieldType.Name, Error: "invalid float"},
				)
			} else {
				field.SetFloat(fv)
			}
		}
	case reflect.Bool:
		field.SetBool(value == "true" || value == "1" || value == "yes")
	}
}

// bindSingleValueWithoutValidation binds a single string value to a field without validation.
// Validation will be performed later if requested.
func bindSingleValueWithoutValidation(
	field reflect.Value,
	fieldType reflect.StructField,
	value string,
	errors *[]ValidationError,
) error {
	isTimeField := field.Type() == reflect.TypeOf(time.Time{})
	isUUIDField := field.Type() == reflect.TypeOf(uuid.UUID{})

	// Handle special types
	//nolint:nestif // TODO: Refactor time field handling to reduce nesting complexity
	if isTimeField {
		// Get the format from the struct tag
		format := fieldType.Tag.Get("format")
		if format == "" {
			format = time.RFC3339 // Default format
		}

		if value != "" {
			t, err := time.Parse(format, value)
			if err != nil {
				*errors = append(
					*errors,
					ValidationError{
						Field: fieldType.Name,
						Error: fmt.Sprintf("invalid time format, expected %s", format),
					},
				)
			} else {
				field.Set(reflect.ValueOf(t))
			}
		}
		return nil
	}

	if isUUIDField {
		if value != "" {
			u, err := uuid.Parse(value)
			if err != nil {
				*errors = append(
					*errors,
					ValidationError{Field: fieldType.Name, Error: "invalid UUID"},
				)
			} else {
				field.Set(reflect.ValueOf(u))
			}
		}
		return nil
	}

	// Bind basic types using the extracted helper
	bindBasicType(field, fieldType, value, errors)
	return nil
}

// Path binds URL path parameters to a struct of type T.
// Path parameters are extracted from the request using PathValue method.
// Struct fields should use the "form" tag to specify parameter names.
// Returns the populated struct, validation errors (if any), and an error if binding fails.
func Path[T any](r *http.Request) (T, []ValidationError, error) {
	var result T
	val := reflect.ValueOf(&result).Elem()
	typ := val.Type()

	errors := []ValidationError{}

	for i := range val.NumField() {
		field := val.Field(i)
		fieldType := typ.Field(i)

		tag := fieldType.Tag.Get("form")
		if tag == "-" || tag == "" {
			continue
		}

		// Get path parameter value
		value := r.PathValue(tag)

		bindSingleValue(field, fieldType, value, &errors)
	}

	return result, errors, nil
}

// Query binds URL query parameters to a struct of type T.
// Query parameters are extracted from r.URL.Query().
// Struct fields should use the "form" tag to specify parameter names.
// Supports slices for multi-value parameters.
// Returns the populated struct, validation errors (if any), and an error if binding fails.
func Query[T any](r *http.Request) (T, []ValidationError, error) {
	var result T
	val := reflect.ValueOf(&result).Elem()

	queryParams := r.URL.Query()
	errors := []ValidationError{}
	err := bindRecursive(queryParams, val, "", &errors)
	return result, errors, err
}

// Cookie binds HTTP cookies to a struct of type T.
// Cookie values are extracted from r.Cookies().
// Struct fields should use the "form" tag to specify cookie names.
// Returns the populated struct, validation errors (if any), and an error if binding fails.
func Cookie[T any](r *http.Request) (T, []ValidationError, error) {
	var result T
	val := reflect.ValueOf(&result).Elem()
	typ := val.Type()

	errors := []ValidationError{}

	// Build a map of cookie values
	cookieMap := make(map[string]string)
	for _, cookie := range r.Cookies() {
		cookieMap[cookie.Name] = cookie.Value
	}

	for i := range val.NumField() {
		field := val.Field(i)
		fieldType := typ.Field(i)

		tag := fieldType.Tag.Get("form")
		if tag == "-" || tag == "" {
			continue
		}

		value := cookieMap[tag]

		bindSingleValue(field, fieldType, value, &errors)
	}

	return result, errors, nil
}

// Header binds HTTP headers to a struct of type T.
// Header values are extracted from r.Header.
// Struct fields should use the "form" tag to specify header names.
// Header names are case-insensitive per HTTP specification.
// Supports slices for multi-value headers.
// Returns the populated struct, validation errors (if any), and an error if binding fails.
func Header[T any](r *http.Request) (T, []ValidationError, error) {
	var result T
	val := reflect.ValueOf(&result).Elem()
	typ := val.Type()

	errors := []ValidationError{}

	for i := range val.NumField() {
		field := val.Field(i)
		fieldType := typ.Field(i)

		tag := fieldType.Tag.Get("form")
		if tag == "-" || tag == "" {
			continue
		}

		kind := field.Kind()
		isTimeField := field.Type() == reflect.TypeOf(time.Time{})

		// Validate field type rules
		validateFieldTypeRules(&fieldType, kind, field.Type())

		// Get header values (headers can have multiple values)
		values := r.Header.Values(tag)

		if len(values) == 0 {
			values = []string{""}
		}

		// Handle slice types
		if kind == reflect.Slice && !isTimeField {
			if errs := validateSliceLength(&fieldType, values); errs != nil {
				errors = append(errors, *errs)
			}

			if errs := validateUniqueItems(&fieldType, values); errs != nil {
				errors = append(errors, *errs)
			}

			if err := bindSliceField(field, fieldType, values, &errors); err != nil {
				return result, errors, err
			}
			continue
		}

		// For non-slice types, use the first value
		value := values[0]

		bindSingleValue(field, fieldType, value, &errors)
	}

	return result, errors, nil
}

// bindSingleValue binds a single string value to a field with validation.
func bindSingleValue(
	field reflect.Value,
	fieldType reflect.StructField,
	value string,
	errors *[]ValidationError,
) {
	kind := field.Kind()
	isTimeField := field.Type() == reflect.TypeOf(time.Time{})
	isUUIDField := field.Type() == reflect.TypeOf(uuid.UUID{})

	// Validate first
	if err := validateField(&fieldType, value, kind); err != nil {
		*errors = append(*errors, *err)
	}

	// Handle special types
	if isTimeField {
		if v, err := validateTimeFieldString(&fieldType, value); err != nil {
			*errors = append(*errors, *err)
		} else {
			field.Set(reflect.ValueOf(v))
		}
	}

	if isUUIDField {
		if v, err := validateUUIDFieldString(&fieldType, value); err != nil {
			*errors = append(*errors, *err)
		} else {
			field.Set(reflect.ValueOf(v))
		}
	}

	// Bind basic types using the extracted helper
	bindBasicType(field, fieldType, value, errors)
}

// bindSliceField binds multiple values to a slice field.
func bindSliceField(
	field reflect.Value,
	fieldType reflect.StructField,
	values []string,
	errors *[]ValidationError,
) error {
	switch field.Type().Elem() {
	case reflect.TypeOf(uuid.UUID{}):
		vs, errs := validateUUIDSliceFieldString(&fieldType, values)
		if len(errs) > 0 {
			*errors = append(*errors, errs...)
		}
		field.Set(reflect.ValueOf(vs))

	case reflect.TypeOf(time.Time{}):
		vs, errs := validateTimeSliceFieldString(&fieldType, values)
		if len(errs) > 0 {
			*errors = append(*errors, errs...)
		}
		field.Set(reflect.ValueOf(vs))
	}

	//nolint:exhaustive // TODO: Add support for complex slice types (complex64/128, channels, etc.)
	switch field.Type().Elem().Kind() {
	case reflect.String:
		field.Set(reflect.ValueOf(values))

	case reflect.Int:
		intSlice := []int{}
		for _, v := range values {
			iv, err := strconv.Atoi(v)
			if err != nil {
				*errors = append(
					*errors,
					ValidationError{Field: fieldType.Name, Error: "invalid int in slice"},
				)
				continue
			}
			intSlice = append(intSlice, iv)
		}
		field.Set(reflect.ValueOf(intSlice))

	case reflect.Int8:
		intSlice := []int8{}
		for _, v := range values {
			iv, err := strconv.ParseInt(v, 10, 8)
			if err != nil {
				*errors = append(
					*errors,
					ValidationError{Field: fieldType.Name, Error: "invalid int8 in slice"},
				)
				continue
			}
			intSlice = append(intSlice, int8(iv))
		}
		field.Set(reflect.ValueOf(intSlice))

	case reflect.Int16:
		intSlice := []int16{}
		for _, v := range values {
			iv, err := strconv.ParseInt(v, 10, 16)
			if err != nil {
				*errors = append(
					*errors,
					ValidationError{Field: fieldType.Name, Error: "invalid int16 in slice"},
				)
				continue
			}
			intSlice = append(intSlice, int16(iv))
		}
		field.Set(reflect.ValueOf(intSlice))

	case reflect.Int32:
		intSlice := []int32{}
		for _, v := range values {
			iv, err := strconv.ParseInt(v, 10, 32)
			if err != nil {
				*errors = append(
					*errors,
					ValidationError{Field: fieldType.Name, Error: "invalid int32 in slice"},
				)
				continue
			}
			intSlice = append(intSlice, int32(iv))
		}
		field.Set(reflect.ValueOf(intSlice))

	case reflect.Int64:
		intSlice := []int64{}
		for _, v := range values {
			iv, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				*errors = append(
					*errors,
					ValidationError{Field: fieldType.Name, Error: "invalid int64 in slice"},
				)
				continue
			}
			intSlice = append(intSlice, iv)
		}
		field.Set(reflect.ValueOf(intSlice))

	case reflect.Float32:
		floatSlice := []float32{}
		for _, v := range values {
			fv, err := strconv.ParseFloat(v, 32)
			if err != nil {
				*errors = append(
					*errors,
					ValidationError{Field: fieldType.Name, Error: "invalid float32 in slice"},
				)
				continue
			}
			floatSlice = append(floatSlice, float32(fv))
		}
		field.Set(reflect.ValueOf(floatSlice))

	case reflect.Float64:
		floatSlice := []float64{}
		for _, v := range values {
			fv, err := strconv.ParseFloat(v, 64)
			if err != nil {
				*errors = append(
					*errors,
					ValidationError{Field: fieldType.Name, Error: "invalid float64 in slice"},
				)
				continue
			}
			floatSlice = append(floatSlice, fv)
		}
		field.Set(reflect.ValueOf(floatSlice))
	}

	return nil
}

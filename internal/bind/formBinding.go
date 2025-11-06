package bind

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func Form[T any](r *http.Request) (T, []ValidationError, error) {
	var result T
	val := reflect.ValueOf(&result).Elem()

	if err := r.ParseForm(); err != nil {
		return result, nil, err
	}

	errors := []ValidationError{}
	err := bindRecursive(r.Form, val, "", &errors)
	return result, errors, err
}

func bindRecursive(form map[string][]string, val reflect.Value, prefix string, errors *[]ValidationError) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		tag := fieldType.Tag.Get("form")

		if tag == "-" {
			continue
		}

		if tag == "" {
			tag = fieldType.Name
		}

		key := tag
		if prefix != "" {
			key = prefix + "." + tag
		}

		values := form[key]
		kind := field.Kind()

		isTimeField := field.Type() == reflect.TypeOf(time.Time{})

		if kind == reflect.Struct && !isTimeField {
			if err := bindRecursive(form, field, key, errors); err != nil {
				return err
			}
			continue
		}

		// Validate that the validation rules are applicable to this field type
		validateFieldTypeRules(fieldType, kind, field.Type())

		if len(values) == 0 {
			values = []string{""}
		}

		// Validate first value
		if err := validateField(fieldType, values[0], kind); err != nil {
			*errors = append(*errors, *err)
		}

		// Bind values
		switch kind {
		case reflect.String:
			field.SetString(values[0])
		case reflect.Int:
			iv, _ := strconv.Atoi(values[0])
			field.SetInt(int64(iv))
		case reflect.Float32, reflect.Float64:
			fv, _ := strconv.ParseFloat(values[0], 64)
			field.SetFloat(fv)
		case reflect.Bool:
			field.SetBool(values[0] == "true")
		case reflect.Slice:

			if errs := validateSliceLength(&fieldType, values); errs != nil {
				*errors = append(*errors, *errs)
			}

			if errs := validateUniqueItems(&fieldType, values); errs != nil {
				*errors = append(*errors, *errs)
			}

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

			switch field.Type().Elem().Kind() {
			case reflect.String:
				field.Set(reflect.ValueOf(values))
			case reflect.Int:
				intSlice := []int{}
				for _, v := range values {
					iv, err := strconv.Atoi(v)
					if err != nil {
						*errors = append(*errors, ValidationError{Field: fieldType.Name, Error: "invalid int in slice"})
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
						*errors = append(*errors, ValidationError{Field: fieldType.Name, Error: "invalid int8 in slice"})
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
						*errors = append(*errors, ValidationError{Field: fieldType.Name, Error: "invalid int16 in slice"})
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
						*errors = append(*errors, ValidationError{Field: fieldType.Name, Error: "invalid int32 in slice"})
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
						*errors = append(*errors, ValidationError{Field: fieldType.Name, Error: "invalid int64 in slice"})
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
						*errors = append(*errors, ValidationError{Field: fieldType.Name, Error: "invalid float32 in slice"})
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
						*errors = append(*errors, ValidationError{Field: fieldType.Name, Error: "invalid float64 in slice"})
						continue
					}
					floatSlice = append(floatSlice, fv)
				}
				field.Set(reflect.ValueOf(floatSlice))
			}

		case reflect.Map:
			// Initialize the map if nil
			if field.IsNil() {
				field.Set(reflect.MakeMap(field.Type()))
			}

			// For form data, convention for map keys:
			// e.g., "metadata[color]=red&metadata[size]=large"
			// Parse keys from form data that match pattern: key[subkey]

			mapKeyType := field.Type().Key()
			mapValueType := field.Type().Elem()

			// Track map size for validation
			mapSize := 0

			// Iterate through form values and extract map entries
			for formKey, formValues := range form {
				if strings.HasPrefix(formKey, key+"[") && strings.HasSuffix(formKey, "]") {
					// Extract the map key
					mapKeyStr := formKey[len(key)+1 : len(formKey)-1]

					// Skip empty keys
					if mapKeyStr == "" {
						continue
					}

					// Convert string key to appropriate type
					mapKey, err := convertStringToType(mapKeyStr, mapKeyType)
					if err != nil {
						*errors = append(*errors, ValidationError{
							Field: fieldType.Name,
							Error: fmt.Sprintf("invalid map key '%s': %v", mapKeyStr, err),
						})
						continue
					}

					// Convert form value to map value type
					if len(formValues) > 0 {
						mapValue, err := convertStringToType(formValues[0], mapValueType)
						if err != nil {
							*errors = append(*errors, ValidationError{
								Field: fieldType.Name,
								Error: fmt.Sprintf("invalid map value for key '%s': %v", mapKeyStr, err),
							})
							continue
						}

						field.SetMapIndex(mapKey, mapValue)
						mapSize++
					}
				}
			}

			// Validate map size
			if err := validateMapSize(fieldType, mapSize); err != nil {
				*errors = append(*errors, *err)
			}
		}

		if isTimeField {
			if v, err := validateTimeFieldString(&fieldType, values[0]); err != nil {
				*errors = append(*errors, *err)
			} else {
				field.Set(reflect.ValueOf(v))
			}
			continue
		}

		if field.Type() == reflect.TypeOf(uuid.UUID{}) {
			if v, err := validateUUIDFieldString(&fieldType, values[0]); err != nil {
				*errors = append(*errors, *err)
			} else {
				field.Set(reflect.ValueOf(v))
			}
			continue
		}
	}

	return nil
}

func validateUniqueItems(fieldType *reflect.StructField, values []string) *ValidationError {
	if !strings.Contains(fieldType.Tag.Get("validate"), "uniqueItems") {
		return nil
	}
	itemMap := make(map[string]bool)
	for _, v := range values {
		if itemMap[v] {
			return &ValidationError{Field: fieldType.Name, Error: "must have unique items"}
		}
		itemMap[v] = true
	}
	return nil
}

func validateSliceLength(field *reflect.StructField, value interface{}) *ValidationError {
	validateTag := field.Tag.Get("validate")
	if validateTag == "" {
		return nil
	}

	rules := strings.Split(validateTag, ",")
	length := reflect.ValueOf(value).Len()

	for _, rule := range rules {
		switch {
		case strings.HasPrefix(rule, "minItems="):
			minLen, _ := strconv.Atoi(strings.TrimPrefix(rule, "minItems="))
			if length < minLen {
				return &ValidationError{Field: field.Name, Error: fmt.Sprintf("must have at least %d items", minLen)}
			}
		case strings.HasPrefix(rule, "maxItems="):
			maxLen, _ := strconv.Atoi(strings.TrimPrefix(rule, "maxItems="))
			if length > maxLen {
				return &ValidationError{Field: field.Name, Error: fmt.Sprintf("must have at most %d items", maxLen)}
			}
		}
	}

	return nil
}

func validateTimeFieldString(field *reflect.StructField, value string) (time.Time, *ValidationError) {
	var rules []string
	validateTag := field.Tag.Get("validate")
	if validateTag != "" {
		rules = strings.Split(validateTag, ",")
	}

	layout := time.RFC3339

	for _, rule := range rules {
		if strings.HasPrefix(rule, "format=") {
			layout = strings.TrimPrefix(rule, "format=")
			break
		}
	}

	v, err := time.Parse(layout, value)
	if err != nil {
		msg := getErrorMessage(field, "format", fmt.Sprintf("must match format %s", layout))
		return time.Time{}, &ValidationError{Field: field.Name, Error: msg}
	}

	if field.Type.Kind() == reflect.Slice {
		if v.IsZero() && !strings.Contains(validateTag, ruleEmptyItemsAllowed) {
			msg := getErrorMessage(field, ruleEmptyItemsAllowed+" (not set)", "empty item not allowed")
			return v, &ValidationError{Field: field.Name, Error: msg}
		}
	} else {
		if v.IsZero() && strings.Contains(validateTag, ruleRequired) {
			msg := getErrorMessage(field, ruleRequired, "is required")
			return time.Time{}, &ValidationError{Field: field.Name, Error: msg}
		}
	}

	return v, nil
}

func validateTimeSliceFieldString(field *reflect.StructField, values []string) ([]time.Time, []ValidationError) {
	var vs []time.Time
	var errors []ValidationError

	for _, value := range values {
		v, err := validateTimeFieldString(field, value)
		if err != nil {
			errors = append(errors, *err)
			v = time.Time{}
		}

		vs = append(vs, v)
	}

	return vs, errors
}

func validateUUIDFieldString(field *reflect.StructField, value string) (uuid.UUID, *ValidationError) {

	v, err := uuid.Parse(value)
	if err != nil {
		msg := getErrorMessage(field, "uuid", "must be a valid UUID")
		return uuid.Nil, &ValidationError{Field: field.Name, Error: msg}
	}

	if field.Type.Kind() == reflect.Slice {
		if v == uuid.Nil && !strings.Contains(field.Tag.Get("validate"), ruleEmptyItemsAllowed) {
			msg := getErrorMessage(field, ruleEmptyItemsAllowed+" (not set)", "empty items not allowed")
			return v, &ValidationError{Field: field.Name, Error: msg}
		}
	} else {
		if v == uuid.Nil && strings.Contains(field.Tag.Get("validate"), ruleRequired) {
			msg := getErrorMessage(field, ruleRequired, "is required")
			return v, &ValidationError{Field: field.Name, Error: msg}
		}
	}

	return v, nil
}

func validateUUIDSliceFieldString(field *reflect.StructField, values []string) ([]uuid.UUID, []ValidationError) {
	var vs []uuid.UUID
	var errors []ValidationError

	for _, value := range values {
		v, err := validateUUIDFieldString(field, value)
		if err != nil {
			errors = append(errors, *err)
			v = uuid.UUID{}
		}
		vs = append(vs, v)
	}

	return vs, errors
}

func validateField(field reflect.StructField, value string, kind reflect.Kind) *ValidationError {
	validateTag := field.Tag.Get("validate")
	if validateTag == "" {
		return nil
	}

	rules := strings.Split(validateTag, ",")
	for _, rule := range rules {
		switch {
		case rule == "required" && value == "":
			msg := getErrorMessage(&field, "required", "is required")
			return &ValidationError{Field: field.Name, Error: msg}

		case strings.HasPrefix(rule, "min=") && IsIntType(kind):
			minVal, _ := strconv.Atoi(strings.TrimPrefix(rule, "min="))
			val, err := strconv.Atoi(value)
			if err != nil || val < minVal {
				msg := getErrorMessage(&field, "min", fmt.Sprintf("must be at least %d", minVal))
				return &ValidationError{Field: field.Name, Error: msg}
			}

		case strings.HasPrefix(rule, "max=") && IsIntType(kind):
			maxVal, _ := strconv.Atoi(strings.TrimPrefix(rule, "max="))
			val, err := strconv.Atoi(value)
			if err != nil || val > maxVal {
				msg := getErrorMessage(&field, "max", fmt.Sprintf("must be at most %d", maxVal))
				return &ValidationError{Field: field.Name, Error: msg}
			}

		case strings.HasPrefix(rule, "min=") && IsFloatType(kind):
			minVal, _ := strconv.ParseFloat(strings.TrimPrefix(rule, "min="), 64)
			val, err := strconv.ParseFloat(value, 64)
			if err != nil || val < minVal {
				msg := getErrorMessage(&field, "min", fmt.Sprintf("must be at least %f", minVal))
				return &ValidationError{Field: field.Name, Error: msg}
			}

		case strings.HasPrefix(rule, "max=") && IsFloatType(kind):
			maxVal, _ := strconv.ParseFloat(strings.TrimPrefix(rule, "max="), 64)
			val, err := strconv.ParseFloat(value, 64)
			if err != nil || val > maxVal {
				msg := getErrorMessage(&field, "max", fmt.Sprintf("must be at most %f", maxVal))
				return &ValidationError{Field: field.Name, Error: msg}
			}

		case strings.HasPrefix(rule, "multipleOf=") && IsIntType(kind):
			multVal, _ := strconv.Atoi(strings.TrimPrefix(rule, "multipleOf="))
			val, err := strconv.Atoi(value)
			if err != nil || val%multVal != 0 {
				msg := getErrorMessage(&field, "multipleOf", fmt.Sprintf("must be a multiple of %d", multVal))
				return &ValidationError{Field: field.Name, Error: msg}
			}

		case strings.HasPrefix(rule, "multipleOf=") && IsFloatType(kind):
			multVal, _ := strconv.ParseFloat(strings.TrimPrefix(rule, "multipleOf="), 64)
			val, err := strconv.ParseFloat(value, 64)
			if err != nil || int(val*1000000)%int(multVal*1000000) != 0 {
				msg := getErrorMessage(&field, "multipleOf", fmt.Sprintf("must be a multiple of %f", multVal))
				return &ValidationError{Field: field.Name, Error: msg}
			}

		case strings.HasPrefix(rule, "minlength=") && kind == reflect.String:
			minLen, _ := strconv.Atoi(strings.TrimPrefix(rule, "minlength="))
			if len(value) < minLen {
				msg := getErrorMessage(&field, "minlength", fmt.Sprintf("must be at least %d characters", minLen))
				return &ValidationError{Field: field.Name, Error: msg}
			}

		case strings.HasPrefix(rule, "maxlength=") && kind == reflect.String:
			maxLen, _ := strconv.Atoi(strings.TrimPrefix(rule, "maxlength="))
			if len(value) > maxLen {
				msg := getErrorMessage(&field, "maxlength", fmt.Sprintf("must be at most %d characters", maxLen))
				return &ValidationError{Field: field.Name, Error: msg}
			}

		case strings.HasPrefix(rule, "pattern=") && kind == reflect.String:
			pattern := strings.TrimPrefix(rule, "pattern=")
			matched, err := regexp.MatchString(pattern, value)
			if err != nil || !matched {
				msg := getErrorMessage(&field, "pattern", "does not match required format")
				return &ValidationError{Field: field.Name, Error: msg}
			}

		case strings.HasPrefix(rule, "format=") && kind == reflect.String:
			format := strings.TrimPrefix(rule, "format=")
			if format == "email" {
				matched := idnEmailRegex.MatchString(value)
				if !matched {
					msg := getErrorMessage(&field, "format", "is not a valid email address")
					return &ValidationError{Field: field.Name, Error: msg}
				}
			}

		case strings.HasPrefix(rule, "enum=") && (kind == reflect.String || IsIntType(kind) || IsFloatType(kind)):
			allowed := strings.Split(strings.TrimPrefix(rule, "enum="), "|")
			found := false
			for _, a := range allowed {
				if value == a {
					found = true
					break
				}
			}
			if !found {
				msg := getErrorMessage(&field, "enum", fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
				return &ValidationError{Field: field.Name, Error: msg}
			}
		}
	}

	return nil
}

// convertStringToType converts a string value to the target reflect.Type
func convertStringToType(value string, targetType reflect.Type) (reflect.Value, error) {
	// Handle pointer types
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	// Handle special types first
	switch targetType {
	case reflect.TypeOf(time.Time{}):
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid time format: %w", err)
		}
		return reflect.ValueOf(t), nil

	case reflect.TypeOf(uuid.UUID{}):
		u, err := uuid.Parse(value)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid UUID: %w", err)
		}
		return reflect.ValueOf(u), nil
	}

	// Handle basic types
	switch targetType.Kind() {
	case reflect.String:
		return reflect.ValueOf(value), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid integer: %w", err)
		}
		convertedVal := reflect.New(targetType).Elem()
		convertedVal.SetInt(intVal)
		return convertedVal, nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid unsigned integer: %w", err)
		}
		convertedVal := reflect.New(targetType).Elem()
		convertedVal.SetUint(uintVal)
		return convertedVal, nil

	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid float: %w", err)
		}
		convertedVal := reflect.New(targetType).Elem()
		convertedVal.SetFloat(floatVal)
		return convertedVal, nil

	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid boolean: %w", err)
		}
		return reflect.ValueOf(boolVal), nil

	case reflect.Slice:
		// For slices, we assume comma-separated values
		elemType := targetType.Elem()
		strValues := strings.Split(value, ",")
		sliceVal := reflect.MakeSlice(targetType, len(strValues), len(strValues))
		for i, strVal := range strValues {
			convertedElem, err := convertStringToType(strings.TrimSpace(strVal), elemType)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("invalid slice element: %w", err)
			}
			sliceVal.Index(i).Set(convertedElem)
		}
		return sliceVal, nil

	default:
		return reflect.Value{}, fmt.Errorf("unsupported type: %s", targetType.Kind())
	}
}

// validateMapSize validates the size of a map based on validation tags
func validateMapSize(field reflect.StructField, size int) *ValidationError {
	validateTag := field.Tag.Get("validate")
	if validateTag == "" {
		return nil
	}

	rules := strings.Split(validateTag, ",")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)

		switch {
		case strings.HasPrefix(rule, ruleMinItems+"="):
			minSize, _ := strconv.Atoi(strings.TrimPrefix(rule, ruleMinItems+"="))
			if size < minSize {
				msg := getErrorMessage(&field, ruleMinItems, fmt.Sprintf("must have at least %d entries", minSize))
				return &ValidationError{Field: field.Name, Error: msg}
			}

		case strings.HasPrefix(rule, ruleMaxItems+"="):
			maxSize, _ := strconv.Atoi(strings.TrimPrefix(rule, ruleMaxItems+"="))
			if size > maxSize {
				msg := getErrorMessage(&field, ruleMaxItems, fmt.Sprintf("must have at most %d entries", maxSize))
				return &ValidationError{Field: field.Name, Error: msg}
			}

		case rule == ruleRequired && size == 0:
			msg := getErrorMessage(&field, ruleRequired, "is required and cannot be empty")
			return &ValidationError{Field: field.Name, Error: msg}
		}
	}

	return nil
}

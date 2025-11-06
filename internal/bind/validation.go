package bind

import (
	"encoding/xml"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ValidationError struct {
	XMLName xml.Name `json:"-" xml:"validationError" form:"-"`
	Field   string   `json:"field" xml:"field" form:"field"`
	Error   string   `json:"error" xml:"error" form:"error"`
}

const (
	// Validation rule names
	ruleRequired          = "required"
	ruleMin               = "min"
	ruleMax               = "max"
	ruleMultipleOf        = "multipleOf"
	ruleMinLength         = "minlength"
	ruleMaxLength         = "maxlength"
	ruleMinItems          = "minItems"
	ruleMaxItems          = "maxItems"
	ruleUniqueItems       = "uniqueItems"
	rulePattern           = "pattern"
	ruleFormat            = "format"
	ruleEnum              = "enum"
	ruleEmptyItemsAllowed = "emptyItemsAllowed"

	// Format types
	formatEmail = "email"
)

var (
	idnEmailRegex = regexp.MustCompile(
		`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?` +
			`(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$|` +
			`^[\p{L}\p{N}.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[\p{L}\p{N}](?:[\p{L}\p{N}-]{0,61}[\p{L}\p{N}])?` +
			`(?:\.[\p{L}\p{N}](?:[\p{L}\p{N}-]{0,61}[\p{L}\p{N}])?)*$`)
)

// isValidationRuleValidForType checks if a validation rule is applicable to the given field type
func isValidationRuleValidForType(rule string, kind reflect.Kind, fieldType reflect.Type) error {
	isTimeType := fieldType == reflect.TypeOf(time.Time{})
	isSliceOfString := kind == reflect.Slice && fieldType.Elem().Kind() == reflect.String
	isSliceOfTime := kind == reflect.Slice && fieldType.Elem() == reflect.TypeOf(time.Time{})
	isSliceOfInt := kind == reflect.Slice && (fieldType.Elem().Kind() == reflect.Int ||
		fieldType.Elem().Kind() == reflect.Int8 || fieldType.Elem().Kind() == reflect.Int16 ||
		fieldType.Elem().Kind() == reflect.Int32 || fieldType.Elem().Kind() == reflect.Int64)
	isSliceOfFloat := kind == reflect.Slice &&
		(fieldType.Elem().Kind() == reflect.Float32 || fieldType.Elem().Kind() == reflect.Float64)

	ruleName := rule
	if idx := strings.Index(rule, "="); idx != -1 {
		ruleName = rule[:idx]
	}

	switch ruleName {
	case ruleRequired:
		// valid for all types
		return nil

	case ruleEmptyItemsAllowed:
		// only valid for slices
		if kind != reflect.Slice {
			return fmt.Errorf("validation rule '%s' can only be applied to slice types, but field is %s", ruleEmptyItemsAllowed, kind)
		}
		return nil

	case ruleMin, ruleMax, ruleMultipleOf:
		// only valid for integers and floats
		if !IsIntType(kind) && !IsFloatType(kind) && !isSliceOfInt && !isSliceOfFloat {
			return fmt.Errorf("validation rule '%s' can only be applied to integer or float types, but field is %s", ruleName, kind)
		}
		return nil

	case ruleMinLength, ruleMaxLength:
		// only valid for strings
		if kind != reflect.String && !isSliceOfString {
			return fmt.Errorf("validation rule '%s' can only be applied to string types, but field is %s", ruleName, kind)
		}
		return nil

	case ruleMinItems, ruleMaxItems:
		// only valid for slices and maps
		if kind != reflect.Slice && kind != reflect.Map {
			return fmt.Errorf("validation rule '%s' can only be applied to slice or map types, but field is %s", ruleName, kind)
		}
		return nil

	case ruleUniqueItems:
		// only valid for slices
		if kind != reflect.Slice {
			return fmt.Errorf("validation rule '%s' can only be applied to slice types, but field is %s", ruleUniqueItems, kind)
		}
		return nil

	case rulePattern:
		// only valid for strings
		if kind != reflect.String && !isSliceOfString {
			return fmt.Errorf("validation rule '%s' can only be applied to string types, but field is %s", rulePattern, kind)
		}
		return nil

	case ruleFormat:
		// valid for strings and time.Time
		if kind != reflect.String && !isSliceOfString && !isTimeType && !isSliceOfTime {
			return fmt.Errorf("validation rule '%s' can only be applied to string or time.Time types, but field is %s", ruleFormat, fieldType)
		}
		return nil

	case ruleEnum:
		// valid for strings, integers, and floats
		if kind != reflect.String && !IsIntType(kind) && !IsFloatType(kind) && !isSliceOfString && !isSliceOfInt && !isSliceOfFloat {
			return fmt.Errorf("validation rule 'enum' can only be applied to string, integer, or float types, but field is %s", kind)
		}
		return nil

	default:
		// Unknown rule - you might want to handle this differently
		return fmt.Errorf("unknown validation rule '%s'", ruleName)
	}
}

// validateFieldTypeRules validates that all validation rules are applicable to the field type
func validateFieldTypeRules(field reflect.StructField, kind reflect.Kind, fieldType reflect.Type) {
	validateTag := field.Tag.Get("validate")
	if validateTag == "" {
		return
	}

	rules := strings.Split(validateTag, ",")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		if err := isValidationRuleValidForType(rule, kind, fieldType); err != nil {
			log.Printf("Validation rule error on field '%s': %v", field.Name, err)
		}
	}
}

func bindValidateRecursive(val reflect.Value, prefix string, errors *[]ValidationError) {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		kind := field.Kind()

		name := fieldType.Tag.Get("json")
		if name == "" {
			name = fieldType.Name
		}

		key := prefix
		if key != "" {
			key += "."
		}
		key += name

		if kind == reflect.Struct && field.Type() != reflect.TypeOf(time.Time{}) {
			bindValidateRecursive(field, key, errors)
			continue
		}

		// Validate that the validation rules are applicable to this field type
		validateFieldTypeRules(fieldType, kind, field.Type())

		validate := fieldType.Tag.Get("validate")
		if validate == "" {
			continue
		}

		rules := strings.Split(validate, ",")
		for _, rule := range rules {
			switch {
			case rule == ruleRequired:
				if isEmpty(field) {
					msg := getErrorMessage(&fieldType, ruleRequired, "is required")
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, ruleMin+"=") && IsIntType(kind):
				minVal, _ := strconv.Atoi(strings.TrimPrefix(rule, ruleMin+"="))
				if field.Int() < int64(minVal) {
					msg := getErrorMessage(&fieldType, ruleMin, fmt.Sprintf("must be ≥ %d", minVal))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, ruleMax+"=") && IsIntType(kind):
				maxVal, _ := strconv.Atoi(strings.TrimPrefix(rule, ruleMax+"="))
				if field.Int() > int64(maxVal) {
					msg := getErrorMessage(&fieldType, ruleMax, fmt.Sprintf("must be ≤ %d", maxVal))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, ruleMin+"=") && IsFloatType(kind):
				minVal, _ := strconv.ParseFloat(strings.TrimPrefix(rule, ruleMin+"="), 64)
				if field.Float() < minVal {
					msg := getErrorMessage(&fieldType, ruleMin, fmt.Sprintf("must be ≥ %f", minVal))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, ruleMax+"=") && IsFloatType(kind):
				maxVal, _ := strconv.ParseFloat(strings.TrimPrefix(rule, ruleMax+"="), 64)
				if field.Float() > maxVal {
					msg := getErrorMessage(&fieldType, ruleMax, fmt.Sprintf("must be ≤ %f", maxVal))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, ruleMultipleOf+"=") && IsIntType(kind):
				multVal, _ := strconv.Atoi(strings.TrimPrefix(rule, ruleMultipleOf+"="))
				if field.Int()%int64(multVal) != 0 {
					msg := getErrorMessage(&fieldType, ruleMultipleOf, fmt.Sprintf("must be a multiple of %d", multVal))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, ruleMultipleOf+"=") && IsFloatType(kind):
				multVal, _ := strconv.ParseFloat(strings.TrimPrefix(rule, ruleMultipleOf+"="), 64)
				if int(field.Float()*1000000)%int(multVal*1000000) != 0 {
					msg := getErrorMessage(&fieldType, ruleMultipleOf, fmt.Sprintf("must be a multiple of %f", multVal))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, ruleMinLength+"=") && kind == reflect.String:
				minLen, _ := strconv.Atoi(strings.TrimPrefix(rule, ruleMinLength+"="))
				if field.Len() < minLen {
					msg := getErrorMessage(&fieldType, ruleMinLength, fmt.Sprintf("must have at least %d characters", minLen))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, ruleMaxLength+"=") && kind == reflect.String:
				maxLen, _ := strconv.Atoi(strings.TrimPrefix(rule, ruleMaxLength+"="))
				if field.Len() > maxLen {
					msg := getErrorMessage(&fieldType, ruleMaxLength, fmt.Sprintf("must have at most %d characters", maxLen))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, ruleMinItems+"=") && kind == reflect.Slice:
				minLen, _ := strconv.Atoi(strings.TrimPrefix(rule, ruleMinItems+"="))
				if field.Len() < minLen {
					msg := getErrorMessage(&fieldType, ruleMinItems, fmt.Sprintf("must have at least %d items", minLen))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, ruleMaxItems+"=") && kind == reflect.Slice:
				maxLen, _ := strconv.Atoi(strings.TrimPrefix(rule, ruleMaxItems+"="))
				if field.Len() > maxLen {
					msg := getErrorMessage(&fieldType, ruleMaxItems, fmt.Sprintf("must have at most %d items", maxLen))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, ruleUniqueItems) && kind == reflect.Slice:
				if !hasUniqueItems(field) {
					msg := getErrorMessage(&fieldType, ruleUniqueItems, "must have unique items")
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, rulePattern+"=") && kind == reflect.String:
				pattern := strings.TrimPrefix(rule, rulePattern+"=")
				matched, err := regexp.MatchString(pattern, field.String())
				if err != nil || !matched {
					msg := getErrorMessage(&fieldType, rulePattern, "invalid format")
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, ruleFormat+"=") && kind == reflect.String:
				format := strings.TrimPrefix(rule, ruleFormat+"=")
				if format == formatEmail {
					matched := idnEmailRegex.MatchString(field.String())
					if !matched {
						msg := getErrorMessage(&fieldType, ruleFormat, "is not a valid email address")
						*errors = append(*errors, ValidationError{Field: key, Error: msg})
					}
				}

			case strings.HasPrefix(rule, ruleEnum+"=") && kind == reflect.String:
				allowed := strings.Split(strings.TrimPrefix(rule, ruleEnum+"="), "|")
				found := false
				for _, a := range allowed {
					if field.String() == a {
						found = true
						break
					}
				}
				if !found {
					msg := getErrorMessage(&fieldType, ruleEnum, fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, ruleEnum+"=") && IsIntType(kind):
				allowed := strings.Split(strings.TrimPrefix(rule, ruleEnum+"="), "|")
				found := false
				for _, a := range allowed {
					allowedVal, _ := strconv.Atoi(a)
					if field.Int() == int64(allowedVal) {
						found = true
						break
					}
				}
				if !found {
					msg := getErrorMessage(&fieldType, ruleEnum, fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, ruleEnum+"=") && IsFloatType(kind):
				allowed := strings.Split(strings.TrimPrefix(rule, ruleEnum+"="), "|")
				found := false
				for _, a := range allowed {
					allowedVal, _ := strconv.ParseFloat(a, 64)
					if field.Float() == allowedVal {
						found = true
						break
					}
				}
				if !found {
					msg := getErrorMessage(&fieldType, ruleEnum, fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}
			}
		}

		if field.Type() == reflect.TypeOf(time.Time{}) {
			v, _ := field.Interface().(time.Time)
			if err := validateTimeField(&fieldType, v); err != nil {
				*errors = append(*errors, *err)
			}
			continue
		}

		if field.Type() == reflect.SliceOf(reflect.TypeOf(time.Time{})) {
			v, _ := field.Interface().([]time.Time)
			errs := validateTimeSliceField(&fieldType, v)
			*errors = append(*errors, errs...)
			continue
		}

		if field.Type() == reflect.TypeOf(uuid.UUID{}) {
			v, _ := field.Interface().(uuid.UUID)
			if err := validateUUIDField(&fieldType, v); err != nil {
				*errors = append(*errors, *err)
			}
			continue
		}

		if field.Type() == reflect.SliceOf(reflect.TypeOf(uuid.UUID{})) {
			v, _ := field.Interface().([]uuid.UUID)
			errs := validateUUIDSliceField(&fieldType, v)
			*errors = append(*errors, errs...)
			continue
		}
	}
}

func hasUniqueItems(field reflect.Value) bool {
	itemMap := make(map[interface{}]bool)
	for i := 0; i < field.Len(); i++ {
		item := field.Index(i).Interface()
		if itemMap[item] {
			return false
		}
		itemMap[item] = true
	}
	return true
}

func validateTimeField(field *reflect.StructField, value time.Time) *ValidationError {
	if field.Type.Kind() == reflect.Slice {
		if value.IsZero() && !strings.Contains(field.Tag.Get("validate"), ruleEmptyItemsAllowed) {
			msg := getErrorMessage(field, ruleEmptyItemsAllowed+" (not set)", "empty items not allowed")
			return &ValidationError{Field: field.Name, Error: msg}
		}
	}
	// Note: 'required' validation for non-slice time fields is already handled in the main validation loop

	return nil
}

func validateTimeSliceField(field *reflect.StructField, values []time.Time) []ValidationError {
	errors := []ValidationError{}

	for _, value := range values {
		if err := validateTimeField(field, value); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

func validateUUIDField(field *reflect.StructField, value uuid.UUID) *ValidationError {
	if field.Type.Kind() == reflect.Slice {
		if value == uuid.Nil && !strings.Contains(field.Tag.Get("validate"), ruleEmptyItemsAllowed) {
			msg := getErrorMessage(field, ruleEmptyItemsAllowed+" (not set)", "empty item not allowed")
			return &ValidationError{Field: field.Name, Error: msg}
		}
	}
	// Note: 'required' validation for non-slice UUID fields is already handled in the main validation loop

	return nil
}

func validateUUIDSliceField(field *reflect.StructField, values []uuid.UUID) []ValidationError {
	errors := []ValidationError{}

	for _, value := range values {
		if err := validateUUIDField(field, value); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

func isEmpty(v reflect.Value) bool {
	if v.Type() == reflect.TypeOf(uuid.UUID{}) {
		return v.Interface().(uuid.UUID) == uuid.Nil
	}

	if v.Type() == reflect.TypeOf(time.Time{}) {
		return v.Interface().(time.Time).IsZero()
	}

	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Int, reflect.Float64:
		return v.Interface() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Struct:
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	default:
		return !v.IsValid()
	}
}

func getErrorMessage(field *reflect.StructField, rule string, fallback string) string {
	tag := field.Tag.Get("errmsg")
	if tag == "" {
		return fallback
	}

	rules := strings.Split(tag, ";")
	for _, r := range rules {
		parts := strings.SplitN(r, "=", 2)
		if len(parts) == 2 && parts[0] == rule {
			return parts[1]
		}
	}

	return fallback
}

// IsIntType returns true if the given reflect.Kind represents an integer type.
// Includes signed and unsigned integers of all sizes (int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64).
func IsIntType(kind reflect.Kind) bool {
	return kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64
}

// IsFloatType returns true if the given reflect.Kind represents a floating-point type.
// Includes float32 and float64.
func IsFloatType(kind reflect.Kind) bool {
	return kind == reflect.Float32 || kind == reflect.Float64
}

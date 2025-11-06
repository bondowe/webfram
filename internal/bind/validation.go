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

/*
type Address struct {
    Street string `form:"street" validate:"required" errmsg:"required=Street is required"`
    City   string `form:"city" validate:"required" errmsg:"required=City is required"`
    Zip    int    `form:"zip" validate:"min=10000,max=99999" errmsg:"min=Zip must be at least 10000,max=Zip must be at most 99999"`
}

type User struct {
    Name  string `form:"name" validate:"required,minlength=3" errmsg:"required=Name is required;minlength=Name must be at least 3 characters"`
    Role  string `form:"role" validate:"enum=admin|user|guest" errmsg:"enum=Role must be one of admin, user, or guest"`
    Birthdate time.Time `form:"birthdate" validate:"required,format=2006-01-02" errmsg:"required=Birthdate is required"`
    Email string `form:"email" validate:"format=email" errmsg:"format=Please enter a valid email address"`
    Hobbies []string `form:"hobbies" validate:"minItems=1,maxItems=5,emptyItemsAllowed,uniqueItems" errmsg:"minItems=At least one hobby is required;maxItems=At most 5 hobbies are allowed"`
    Address Address `form:"address" validate:"required" errmsg:"required=Address is required"`
}
Tag        							Applies to        		Description
required   							all types 				Field must be present and non-empty
emptyItemsAllowed  					slices      			Field may contain empty items in the slice
min=10      						integers, floats   		Field value must be at least 10 (inclusive).
max=100      						integers, floats   		Field value must be at most 100 (inclusive).
multipleOf=5      					integers, floats   		Field value must be a multiple of 5.
minlength=3 						string     				Field length must be at least 3 characters.
maxlength=10 						string     				Field length must be at most 10 characters.
minItems=2 							slices, maps       		Field must contain at least 2 items.
maxItems=5 							slices, maps       		Field must contain at most 5 items.
uniqueItems 						slices       			All items in the slice must be unique.
pattern=^\\d+\\.\\w{2}\\.\\w{2}$ 	string         			Field value must match the regular expression PATTERN (e.g. ^\\w+@\\w+\\.com$).
enum=val1|val2|val3 				string, int, floats 	Field value must be one of the specified values (e.g. enum=val1|val2|val3).
format=2006-01-02 					time.Time, string      	Specifies the time layout for parsing time (for form binding only, default is RFC3339) or format of strings (i.e. email).
*/

type ValidationError struct {
	XMLName xml.Name `json:"-" xml:"validationError" form:"-"`
	Field   string   `json:"field" xml:"field" form:"field"`
	Error   string   `json:"error" xml:"error" form:"error"`
}

var (
	idnEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$|^[\p{L}\p{N}.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[\p{L}\p{N}](?:[\p{L}\p{N}-]{0,61}[\p{L}\p{N}])?(?:\.[\p{L}\p{N}](?:[\p{L}\p{N}-]{0,61}[\p{L}\p{N}])?)*$`)
)

// isValidationRuleValidForType checks if a validation rule is applicable to the given field type
func isValidationRuleValidForType(rule string, kind reflect.Kind, fieldType reflect.Type) error {
	isTimeType := fieldType == reflect.TypeOf(time.Time{})
	isSliceOfString := kind == reflect.Slice && fieldType.Elem().Kind() == reflect.String
	isSliceOfTime := kind == reflect.Slice && fieldType.Elem() == reflect.TypeOf(time.Time{})
	isSliceOfInt := kind == reflect.Slice && (fieldType.Elem().Kind() == reflect.Int || fieldType.Elem().Kind() == reflect.Int8 || fieldType.Elem().Kind() == reflect.Int16 || fieldType.Elem().Kind() == reflect.Int32 || fieldType.Elem().Kind() == reflect.Int64)
	isSliceOfFloat := kind == reflect.Slice && (fieldType.Elem().Kind() == reflect.Float32 || fieldType.Elem().Kind() == reflect.Float64)

	ruleName := rule
	if idx := strings.Index(rule, "="); idx != -1 {
		ruleName = rule[:idx]
	}

	switch ruleName {
	case "required":
		// valid for all types
		return nil

	case "emptyItemsAllowed":
		// only valid for slices
		if kind != reflect.Slice {
			return fmt.Errorf("validation rule 'emptyItemsAllowed' can only be applied to slice types, but field is %s", kind)
		}
		return nil

	case "min", "max", "multipleOf":
		// only valid for integers and floats
		if !IsIntType(kind) && !IsFloatType(kind) && !isSliceOfInt && !isSliceOfFloat {
			return fmt.Errorf("validation rule '%s' can only be applied to integer or float types, but field is %s", ruleName, kind)
		}
		return nil

	case "minlength", "maxlength":
		// only valid for strings
		if kind != reflect.String && !isSliceOfString {
			return fmt.Errorf("validation rule '%s' can only be applied to string types, but field is %s", ruleName, kind)
		}
		return nil

	case "minItems", "maxItems":
		// only valid for slices and maps
		if kind != reflect.Slice && kind != reflect.Map {
			return fmt.Errorf("validation rule '%s' can only be applied to slice or map types, but field is %s", ruleName, kind)
		}
		return nil

	case "uniqueItems":
		// only valid for slices
		if kind != reflect.Slice {
			return fmt.Errorf("validation rule 'uniqueItems' can only be applied to slice types, but field is %s", kind)
		}
		return nil

	case "pattern":
		// only valid for strings
		if kind != reflect.String && !isSliceOfString {
			return fmt.Errorf("validation rule 'pattern' can only be applied to string types, but field is %s", kind)
		}
		return nil

	case "format":
		// valid for strings and time.Time
		if kind != reflect.String && !isSliceOfString && !isTimeType && !isSliceOfTime {
			return fmt.Errorf("validation rule 'format' can only be applied to string or time.Time types, but field is %s", fieldType)
		}
		return nil

	case "enum":
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
			case rule == "required":
				if isEmpty(field) {
					msg := getErrorMessage(fieldType, "required", "is required")
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "min=") && IsIntType(kind):
				minVal, _ := strconv.Atoi(strings.TrimPrefix(rule, "min="))
				if field.Int() < int64(minVal) {
					msg := getErrorMessage(fieldType, "min", fmt.Sprintf("must be ≥ %d", minVal))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "max=") && IsIntType(kind):
				maxVal, _ := strconv.Atoi(strings.TrimPrefix(rule, "max="))
				if field.Int() > int64(maxVal) {
					msg := getErrorMessage(fieldType, "max", fmt.Sprintf("must be ≤ %d", maxVal))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "min=") && IsFloatType(kind):
				minVal, _ := strconv.ParseFloat(strings.TrimPrefix(rule, "min="), 64)
				if field.Float() < minVal {
					msg := getErrorMessage(fieldType, "min", fmt.Sprintf("must be ≥ %f", minVal))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "max=") && IsFloatType(kind):
				maxVal, _ := strconv.ParseFloat(strings.TrimPrefix(rule, "max="), 64)
				if field.Float() > maxVal {
					msg := getErrorMessage(fieldType, "max", fmt.Sprintf("must be ≤ %f", maxVal))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "multipleOf=") && IsIntType(kind):
				multVal, _ := strconv.Atoi(strings.TrimPrefix(rule, "multipleOf="))
				if field.Int()%int64(multVal) != 0 {
					msg := getErrorMessage(fieldType, "multipleOf", fmt.Sprintf("must be a multiple of %d", multVal))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "multipleOf=") && IsFloatType(kind):
				multVal, _ := strconv.ParseFloat(strings.TrimPrefix(rule, "multipleOf="), 64)
				if int(field.Float()*1000000)%int(multVal*1000000) != 0 {
					msg := getErrorMessage(fieldType, "multipleOf", fmt.Sprintf("must be a multiple of %f", multVal))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "minlength=") && kind == reflect.String:
				minLen, _ := strconv.Atoi(strings.TrimPrefix(rule, "minlength="))
				if field.Len() < minLen {
					msg := getErrorMessage(fieldType, "minlength", fmt.Sprintf("must have at least %d characters", minLen))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "maxlength=") && kind == reflect.String:
				maxLen, _ := strconv.Atoi(strings.TrimPrefix(rule, "maxlength="))
				if field.Len() > maxLen {
					msg := getErrorMessage(fieldType, "maxlength", fmt.Sprintf("must have at most %d characters", maxLen))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "minItems=") && kind == reflect.Slice:
				minLen, _ := strconv.Atoi(strings.TrimPrefix(rule, "minItems="))
				if field.Len() < minLen {
					msg := getErrorMessage(fieldType, "minItems", fmt.Sprintf("must have at least %d items", minLen))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "maxItems=") && kind == reflect.Slice:
				maxLen, _ := strconv.Atoi(strings.TrimPrefix(rule, "maxItems="))
				if field.Len() > maxLen {
					msg := getErrorMessage(fieldType, "maxItems", fmt.Sprintf("must have at most %d items", maxLen))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "uniqueItems") && kind == reflect.Slice:
				if !hasUniqueItems(field) {
					msg := getErrorMessage(fieldType, "uniqueItems", "must have unique items")
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "pattern=") && kind == reflect.String:
				pattern := strings.TrimPrefix(rule, "pattern=")
				matched, err := regexp.MatchString(pattern, field.String())
				if err != nil || !matched {
					msg := getErrorMessage(fieldType, "pattern", "invalid format")
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "format=") && kind == reflect.String:
				format := strings.TrimPrefix(rule, "format=")
				switch format {
				case "email":
					matched := idnEmailRegex.MatchString(field.String())
					if !matched {
						msg := getErrorMessage(fieldType, "format", "is not a valid email address")
						*errors = append(*errors, ValidationError{Field: key, Error: msg})
					}
				}

			case strings.HasPrefix(rule, "enum=") && kind == reflect.String:
				allowed := strings.Split(strings.TrimPrefix(rule, "enum="), "|")
				found := false
				for _, a := range allowed {
					if field.String() == a {
						found = true
						break
					}
				}
				if !found {
					msg := getErrorMessage(fieldType, "enum", fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "enum=") && IsIntType(kind):
				allowed := strings.Split(strings.TrimPrefix(rule, "enum="), "|")
				found := false
				for _, a := range allowed {
					allowedVal, _ := strconv.Atoi(a)
					if field.Int() == int64(allowedVal) {
						found = true
						break
					}
				}
				if !found {
					msg := getErrorMessage(fieldType, "enum", fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}

			case strings.HasPrefix(rule, "enum=") && IsFloatType(kind):
				allowed := strings.Split(strings.TrimPrefix(rule, "enum="), "|")
				found := false
				for _, a := range allowed {
					allowedVal, _ := strconv.ParseFloat(a, 64)
					if field.Float() == allowedVal {
						found = true
						break
					}
				}
				if !found {
					msg := getErrorMessage(fieldType, "enum", fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")))
					*errors = append(*errors, ValidationError{Field: key, Error: msg})
				}
			}
		}

		if field.Type() == reflect.TypeOf(time.Time{}) {
			v, _ := field.Interface().(time.Time)
			if err := validateTimeField(fieldType, v); err != nil {
				*errors = append(*errors, *err)
			}
			continue
		}

		if field.Type() == reflect.SliceOf(reflect.TypeOf(time.Time{})) {
			v, _ := field.Interface().([]time.Time)
			errs := validateTimeSliceField(fieldType, v)
			*errors = append(*errors, errs...)
			continue
		}

		if field.Type() == reflect.TypeOf(uuid.UUID{}) {
			v, _ := field.Interface().(uuid.UUID)
			if err := validateUUIDField(fieldType, v); err != nil {
				*errors = append(*errors, *err)
			}
			continue
		}

		if field.Type() == reflect.SliceOf(reflect.TypeOf(uuid.UUID{})) {
			v, _ := field.Interface().([]uuid.UUID)
			errs := validateUUIDSliceField(fieldType, v)
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

func validateTimeField(field reflect.StructField, value time.Time) *ValidationError {
	if field.Type.Kind() == reflect.Slice {
		if value.IsZero() && !strings.Contains(field.Tag.Get("validate"), "emptyItemsAllowed") {
			msg := getErrorMessage(field, "emptyItemsAllowed (not set)", "empty items not allowed")
			return &ValidationError{Field: field.Name, Error: msg}
		}
	}
	// Note: 'required' validation for non-slice time fields is already handled in the main validation loop

	return nil
}

func validateTimeSliceField(field reflect.StructField, values []time.Time) []ValidationError {
	errors := []ValidationError{}

	for _, value := range values {
		if err := validateTimeField(field, value); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

func validateUUIDField(field reflect.StructField, value uuid.UUID) *ValidationError {
	if field.Type.Kind() == reflect.Slice {
		if value == uuid.Nil && !strings.Contains(field.Tag.Get("validate"), "emptyItemsAllowed") {
			msg := getErrorMessage(field, "emptyItemsAllowed (not set)", "empty item not allowed")
			return &ValidationError{Field: field.Name, Error: msg}
		}
	}
	// Note: 'required' validation for non-slice UUID fields is already handled in the main validation loop

	return nil
}

func validateUUIDSliceField(field reflect.StructField, values []uuid.UUID) []ValidationError {
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

func getErrorMessage(field reflect.StructField, rule string, fallback string) string {
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

func IsIntType(kind reflect.Kind) bool {
	return kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64
}

func IsFloatType(kind reflect.Kind) bool {
	return kind == reflect.Float32 || kind == reflect.Float64
}

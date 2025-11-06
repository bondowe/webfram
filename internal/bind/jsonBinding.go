package bind

import (
	"encoding/json"
	"net/http"
	"reflect"
)

// ValidateJSON validates a struct according to its validation tags.
// It recursively checks all fields and nested structs for compliance with constraints
// such as required, min, max, pattern, format, etc.
// Returns a slice of validation errors, empty if validation passes.
func ValidateJSON[T any](data *T) []ValidationError {
	val := reflect.ValueOf(data).Elem()
	errors := []ValidationError{}

	bindValidateRecursive(val, "", &errors)

	return errors
}

// JSON parses JSON from an HTTP request body and binds it to a struct of type T.
// If validate is true, performs validation according to struct tags after decoding.
// Returns the populated struct, validation errors (if validation is enabled), and a decoding error (if parsing fails).
func JSON[T any](r *http.Request, validate bool) (T, []ValidationError, error) {
	var result T
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&result); err != nil {
		return result, nil, err
	}

	if !validate {
		return result, nil, nil
	}

	val := reflect.ValueOf(&result).Elem()
	errors := []ValidationError{}

	bindValidateRecursive(val, "", &errors)

	return result, errors, nil
}

package bind

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"reflect"
)

// XML parses XML from an HTTP request body and binds it to a struct of type T.
// If validate is true, performs validation according to struct tags after decoding.
// Returns the populated struct, validation errors (if validation is enabled), and a decoding error (if parsing fails).
func XML[T any](r *http.Request, validate bool) (T, []ValidationError, error) {
	var result T
	decoder := xml.NewDecoder(r.Body)
	err := decoder.Decode(&result)
	if err != nil {
		return result, nil, fmt.Errorf("failed to decode XML: %w", err)
	}

	if !validate {
		return result, nil, nil
	}

	val := reflect.ValueOf(&result).Elem()
	errors := []ValidationError{}

	bindValidateRecursive(val, "", &errors)

	return result, errors, nil
}

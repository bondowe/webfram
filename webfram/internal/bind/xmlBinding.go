package bind

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"reflect"
)

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

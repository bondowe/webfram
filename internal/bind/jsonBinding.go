package bind

import (
	"encoding/json"
	"net/http"
	"reflect"
)

func ValidateJSON[T any](data *T) []ValidationError {
	val := reflect.ValueOf(data).Elem()
	errors := []ValidationError{}

	bindValidateRecursive(val, "", &errors)

	return errors
}

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

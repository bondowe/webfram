package bind

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newPost(values url.Values) *http.Request {
	body := values.Encode()
	r, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestFormBinding_BasicTypes(t *testing.T) {
	type Person struct {
		Name   string `form:"name"   validate:"required"`
		Age    int    `form:"age"    validate:"min=1"`
		Active bool   `form:"active"`
	}

	values := url.Values{
		"name":   {"Alice"},
		"age":    {"25"},
		"active": {"true"},
	}
	req := newPost(values)

	res, errs, err := Form[Person](req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors, got: %#v", errs)
	}
	if res.Name != "Alice" || res.Age != 25 || res.Active != true {
		t.Fatalf("unexpected binding result: %#v", res)
	}
}

func TestFormBinding_SliceUniqueAndLengthValidation(t *testing.T) {
	type S struct {
		Tags []string `form:"tags" validate:"uniqueItems,minItems=1"`
	}

	values := url.Values{
		"tags": {"go", "go"},
	}
	req := newPost(values)

	res, errs, err := Form[S](req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// uniqueItems should trigger one validation error
	if len(errs) == 0 {
		t.Fatalf("expected validation errors for uniqueItems, got none")
	}
	found := false
	for _, e := range errs {
		if e.Field == "Tags" && strings.Contains(e.Error, "unique") {
			found = true
		}
	}
	if !found {
		t.Fatalf("uniqueItems error not found in %#v", errs)
	}
	// binding should still populate the slice
	if len(res.Tags) != 2 || res.Tags[0] != "go" || res.Tags[1] != "go" {
		t.Fatalf("unexpected slice binding: %#v", res.Tags)
	}
}

func TestFormBinding_EqualsValidation_String(t *testing.T) {
	type T struct {
		Status string `form:"status" validate:"equals=active"`
	}

	tests := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"valid_equals", "active", false},
		{"invalid_equals", "inactive", true},
		{"empty_fails", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{"status": {tt.value}}
			req := newPost(values)

			_, errs, err := Form[T](req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectErr && len(errs) == 0 {
				t.Error("expected validation error but got none")
			}
			if !tt.expectErr && len(errs) > 0 {
				t.Errorf("expected no errors, got: %v", errs)
			}
		})
	}
}

func TestFormBinding_EqualsValidation_Int(t *testing.T) {
	type T struct {
		Count int `form:"count" validate:"equals=42"`
	}

	tests := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"valid_equals", "42", false},
		{"invalid_equals", "41", true},
		{"zero_fails", "0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{"count": {tt.value}}
			req := newPost(values)

			_, errs, err := Form[T](req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectErr && len(errs) == 0 {
				t.Error("expected validation error but got none")
			}
			if !tt.expectErr && len(errs) > 0 {
				t.Errorf("expected no errors, got: %v", errs)
			}
		})
	}
}

func TestFormBinding_EqualsValidation_Float(t *testing.T) {
	type T struct {
		Price float64 `form:"price" validate:"equals=19.99"`
	}

	tests := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"valid_equals", "19.99", false},
		{"invalid_equals", "20.00", true},
		{"zero_fails", "0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{"price": {tt.value}}
			req := newPost(values)

			_, errs, err := Form[T](req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectErr && len(errs) == 0 {
				t.Error("expected validation error but got none")
			}
			if !tt.expectErr && len(errs) > 0 {
				t.Errorf("expected no errors, got: %v", errs)
			}
		})
	}
}

func TestFormBinding_URLFormatValidation(t *testing.T) {
	type T struct {
		Website string `form:"website" validate:"format=url"`
	}

	tests := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"valid_http", "http://example.com", false},
		{"valid_https", "https://example.com", false},
		{"valid_with_path", "https://example.com/path/to/page", false},
		{"valid_with_query", "https://example.com?query=param", false},
		{"invalid_no_protocol", "example.com", true},
		{"invalid_ftp", "ftp://example.com", true},
		{"invalid_empty", "", true},
		{"invalid_malformed", "not a url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{"website": {tt.value}}
			req := newPost(values)

			_, errs, err := Form[T](req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectErr && len(errs) == 0 {
				t.Errorf("expected validation error for %q but got none", tt.value)
			}
			if !tt.expectErr && len(errs) > 0 {
				t.Errorf("expected no errors for %q, got: %v", tt.value, errs)
			}
		})
	}
}

func TestFormBinding_UUIDAndTime(t *testing.T) {
	type T struct {
		Times []time.Time `form:"times" validate:"minItems=1"`
		ID    uuid.UUID   `form:"id"    validate:"required"`
	}

	values := url.Values{
		"id":    {"not-a-uuid"},
		"times": {"2020-01-01T00:00:00Z"},
	}
	req := newPost(values)

	res, errs, err := Form[T](req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// invalid UUID should produce an error
	if len(errs) == 0 {
		t.Fatalf("expected validation error for invalid UUID, got none")
	}
	hasUUIDErr := false
	for _, e := range errs {
		if e.Field == "ID" {
			hasUUIDErr = true
		}
	}
	if !hasUUIDErr {
		t.Fatalf("expected UUID error in %#v", errs)
	}
	// times should be parsed correctly
	if len(res.Times) != 1 {
		t.Fatalf("expected one time parsed, got %#v", res.Times)
	}
	if res.Times[0].Year() != 2020 {
		t.Fatalf("unexpected parsed time: %#v", res.Times[0])
	}
}

func TestFormBinding_MapBindingAndValidation(t *testing.T) {
	type M struct {
		Meta map[string]int `form:"metadata" validate:"minItems=1"`
	}

	values := url.Values{
		"metadata[color]": {"5"},
		"metadata[size]":  {"10"},
	}
	req := newPost(values)

	res, errs, err := Form[M](req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected validation errors: %#v", errs)
	}
	if res.Meta == nil {
		t.Fatalf("expected map to be initialized")
	}
	if res.Meta["color"] != 5 || res.Meta["size"] != 10 {
		t.Fatalf("unexpected map values: %#v", res.Meta)
	}
}

func TestFormBinding_NestedStruct(t *testing.T) {
	type Parent struct {
		Child struct {
			Field string `form:"field" validate:"required"`
		} `form:"child"`
	}

	values := url.Values{
		"child.field": {"nested value"},
	}
	req := newPost(values)

	res, errs, err := Form[Parent](req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected validation errors: %#v", errs)
	}
	if res.Child.Field != "nested value" {
		t.Fatalf("nested field not bound correctly, got: %q", res.Child.Field)
	}
}

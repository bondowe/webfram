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
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestFormBinding_BasicTypes(t *testing.T) {
	type Person struct {
		Name   string `form:"name" validate:"required"`
		Age    int    `form:"age" validate:"min=1"`
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

func TestFormBinding_UUIDAndTime(t *testing.T) {
	type T struct {
		Times []time.Time `form:"times" validate:"minItems=1"`
		ID    uuid.UUID   `form:"id" validate:"required"`
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

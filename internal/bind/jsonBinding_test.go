package bind

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONDecodeSuccess_NoValidation(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		Age  int    `json:"age,omitempty"`
	}

	body := `{"name":"Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))

	got, errs, err := JSON[payload](req, false)
	if err != nil {
		t.Fatalf("expected no error decoding JSON, got: %v", err)
	}
	if errs != nil {
		t.Fatalf("expected nil validation errors when validate=false, got: %v", errs)
	}
	if got.Name != "Alice" {
		t.Fatalf("expected Name to be Alice, got: %s", got.Name)
	}
}

func TestJSONDecodeSuccess_WithValidation(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	body := `{"name":"Bob"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))

	got, errs, err := JSON[payload](req, true)
	if err != nil {
		t.Fatalf("expected no error decoding JSON, got: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors for simple payload, got: %v", errs)
	}
	if got.Name != "Bob" {
		t.Fatalf("expected Name to be Bob, got: %s", got.Name)
	}
}

func TestJSONDisallowUnknownFields_ReturnsError(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	// includes an unknown field "extra" which should trigger DisallowUnknownFields
	body := `{"name":"Carol","extra":"value"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))

	_, errs, err := JSON[payload](req, true)
	if err == nil {
		t.Fatalf("expected error due to unknown field, got nil")
	}
	if len(errs) != 0 {
		t.Fatalf("expected validation errors to be nil when decode fails, got: %v", errs)
	}
}

func TestJSONInvalidJSON_ReturnsError(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	body := `{"name":"MissingEnd"`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))

	_, errs, err := JSON[payload](req, false)
	if err == nil {
		t.Fatalf("expected error for invalid JSON, got nil")
	}
	if len(errs) != 0 {
		t.Fatalf("expected validation errors to be nil when decode fails, got: %v", errs)
	}
}

func TestValidateJSON_PointerInput(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	v := payload{Name: "Eve"}
	errs := ValidateJSON(&v)
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors for simple payload, got: %v", errs)
	}
}

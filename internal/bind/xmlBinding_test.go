package bind

import (
	"net/http"
	"strings"
	"testing"
)

type person struct {
	Name string `xml:"Name"`
	Age  int    `xml:"Age"`
}

func TestXMLDecode_Success_NoValidate(t *testing.T) {
	xml := `<person><Name>John</Name><Age>30</Age></person>`
	req, err := http.NewRequest("POST", "/", strings.NewReader(xml))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	got, errs, err := XML[person](req, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if errs != nil {
		t.Fatalf("expected nil errors when validate=false, got: %#v", errs)
	}
	if got.Name != "John" {
		t.Fatalf("expected Name=John, got %q", got.Name)
	}
	if got.Age != 30 {
		t.Fatalf("expected Age=30, got %d", got.Age)
	}
}

func TestXMLDecode_BadXML_ReturnsError(t *testing.T) {
	// malformed XML
	xml := `<person><Name>John</Name>`
	req, err := http.NewRequest("POST", "/", strings.NewReader(xml))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, _, err = XML[person](req, false)
	if err == nil {
		t.Fatalf("expected error for malformed XML, got nil")
	}
}

func TestXMLDecode_ValidateTrue_ErrSliceNonNil(t *testing.T) {
	xml := `<person><Name>Jane</Name><Age>25</Age></person>`
	req, err := http.NewRequest("POST", "/", strings.NewReader(xml))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	got, errs, err := XML[person](req, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When validate=true the function returns a (possibly empty) slice of ValidationError.
	// It should be a non-nil slice (per implementation).
	if errs == nil {
		t.Fatalf("expected non-nil errors slice when validate=true, got nil")
	}
	if got.Name != "Jane" || got.Age != 25 {
		t.Fatalf("unexpected decoded value: %+v", got)
	}
}

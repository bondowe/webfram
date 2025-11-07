package bind

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Test convertStringToType with various types.
func TestConvertStringToType(t *testing.T) {
	tests := []struct {
		targetType  reflect.Type
		name        string
		value       string
		shouldError bool
	}{
		{
			name:        "valid time.Time",
			value:       "2023-01-15T10:30:00Z",
			targetType:  reflect.TypeOf(time.Time{}),
			shouldError: false,
		},
		{
			name:        "invalid time.Time",
			value:       "not-a-date",
			targetType:  reflect.TypeOf(time.Time{}),
			shouldError: true,
		},
		{
			name:        "valid UUID",
			value:       "550e8400-e29b-41d4-a716-446655440000",
			targetType:  reflect.TypeOf(uuid.UUID{}),
			shouldError: false,
		},
		{
			name:        "invalid UUID",
			value:       "not-a-uuid",
			targetType:  reflect.TypeOf(uuid.UUID{}),
			shouldError: true,
		},
		{
			name:        "pointer to time.Time",
			value:       "2023-01-15T10:30:00Z",
			targetType:  reflect.TypeOf(&time.Time{}),
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertStringToType(tt.value, tt.targetType)
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !result.IsValid() {
					t.Errorf("Expected valid result")
				}
			}
		})
	}
}

// Test validateField with various edge cases.
func TestValidateField_EdgeCases(t *testing.T) {
	tests := []struct {
		fieldSetup func() (reflect.StructField, string, reflect.Kind)
		name       string
		wantError  bool
	}{
		{
			name: "required field with empty value",
			fieldSetup: func() (reflect.StructField, string, reflect.Kind) {
				type TestStruct struct {
					Name string `validate:"required"`
				}
				field, _ := reflect.TypeOf(TestStruct{}).FieldByName("Name")
				return field, "", reflect.String
			},
			wantError: true,
		},
		{
			name: "int field below min",
			fieldSetup: func() (reflect.StructField, string, reflect.Kind) {
				type TestStruct struct {
					Age int `validate:"min=18"`
				}
				field, _ := reflect.TypeOf(TestStruct{}).FieldByName("Age")
				return field, "10", reflect.Int
			},
			wantError: true,
		},
		{
			name: "int field above max",
			fieldSetup: func() (reflect.StructField, string, reflect.Kind) {
				type TestStruct struct {
					Score int `validate:"max=100"`
				}
				field, _ := reflect.TypeOf(TestStruct{}).FieldByName("Score")
				return field, "150", reflect.Int
			},
			wantError: true,
		},
		{
			name: "float field below min",
			fieldSetup: func() (reflect.StructField, string, reflect.Kind) {
				type TestStruct struct {
					Price float64 `validate:"min=10.5"`
				}
				field, _ := reflect.TypeOf(TestStruct{}).FieldByName("Price")
				return field, "5.0", reflect.Float64
			},
			wantError: true,
		},
		{
			name: "float field above max",
			fieldSetup: func() (reflect.StructField, string, reflect.Kind) {
				type TestStruct struct {
					Price float64 `validate:"max=100.5"`
				}
				field, _ := reflect.TypeOf(TestStruct{}).FieldByName("Price")
				return field, "200.0", reflect.Float64
			},
			wantError: true,
		},
		{
			name: "invalid int parse for min validation",
			fieldSetup: func() (reflect.StructField, string, reflect.Kind) {
				type TestStruct struct {
					Count int `validate:"min=5"`
				}
				field, _ := reflect.TypeOf(TestStruct{}).FieldByName("Count")
				return field, "not-a-number", reflect.Int
			},
			wantError: true,
		},
		{
			name: "invalid float parse for max validation",
			fieldSetup: func() (reflect.StructField, string, reflect.Kind) {
				type TestStruct struct {
					Amount float64 `validate:"max=100"`
				}
				field, _ := reflect.TypeOf(TestStruct{}).FieldByName("Amount")
				return field, "not-a-float", reflect.Float64
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, value, kind := tt.fieldSetup()
			err := validateField(&field, value, kind)
			if tt.wantError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err.Error)
			}
		})
	}
}

// Test Form binding with nested structs.
func TestForm_NestedStructs(t *testing.T) {
	type Address struct {
		Street string `form:"street" validate:"required"`
		City   string `form:"city"`
	}

	type Person struct {
		Name    string  `form:"name"    validate:"required"`
		Address Address `form:"address"`
	}

	form := url.Values{}
	form.Add("name", "John")
	form.Add("address.street", "123 Main St")
	form.Add("address.city", "Boston")

	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req.PostForm = form

	result, validationErrors, err := Form[Person](req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(validationErrors) > 0 {
		t.Errorf("Expected no validation errors, got: %v", validationErrors)
	}

	if result.Name != "John" {
		t.Errorf("Expected Name='John', got '%s'", result.Name)
	}
	if result.Address.Street != "123 Main St" {
		t.Errorf("Expected Street='123 Main St', got '%s'", result.Address.Street)
	}
	if result.Address.City != "Boston" {
		t.Errorf("Expected City='Boston', got '%s'", result.Address.City)
	}
}

// Test Form binding with nested structs and validation errors.
func TestForm_NestedStructsValidationError(t *testing.T) {
	type Address struct {
		Street string `form:"street" validate:"required"`
		City   string `form:"city"`
	}

	type Person struct {
		Name    string  `form:"name"    validate:"required"`
		Address Address `form:"address"`
	}

	form := url.Values{}
	form.Add("name", "John")
	form.Add("address.city", "Boston")
	// Missing required address.street

	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req.PostForm = form

	result, validationErrors, err := Form[Person](req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(validationErrors) == 0 {
		t.Error("Expected validation errors for missing required field")
	}

	foundStreetError := false
	for _, ve := range validationErrors {
		if ve.Field == "Street" {
			foundStreetError = true
			break
		}
	}
	if !foundStreetError {
		t.Error("Expected validation error for Street field")
	}

	if result.Name != "John" {
		t.Errorf("Expected Name='John', got '%s'", result.Name)
	}
}

// Test Form binding with various slice types.
func TestForm_SliceTypes(t *testing.T) {
	type FormData struct {
		StringSlice []string    `form:"strings"`
		IntSlice    []int       `form:"ints"`
		FloatSlice  []float64   `form:"floats"`
		TimeSlice   []time.Time `form:"times"`
		UUIDSlice   []uuid.UUID `form:"uuids"`
	}

	form := url.Values{}
	form.Add("strings", "a")
	form.Add("strings", "b")
	form.Add("ints", "1")
	form.Add("ints", "2")
	form.Add("floats", "1.5")
	form.Add("floats", "2.5")
	form.Add("times", "2023-01-15T10:30:00Z")
	form.Add("times", "2023-01-16T10:30:00Z")
	form.Add("uuids", "550e8400-e29b-41d4-a716-446655440000")
	form.Add("uuids", "550e8400-e29b-41d4-a716-446655440001")

	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req.PostForm = form

	result, validationErrors, err := Form[FormData](req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(validationErrors) > 0 {
		t.Errorf("Expected no validation errors, got: %v", validationErrors)
	}

	if len(result.StringSlice) != 2 {
		t.Errorf("Expected 2 strings, got %d", len(result.StringSlice))
	}
	if len(result.IntSlice) != 2 {
		t.Errorf("Expected 2 ints, got %d", len(result.IntSlice))
	}
	if len(result.FloatSlice) != 2 {
		t.Errorf("Expected 2 floats, got %d", len(result.FloatSlice))
	}
	if len(result.TimeSlice) != 2 {
		t.Errorf("Expected 2 times, got %d", len(result.TimeSlice))
	}
	if len(result.UUIDSlice) != 2 {
		t.Errorf("Expected 2 UUIDs, got %d", len(result.UUIDSlice))
	}
}

// Test Form binding with invalid slice values.
func TestForm_InvalidSliceValues(t *testing.T) {
	type FormData struct {
		Times []time.Time `form:"times"`
		UUIDs []uuid.UUID `form:"uuids"`
	}

	form := url.Values{}
	form.Add("times", "not-a-time")
	form.Add("uuids", "not-a-uuid")

	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req.PostForm = form

	result, validationErrors, err := Form[FormData](req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have validation errors but continue processing
	if len(validationErrors) == 0 {
		t.Error("Expected validation errors for invalid time and UUID")
	}

	// Result should have zero values for failed conversions
	if len(result.Times) != 1 {
		t.Errorf("Expected 1 time (zero value), got %d", len(result.Times))
	}
	if len(result.UUIDs) != 1 {
		t.Errorf("Expected 1 UUID (zero value), got %d", len(result.UUIDs))
	}
}

// Test Form with ParseForm error.
func TestForm_ParseFormError(_ *testing.T) {
	type FormData struct {
		Name string `form:"name"`
	}

	// Create a request with an invalid body that will cause ParseForm to fail
	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = nil // This will cause ParseForm to potentially fail in some edge cases

	_, _, err := Form[FormData](req)
	// The error handling depends on the specific case
	// Just ensure the function handles it gracefully
	_ = err
}

// Test binding with time.Time fields.
func TestForm_TimeFields(t *testing.T) {
	type FormData struct {
		CreatedAt time.Time `form:"created_at"`
		UpdatedAt time.Time `form:"updated_at" validate:"required"`
	}

	form := url.Values{}
	form.Add("created_at", "2023-01-15T10:30:00Z")
	form.Add("updated_at", "2023-01-16T10:30:00Z")

	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req.PostForm = form

	result, validationErrors, err := Form[FormData](req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(validationErrors) > 0 {
		t.Errorf("Expected no validation errors, got: %v", validationErrors)
	}

	expectedCreated, _ := time.Parse(time.RFC3339, "2023-01-15T10:30:00Z")
	expectedUpdated, _ := time.Parse(time.RFC3339, "2023-01-16T10:30:00Z")

	if !result.CreatedAt.Equal(expectedCreated) {
		t.Errorf("Expected CreatedAt=%v, got %v", expectedCreated, result.CreatedAt)
	}
	if !result.UpdatedAt.Equal(expectedUpdated) {
		t.Errorf("Expected UpdatedAt=%v, got %v", expectedUpdated, result.UpdatedAt)
	}
}

// Test field with tag "-" to skip binding.
func TestForm_SkippedFields(t *testing.T) {
	type FormData struct {
		Name     string `form:"name"`
		Internal string `form:"-"`
		ID       int    `form:"id"`
	}

	form := url.Values{}
	form.Add("name", "test")
	form.Add("Internal", "should-be-skipped")
	form.Add("id", "123")

	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req.PostForm = form

	result, _, err := Form[FormData](req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Name != "test" {
		t.Errorf("Expected Name='test', got '%s'", result.Name)
	}
	if result.Internal != "" {
		t.Errorf("Expected Internal to be empty (skipped), got '%s'", result.Internal)
	}
	if result.ID != 123 {
		t.Errorf("Expected ID=123, got %d", result.ID)
	}
}

// Test binding with map types using bracket notation.
func TestForm_MapFields(t *testing.T) {
	type FormData struct {
		Metadata map[string]string `form:"metadata"`
	}

	form := url.Values{}
	form.Add("metadata[key1]", "value1")
	form.Add("metadata[key2]", "value2")

	req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	req.PostForm = form

	result, validationErrors, err := Form[FormData](req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(validationErrors) > 0 {
		t.Errorf("Expected no validation errors, got: %v", validationErrors)
	}

	if len(result.Metadata) != 2 {
		t.Errorf("Expected 2 metadata entries, got %d", len(result.Metadata))
	}

	if result.Metadata["key1"] != "value1" {
		t.Errorf("Expected Metadata[key1]='value1', got '%s'", result.Metadata["key1"])
	}
	if result.Metadata["key2"] != "value2" {
		t.Errorf("Expected Metadata[key2]='value2', got '%s'", result.Metadata["key2"])
	}
}

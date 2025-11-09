package bind

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

func runValidate(v interface{}) []ValidationError {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	errs := []ValidationError{}
	bindValidateRecursive(val, "", &errs)
	return errs
}

func findByField(errs []ValidationError, field string) *ValidationError {
	for _, e := range errs {
		if e.Field == field {
			return &e
		}
	}
	return nil
}

func TestUnsignedIntegerValidation(t *testing.T) {
	type UintStruct struct {
		Count    uint   `json:"count"    validate:"min=10,max=100"`
		Port     uint16 `json:"port"     validate:"min=1024,max=65535"`
		Age      uint8  `json:"age"      validate:"min=18,max=120"`
		ID       uint64 `json:"id"       validate:"min=1"`
		Multiple uint32 `json:"multiple" validate:"multipleOf=5"`
		Status   uint   `json:"status"   validate:"enum=0|1|2"`
	}

	// Test valid values
	validStruct := UintStruct{
		Count:    50,
		Port:     8080,
		Age:      25,
		ID:       1000,
		Multiple: 15,
		Status:   1,
	}
	errs := runValidate(validStruct)
	if len(errs) > 0 {
		t.Errorf("expected no errors for valid unsigned integers, got: %+v", errs)
	}

	// Test violations
	invalidStruct := UintStruct{
		Count:    5,   // min violation
		Port:     100, // min violation
		Age:      200, // max violation (exceeds uint8 range in validation)
		ID:       0,   // min violation
		Multiple: 12,  // multipleOf violation
		Status:   99,  // enum violation
	}
	errs = runValidate(invalidStruct)
	if len(errs) != 6 {
		t.Errorf("expected 6 errors for invalid unsigned integers, got %d: %+v", len(errs), errs)
	}

	// Verify specific errors
	if e := findByField(errs, "count"); e == nil {
		t.Error("expected error for count field")
	}
	if e := findByField(errs, "port"); e == nil {
		t.Error("expected error for port field")
	}
	if e := findByField(errs, "age"); e == nil {
		t.Error("expected error for age field")
	}
	if e := findByField(errs, "id"); e == nil {
		t.Error("expected error for id field")
	}
	if e := findByField(errs, "multiple"); e == nil {
		t.Error("expected error for multiple field")
	}
	if e := findByField(errs, "status"); e == nil {
		t.Error("expected error for status field")
	}
}

func TestRequiredAndMinIntValidation(t *testing.T) {
	type User struct {
		Name string `json:"name" validate:"required"      errmsg:"required=Name is required"`
		Age  int    `json:"age"  validate:"min=18,max=65" errmsg:"min=Age must be at least 18;max=Age must be at most 65"`
	}

	u := User{
		Name: "",
		Age:  16,
	}

	errs := runValidate(u)
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d: %+v", len(errs), errs)
	}

	if e := findByField(errs, "name"); e == nil {
		t.Errorf("expected error for field 'name'")
	} else if e.Error != "Name is required" {
		t.Errorf("unexpected error message for name: %s", e.Error)
	}

	if e := findByField(errs, "age"); e == nil {
		t.Errorf("expected error for field 'age'")
	} else if e.Error != "Age must be at least 18" {
		t.Errorf("unexpected error message for age: %s", e.Error)
	}
}

func TestUniqueItemsValidation(t *testing.T) {
	type S struct {
		Items []string `json:"items" validate:"uniqueItems" errmsg:"uniqueItems=Items must be unique"`
	}

	s := S{
		Items: []string{"a", "b", "a"},
	}

	errs := runValidate(s)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %+v", len(errs), errs)
	}

	if e := findByField(errs, "items"); e == nil {
		t.Errorf("expected error for field 'items'")
	} else if e.Error != "Items must be unique" {
		t.Errorf("unexpected error message for items: %s", e.Error)
	}
}

func TestFormatEmailValidation(t *testing.T) {
	type E struct {
		Email string `json:"email" validate:"format=email" errmsg:"format=Please enter a valid email address"`
	}

	e := E{
		Email: "not-an-email",
	}

	errs := runValidate(e)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %+v", len(errs), errs)
	}

	if ev := findByField(errs, "email"); ev == nil {
		t.Errorf("expected error for field 'email'")
	} else if ev.Error != "Please enter a valid email address" {
		t.Errorf("unexpected error message for email: %s", ev.Error)
	}
}

func TestTimeRequiredValidation(t *testing.T) {
	type T struct {
		Birthdate time.Time `json:"birthdate" validate:"required,format=2006-01-02" errmsg:"required=Birthdate is required"`
	}

	tu := T{
		Birthdate: time.Time{},
	}

	errs := runValidate(tu)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %+v", len(errs), errs)
	}

	if ev := findByField(errs, "birthdate"); ev == nil {
		t.Errorf("expected error for field 'birthdate'")
	} else if ev.Error != "Birthdate is required" {
		t.Errorf("unexpected error message for birthdate: %s", ev.Error)
	}
}

func TestUUIDRequiredValidation(t *testing.T) {
	type R struct {
		ID uuid.UUID `json:"id" validate:"required" errmsg:"required=ID is required"`
	}

	r := R{
		ID: uuid.Nil,
	}

	errs := runValidate(r)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %+v", len(errs), errs)
	}

	if ev := findByField(errs, "id"); ev == nil {
		t.Errorf("expected error for field 'id'")
	} else if ev.Error != "ID is required" {
		t.Errorf("unexpected error message for id: %s", ev.Error)
	}
}

func TestMultipleRulesCombination(t *testing.T) {
	type X struct {
		Title string  `json:"title" validate:"required,minlength=3,maxlength=10" errmsg:"required=Required;minlength=Short;maxlength=Long"`
		Role  string  `json:"role"  validate:"enum=admin|user|guest"             errmsg:"enum=Invalid"`
		Nums  []int   `json:"nums"  validate:"minItems=1,maxItems=3,uniqueItems" errmsg:"minItems=Need 1;maxItems=Max 3;uniqueItems=Unique"`
		Score float64 `json:"score" validate:"min=0.5,max=10"                    errmsg:"min=Too low;max=Too high"`
	}

	x := X{
		Title: "ab",        // minlength violation
		Nums:  []int{},     // minItems violation
		Role:  "superuser", // enum violation
		Score: 0.1,         // min violation for float
	}

	errs := runValidate(x)
	if len(errs) != 4 {
		t.Fatalf("expected 4 errors, got %d: %+v", len(errs), errs)
	}

	if e := findByField(errs, "title"); e == nil || e.Error != "Short" {
		t.Errorf("title error missing or unexpected: %+v", e)
	}
	if e := findByField(errs, "nums"); e == nil || e.Error != "Need 1" {
		t.Errorf("nums error missing or unexpected: %+v", e)
	}
	if e := findByField(errs, "role"); e == nil || e.Error != "Invalid" {
		t.Errorf("role error missing or unexpected: %+v", e)
	}
	if e := findByField(errs, "score"); e == nil {
		t.Errorf("score error missing")
	}
}

// TestValidateTimeSliceField tests time slice validation.
func TestValidateTimeSliceField(t *testing.T) {
	type TimeSliceStruct struct {
		Dates []time.Time `json:"dates" validate:"required,format=2006-01-02"`
	}

	// Valid case - all dates are valid
	validTime := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
	s := TimeSliceStruct{
		Dates: []time.Time{validTime, validTime},
	}
	errs := runValidate(s)
	if len(errs) > 0 {
		t.Errorf("expected no errors for valid time slice, got: %+v", errs)
	}

	// Test with required validation on empty slice
	emptyS := TimeSliceStruct{
		Dates: []time.Time{},
	}
	errs = runValidate(emptyS)
	if len(errs) == 0 {
		t.Error("expected error for empty required time slice")
	}
}

// TestValidateUUIDSliceField tests UUID slice validation.
func TestValidateUUIDSliceField(t *testing.T) {
	type UUIDSliceStruct struct {
		IDs []uuid.UUID `json:"ids" validate:"required"`
	}

	// Valid case
	validUUID := uuid.New()
	s := UUIDSliceStruct{
		IDs: []uuid.UUID{validUUID, validUUID},
	}
	errs := runValidate(s)
	if len(errs) > 0 {
		t.Errorf("expected no errors for valid UUID slice, got: %+v", errs)
	}

	// Empty slice should trigger required error
	emptyS := UUIDSliceStruct{
		IDs: []uuid.UUID{},
	}
	errs = runValidate(emptyS)
	if len(errs) == 0 {
		t.Error("expected error for empty required UUID slice")
	}
}

// TestValidateWithEmptyItems tests validation with emptyItemsAllowed.
func TestValidateWithEmptyItems(t *testing.T) {
	type SliceStruct struct {
		UUIDs []uuid.UUID `json:"uuids" validate:"emptyItemsAllowed"`
	}

	// Should allow nil UUID in slice with emptyItemsAllowed
	s := SliceStruct{
		UUIDs: []uuid.UUID{uuid.Nil, uuid.New()},
	}
	errs := runValidate(s)
	if len(errs) > 0 {
		t.Errorf("expected no errors with emptyItemsAllowed, got: %+v", errs)
	}
}

// TestValidationRuleValidForType tests error cases in rule validation.
func TestValidationRuleValidForType(t *testing.T) {
	tests := []struct {
		name        string
		rule        string
		kind        reflect.Kind
		fieldType   reflect.Type
		expectError bool
	}{
		{"emptyItemsAllowed on non-slice", "emptyItemsAllowed", reflect.String, reflect.TypeOf(""), true},
		{"min on string", "min=5", reflect.String, reflect.TypeOf(""), true},
		{"minlength on int", "minlength=5", reflect.Int, reflect.TypeOf(0), true},
		{"minItems on string", "minItems=1", reflect.String, reflect.TypeOf(""), true},
		{"uniqueItems on non-slice", "uniqueItems", reflect.String, reflect.TypeOf(""), true},
		{"pattern on int", "pattern=\\d+", reflect.Int, reflect.TypeOf(0), true},
		{"format on int", "format=email", reflect.Int, reflect.TypeOf(0), true},
		{"enum on bool", "enum=true|false", reflect.Bool, reflect.TypeOf(false), true},
		{"valid min on int", "min=5", reflect.Int, reflect.TypeOf(0), false},
		{"unknown rule", "unknownRule=value", reflect.String, reflect.TypeOf(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isValidationRuleValidForType(tt.rule, tt.kind, tt.fieldType)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestIsEmpty tests the isEmpty helper function.
func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		val      reflect.Value
		expected bool
	}{
		{"empty string", reflect.ValueOf(""), true},
		{"non-empty string", reflect.ValueOf("hello"), false},
		{"zero int", reflect.ValueOf(0), true},
		{"non-zero int", reflect.ValueOf(42), false},
		{"non-zero float64", reflect.ValueOf(float64(3.14)), false},
		{"false bool", reflect.ValueOf(false), true},
		{"true bool", reflect.ValueOf(true), false},
		{"nil slice", reflect.ValueOf([]int(nil)), true},
		{"empty slice", reflect.ValueOf([]int{}), true},
		{"non-empty slice", reflect.ValueOf([]int{1, 2}), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEmpty(tt.val)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestGetErrorMessage tests custom error message retrieval.
func TestGetErrorMessage(t *testing.T) {
	// Test with custom error message
	fieldWithMsg := reflect.StructField{
		Name: "Email",
		Type: reflect.TypeOf(""),
		Tag:  reflect.StructTag(`errmsg:"required=Email is required;format=Invalid email"`),
	}

	msg := getErrorMessage(&fieldWithMsg, "required", "default message")
	if msg != "Email is required" {
		t.Errorf("expected 'Email is required', got '%s'", msg)
	}

	msg = getErrorMessage(&fieldWithMsg, "format", "default message")
	if msg != "Invalid email" {
		t.Errorf("expected 'Invalid email', got '%s'", msg)
	}

	// Test with missing rule - should return default
	msg = getErrorMessage(&fieldWithMsg, "min", "default message")
	if msg != "default message" {
		t.Errorf("expected 'default message', got '%s'", msg)
	}

	// Test without errmsg tag - should return default
	fieldWithoutMsg := reflect.StructField{
		Name: "Name",
		Type: reflect.TypeOf(""),
		Tag:  reflect.StructTag(``),
	}
	msg = getErrorMessage(&fieldWithoutMsg, "required", "default message")
	if msg != "default message" {
		t.Errorf("expected 'default message', got '%s'", msg)
	}
}

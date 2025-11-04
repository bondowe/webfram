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

func TestRequiredAndMinIntValidation(t *testing.T) {
	type User struct {
		Name string `json:"name" validate:"required" errmsg:"required=Name is required"`
		Age  int    `json:"age" validate:"min=18,max=65" errmsg:"min=Age must be at least 18;max=Age must be at most 65"`
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
		Title string  `json:"title" validate:"required,minlength=3,maxlength=10" errmsg:"required=Title required;minlength=Title too short;maxlength=Title too long"`
		Nums  []int   `json:"nums" validate:"minItems=1,maxItems=3,uniqueItems" errmsg:"minItems=At least one number;maxItems=At most 3 numbers;uniqueItems=Numbers must be unique"`
		Role  string  `json:"role" validate:"enum=admin|user|guest" errmsg:"enum=Role invalid"`
		Score float64 `json:"score" validate:"min=0.5,max=10" errmsg:"min=Score too low;max=Score too high"`
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

	if e := findByField(errs, "title"); e == nil || e.Error != "Title too short" {
		t.Errorf("title error missing or unexpected: %+v", e)
	}
	if e := findByField(errs, "nums"); e == nil || e.Error != "At least one number" {
		t.Errorf("nums error missing or unexpected: %+v", e)
	}
	if e := findByField(errs, "role"); e == nil || e.Error != "Role invalid" {
		t.Errorf("role error missing or unexpected: %+v", e)
	}
	if e := findByField(errs, "score"); e == nil {
		t.Errorf("score error missing")
	}
}

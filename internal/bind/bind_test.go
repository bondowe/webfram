package bind

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestBind_Auto tests the unified Bind method with auto source detection.
func TestBind_Auto(t *testing.T) {
	type TestStruct struct {
		ID       string `form:"id"`
		Name     string `form:"name"`
		Age      int    `form:"age"`
		Email    string `form:"email"`
		IsActive bool   `form:"active"`
	}

	tests := []struct {
		name        string
		setupReq    func() *http.Request
		validate    bool
		wantData    TestStruct
		wantErr     bool
		wantValErrs bool
	}{
		{
			name: "query parameters",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(
					http.MethodGet,
					"/test?id=123&name=John&age=30&email=john@example.com&active=true",
					nil,
				)
				return req
			},
			validate: false,
			wantData: TestStruct{
				ID:       "123",
				Name:     "John",
				Age:      30,
				Email:    "john@example.com",
				IsActive: true,
			},
			wantErr:     false,
			wantValErrs: false,
		},
		{
			name: "header parameters",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Id", "456")
				req.Header.Set("Name", "Jane")
				req.Header.Set("Age", "25")
				req.Header.Set("Email", "jane@example.com")
				req.Header.Set("Active", "true")
				return req
			},
			validate: false,
			wantData: TestStruct{
				ID:       "456",
				Name:     "Jane",
				Age:      25,
				Email:    "jane@example.com",
				IsActive: true,
			},
			wantErr:     false,
			wantValErrs: false,
		},
		{
			name: "cookie parameters",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.AddCookie(&http.Cookie{Name: "id", Value: "789"})
				req.AddCookie(&http.Cookie{Name: "name", Value: "Bob"})
				req.AddCookie(&http.Cookie{Name: "age", Value: "40"})
				req.AddCookie(&http.Cookie{Name: "email", Value: "bob@example.com"})
				req.AddCookie(&http.Cookie{Name: "active", Value: "false"})
				return req
			},
			validate: false,
			wantData: TestStruct{
				ID:       "789",
				Name:     "Bob",
				Age:      40,
				Email:    "bob@example.com",
				IsActive: false,
			},
			wantErr:     false,
			wantValErrs: false,
		},
		{
			name: "precedence - query over header",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test?name=QueryName&age=30", nil)
				req.Header.Set("Name", "HeaderName")
				req.Header.Set("Age", "25")
				req.Header.Set("Email", "header@example.com")
				return req
			},
			validate: false,
			wantData: TestStruct{
				Name:  "QueryName", // Query takes precedence
				Age:   30,          // Query takes precedence
				Email: "header@example.com",
			},
			wantErr:     false,
			wantValErrs: false,
		},
		{
			name: "precedence - header over cookie",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("Name", "HeaderName")
				req.AddCookie(&http.Cookie{Name: "name", Value: "CookieName"})
				req.AddCookie(&http.Cookie{Name: "age", Value: "40"})
				return req
			},
			validate: false,
			wantData: TestStruct{
				Name: "HeaderName", // Header takes precedence
				Age:  40,
			},
			wantErr:     false,
			wantValErrs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			result, valErrs, err := Bind[TestStruct](req, tt.validate)

			if (err != nil) != tt.wantErr {
				t.Errorf("Bind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (len(valErrs) > 0) != tt.wantValErrs {
				t.Errorf("Bind() validation errors = %v, wantValErrs %v", valErrs, tt.wantValErrs)
				return
			}

			if result != tt.wantData {
				t.Errorf("Bind() result = %+v, want %+v", result, tt.wantData)
			}
		})
	}
}

// TestBind_ExplicitSource tests the unified Bind method with explicit bindFrom tags.
func TestBind_ExplicitSource(t *testing.T) {
	type TestStruct struct {
		QueryParam  string `form:"q" bindFrom:"query"`
		HeaderParam string `form:"h" bindFrom:"header"`
		CookieParam string `form:"c" bindFrom:"cookie"`
	}

	tests := []struct {
		name        string
		setupReq    func() *http.Request
		validate    bool
		wantData    TestStruct
		wantErr     bool
		wantValErrs bool
	}{
		{
			name: "explicit sources",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(
					http.MethodGet,
					"/test?q=query_value&h=wrong_value&c=wrong_value",
					nil,
				)
				req.Header.Set("H", "header_value")
				req.AddCookie(&http.Cookie{Name: "c", Value: "cookie_value"})
				return req
			},
			validate: false,
			wantData: TestStruct{
				QueryParam:  "query_value",
				HeaderParam: "header_value",
				CookieParam: "cookie_value",
			},
			wantErr:     false,
			wantValErrs: false,
		},
		{
			name: "explicit source overrides precedence",
			setupReq: func() *http.Request {
				// Even though query has higher precedence, explicit binding should work
				req := httptest.NewRequest(http.MethodGet, "/test?h=query_h_value", nil)
				req.Header.Set("H", "header_value")
				return req
			},
			validate: false,
			wantData: TestStruct{
				HeaderParam: "header_value", // Should bind from header, not query
			},
			wantErr:     false,
			wantValErrs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			result, valErrs, err := Bind[TestStruct](req, tt.validate)

			if (err != nil) != tt.wantErr {
				t.Errorf("Bind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (len(valErrs) > 0) != tt.wantValErrs {
				t.Errorf("Bind() validation errors = %v, wantValErrs %v", valErrs, tt.wantValErrs)
				return
			}

			if result != tt.wantData {
				t.Errorf("Bind() result = %+v, want %+v", result, tt.wantData)
			}
		})
	}
}

// TestBind_Validation tests validation with unified Bind method.
func TestBind_Validation(t *testing.T) {
	type TestStruct struct {
		Name  string `form:"name"  validate:"required,minlength=3"`
		Email string `form:"email" validate:"required,format=email"`
		Age   int    `form:"age"   validate:"min=18,max=120"`
	}

	tests := []struct {
		name         string
		setupReq     func() *http.Request
		validate     bool
		wantErr      bool
		wantValErrs  int
		wantValField string
	}{
		{
			name: "valid data",
			setupReq: func() *http.Request {
				return httptest.NewRequest(
					http.MethodGet,
					"/test?name=John&email=john@example.com&age=30",
					nil,
				)
			},
			validate:    true,
			wantErr:     false,
			wantValErrs: 0,
		},
		{
			name: "missing required field",
			setupReq: func() *http.Request {
				return httptest.NewRequest(
					http.MethodGet,
					"/test?email=john@example.com&age=30",
					nil,
				)
			},
			validate:     true,
			wantErr:      false,
			wantValErrs:  2, // Both required and minlength will fail for empty string
			wantValField: "name",
		},
		{
			name: "invalid email format",
			setupReq: func() *http.Request {
				return httptest.NewRequest(
					http.MethodGet,
					"/test?name=John&email=invalid-email&age=30",
					nil,
				)
			},
			validate:     true,
			wantErr:      false,
			wantValErrs:  1,
			wantValField: "email",
		},
		{
			name: "age below minimum",
			setupReq: func() *http.Request {
				return httptest.NewRequest(
					http.MethodGet,
					"/test?name=John&email=john@example.com&age=15",
					nil,
				)
			},
			validate:     true,
			wantErr:      false,
			wantValErrs:  1,
			wantValField: "age",
		},
		{
			name: "age above maximum",
			setupReq: func() *http.Request {
				return httptest.NewRequest(
					http.MethodGet,
					"/test?name=John&email=john@example.com&age=150",
					nil,
				)
			},
			validate:     true,
			wantErr:      false,
			wantValErrs:  1,
			wantValField: "age",
		},
		{
			name: "multiple validation errors",
			setupReq: func() *http.Request {
				return httptest.NewRequest(
					http.MethodGet,
					"/test?name=Jo&email=invalid&age=15",
					nil,
				)
			},
			validate:    true,
			wantErr:     false,
			wantValErrs: 3, // Short name, invalid email, age too low
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			_, valErrs, err := Bind[TestStruct](req, tt.validate)

			if (err != nil) != tt.wantErr {
				t.Errorf("Bind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(valErrs) != tt.wantValErrs {
				t.Errorf(
					"Bind() validation errors count = %d, want %d (errors: %v)",
					len(valErrs),
					tt.wantValErrs,
					valErrs,
				)
				return
			}

			if tt.wantValErrs > 0 && tt.wantValField != "" {
				found := false
				for _, ve := range valErrs {
					if strings.Contains(
						strings.ToLower(ve.Field),
						strings.ToLower(tt.wantValField),
					) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf(
						"Bind() expected validation error for field %s, but not found in: %v",
						tt.wantValField,
						valErrs,
					)
				}
			}
		})
	}
}

// TestBind_ComplexTypes tests binding of complex types.
func TestBind_ComplexTypes(t *testing.T) {
	type TestStruct struct {
		ID        uuid.UUID `form:"id"      bindFrom:"query"`
		CreatedAt time.Time `form:"created" bindFrom:"query" validate:"required" format:"2006-01-02"`
		Tags      []string  `form:"tags"    bindFrom:"query"`
		Scores    []int     `form:"scores"  bindFrom:"query"`
	}

	testID := uuid.New()
	testDate := "2024-01-15"

	tests := []struct {
		name        string
		setupReq    func() *http.Request
		validate    bool
		wantErr     bool
		wantValErrs bool
		checkResult func(t *testing.T, result TestStruct)
	}{
		{
			name: "uuid and time binding",
			setupReq: func() *http.Request {
				return httptest.NewRequest(
					http.MethodGet,
					"/test?id="+testID.String()+"&created="+testDate,
					nil,
				)
			},
			validate:    false,
			wantErr:     false,
			wantValErrs: false,
			checkResult: func(t *testing.T, result TestStruct) {
				if result.ID != testID {
					t.Errorf("ID = %v, want %v", result.ID, testID)
				}
				expectedDate, _ := time.Parse("2006-01-02", testDate)
				if !result.CreatedAt.Equal(expectedDate) {
					t.Errorf("CreatedAt = %v, want %v", result.CreatedAt, expectedDate)
				}
			},
		},
		{
			name: "slice binding",
			setupReq: func() *http.Request {
				req := httptest.NewRequest(
					http.MethodGet,
					"/test?tags=go&tags=web&tags=framework&scores=100&scores=95&scores=98",
					nil,
				)
				return req
			},
			validate:    false,
			wantErr:     false,
			wantValErrs: false,
			checkResult: func(t *testing.T, result TestStruct) {
				if len(result.Tags) != 3 {
					t.Errorf("Tags length = %d, want 3", len(result.Tags))
				}
				if len(result.Scores) != 3 {
					t.Errorf("Scores length = %d, want 3", len(result.Scores))
				}
				expectedTags := []string{"go", "web", "framework"}
				for i, tag := range expectedTags {
					if i >= len(result.Tags) || result.Tags[i] != tag {
						t.Errorf("Tags[%d] = %v, want %v", i, result.Tags[i], tag)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			result, valErrs, err := Bind[TestStruct](req, tt.validate)

			if (err != nil) != tt.wantErr {
				t.Errorf("Bind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (len(valErrs) > 0) != tt.wantValErrs {
				t.Errorf("Bind() validation errors = %v, wantValErrs %v", valErrs, tt.wantValErrs)
				return
			}

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

// TestBind_FormData tests form data binding.
func TestBind_FormData(t *testing.T) {
	type TestStruct struct {
		Username string `form:"username" bindFrom:"form"`
		Password string `form:"password" bindFrom:"form"`
		Remember bool   `form:"remember" bindFrom:"form"`
	}

	tests := []struct {
		name        string
		setupReq    func() *http.Request
		validate    bool
		wantData    TestStruct
		wantErr     bool
		wantValErrs bool
	}{
		{
			name: "form urlencoded",
			setupReq: func() *http.Request {
				formData := url.Values{}
				formData.Set("username", "testuser")
				formData.Set("password", "secret123")
				formData.Set("remember", "true")

				req := httptest.NewRequest(
					http.MethodPost,
					"/login",
					strings.NewReader(formData.Encode()),
				)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			validate: false,
			wantData: TestStruct{
				Username: "testuser",
				Password: "secret123",
				Remember: true,
			},
			wantErr:     false,
			wantValErrs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			result, valErrs, err := Bind[TestStruct](req, tt.validate)

			if (err != nil) != tt.wantErr {
				t.Errorf("Bind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (len(valErrs) > 0) != tt.wantValErrs {
				t.Errorf("Bind() validation errors = %v, wantValErrs %v", valErrs, tt.wantValErrs)
				return
			}

			if result != tt.wantData {
				t.Errorf("Bind() result = %+v, want %+v", result, tt.wantData)
			}
		})
	}
}

// TestBind_MixedSources tests binding from multiple sources simultaneously.
func TestBind_MixedSources(t *testing.T) {
	type TestStruct struct {
		ID       string `form:"id"            bindFrom:"path"`
		Search   string `form:"q"             bindFrom:"query"`
		Token    string `form:"Authorization" bindFrom:"header"`
		Session  string `form:"session_id"    bindFrom:"cookie"`
		Username string `form:"username"      bindFrom:"form"`
	}

	tests := []struct {
		name        string
		setupReq    func() *http.Request
		validate    bool
		wantData    TestStruct
		wantErr     bool
		wantValErrs bool
	}{
		{
			name: "all sources",
			setupReq: func() *http.Request {
				formData := url.Values{}
				formData.Set("username", "testuser")

				req := httptest.NewRequest(
					http.MethodPost,
					"/users/123?q=golang",
					strings.NewReader(formData.Encode()),
				)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Header.Set("Authorization", "Bearer token123")
				req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess_abc123"})

				// Simulate path parameter
				req.SetPathValue("id", "123")

				return req
			},
			validate: false,
			wantData: TestStruct{
				ID:       "123",
				Search:   "golang",
				Token:    "Bearer token123",
				Session:  "sess_abc123",
				Username: "testuser",
			},
			wantErr:     false,
			wantValErrs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()
			result, valErrs, err := Bind[TestStruct](req, tt.validate)

			if (err != nil) != tt.wantErr {
				t.Errorf("Bind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if (len(valErrs) > 0) != tt.wantValErrs {
				t.Errorf("Bind() validation errors = %v, wantValErrs %v", valErrs, tt.wantValErrs)
				return
			}

			if result != tt.wantData {
				t.Errorf("Bind() result = %+v, want %+v", result, tt.wantData)
			}
		})
	}
}

// TestBind_SkipFields tests that fields with "-" tag are skipped.
func TestBind_SkipFields(t *testing.T) {
	type TestStruct struct {
		Name     string `form:"name"`
		Internal string `form:"-"`             // Should be skipped
		Hidden   string `            json:"-"` // Should be skipped
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/test?name=John&Internal=secret&Hidden=secret2",
		nil,
	)
	result, _, err := Bind[TestStruct](req, false)

	if err != nil {
		t.Errorf("Bind() error = %v", err)
		return
	}

	if result.Name != "John" {
		t.Errorf("Name = %s, want John", result.Name)
	}

	if result.Internal != "" {
		t.Errorf("Internal = %s, want empty (should be skipped)", result.Internal)
	}

	if result.Hidden != "" {
		t.Errorf("Hidden = %s, want empty (should be skipped)", result.Hidden)
	}
}

// TestBind_TagFallback tests how bindFrom works with different struct tags.
func TestBind_TagFallback(t *testing.T) {
	tests := []struct {
		name     string
		testType interface{}
		setupReq func() *http.Request
		validate func(t *testing.T, result interface{})
	}{
		{
			name: "bindFrom with form tag",
			testType: struct {
				Name string `form:"name" bindFrom:"query"`
			}{},
			setupReq: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/test?name=John", nil)
			},
			validate: func(t *testing.T, result interface{}) {
				v := result.(struct {
					Name string `form:"name" bindFrom:"query"`
				})
				if v.Name != "John" {
					t.Errorf("Name = %s, want John", v.Name)
				}
			},
		},
		{
			name: "bindFrom with json tag (no form tag)",
			testType: struct {
				Email string `json:"email" bindFrom:"query"`
			}{},
			setupReq: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/test?email=test@example.com", nil)
			},
			validate: func(t *testing.T, result interface{}) {
				v := result.(struct {
					Email string `json:"email" bindFrom:"query"`
				})
				if v.Email != "test@example.com" {
					t.Errorf("Email = %s, want test@example.com", v.Email)
				}
			},
		},
		{
			name: "bindFrom with xml tag (no form or json tag)",
			testType: struct {
				Age int `xml:"age" bindFrom:"query"`
			}{},
			setupReq: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/test?age=30", nil)
			},
			validate: func(t *testing.T, result interface{}) {
				v := result.(struct {
					Age int `xml:"age" bindFrom:"query"`
				})
				if v.Age != 30 {
					t.Errorf("Age = %d, want 30", v.Age)
				}
			},
		},
		{
			name: "bindFrom with json tag containing options",
			testType: struct {
				Status string `json:"status,omitempty" bindFrom:"query"`
			}{},
			setupReq: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/test?status=active", nil)
			},
			validate: func(t *testing.T, result interface{}) {
				v := result.(struct {
					Status string `json:"status,omitempty" bindFrom:"query"`
				})
				if v.Status != "active" {
					t.Errorf("Status = %s, want active", v.Status)
				}
			},
		},
		{
			name: "bindFrom with no tags - uses field name",
			testType: struct {
				Username string `bindFrom:"query"`
			}{},
			setupReq: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/test?Username=testuser", nil)
			},
			validate: func(t *testing.T, result interface{}) {
				v := result.(struct {
					Username string `bindFrom:"query"`
				})
				if v.Username != "testuser" {
					t.Errorf("Username = %s, want testuser", v.Username)
				}
			},
		},
		{
			name: "form tag takes precedence over json tag",
			testType: struct {
				Data string `form:"data" json:"json_data" bindFrom:"query"`
			}{},
			setupReq: func() *http.Request {
				// Both parameter names present
				return httptest.NewRequest(
					http.MethodGet,
					"/test?data=from_form&json_data=from_json",
					nil,
				)
			},
			validate: func(t *testing.T, result interface{}) {
				v := result.(struct {
					Data string `form:"data" json:"json_data" bindFrom:"query"`
				})
				// Should use "data" (form tag) not "json_data" (json tag)
				if v.Data != "from_form" {
					t.Errorf("Data = %s, want from_form (form tag should take precedence)", v.Data)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()

			// Use reflection to call Bind with the correct type
			switch tt.testType.(type) {
			case struct {
				Name string `form:"name" bindFrom:"query"`
			}:
				result, _, err := Bind[struct {
					Name string `form:"name" bindFrom:"query"`
				}](req, false)
				if err != nil {
					t.Errorf("Bind() error = %v", err)
					return
				}
				tt.validate(t, result)

			case struct {
				Email string `json:"email" bindFrom:"query"`
			}:
				result, _, err := Bind[struct {
					Email string `json:"email" bindFrom:"query"`
				}](req, false)
				if err != nil {
					t.Errorf("Bind() error = %v", err)
					return
				}
				tt.validate(t, result)

			case struct {
				Age int `xml:"age" bindFrom:"query"`
			}:
				result, _, err := Bind[struct {
					Age int `xml:"age" bindFrom:"query"`
				}](req, false)
				if err != nil {
					t.Errorf("Bind() error = %v", err)
					return
				}
				tt.validate(t, result)

			case struct {
				Status string `json:"status,omitempty" bindFrom:"query"`
			}:
				result, _, err := Bind[struct {
					Status string `json:"status,omitempty" bindFrom:"query"`
				}](req, false)
				if err != nil {
					t.Errorf("Bind() error = %v", err)
					return
				}
				tt.validate(t, result)

			case struct {
				Username string `bindFrom:"query"`
			}:
				result, _, err := Bind[struct {
					Username string `bindFrom:"query"`
				}](req, false)
				if err != nil {
					t.Errorf("Bind() error = %v", err)
					return
				}
				tt.validate(t, result)

			case struct {
				Data string `form:"data" json:"json_data" bindFrom:"query"`
			}:
				result, _, err := Bind[struct {
					Data string `form:"data" json:"json_data" bindFrom:"query"`
				}](req, false)
				if err != nil {
					t.Errorf("Bind() error = %v", err)
					return
				}
				tt.validate(t, result)
			}
		})
	}
}

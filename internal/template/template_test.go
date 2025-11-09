package template

import (
	"embed"
	"errors"
	"fmt"
	htmlTemplate "html/template"
	"io/fs"
	"strings"
	"sync"
	"testing"
)

//go:embed all:testdata/**
var testFS embed.FS

func resetTemplateConfig() {
	config = nil
	templatesCache = sync.Map{}
	partialsCache = sync.Map{}
	layoutsCache = make(map[string]any)
	layoutPattern = nil
	funcMap = htmlTemplate.FuncMap{}
}

func setupTestTemplateConfig(t *testing.T) {
	t.Helper()
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)
}

func TestConfigure(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	if config == nil {
		t.Fatal("Config was not set")
	}

	if htmlLayoutFileName != "layout.go.html" {
		t.Errorf("Expected htmlLayoutFileName 'layout.go.html', got %q", htmlLayoutFileName)
	}

	if textLayoutFileName != "layout.go.txt" {
		t.Errorf("Expected textLayoutFileName 'layout.go.txt', got %q", textLayoutFileName)
	}

	if layoutPattern == nil {
		t.Fatal("layoutPattern was not set")
	}

	if _, ok := funcMap[cfg.I18nFuncName]; !ok {
		t.Errorf("I18n function %q was not added to funcMap", cfg.I18nFuncName)
	}
}

func TestConfiguration(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	result, ok := Configuration()

	if !ok {
		t.Fatal("Expected valid configuration")
	}

	if result.LayoutBaseName != cfg.LayoutBaseName {
		t.Errorf("Expected LayoutBaseName %q, got %q", cfg.LayoutBaseName, result.LayoutBaseName)
	}

	if result.HTMLTemplateExtension != cfg.HTMLTemplateExtension {
		t.Errorf("Expected HTMLTemplateExtension %q, got %q", cfg.HTMLTemplateExtension, result.HTMLTemplateExtension)
	}

	if result.TextTemplateExtension != cfg.TextTemplateExtension {
		t.Errorf("Expected TextTemplateExtension %q, got %q", cfg.TextTemplateExtension, result.TextTemplateExtension)
	}
}

func TestMust_Success(t *testing.T) {
	result := Must("test", nil)
	if result != "test" {
		t.Errorf("Expected 'test', got %q", result)
	}
}

func TestMust_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic but got none")
		}
	}()

	Must("test", errors.New("test error"))
}

func TestLookupTemplate_Absolute(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	// Store a test template
	tmpl := htmlTemplate.Must(htmlTemplate.New("test").Parse("<h1>Test</h1>"))
	templatesCache.Store("testdata/test.go.html", [2]any{"test", tmpl})

	result, ok := LookupTemplate("testdata/test.go.html", true)
	if !ok {
		t.Fatal("Template not found")
	}

	if result == nil {
		t.Error("Expected non-nil template")
	}
}

func TestLookupTemplate_Relative(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	// Store a test template
	tmpl := htmlTemplate.Must(htmlTemplate.New("test").Parse("<h1>Test</h1>"))
	templatesCache.Store("testdata/test.go.html", [2]any{"test", tmpl})

	result, ok := LookupTemplate("test.go.html", false)
	if !ok {
		t.Fatal("Template not found")
	}

	if result == nil {
		t.Error("Expected non-nil template")
	}
}

func TestLookupTemplate_NotFound(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	result, ok := LookupTemplate("nonexistent.go.html", false)
	if ok {
		t.Error("Expected template not to be found")
	}

	if result != nil {
		t.Error("Expected nil template")
	}
}

func TestLayoutExists(t *testing.T) {
	tests := []struct {
		name        string
		layoutName  string
		expected    bool
		expectPanic bool
	}{
		{
			name:       "existing layout",
			layoutName: "layout.go.html",
			expected:   true,
		},
		{
			name:       "non-existing layout",
			layoutName: "nonexistent.go.html",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic but got none")
					}
				}()
			}

			dir, err := fs.Sub(testFS, "testdata")
			if err != nil {
				t.Fatal(err)
			}

			result := layoutExists(dir, tt.layoutName)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetLayout(t *testing.T) {
	tests := []struct {
		name          string
		layoutName    string
		expectedName  string
		expectedFound bool
		expectPanic   bool
	}{
		{
			name:          "standard layout exists",
			layoutName:    "layout.go.html",
			expectedName:  "layout.go.html",
			expectedFound: true,
		},
		{
			name:          "no inherit layout exists",
			layoutName:    "special.go.html",
			expectedName:  "_special.go.html",
			expectedFound: true,
		},
		{
			name:          "layout doesn't exist",
			layoutName:    "nonexistent.go.html",
			expectedName:  "",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic but got none")
					}
				}()
			}

			dir, err := fs.Sub(testFS, "testdata")
			if err != nil {
				t.Fatal(err)
			}

			name, found := getLayout(dir, tt.layoutName)
			if found != tt.expectedFound {
				t.Errorf("Expected found=%v, got %v", tt.expectedFound, found)
			}

			if name != tt.expectedName {
				t.Errorf("Expected name %q, got %q", tt.expectedName, name)
			}
		})
	}
}

func TestGetLayout_AmbiguousLayouts(t *testing.T) {
	t.Skip("Ambiguous test directory not available - test skipped")
}

func TestParseHTMLTemplate_WithoutLayout(t *testing.T) {
	setupTestTemplateConfig(t)

	templatePath := "testdata/simple.go.html"
	layouts := []string{}

	name, tmpl := parseHTMLTemplate(templatePath, layouts)

	if name == "" {
		t.Error("Expected non-empty template name")
	}

	if tmpl == nil {
		t.Fatal("Expected non-nil template")
	}

	// Test execution
	var buf strings.Builder
	data := map[string]string{"Title": "Test"}
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Errorf("Template execution failed: %v", err)
	}
}

func TestParseHTMLTemplate_WithLayout(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	templatePath := "testdata/withLayout.go.html"
	layouts := []string{"testdata/layout.go.html"}

	name, tmpl := parseHTMLTemplate(templatePath, layouts)

	if name == "" {
		t.Error("Expected non-empty template name")
	}

	if tmpl == nil {
		t.Fatal("Expected non-nil template")
	}
}

func TestParseTextTemplate_WithoutLayout(t *testing.T) {
	setupTestTemplateConfig(t)

	templatePath := "testdata/simple.go.txt"
	layouts := []string{}

	name, tmpl := parseTextTemplate(templatePath, layouts)

	if name == "" {
		t.Error("Expected non-empty template name")
	}

	if tmpl == nil {
		t.Fatal("Expected non-nil template")
	}

	// Test execution
	var buf strings.Builder
	data := map[string]string{"Name": "Test"}
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Errorf("Template execution failed: %v", err)
	}
}

func TestParseTextTemplate_WithLayout(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	templatePath := "testdata/withLayout.go.txt"
	layouts := []string{"testdata/layout.go.txt"}

	name, tmpl := parseTextTemplate(templatePath, layouts)

	if name == "" {
		t.Error("Expected non-empty template name")
	}

	if tmpl == nil {
		t.Fatal("Expected non-nil template")
	}
}

func TestGetPartialFunc(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	// Store a partial template
	partialTmpl := htmlTemplate.Must(htmlTemplate.New("_header").Parse("<header>{{.Title}}</header>"))
	templatesCache.Store("testdata/_header.go.html", [2]any{"_header", partialTmpl})

	partialFunc := getPartialFunc("testdata/page.go.html")

	data := map[string]string{"Title": "Test Header"}
	result, err := partialFunc("header", data)

	if err != nil {
		t.Fatalf("Partial function returned error: %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "Test Header") {
		t.Errorf("Expected result to contain 'Test Header', got %q", resultStr)
	}
}

func TestGetPartialFunc_NotFound(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	partialFunc := getPartialFunc("testdata/page.go.html")

	data := map[string]string{"Title": "Test"}
	_, err := partialFunc("nonexistent", data)

	if err == nil {
		t.Error("Expected error for non-existent partial")
	}

	if !strings.Contains(err.Error(), "template not found") {
		t.Errorf("Expected error message to contain 'template not found', got %q", err.Error())
	}
}

func TestLookUpPartial(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	// Store a partial at the root
	partialTmpl := htmlTemplate.Must(htmlTemplate.New("_partial").Parse("<div>Partial</div>"))
	templatesCache.Store("testdata/_partial.go.html", [2]any{"_partial", partialTmpl})

	// Look up from a nested folder
	result := lookUpPartial("testdata/nested/deep", "_partial.go.html")

	if result == nil {
		t.Error("Expected to find partial in parent directory")
	}
}

func TestLookUpPartial_NotFound(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	result := lookUpPartial("testdata", "_nonexistent.go.html")

	if result != nil {
		t.Error("Expected nil for non-existent partial")
	}
}

// TestLookUpPartial_RootLevel tests that partials at the root level are found
// when looking up from a subdirectory. This is a critical test for the bug fix
// where the condition check was happening before the root lookup.
func TestLookUpPartial_RootLevel(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	tests := []struct {
		name           string
		startFolder    string
		partialFile    string
		shouldFind     bool
		expectedInBody string
	}{
		{
			name:           "Find root partial from deep nested folder",
			startFolder:    "testdata/nested/deep",
			partialFile:    "_root_partial.go.html",
			shouldFind:     true,
			expectedInBody: "Root Level Partial",
		},
		{
			name:           "Find root partial from single nested folder",
			startFolder:    "testdata/nested",
			partialFile:    "_root_partial.go.html",
			shouldFind:     true,
			expectedInBody: "Root Level Partial",
		},
		{
			name:           "Find root partial from same level",
			startFolder:    "testdata",
			partialFile:    "_root_partial.go.html",
			shouldFind:     true,
			expectedInBody: "Root Level Partial",
		},
		{
			name:           "Find nested partial from deep folder",
			startFolder:    "testdata/nested/deep",
			partialFile:    "_nested_partial.go.html",
			shouldFind:     true,
			expectedInBody: "Nested Level Partial",
		},
		{
			name:           "Find deep partial from deep folder",
			startFolder:    "testdata/nested/deep",
			partialFile:    "_deep_partial.go.html",
			shouldFind:     true,
			expectedInBody: "Deep Level Partial",
		},
		{
			name:        "Don't find deep partial from root",
			startFolder: "testdata",
			partialFile: "_deep_partial.go.html",
			shouldFind:  false,
		},
		{
			name:        "Don't find nested partial from root",
			startFolder: "testdata",
			partialFile: "_nested_partial.go.html",
			shouldFind:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lookUpPartial(tt.startFolder, tt.partialFile)

			if tt.shouldFind {
				if result == nil {
					t.Errorf("Expected to find partial %s from %s, but got nil", tt.partialFile, tt.startFolder)
					return
				}

				// Execute the template to verify it contains expected content
				var sb strings.Builder
				err := result.Execute(&sb, nil)
				if err != nil {
					t.Errorf("Failed to execute partial: %v", err)
					return
				}

				if !strings.Contains(sb.String(), tt.expectedInBody) {
					t.Errorf("Expected partial to contain %q, got %q", tt.expectedInBody, sb.String())
				}
			} else {
				if result != nil {
					t.Errorf("Expected not to find partial %s from %s, but got a template", tt.partialFile, tt.startFolder)
				}
			}
		})
	}
}

// TestLookUpPartial_Caching tests that the partial lookup results are properly cached
func TestLookUpPartial_Caching(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	// First lookup - should cache the result
	result1 := lookUpPartial("testdata/nested/deep", "_root_partial.go.html")
	if result1 == nil {
		t.Fatal("Expected to find partial on first lookup")
	}

	// Second lookup - should return cached result
	result2 := lookUpPartial("testdata/nested/deep", "_root_partial.go.html")
	if result2 == nil {
		t.Fatal("Expected to find cached partial on second lookup")
	}

	if result1 != result2 {
		t.Error("Expected cached lookup to return same template instance")
	}

	// Test caching of not-found results
	notFound1 := lookUpPartial("testdata", "_nonexistent.go.html")
	if notFound1 != nil {
		t.Error("Expected nil for non-existent partial")
	}

	notFound2 := lookUpPartial("testdata", "_nonexistent.go.html")
	if notFound2 != nil {
		t.Error("Expected cached nil for non-existent partial")
	}
}

// TestLookUpPartial_EmptyFolder tests edge cases with empty or root folder paths
func TestLookUpPartial_EmptyFolder(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	tests := []struct {
		name        string
		folder      string
		partialFile string
		shouldFind  bool
	}{
		{
			name:        "Empty folder string - won't find testdata partial",
			folder:      "",
			partialFile: "_root_partial.go.html",
			shouldFind:  false, // Partial is at testdata/_root_partial.go.html, not at root
		},
		{
			name:        "Dot folder - won't find testdata partial",
			folder:      ".",
			partialFile: "_root_partial.go.html",
			shouldFind:  false, // Partial is at testdata/_root_partial.go.html, not at root
		},
		{
			name:        "Testdata folder - finds partial at same level",
			folder:      "testdata",
			partialFile: "_root_partial.go.html",
			shouldFind:  true,
		},
		{
			name:        "Header partial exists at testdata",
			folder:      "testdata",
			partialFile: "_header.go.html",
			shouldFind:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear cache for this test
			cacheKey := tt.folder + "|" + tt.partialFile
			partialsCache.Delete(cacheKey)

			result := lookUpPartial(tt.folder, tt.partialFile)

			if tt.shouldFind && result == nil {
				t.Errorf("Expected to find partial %s from folder %q, but got nil", tt.partialFile, tt.folder)
			} else if !tt.shouldFind && result != nil {
				t.Errorf("Expected not to find partial %s from folder %q, but got a template", tt.partialFile, tt.folder)
			}
		})
	}
}

// TestGetPartialFuncWithI18n tests the i18n-aware partial function creation
func TestGetPartialFuncWithI18n(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	tests := []struct {
		name           string
		i18nFunc       func(string, ...any) string
		partialName    string
		data           map[string]string
		expectedSubstr string
		shouldError    bool
	}{
		{
			name: "Partial with custom i18n function",
			i18nFunc: func(format string, args ...any) string {
				// Simple test i18n that prefixes with [FR]
				return "[FR] " + format
			},
			partialName:    "i18n_test",
			data:           map[string]string{"Name": "John"},
			expectedSubstr: "[FR] Hello, %s!",
			shouldError:    false,
		},
		{
			name:           "Partial with nil i18n function falls back to default",
			i18nFunc:       nil,
			partialName:    "root_partial",
			expectedSubstr: "Root Level Partial",
			shouldError:    false,
		},
		{
			name:        "Non-existent partial returns error",
			i18nFunc:    nil,
			partialName: "nonexistent",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partialFunc := GetPartialFuncWithI18n("testdata/page.go.html", tt.i18nFunc)

			result, err := partialFunc(tt.partialName, tt.data)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			resultStr := string(result)
			if !strings.Contains(resultStr, tt.expectedSubstr) {
				t.Errorf("Expected result to contain %q, got %q", tt.expectedSubstr, resultStr)
			}
		})
	}
}

// TestGetPartialFuncWithI18n_NestedPartials tests that nested partials inherit i18n
func TestGetPartialFuncWithI18n_NestedPartials(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	// Create a custom i18n function that prefixes with language tag
	spanishFunc := func(format string, args ...any) string {
		return "[ES] " + format
	}

	partialFunc := GetPartialFuncWithI18n("testdata/page.go.html", spanishFunc)

	result, err := partialFunc("nested_i18n", map[string]string{"Name": "Maria"})

	if err != nil {
		t.Fatalf("Unexpected error executing nested partial: %v", err)
	}

	resultStr := string(result)

	// Check that both the outer partial and nested partial have i18n applied
	if !strings.Contains(resultStr, "[ES] Nested partial message") {
		t.Errorf("Expected outer partial to have i18n, got %q", resultStr)
	}

	if !strings.Contains(resultStr, "[ES] Hello, %s!") {
		t.Errorf("Expected nested partial to have i18n, got %q", resultStr)
	}
}

// TestGetPartialFuncWithI18n_TemplateCloning tests that i18n injection doesn't affect cached templates
func TestGetPartialFuncWithI18n_TemplateCloning(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	// Create two different i18n functions
	englishFunc := func(format string, args ...any) string {
		return "[EN] " + format
	}

	frenchFunc := func(format string, args ...any) string {
		return "[FR] " + format
	}

	// Create partial functions with different i18n
	partialFuncEN := GetPartialFuncWithI18n("testdata/page.go.html", englishFunc)
	partialFuncFR := GetPartialFuncWithI18n("testdata/page.go.html", frenchFunc)

	// Execute with English
	resultEN, err := partialFuncEN("i18n_test", map[string]string{"Name": "John"})
	if err != nil {
		t.Fatalf("Unexpected error with English: %v", err)
	}

	// Execute with French
	resultFR, err := partialFuncFR("i18n_test", map[string]string{"Name": "Jean"})
	if err != nil {
		t.Fatalf("Unexpected error with French: %v", err)
	}

	// Verify each has the correct language tag
	if !strings.Contains(string(resultEN), "[EN]") {
		t.Errorf("Expected English result to contain [EN], got %q", string(resultEN))
	}

	if !strings.Contains(string(resultFR), "[FR]") {
		t.Errorf("Expected French result to contain [FR], got %q", string(resultFR))
	}

	// Verify they're different
	if string(resultEN) == string(resultFR) {
		t.Error("Expected different results for different i18n functions")
	}
}

// TestGetPartialFunc_DefaultBehavior tests that getPartialFunc works without custom i18n injection
func TestGetPartialFunc_DefaultBehavior(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	// Set up a default i18n in funcMap (simulates what the template system does)
	funcMap[cfg.I18nFuncName] = func(format string, args ...any) string {
		// This simulates the default fmt.Sprintf behavior
		if len(args) > 0 {
			return fmt.Sprintf(format, args...)
		}
		return format
	}

	partialFunc := getPartialFunc("testdata/page.go.html")

	result, err := partialFunc("i18n_test", map[string]string{"Name": "Default"})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// getPartialFunc passes nil i18nFunc, so it uses whatever T function is in the cached template's funcMap
	// The cached template will use the funcMap's T function, which formats the string
	resultStr := string(result)
	if !strings.Contains(resultStr, "Hello") {
		t.Errorf("Expected result to contain 'Hello', got %q", resultStr)
	}
	if !strings.Contains(resultStr, "Default") {
		t.Errorf("Expected result to contain 'Default', got %q", resultStr)
	}
}

func TestGetOrCreateHTMLLayoutChain(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	layouts := []string{"testdata/layout.go.html"}

	tmpl := getOrCreateHTMLLayoutChain(layouts)

	if tmpl == nil {
		t.Fatal("Expected non-nil template")
	}

	// Call again to test caching
	tmpl2 := getOrCreateHTMLLayoutChain(layouts)

	if tmpl2 == nil {
		t.Fatal("Expected non-nil cached template")
	}
}

func TestGetOrCreateTextLayoutChain(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	layouts := []string{"testdata/layout.go.txt"}

	tmpl := getOrCreateTextLayoutChain(layouts)

	if tmpl == nil {
		t.Fatal("Expected non-nil template")
	}

	// Call again to test caching
	tmpl2 := getOrCreateTextLayoutChain(layouts)

	if tmpl2 == nil {
		t.Fatal("Expected non-nil cached template")
	}
}

func TestCacheTemplates(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	// After Configure, templates should be cached
	// Check that some templates exist
	_, ok := LookupTemplate("simple.go.html", false)
	if !ok {
		t.Error("Expected simple.go.html to be cached")
	}
}

func TestLayoutPattern(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"html layout", "layout.go.html", true},
		{"html _layout", "_layout.go.html", true},
		{"text layout", "layout.go.txt", true},
		{"text _layout", "_layout.go.txt", true},
		{"regular file", "page.go.html", false},
		{"partial file", "_partial.go.html", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := layoutPattern.MatchString(tt.filename)
			if result != tt.expected {
				t.Errorf("Expected match=%v for %q, got %v", tt.expected, tt.filename, result)
			}
		})
	}
}

func TestFuncMap_I18nFunction(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	// Test that the I18n function works
	i18nFunc, ok := funcMap["T"].(func(string, ...any) string)
	if !ok {
		t.Fatal("I18n function not found in funcMap")
	}

	result := i18nFunc("Hello %s", "World")
	if result != "Hello World" {
		t.Errorf("Expected 'Hello World', got %q", result)
	}
}

func BenchmarkLookupTemplate(b *testing.B) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	// Store a test template
	tmpl := htmlTemplate.Must(htmlTemplate.New("test").Parse("<h1>Test</h1>"))
	templatesCache.Store("testdata/test.go.html", [2]any{"test", tmpl})

	b.ResetTimer()
	for b.Loop() {
		LookupTemplate("test.go.html", false)
	}
}

func BenchmarkParseHTMLTemplate(b *testing.B) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	templatePath := "testdata/simple.go.html"
	layouts := []string{}

	b.ResetTimer()
	for b.Loop() {
		parseHTMLTemplate(templatePath, layouts)
	}
}

// Text template partial tests

func TestLookUpTextPartial_RootLevel(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	tests := []struct {
		name           string
		startFolder    string
		partialFile    string
		shouldFind     bool
		expectedInBody string
	}{
		{
			name:           "Find root text partial from deep nested folder",
			startFolder:    "testdata/nested/deep",
			partialFile:    "_root_partial.go.txt",
			shouldFind:     true,
			expectedInBody: "Root Level Text Partial",
		},
		{
			name:           "Find nested text partial from deep folder",
			startFolder:    "testdata/nested/deep",
			partialFile:    "_nested_partial.go.txt",
			shouldFind:     true,
			expectedInBody: "Nested Level Text Partial",
		},
		{
			name:           "Find deep text partial from deep folder",
			startFolder:    "testdata/nested/deep",
			partialFile:    "_deep_partial.go.txt",
			shouldFind:     true,
			expectedInBody: "Deep Level Text Partial",
		},
		{
			name:        "Don't find deep text partial from root",
			startFolder: "testdata",
			partialFile: "_deep_partial.go.txt",
			shouldFind:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lookUpTextPartial(tt.startFolder, tt.partialFile)

			if tt.shouldFind {
				if result == nil {
					t.Errorf("Expected to find partial %s from %s, but got nil", tt.partialFile, tt.startFolder)
					return
				}

				// Execute the template to verify it contains expected content
				var sb strings.Builder
				err := result.Execute(&sb, nil)
				if err != nil {
					t.Errorf("Failed to execute partial: %v", err)
					return
				}

				if !strings.Contains(sb.String(), tt.expectedInBody) {
					t.Errorf("Expected partial to contain %q, got %q", tt.expectedInBody, sb.String())
				}
			} else {
				if result != nil {
					t.Errorf("Expected not to find partial %s from %s, but got a template", tt.partialFile, tt.startFolder)
				}
			}
		})
	}
}

func TestGetTextPartialFuncWithI18n(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	tests := []struct {
		name           string
		i18nFunc       func(string, ...any) string
		partialName    string
		data           map[string]string
		expectedSubstr string
		shouldError    bool
	}{
		{
			name: "Text partial with custom i18n function",
			i18nFunc: func(format string, args ...any) string {
				return "[FR] " + format
			},
			partialName:    "i18n_test",
			data:           map[string]string{"Name": "John"},
			expectedSubstr: "[FR] Hello, %s!",
			shouldError:    false,
		},
		{
			name:           "Text partial with nil i18n function",
			i18nFunc:       nil,
			partialName:    "root_partial",
			expectedSubstr: "Root Level Text Partial",
			shouldError:    false,
		},
		{
			name:        "Non-existent text partial returns error",
			i18nFunc:    nil,
			partialName: "nonexistent",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partialFunc := GetTextPartialFuncWithI18n("testdata/page.go.txt", tt.i18nFunc)

			result, err := partialFunc(tt.partialName, tt.data)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !strings.Contains(result, tt.expectedSubstr) {
				t.Errorf("Expected result to contain %q, got %q", tt.expectedSubstr, result)
			}
		})
	}
}

func TestGetTextPartialFuncWithI18n_NestedPartials(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	spanishFunc := func(format string, args ...any) string {
		return "[ES] " + format
	}

	partialFunc := GetTextPartialFuncWithI18n("testdata/page.go.txt", spanishFunc)

	result, err := partialFunc("nested_i18n", map[string]string{"Name": "Maria"})

	if err != nil {
		t.Fatalf("Unexpected error executing nested text partial: %v", err)
	}

	// Check that both the outer partial and nested partial have i18n applied
	if !strings.Contains(result, "[ES] Nested partial message") {
		t.Errorf("Expected outer partial to have i18n, got %q", result)
	}

	if !strings.Contains(result, "[ES] Hello, %s!") {
		t.Errorf("Expected nested partial to have i18n, got %q", result)
	}
}

func TestGetTextPartialFuncWithI18n_TemplateCloning(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	englishFunc := func(format string, args ...any) string {
		return "[EN] " + format
	}

	frenchFunc := func(format string, args ...any) string {
		return "[FR] " + format
	}

	partialFuncEN := GetTextPartialFuncWithI18n("testdata/page.go.txt", englishFunc)
	partialFuncFR := GetTextPartialFuncWithI18n("testdata/page.go.txt", frenchFunc)

	resultEN, err := partialFuncEN("i18n_test", map[string]string{"Name": "John"})
	if err != nil {
		t.Fatalf("Unexpected error with English: %v", err)
	}

	resultFR, err := partialFuncFR("i18n_test", map[string]string{"Name": "Jean"})
	if err != nil {
		t.Fatalf("Unexpected error with French: %v", err)
	}

	if !strings.Contains(resultEN, "[EN]") {
		t.Errorf("Expected English result to contain [EN], got %q", resultEN)
	}

	if !strings.Contains(resultFR, "[FR]") {
		t.Errorf("Expected French result to contain [FR], got %q", resultFR)
	}

	if resultEN == resultFR {
		t.Error("Expected different results for different i18n functions")
	}
}

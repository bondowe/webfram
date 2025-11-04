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
	layoutsCache = make(map[string]any)
	layoutPattern = nil
	funcMap = htmlTemplate.FuncMap{}
}

func TestConfigure(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		TemplatesPath:         "testdata",
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
		TemplatesPath:         "testdata",
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	result := Configuration()

	if result.TemplatesPath != cfg.TemplatesPath {
		t.Errorf("Expected TemplatesPath %q, got %q", cfg.TemplatesPath, result.TemplatesPath)
	}

	if result.LayoutBaseName != cfg.LayoutBaseName {
		t.Errorf("Expected LayoutBaseName %q, got %q", cfg.LayoutBaseName, result.LayoutBaseName)
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
		TemplatesPath:         "testdata",
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
		TemplatesPath:         "testdata",
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
		TemplatesPath:         "testdata",
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
	// Create a test directory with both layout and _layout
	dir, err := fs.Sub(testFS, "testdata/ambiguous")
	if err != nil {
		t.Skip("Ambiguous test directory not available")
		return
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for ambiguous layouts")
		} else {
			errMsg := fmt.Sprint(r)
			if !strings.Contains(errMsg, "ambiguous") {
				t.Errorf("Expected panic message to contain 'ambiguous', got %q", errMsg)
			}
		}
	}()

	getLayout(dir, "both.go.html")
}

func TestParseHTMLTemplate_WithoutLayout(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		TemplatesPath:         "testdata",
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

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
		TemplatesPath:         "testdata",
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
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		TemplatesPath:         "testdata",
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

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
		TemplatesPath:         "testdata",
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
		TemplatesPath:         "testdata",
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
		TemplatesPath:         "testdata",
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
		TemplatesPath:         "testdata",
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
		TemplatesPath:         "testdata",
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

func TestGetOrCreateHTMLLayoutChain(t *testing.T) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		TemplatesPath:         "testdata",
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
		TemplatesPath:         "testdata",
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
		TemplatesPath:         "testdata",
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
		TemplatesPath:         "testdata",
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
		TemplatesPath:         "testdata",
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
		TemplatesPath:         "testdata",
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
	for i := 0; i < b.N; i++ {
		LookupTemplate("test.go.html", false)
	}
}

func BenchmarkParseHTMLTemplate(b *testing.B) {
	resetTemplateConfig()

	cfg := &Config{
		FS:                    testFS,
		TemplatesPath:         "testdata",
		LayoutBaseName:        "layout",
		HTMLTemplateExtension: ".go.html",
		TextTemplateExtension: ".go.txt",
		I18nFuncName:          "T",
	}

	Configure(cfg)

	templatePath := "testdata/simple.go.html"
	layouts := []string{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseHTMLTemplate(templatePath, layouts)
	}
}

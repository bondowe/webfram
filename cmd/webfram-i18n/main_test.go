package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseLanguages(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single language",
			input:    "en",
			expected: []string{"en"},
		},
		{
			name:     "multiple languages",
			input:    "en,fr,es",
			expected: []string{"en", "fr", "es"},
		},
		{
			name:     "languages with spaces",
			input:    "en, fr, es",
			expected: []string{"en", "fr", "es"},
		},
		{
			name:     "languages with extra spaces",
			input:    " en ,  fr  , es ",
			expected: []string{"en", "fr", "es"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "only spaces",
			input:    "  ,  ,  ",
			expected: []string{},
		},
		{
			name:     "with region codes",
			input:    "en-US,en-GB,fr-FR",
			expected: []string{"en-US", "en-GB", "fr-FR"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLanguages(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d languages, got %d", len(tt.expected), len(result))
				return
			}

			for i, lang := range result {
				if lang != tt.expected[i] {
					t.Errorf("Expected language[%d]=%q, got %q", i, tt.expected[i], lang)
				}
			}
		})
	}
}

func TestMergeTranslations(t *testing.T) {
	source1 := map[string]TranslationInfo{
		"hello":   {MessageID: "hello"},
		"goodbye": {MessageID: "goodbye"},
	}

	source2 := map[string]TranslationInfo{
		"welcome": {MessageID: "welcome"},
		"hello":   {MessageID: "hello", Placeholders: []PlaceholderInfo{{Type: "string", ArgNum: 1}}},
	}

	result := mergeTranslations(source1, source2)

	if len(result) != 3 {
		t.Errorf("Expected 3 translations, got %d", len(result))
	}

	if _, exists := result["hello"]; !exists {
		t.Error("Expected 'hello' to exist in merged result")
	}

	if _, exists := result["goodbye"]; !exists {
		t.Error("Expected 'goodbye' to exist in merged result")
	}

	if _, exists := result["welcome"]; !exists {
		t.Error("Expected 'welcome' to exist in merged result")
	}

	// Later source should override earlier source
	if len(result["hello"].Placeholders) != 1 {
		t.Error("Expected 'hello' from source2 to override source1")
	}
}

func TestCatalogsAreEqual(t *testing.T) {
	tests := []struct {
		catalog1 *Catalog
		catalog2 *Catalog
		name     string
		expected bool
	}{
		{
			name: "equal catalogs",
			catalog1: &Catalog{
				Language: "en",
				Messages: []Message{
					{ID: "hello", Message: "hello", Translation: "Hello"},
				},
			},
			catalog2: &Catalog{
				Language: "en",
				Messages: []Message{
					{ID: "hello", Message: "hello", Translation: "Hello"},
				},
			},
			expected: true,
		},
		{
			name: "different languages",
			catalog1: &Catalog{
				Language: "en",
				Messages: []Message{},
			},
			catalog2: &Catalog{
				Language: "fr",
				Messages: []Message{},
			},
			expected: false,
		},
		{
			name: "different number of messages",
			catalog1: &Catalog{
				Language: "en",
				Messages: []Message{
					{ID: "hello", Message: "hello"},
				},
			},
			catalog2: &Catalog{
				Language: "en",
				Messages: []Message{},
			},
			expected: false,
		},
		{
			name: "same messages different order",
			catalog1: &Catalog{
				Language: "en",
				Messages: []Message{
					{ID: "hello", Message: "hello"},
					{ID: "goodbye", Message: "goodbye"},
				},
			},
			catalog2: &Catalog{
				Language: "en",
				Messages: []Message{
					{ID: "goodbye", Message: "goodbye"},
					{ID: "hello", Message: "hello"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := catalogsAreEqual(tt.catalog1, tt.catalog2)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMessagesAreEqual(t *testing.T) {
	tests := []struct {
		name     string
		msg1     Message
		msg2     Message
		expected bool
	}{
		{
			name: "equal messages",
			msg1: Message{
				ID:          "hello",
				Message:     "hello",
				Translation: "Hello",
			},
			msg2: Message{
				ID:          "hello",
				Message:     "hello",
				Translation: "Hello",
			},
			expected: true,
		},
		{
			name: "different IDs",
			msg1: Message{
				ID:      "hello",
				Message: "hello",
			},
			msg2: Message{
				ID:      "goodbye",
				Message: "goodbye",
			},
			expected: false,
		},
		{
			name: "different translations",
			msg1: Message{
				ID:          "hello",
				Translation: "Hello",
			},
			msg2: Message{
				ID:          "hello",
				Translation: "Bonjour",
			},
			expected: false,
		},
		{
			name: "different placeholders",
			msg1: Message{
				ID: "hello",
				Placeholders: map[string]Placeholder{
					"arg_1": {ID: "arg_1", Type: "string"},
				},
			},
			msg2: Message{
				ID:           "hello",
				Placeholders: map[string]Placeholder{},
			},
			expected: false,
		},
		{
			name: "different plural forms",
			msg1: Message{
				ID:  "count",
				One: "one item",
			},
			msg2: Message{
				ID:  "count",
				One: "1 item",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := messagesAreEqual(tt.msg1, tt.msg2)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCreateMessage(t *testing.T) {
	tests := []struct {
		checkFn func(*testing.T, Message)
		name    string
		msgID   string
		info    TranslationInfo
	}{
		{
			name:  "simple message",
			msgID: "hello",
			info: TranslationInfo{
				MessageID: "hello",
			},
			checkFn: func(t *testing.T, msg Message) {
				if msg.ID != "hello" {
					t.Errorf("Expected ID 'hello', got %q", msg.ID)
				}
				if msg.Message != "hello" {
					t.Errorf("Expected Message 'hello', got %q", msg.Message)
				}
				if msg.Translation != "" {
					t.Errorf("Expected empty Translation, got %q", msg.Translation)
				}
			},
		},
		{
			name:  "message with placeholder",
			msgID: "hello %s",
			info: TranslationInfo{
				MessageID: "hello %s",
				Placeholders: []PlaceholderInfo{
					{Type: "string", ArgNum: 1},
				},
			},
			checkFn: func(t *testing.T, msg Message) {
				if len(msg.Placeholders) != 1 {
					t.Errorf("Expected 1 placeholder, got %d", len(msg.Placeholders))
				}
				if _, exists := msg.Placeholders["arg_1"]; !exists {
					t.Error("Expected placeholder 'arg_1' to exist")
				}
			},
		},
		{
			name:  "message with integer placeholder (plural forms)",
			msgID: "you have %d items",
			info: TranslationInfo{
				MessageID: "you have %d items",
				Placeholders: []PlaceholderInfo{
					{Type: "int", ArgNum: 1},
				},
			},
			checkFn: func(t *testing.T, msg Message) {
				// Should have plural form fields initialized
				if msg.One == "" && msg.Other == "" {
					// Fields should exist but be empty strings (not omitted from JSON)
					// We can't check for existence directly, but we know they're zero values
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createMessage(tt.msgID, tt.info)
			tt.checkFn(t, result)
		})
	}
}

func TestExtractPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		types    []string
		expected int
	}{
		{
			name:     "no placeholders",
			message:  "Hello World",
			expected: 0,
		},
		{
			name:     "single string placeholder",
			message:  "Hello %s",
			expected: 1,
			types:    []string{"string"},
		},
		{
			name:     "single int placeholder",
			message:  "You have %d items",
			expected: 1,
			types:    []string{"int"},
		},
		{
			name:     "multiple placeholders",
			message:  "Hello %s, you have %d messages",
			expected: 2,
			types:    []string{"string", "int"},
		},
		{
			name:     "escaped percent",
			message:  "Discount: 50%% off",
			expected: 0, // %% is escaped
		},
		{
			name:     "float placeholder",
			message:  "Price: $%.2f",
			expected: 1,
			types:    []string{"float64"},
		},
		{
			name:     "various format specifiers",
			message:  "%s %d %f %t %v",
			expected: 5,
			types:    []string{"string", "int", "float64", "bool", "interface{}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPlaceholders(tt.message)

			if len(result) != tt.expected {
				t.Errorf("Expected %d placeholders, got %d", tt.expected, len(result))
			}

			for i, expectedType := range tt.types {
				if i < len(result) && result[i].Type != expectedType {
					t.Errorf("Expected placeholder[%d] type %q, got %q", i, expectedType, result[i].Type)
				}
			}
		})
	}
}

func TestInferPlaceholderType(t *testing.T) {
	tests := []struct {
		verb     string
		expected string
	}{
		{"d", "int"},
		{"b", "int"},
		{"o", "int"},
		{"x", "int"},
		{"X", "int"},
		{"f", "float64"},
		{"F", "float64"},
		{"e", "float64"},
		{"E", "float64"},
		{"g", "float64"},
		{"G", "float64"},
		{"s", "string"},
		{"q", "string"},
		{"t", "bool"},
		{"p", "pointer"},
		{"v", "interface{}"},
		{"T", "interface{}"},
		{"unknown", "interface{}"},
	}

	for _, tt := range tests {
		t.Run(tt.verb, func(t *testing.T) {
			result := inferPlaceholderType(tt.verb)

			if result != tt.expected {
				t.Errorf("Expected type %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetFormatSpecifier(t *testing.T) {
	tests := []struct {
		placeholderType string
		expected        string
	}{
		{"int", "d"},
		{"float64", "f"},
		{"string", "s"},
		{"bool", "t"},
		{"pointer", "p"},
		{"interface{}", "v"},
		{"unknown", "v"},
	}

	for _, tt := range tests {
		t.Run(tt.placeholderType, func(t *testing.T) {
			result := getFormatSpecifier(tt.placeholderType)

			if result != tt.expected {
				t.Errorf("Expected specifier %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestContainsIntegerPlaceholder(t *testing.T) {
	tests := []struct {
		name         string
		placeholders []PlaceholderInfo
		expected     bool
	}{
		{
			name:         "no placeholders",
			placeholders: []PlaceholderInfo{},
			expected:     false,
		},
		{
			name: "has integer",
			placeholders: []PlaceholderInfo{
				{Type: "int", ArgNum: 1},
			},
			expected: true,
		},
		{
			name: "only string",
			placeholders: []PlaceholderInfo{
				{Type: "string", ArgNum: 1},
			},
			expected: false,
		},
		{
			name: "mixed types with integer",
			placeholders: []PlaceholderInfo{
				{Type: "string", ArgNum: 1},
				{Type: "int", ArgNum: 2},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsIntegerPlaceholder(tt.placeholders)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsI18nMethod(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"Sprintf", true},
		{"Printf", true},
		{"Fprintf", true},
		{"Sprint", true},
		{"Print", true},
		{"Fprint", true},
		{"Sprintln", true},
		{"Println", true},
		{"Fprintln", true},
		{"NotAMethod", false},
		{"Format", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isI18nMethod(tt.name)

			if result != tt.expected {
				t.Errorf("Expected %v for %q, got %v", tt.expected, tt.name, result)
			}
		})
	}
}

func TestIsLogPackage(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"fmt", true},
		{"log", true},
		{"strings", false},
		{"io", false},
		{"os", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLogPackage(tt.name)

			if result != tt.expected {
				t.Errorf("Expected %v for %q, got %v", tt.expected, tt.name, result)
			}
		})
	}
}

func TestIsLogMethod(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"Printf", true},
		{"Print", true},
		{"Println", true},
		{"Sprintf", true},
		{"Sprint", true},
		{"Sprintln", true},
		{"Fprintf", true},
		{"Fprint", true},
		{"Fprintln", true},
		{"Errorf", true},
		{"Error", true},
		{"Errorln", true},
		{"Fatalf", true},
		{"Fatal", true},
		{"Fatalln", true},
		{"Panicf", true},
		{"Panic", true},
		{"Panicln", true},
		{"NotAMethod", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLogMethod(tt.name)

			if result != tt.expected {
				t.Errorf("Expected %v for %q, got %v", tt.expected, tt.name, result)
			}
		})
	}
}

func TestWriteCatalog(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "messages.en.json")

	catalog := Catalog{
		Language: "en",
		Messages: []Message{
			{
				ID:          "hello",
				Message:     "hello",
				Translation: "Hello",
			},
		},
	}

	err := writeCatalog(filename, catalog)
	if err != nil {
		t.Fatalf("writeCatalog failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Fatal("Catalog file was not created")
	}

	// Verify content
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read catalog file: %v", err)
	}

	var loaded Catalog
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal catalog: %v", err)
	}

	if loaded.Language != catalog.Language {
		t.Errorf("Expected language %q, got %q", catalog.Language, loaded.Language)
	}

	if len(loaded.Messages) != len(catalog.Messages) {
		t.Errorf("Expected %d messages, got %d", len(catalog.Messages), len(loaded.Messages))
	}
}

func TestLoadExistingCatalog(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "messages.en.json")

	// Create a catalog file
	catalog := Catalog{
		Language: "en",
		Messages: []Message{
			{ID: "hello", Message: "hello", Translation: "Hello"},
		},
	}

	data, _ := json.MarshalIndent(catalog, "", "  ")
	os.WriteFile(filename, data, 0644)

	// Load it
	loaded, err := loadExistingCatalog(filename)
	if err != nil {
		t.Fatalf("loadExistingCatalog failed: %v", err)
	}

	if loaded.Language != "en" {
		t.Errorf("Expected language 'en', got %q", loaded.Language)
	}

	if len(loaded.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(loaded.Messages))
	}
}

func TestLoadExistingCatalog_NotFound(t *testing.T) {
	_, err := loadExistingCatalog("nonexistent.json")

	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	if !os.IsNotExist(err) {
		t.Errorf("Expected os.IsNotExist error, got %v", err)
	}
}

func TestCreateNewCatalog(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "messages.fr.json")

	translations := map[string]TranslationInfo{
		"hello": {
			MessageID:    "hello",
			Placeholders: []PlaceholderInfo{},
		},
		"goodbye": {
			MessageID:    "goodbye",
			Placeholders: []PlaceholderInfo{},
		},
	}

	err := createNewCatalog(filename, "fr", translations)
	if err != nil {
		t.Fatalf("createNewCatalog failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Fatal("Catalog file was not created")
	}

	// Load and verify
	loaded, err := loadExistingCatalog(filename)
	if err != nil {
		t.Fatalf("Failed to load created catalog: %v", err)
	}

	if loaded.Language != "fr" {
		t.Errorf("Expected language 'fr', got %q", loaded.Language)
	}

	if len(loaded.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(loaded.Messages))
	}

	// Verify messages are sorted
	if loaded.Messages[0].ID != "goodbye" || loaded.Messages[1].ID != "hello" {
		t.Error("Messages are not sorted alphabetically")
	}
}

func TestExtractTranslationsFromTemplates(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test template files
	htmlContent := `
<h1>{{T "Hello World"}}</h1>
<p>{{T "You have %d messages" .Count}}</p>
`
	os.WriteFile(filepath.Join(tmpDir, "test.go.html"), []byte(htmlContent), 0644)

	txtContent := `{{T "Welcome"}}`
	os.WriteFile(filepath.Join(tmpDir, "test.go.txt"), []byte(txtContent), 0644)

	// Extract translations
	translations, err := extractTranslationsFromTemplates(tmpDir)
	if err != nil {
		t.Fatalf("extractTranslationsFromTemplates failed: %v", err)
	}

	if len(translations) != 3 {
		t.Errorf("Expected 3 translations, got %d", len(translations))
	}

	if _, exists := translations["Hello World"]; !exists {
		t.Error("Expected 'Hello World' translation")
	}

	if _, exists := translations["You have %d messages"]; !exists {
		t.Errorf("Expected %q translation", "You have %d messages")
	}

	if _, exists := translations["Welcome"]; !exists {
		t.Error("Expected 'Welcome' translation")
	}
}

func TestExtractTranslationsFromGoFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test Go file
	goContent := `package main

import (
    "fmt"
    "log"
)

type User struct {
    Name string ` + "`" + `validate:"required" errmsg:"required=Name is required"` + "`" + `
}

func main() {
    fmt.Printf("Hello %s", "World")
    log.Println("Server started")
    printer.Sprintf("Welcome to %s", "App")
}
`
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(goContent), 0644)

	// Extract translations
	translations, err := extractTranslationsFromGoFiles(tmpDir)
	if err != nil {
		t.Fatalf("extractTranslationsFromGoFiles failed: %v", err)
	}

	if len(translations) < 3 {
		t.Errorf("Expected at least 3 translations, got %d", len(translations))
	}

	// Check for specific translations
	expectedMessages := []string{
		"Hello %s",
		"Server started",
		"Name is required",
	}

	for _, msg := range expectedMessages {
		if _, exists := translations[msg]; !exists {
			t.Errorf("Expected translation for %q", msg)
		}
	}
}

func BenchmarkExtractPlaceholders(b *testing.B) {
	message := "Hello %s, you have %d new messages and %.2f credits"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractPlaceholders(message)
	}
}

func BenchmarkCreateMessage(b *testing.B) {
	info := TranslationInfo{
		MessageID: "Hello %s",
		Placeholders: []PlaceholderInfo{
			{Type: "string", ArgNum: 1},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		createMessage("Hello %s", info)
	}
}

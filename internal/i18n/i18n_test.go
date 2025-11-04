package i18n

import (
	"context"
	"embed"
	"encoding/json"
	"strings"
	"testing"

	"golang.org/x/text/language"
	"golang.org/x/text/message/catalog"
)

//go:embed testdata/locales/*.json
var testFS embed.FS

func resetI18nConfig() {
	config = nil
	msgCatalog = nil
}

func TestConfigure(t *testing.T) {
	resetI18nConfig()

	cfg := &Config{
		FS: testFS,
	}

	Configure(cfg)

	if config == nil {
		t.Fatal("Config was not set")
	}

	if config.FS == nil {
		t.Error("FS was not set in config")
	}

	if msgCatalog == nil {
		t.Error("Message catalog was not initialized")
	}
}

func TestConfiguration(t *testing.T) {
	resetI18nConfig()

	cfg := &Config{
		FS: testFS,
	}

	Configure(cfg)

	result, ok := Configuration()

	if !ok {
		t.Error("Expected valid configuration")
		return
	}

	if result.FS == nil {
		t.Error("Expected non-nil FS in configuration")
	}
}

func TestGetI18nPrinter(t *testing.T) {
	resetI18nConfig()

	cfg := &Config{
		FS: testFS,
	}

	Configure(cfg)

	tests := []struct {
		name string
		tag  language.Tag
	}{
		{
			name: "English",
			tag:  language.English,
		},
		{
			name: "French",
			tag:  language.French,
		},
		{
			name: "Spanish",
			tag:  language.Spanish,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := GetI18nPrinter(tt.tag)

			if printer == nil {
				t.Fatal("GetI18nPrinter returned nil")
			}

			// Test that the printer works
			result := printer.Sprintf("Hello %s", "World")
			if result == "" {
				t.Error("Expected non-empty result from printer")
			}
		})
	}
}

func TestContextWithI18nPrinter(t *testing.T) {
	resetI18nConfig()

	cfg := &Config{
		FS: testFS,
	}

	Configure(cfg)

	printer := GetI18nPrinter(language.English)
	ctx := context.Background()

	newCtx := ContextWithI18nPrinter(ctx, printer)

	if newCtx == nil {
		t.Fatal("ContextWithI18nPrinter returned nil")
	}

	// Verify the printer can be retrieved
	retrievedPrinter, ok := I18nPrinterFromContext(newCtx)
	if !ok {
		t.Error("Expected to find printer in context")
	}

	if retrievedPrinter == nil {
		t.Error("Expected non-nil printer from context")
	}
}

func TestI18nPrinterFromContext(t *testing.T) {
	resetI18nConfig()

	cfg := &Config{
		FS: testFS,
	}

	Configure(cfg)

	tests := []struct {
		name         string
		setupContext func() context.Context
		expectFound  bool
	}{
		{
			name: "with printer in context",
			setupContext: func() context.Context {
				printer := GetI18nPrinter(language.English)
				return ContextWithI18nPrinter(context.Background(), printer)
			},
			expectFound: true,
		},
		{
			name: "without printer in context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectFound: false,
		},
		{
			name: "with wrong type in context",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), i18nPrinterKey, "not a printer")
			},
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()

			printer, found := I18nPrinterFromContext(ctx)

			if found != tt.expectFound {
				t.Errorf("Expected found=%v, got %v", tt.expectFound, found)
			}

			if tt.expectFound && printer == nil {
				t.Error("Expected non-nil printer when found=true")
			}

			if !tt.expectFound && printer != nil {
				t.Error("Expected nil printer when found=false")
			}
		})
	}
}

func TestExtractLangTagFromFilename(t *testing.T) {
	tests := []struct {
		name     string
		filepath string
		expected language.Tag
	}{
		{
			name:     "English",
			filepath: "locales/messages.en.json",
			expected: language.English,
		},
		{
			name:     "English GB",
			filepath: "locales/messages.en-GB.json",
			expected: language.BritishEnglish,
		},
		{
			name:     "French",
			filepath: "locales/messages.fr.json",
			expected: language.French,
		},
		{
			name:     "Spanish",
			filepath: "locales/messages.es.json",
			expected: language.Spanish,
		},
		{
			name:     "invalid format",
			filepath: "messages.json",
			expected: language.Und,
		},
		{
			name:     "invalid language",
			filepath: "messages.invalid.json",
			expected: language.Und,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractLangTagFromFilename(tt.filepath)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestLoadJSONMessages(t *testing.T) {
	resetI18nConfig()

	builder := catalog.NewBuilder()

	tests := []struct {
		name        string
		tag         language.Tag
		jsonData    string
		expectError bool
	}{
		{
			name: "valid messages",
			tag:  language.English,
			jsonData: `{
                "language": "en",
                "messages": [
                    {
                        "id": "hello",
                        "message": "Hello",
                        "translation": "Hello"
                    },
                    {
                        "id": "goodbye",
                        "message": "Goodbye",
                        "translation": "Goodbye"
                    }
                ]
            }`,
			expectError: false,
		},
		{
			name: "messages with placeholders",
			tag:  language.French,
			jsonData: `{
                "language": "fr",
                "messages": [
                    {
                        "id": "hello_name",
                        "message": "Hello %s",
                        "translation": "Bonjour %s",
                        "placeholders": {
                            "arg_1": {
                                "id": "arg_1",
                                "string": "%s",
                                "type": "string",
                                "underlyingType": "string",
                                "argNum": 1,
                                "expr": "arg1"
                            }
                        }
                    }
                ]
            }`,
			expectError: false,
		},
		{
			name: "messages without translation",
			tag:  language.Spanish,
			jsonData: `{
                "language": "es",
                "messages": [
                    {
                        "id": "test",
                        "message": "Test Message"
                    }
                ]
            }`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			tag:         language.English,
			jsonData:    `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loadJSONMessages(builder, tt.tag, []byte(tt.jsonData))

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestLoadI18nCatalogs_NilConfig(t *testing.T) {
	resetI18nConfig()

	// Should not panic with nil config
	loadI18nCatalogs()
}

func TestLoadI18nCatalogs_NilFS(t *testing.T) {
	resetI18nConfig()

	config = &Config{
		FS: nil,
	}

	// Should not panic with nil FS
	loadI18nCatalogs()
}

func TestLoadI18nCatalogs_WithTestData(t *testing.T) {
	resetI18nConfig()

	cfg := &Config{
		FS: testFS,
	}

	Configure(cfg)

	if msgCatalog == nil {
		t.Error("Expected message catalog to be loaded")
	}

	// Test that we can create a printer and use it
	printer := GetI18nPrinter(language.English)
	result := printer.Sprintf("Test message")

	if result == "" {
		t.Error("Expected non-empty result from printer")
	}
}

func TestMessageFileStruct(t *testing.T) {
	msgFile := MessageFile{
		Language: "en",
		Messages: []MessageEntry{
			{
				ID:          "test",
				Message:     "Test Message",
				Translation: "Test Message",
				Placeholders: map[string]Placeholder{
					"arg_1": {
						ID:             "arg_1",
						String:         "%s",
						Type:           "string",
						UnderlyingType: "string",
						ArgNum:         1,
						Expr:           "arg1",
					},
				},
			},
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(msgFile)
	if err != nil {
		t.Fatalf("Failed to marshal MessageFile: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled MessageFile
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal MessageFile: %v", err)
	}

	if unmarshaled.Language != msgFile.Language {
		t.Errorf("Expected language %q, got %q", msgFile.Language, unmarshaled.Language)
	}

	if len(unmarshaled.Messages) != len(msgFile.Messages) {
		t.Errorf("Expected %d messages, got %d", len(msgFile.Messages), len(unmarshaled.Messages))
	}
}

func TestMessageEntryStruct(t *testing.T) {
	entry := MessageEntry{
		ID:          "hello",
		Message:     "Hello",
		Translation: "Hello",
		Placeholders: map[string]Placeholder{
			"arg_1": {
				ID:             "arg_1",
				String:         "%s",
				Type:           "string",
				UnderlyingType: "string",
				ArgNum:         1,
				Expr:           "arg1",
			},
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal MessageEntry: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled MessageEntry
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal MessageEntry: %v", err)
	}

	if unmarshaled.ID != entry.ID {
		t.Errorf("Expected ID %q, got %q", entry.ID, unmarshaled.ID)
	}

	if len(unmarshaled.Placeholders) != len(entry.Placeholders) {
		t.Errorf("Expected %d placeholders, got %d", len(entry.Placeholders), len(unmarshaled.Placeholders))
	}
}

func TestPlaceholderStruct(t *testing.T) {
	placeholder := Placeholder{
		ID:             "arg_1",
		String:         "%s",
		Type:           "string",
		UnderlyingType: "string",
		ArgNum:         1,
		Expr:           "arg1",
	}

	// Test JSON marshaling
	data, err := json.Marshal(placeholder)
	if err != nil {
		t.Fatalf("Failed to marshal Placeholder: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled Placeholder
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Placeholder: %v", err)
	}

	if unmarshaled.ID != placeholder.ID {
		t.Errorf("Expected ID %q, got %q", placeholder.ID, unmarshaled.ID)
	}

	if unmarshaled.Type != placeholder.Type {
		t.Errorf("Expected Type %q, got %q", placeholder.Type, unmarshaled.Type)
	}

	if unmarshaled.ArgNum != placeholder.ArgNum {
		t.Errorf("Expected ArgNum %d, got %d", placeholder.ArgNum, unmarshaled.ArgNum)
	}
}

func TestI18nPrinterTranslation(t *testing.T) {
	resetI18nConfig()

	cfg := &Config{
		FS: testFS,
	}

	Configure(cfg)

	tests := []struct {
		name     string
		lang     language.Tag
		message  string
		args     []interface{}
		contains string
	}{
		{
			name:     "simple message",
			lang:     language.English,
			message:  "Hello",
			args:     nil,
			contains: "Hello",
		},
		{
			name:     "message with placeholder",
			lang:     language.English,
			message:  "Hello %s",
			args:     []interface{}{"World"},
			contains: "World",
		},
		{
			name:     "message with multiple placeholders",
			lang:     language.English,
			message:  "Hello %s, you have %d messages",
			args:     []interface{}{"Alice", 5},
			contains: "Alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := GetI18nPrinter(tt.lang)

			var result string
			if len(tt.args) > 0 {
				result = printer.Sprintf(tt.message, tt.args...)
			} else {
				result = printer.Sprintf(tt.message)
			}

			if !strings.Contains(result, tt.contains) {
				t.Errorf("Expected result to contain %q, got %q", tt.contains, result)
			}
		})
	}
}

func TestContextKey(t *testing.T) {
	key1 := i18nPrinterKey
	key2 := contextKey("i18nPrinter")

	if key1 != key2 {
		t.Error("Context keys should be equal")
	}
}

func BenchmarkGetI18nPrinter(b *testing.B) {
	resetI18nConfig()

	cfg := &Config{
		FS: testFS,
	}

	Configure(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetI18nPrinter(language.English)
	}
}

func BenchmarkContextWithI18nPrinter(b *testing.B) {
	resetI18nConfig()

	cfg := &Config{
		FS: testFS,
	}

	Configure(cfg)

	printer := GetI18nPrinter(language.English)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ContextWithI18nPrinter(ctx, printer)
	}
}

func BenchmarkI18nPrinterFromContext(b *testing.B) {
	resetI18nConfig()

	cfg := &Config{
		FS: testFS,
	}

	Configure(cfg)

	printer := GetI18nPrinter(language.English)
	ctx := ContextWithI18nPrinter(context.Background(), printer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		I18nPrinterFromContext(ctx)
	}
}

func BenchmarkPrinterSprintf(b *testing.B) {
	resetI18nConfig()

	cfg := &Config{
		FS: testFS,
	}

	Configure(cfg)

	printer := GetI18nPrinter(language.English)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		printer.Sprintf("Hello %s", "World")
	}
}

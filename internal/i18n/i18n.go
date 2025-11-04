package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

type (
	contextKey string
	Config     struct {
		FS fs.FS
	}

	// MessageFile represents the structure of the JSON message files
	MessageFile struct {
		Language string         `json:"language"`
		Messages []MessageEntry `json:"messages"`
	}

	// MessageEntry represents a single message with its translations and placeholders
	MessageEntry struct {
		ID           string                 `json:"id"`
		Message      string                 `json:"message"`
		Translation  string                 `json:"translation,omitempty"`
		Placeholders map[string]Placeholder `json:"placeholders,omitempty"`
	}

	// Placeholder represents a placeholder in a message
	Placeholder struct {
		ID             string `json:"id"`
		String         string `json:"string"`
		Type           string `json:"type"`
		UnderlyingType string `json:"underlyingType"`
		ArgNum         int    `json:"argNum"`
		Expr           string `json:"expr"`
	}
)

const (
	i18nPrinterKey contextKey = "i18nPrinter"
)

var (
	config     *Config
	msgCatalog catalog.Catalog
)

func Configure(cfg *Config) {
	config = cfg
	loadI18nCatalogs()
}

func Configuration() Config {
	return *config
}

// GetI18nPrinter creates a message printer for the given language tag
func GetI18nPrinter(langTag language.Tag) *message.Printer {
	p := message.NewPrinter(langTag, message.Catalog(msgCatalog))
	return p
}

// ContextWithI18nPrinter adds the message printer to the context
func ContextWithI18nPrinter(ctx context.Context, printer *message.Printer) context.Context {
	return context.WithValue(ctx, i18nPrinterKey, printer)
}

// I18nPrinterFromContext retrieves the message printer from the context
func I18nPrinterFromContext(ctx context.Context) (*message.Printer, bool) {
	printer, ok := ctx.Value(i18nPrinterKey).(*message.Printer)
	return printer, ok
}

func loadI18nCatalogs() {
	if config == nil || config.FS == nil {
		fmt.Println("Warning: i18n config not set, skipping catalog loading")
		return
	}

	builder := catalog.NewBuilder()

	// Walk through the file system to find all message files
	err := fs.WalkDir(config.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Only process JSON files with "messages." prefix
		if !strings.HasPrefix(filepath.Base(path), "messages.") || filepath.Ext(path) != ".json" {
			return nil
		}

		// Extract language tag from filename
		langTag := extractLangTagFromFilename(path)
		if langTag == language.Und {
			fmt.Printf("Warning: could not determine language for file: %s\n", path)
			return nil
		}

		// Load messages from the file
		data, err := fs.ReadFile(config.FS, path)
		if err != nil {
			return fmt.Errorf("error reading file %s: %w", path, err)
		}

		if err := loadJSONMessages(builder, langTag, data); err != nil {
			return fmt.Errorf("error loading messages from %s: %w", path, err)
		}

		fmt.Printf("Loaded messages for language: %s from %s\n", langTag, path)
		return nil
	})

	if err != nil {
		fmt.Printf("Error loading i18n catalogs: %v\n", err)
	}

	msgCatalog = builder
}

func extractLangTagFromFilename(filePath string) language.Tag {
	base := filepath.Base(filePath)
	nameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))
	parts := strings.Split(nameWithoutExt, ".")
	if len(parts) < 2 {
		return language.Und
	}
	lang := parts[len(parts)-1]
	langTag, err := language.Parse(lang)
	if err != nil {
		return language.Und
	}
	return langTag
}

// loadJSONMessages loads messages from JSON data into the catalog builder
func loadJSONMessages(builder *catalog.Builder, tag language.Tag, data []byte) error {
	var msgFile MessageFile
	if err := json.Unmarshal(data, &msgFile); err != nil {
		return fmt.Errorf("error parsing JSON: %w", err)
	}

	for _, entry := range msgFile.Messages {
		// Use the translation if available, otherwise use the message itself
		translation := entry.Message
		if entry.Translation != "" {
			translation = entry.Translation
		}

		// Add the message to the catalog
		// The ID is the key, and the translated message is the value
		builder.SetString(tag, entry.ID, translation)
	}

	return nil
}

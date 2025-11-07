// Package i18n provides internationalization support for the framework.
package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

type (
	contextKey string
	// Config holds i18n configuration.
	Config struct {
		FS fs.FS
	}

	// MessageFile represents the structure of the JSON message files.
	MessageFile struct {
		Language string         `json:"language"`
		Messages []MessageEntry `json:"messages"`
	}

	// MessageEntry represents a single message with its translations and placeholders.
	MessageEntry struct {
		Placeholders map[string]Placeholder `json:"placeholders,omitempty"`
		ID           string                 `json:"id"`
		Message      string                 `json:"message"`
		Translation  string                 `json:"translation,omitempty"`
	}

	// Placeholder represents a placeholder in a message.
	Placeholder struct {
		ID             string `json:"id"`
		String         string `json:"string"`
		Type           string `json:"type"`
		UnderlyingType string `json:"underlyingType"`
		Expr           string `json:"expr"`
		ArgNum         int    `json:"argNum"`
	}
)

const (
	i18nPrinterKey contextKey = "i18nPrinter"
)

//nolint:gochecknoglobals // Package-level state for i18n configuration and message catalog
var (
	config     *Config
	msgCatalog catalog.Catalog
)

// Configure initializes the internationalization system with the provided configuration.
// It sets up the filesystem and base path for locale files, then loads all message catalogs.
// Panics if locales directory or filesystem is missing.
func Configure(cfg *Config) {
	config = cfg
	loadI18nCatalogs()
}

// Configuration returns the current i18n configuration.
// Returns the config and true if i18n is configured, or an empty config and false if not configured.
func Configuration() (Config, bool) {
	if config == nil {
		return Config{}, false
	}
	return *config, true
}

// GetI18nPrinter creates a message printer for the given language tag
// GetI18nPrinter creates a message printer for the specified language tag.
// The printer can be used to translate messages according to the loaded message catalogs.
// Returns a printer configured for the given language tag.
func GetI18nPrinter(langTag language.Tag) *message.Printer {
	p := message.NewPrinter(langTag, message.Catalog(msgCatalog))
	return p
}

// ContextWithI18nPrinter adds the message printer to the context
// ContextWithI18nPrinter stores a message printer in the context.
// Returns a new context containing the printer, which can be retrieved later with PrinterFromContext.
func ContextWithI18nPrinter(ctx context.Context, printer *message.Printer) context.Context {
	return context.WithValue(ctx, i18nPrinterKey, printer)
}

// PrinterFromContext retrieves a message printer from the context.
// Returns the printer and true if found, or nil and false if not present.
func PrinterFromContext(ctx context.Context) (*message.Printer, bool) {
	printer, ok := ctx.Value(i18nPrinterKey).(*message.Printer)
	return printer, ok
}

func loadI18nCatalogs() {
	if config == nil || config.FS == nil {
		slog.Default().Warn("i18n config not set, skipping catalog loading")
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
			slog.Default().Warn("could not determine language for file", "path", path)
			return nil
		}

		// Load messages from the file
		data, err := fs.ReadFile(config.FS, path)
		if err != nil {
			return fmt.Errorf("error reading file %s: %w", path, err)
		}

		if loadErr := loadJSONMessages(builder, langTag, data); loadErr != nil {
			return fmt.Errorf("error loading messages from %s: %w", path, loadErr)
		}

		slog.Default().Info("Loaded messages for language", "language", langTag, "path", path)
		return nil
	})

	if err != nil {
		slog.Default().Error("Error loading i18n catalogs", "error", err)
	}

	msgCatalog = builder
}

func extractLangTagFromFilename(filePath string) language.Tag {
	base := filepath.Base(filePath)
	nameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))
	parts := strings.Split(nameWithoutExt, ".")
	if len(parts) < 2 { //nolint:mnd // need at least name and language parts
		return language.Und
	}
	lang := parts[len(parts)-1]
	langTag, err := language.Parse(lang)
	if err != nil {
		return language.Und
	}
	return langTag
}

// loadJSONMessages loads messages from JSON data into the catalog builder.
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
		_ = builder.SetString(tag, entry.ID, translation)
	}

	return nil
}

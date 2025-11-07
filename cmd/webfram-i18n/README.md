# WebFram I18n Extraction Tool

A powerful command-line tool for extracting translatable strings from Go code and templates, and managing translation catalogs in JSON format for WebFram applications. Streamlines the internationalization workflow with automatic placeholder detection, plural forms support, and intelligent catalog merging.

## Overview

This tool automates the tedious process of managing translations by extracting translatable strings from:

- **Go source files**: Automatically detects i18n printer methods (`Sprintf`, `Printf`, `Fprintf`, `Sprint`, `Print`, `Fprint`, `Sprintln`, `Println`, `Fprintln`)
- **Go HTML templates**: Detects `{{T "..." ...}}` template function calls in `.go.html` files
- **Go text templates**: Detects `{{T "..." ...}}` template function calls in `.go.txt` files

It generates and maintains JSON catalog files compatible with Go's `golang.org/x/text/message` package, preserving existing translations while intelligently adding new strings and removing obsolete ones.

## Features

- ✅ **Dual Source Extraction**: Extracts translations from both Go code and HTML/text templates in a single run
- ✅ **Intelligent Type Detection**: Automatically detects placeholder types (`%s`, `%d`, `%v`, etc.) with high accuracy
- ✅ **Plural Forms Support**: Generates plural form fields (`zero`, `one`, `two`, `few`, `many`, `other`) for messages with integer placeholders
- ✅ **Smart Catalog Merging**: Preserves existing translations when updating catalogs, never overwrites your work
- ✅ **Alphabetical Sorting**: Maintains alphabetically sorted message entries for easy navigation and diff tracking
- ✅ **Detailed Reporting**: Generates comprehensive extraction reports showing new, updated, and removed translations
- ✅ **Flexible Modes**: Configure extraction for templates-only, code-only, or both
- ✅ **Multi-Language Support**: Generate catalogs for unlimited languages in a single command
- ✅ **Validation**: Detects and reports duplicate message IDs and malformed translations
- ✅ **Performance**: Fast extraction even for large codebases with thousands of translatable strings
- ✅ **Cross-Platform**: Works on Windows, macOS, and Linux
- ✅ **Zero Configuration**: Sensible defaults with optional customization

## Installation

```bash
cd cmd/webfram-i18n
go build -o i18n
```

Or run directly:

```bash
go run cmd/webfram-i18n/main.go [flags]
```

## Usage

### Basic Usage

Extract translations from both templates and code:

```bash
go run cmd/webfram-i18n/main.go -languages "en,fr" -templates ./templates
```

### Command-Line Flags

| Flag | Default | Required | Description |
|------|---------|----------|-------------|
| `-languages` | _(none)_ | **YES** | Comma-separated list of language codes (e.g., `en,fr,es,de`) |
| `-templates` | _(none)_ | **YES** for `templates` and `both` modes | Directory containing template files |
| `-mode` | `both` | No | Extraction mode: `templates`, `code`, or `both` |
| `-code` | `.` (current directory) | No | Directory containing Go source files |
| `-locales` | `./locales` | No | Directory for message files (input/output) |

**Note:** The `-languages` flag is always required. The `-templates` flag is required when using `-mode templates` or `-mode both` (default).

### Examples

#### Extract from both templates and code (most common)

Ideal for full-stack applications with server-side rendering:

```bash
go run cmd/webfram-i18n/main.go -languages "en,fr" -templates ./templates
```

#### Extract only from Go code

Perfect for API-only services or microservices:

```bash
go run cmd/webfram-i18n/main.go -languages "en,fr,es" -mode code
```

#### Extract from custom directories

For projects with non-standard structure:

```bash
go run cmd/webfram-i18n/main.go \
  -languages "en,fr" \
  -mode both \
  -code ./src \
  -templates ./views \
  -locales ./i18n
```

#### Extract for multiple languages

Large international application:

```bash
go run cmd/webfram-i18n/main.go -languages "en,de,ja,zh" -templates ./templates
```

#### Extract for a single language

During initial development:

```bash
go run cmd/webfram-i18n/main.go -languages "en" -mode code
```

#### Extract with custom locale directory

For projects using a different translations structure:

```bash
go run cmd/webfram-i18n/main.go \
  -languages "en,fr,de,es,it,pt" \
  -mode code \
  -locales ./translations
```

#### CI/CD Integration

Automate translation extraction in your build pipeline:

```bash
#!/bin/bash
# Extract translations and check for changes
go run cmd/webfram-i18n/main.go -languages "en,fr,de,es" -templates ./templates

# Check if any translations were added or modified
if git diff --quiet locales/; then
    echo "No translation changes detected"
else
    echo "Translation files have been updated"
    git add locales/
    git commit -m "chore: update translation catalogs"
fi
```
```

## Using Translations in WebFram Applications

### 1. Configure I18n in Your Application

```go
package main

import (
    "embed"
    app "github.com/bondowe/webfram"
    "golang.org/x/text/language"
)

//go:embed locales
var assetsFS embed.FS

func main() {
    app.Configure(&app.Config{
        Assets: &app.Assets{
            FS: assetsFS,
            I18nMessages: &app.I18nMessages{
                Dir: "locales",
            },
        },
    })
    
    // Your application code...
}
```

### 2. Using Translations in Go Code

Get a printer for a specific language and use it to format translatable strings:

```go
package main

import (
    app "github.com/bondowe/webfram"
    "golang.org/x/text/language"
)

func handler(w app.ResponseWriter, r *app.Request) {
    // Get a printer for the user's language
    printer := app.GetI18nPrinter(language.French)
    
    // Simple string
    msg := printer.Sprintf("Client disconnected!")
    
    // String with placeholders
    greeting := printer.Sprintf("Welcome to %s! Today is %s.", "WebFram", "2024-01-01")
    
    // Use in responses
    w.WriteString(greeting)
}
```

**Supported printer methods:**

- `Sprintf(format string, args ...interface{}) string`
- `Printf(format string, args ...interface{})`
- `Fprintf(w io.Writer, format string, args ...interface{})`
- `Sprint(args ...interface{}) string`
- `Print(args ...interface{})`
- `Fprint(w io.Writer, args ...interface{})`
- `Sprintln(args ...interface{}) string`
- `Println(args ...interface{})`
- `Fprintln(w io.Writer, args ...interface{})`

### 3. Using Translations in Templates

Use the `T` function in your Go templates:

```html
<!-- users/manage/update.go.html -->
<!DOCTYPE html>
<html>
<head>
    <title>{{T "Welcome"}}</title>
</head>
<body>
    <h1>{{T "Welcome to %s! Today's date is %s." "Our World" "2023-01-01"}}</h1>
    
    <!-- Simple string -->
    <p>{{T "Click here to continue"}}</p>
    
    <!-- With placeholders -->
    <p>{{T "You have %d new messages" .MessageCount}}</p>
</body>
</html>
```

### 4. Context-Based I18n (Recommended for Web Handlers)

Store the printer in the request context for easy access:

```go
func languageMiddleware(next app.Handler) app.Handler {
    return app.HandlerFunc(func(w app.ResponseWriter, r *app.Request) {
        // Detect user's preferred language
        acceptLang := r.Header.Get("Accept-Language")
        tag, _ := language.MatchStrings(
            language.NewMatcher([]language.Tag{
                language.English,
                language.French,
                language.Spanish,
            }),
            acceptLang,
        )
        
        // Create and store printer in context
        printer := app.GetI18nPrinter(tag)
        ctx := app.ContextWithI18nPrinter(r.Context(), printer)
        
        // Call next handler with updated context
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func handler(w app.ResponseWriter, r *app.Request) {
    // Retrieve printer from context
    if printer, ok := app.PrinterFromContext(r.Context()); ok {
        msg := printer.Sprintf("Welcome to %s!", "WebFram")
        w.WriteString(msg)
    }
}
```

## Message Catalog Format

The tool generates JSON files following this structure:

```json
{
  "language": "en",
  "messages": [
    {
      "id": "Welcome to %s! Today is %s.",
      "message": "Welcome to %s! Today is %s.",
      "translation": "Welcome to %s! Today is %s.",
      "placeholders": {
        "arg_1": {
          "id": "arg_1",
          "string": "%s",
          "type": "string",
          "underlyingType": "string",
          "argNum": 1,
          "expr": "arg1"
        },
        "arg_2": {
          "id": "arg_2",
          "string": "%s",
          "type": "string",
          "underlyingType": "string",
          "argNum": 2,
          "expr": "arg2"
        }
      }
    }
  ]
}
```

### Supported Placeholder Types

The tool automatically detects placeholder types based on format verbs:

| Format Verb | Type | Example |
|-------------|------|---------|
| `%d`, `%b`, `%o`, `%x`, `%X` | `int` | `"You have %d items"` |
| `%f`, `%e`, `%E`, `%g`, `%G` | `float64` | `"Price: $%.2f"` |
| `%s`, `%q` | `string` | `"Hello, %s!"` |
| `%t` | `bool` | `"Active: %t"` |
| `%p` | `pointer` | `"Address: %p"` |
| `%v`, `%T` | `interface{}` | `"Value: %v"` |

## Workflow: Adding and Updating Translations

### 1. Write Code with Translatable Strings

In Go code:

```go
printer := app.GetI18nPrinter(language.French)
msg := printer.Sprintf("You have %d new messages", count)
```

In templates:

```html
<h1>{{T "Welcome to %s!" .AppName}}</h1>
```

### 2. Extract Translations

Run the extraction tool with required flags:

```bash
go run cmd/webfram-i18n/main.go -languages "en,fr,es" -templates ./templates
```

Output example:

```
=== Extracting Translations from Templates and Code ===
Found 5 translations in templates
Found 12 translations in Go code
Total unique translations: 15

=== Updating Message Catalogs ===
Updated ./locales/messages.en.json: +3 new
Updated ./locales/messages.fr.json: +3 new
Updated ./locales/messages.es.json: +3 new

=== Translation Summary ===
Total unique translation strings: 15

Translation strings found:
  - Client disconnected!
  - Welcome to %s! (placeholders: %s)
  - You have %d new messages (placeholders: %d)
  ...

✓ Extraction and merge completed successfully
```

### 3. Translate the Messages

Open each language file and add translations:

**messages.en.json** (source):

```json
{
  "id": "You have %d new messages",
  "message": "You have %d new messages",
  "translation": "You have %d new messages",
  ...
}
```

**messages.fr.json** (translate):

```json
{
  "id": "You have %d new messages",
  "message": "You have %d new messages",
  "translation": "Vous avez %d nouveaux messages",
  ...
}
```

**messages.es.json** (translate):

```json
{
  "id": "You have %d new messages",
  "message": "You have %d new messages",
  "translation": "Tienes %d mensajes nuevos",
  ...
}
```

### 4. Embed and Use in Your Application

The translations are automatically loaded when your application starts:

```go
//go:embed locales/*.json
var i18nFS embed.FS

func main() {
    app.Configure(&app.Config{
        I18n: &app.I18nConfig{
            FS: i18nFS,
        },
    })
    // Prints: "Loaded messages for language: en from locales/messages.en.json"
    // Prints: "Loaded messages for language: fr from locales/messages.fr.json"
    // ...
}
```

## Plural Forms Support

Messages with integer placeholders automatically include plural form fields:

```json
{
  "id": "You have %d items",
  "message": "You have %d items",
  "translation": "",
  "placeholders": { ... },
  "zero": "",
  "one": "",
  "two": "",
  "few": "",
  "many": "",
  "other": ""
}
```

Fill in the appropriate plural forms for each language:

```json
{
  "id": "You have %d items",
  "translation": "You have %d items",
  "one": "You have 1 item",
  "other": "You have %d items"
}
```

## Supported Languages

The tool creates message catalog files in the format `messages.{language}.json` for each specified language.

### Default Languages

By default, the tool extracts translations for these languages:

- `en-GB` - English
- `en-US` - English

### Custom Languages

You can specify any languages using the **required** `-languages` flag:

```bash
# Extract for specific languages with templates
go run cmd/webfram-i18n/main.go -languages "en-GB,en-US,fr-FR" -templates ./templates

# Single language from code only
go run cmd/webfram-i18n/main.go -languages "en-US" -mode code

# Many languages with custom directories
go run cmd/webfram-i18n/main.go \
  -languages "en-GB,fr-FR,de,es,it,pt,ja,zh,ko,ru,ar,hi" \
  -templates ./views \
  -code ./src
```

The flag accepts a comma-separated list of language codes. Spaces around commas are automatically trimmed.

### Common Language Codes

**European Languages:**

- `en` - English
- `en-US` - US English
- `en-GB` - British English
- `fr` - French
- `fr-CA` - Canadian French
- `fr-FR` - France French
- `es` - Spanish
- `es-ES` - Spain Spanish
- `es-MX` - Mexican Spanish
- `de` - German
- `de-DE` - Germany German
- `de-AT` - Austrian German
- `it` - Italian
- `pt` - Portuguese
- `pt-BR` - Brazilian Portuguese
- `pt-PT` - European Portuguese
- `nl` - Dutch
- `sv` - Swedish
- `no` - Norwegian
- `da` - Danish
- `fi` - Finnish
- `pl` - Polish
- `cs` - Czech
- `sk` - Slovak
- `hu` - Hungarian
- `ro` - Romanian
- `el` - Greek
- `tr` - Turkish

**Asian Languages:**

- `ja` - Japanese
- `zh` - Chinese (Mandarin)
- `zh-CN` - Simplified Chinese (China)
- `zh-TW` - Traditional Chinese (Taiwan)
- `zh-HK` - Traditional Chinese (Hong Kong)
- `ko` - Korean
- `th` - Thai
- `vi` - Vietnamese
- `id` - Indonesian
- `ms` - Malay
- `tl` - Tagalog (Filipino)
- `hi` - Hindi
- `bn` - Bengali
- `ta` - Tamil
- `te` - Telugu
- `ur` - Urdu

**Other Languages:**

- `ar` - Arabic
- `he` - Hebrew
- `fa` - Persian (Farsi)
- `ru` - Russian
- `uk` - Ukrainian
- `sw` - Swahili
- `am` - Amharic
- `af` - Afrikaans

### Changing Default Languages

If you prefer to modify the default languages in the code, edit the default value in [`main.go`](main.go):

```go
languagesFlag := flag.String("languages", "en,fr,de,es,it", "Comma-separated list of language codes")
```

## How It Works

1. **Template Scanning**: Uses regex to find `{{T "..." ...}}` patterns in `.go.html` and `.go.txt` files
2. **Code Scanning**: Uses Go's AST parser to find i18n printer method calls in `.go` files
3. **Placeholder Detection**: Analyzes format strings to identify placeholder types
4. **Catalog Merging**: Combines new translations with existing catalogs, preserving translations
5. **File Writing**: Saves updated catalogs with consistent formatting and alphabetical ordering

## Tips and Best Practices

### ✅ Do

- Use descriptive message IDs that clearly indicate the context
- Keep format placeholders simple and consistent
- Run extraction after adding new translatable strings
- Review the extraction summary for accuracy
- Commit generated catalog files to version control
- Use regional variants (e.g., `en-US`, `en-GB`) when needed for locale-specific content

### ❌ Don't

- Don't manually edit the `id` or `message` fields (only edit `translation`)
- Don't remove placeholder definitions
- Don't concatenate translatable strings in code
- Don't use complex expressions in template `T` calls

### Best Practices

- Always specify required languages explicitly with `-languages` flag
- Run extraction regularly as part of your development workflow
- Use a consistent naming convention for message IDs
- Test translations in different languages before deployment
- Keep your language list consistent across development, staging, and production
- Use the `-languages` flag in CI/CD pipelines to ensure all required languages are generated
- For code-only projects, use `-mode code` to skip template extraction

### Working with Language Variants

When using language variants (like `en-US` vs `en-GB`), consider:

```bash
# Generate both base and variant
go run cmd/webfram-i18n/main.go \
  -languages "en,en-US,en-GB" \
  -mode code

# The tool will create:
# - messages.en.json (base English)
# - messages.en-US.json (US English)
# - messages.en-GB.json (British English)
```

Then translate only the differences in the variant files.

### Advanced Workflow Patterns

#### Continuous Translation Updates

```bash
#!/bin/bash
# run_i18n_update.sh - Run after adding new translatable strings

echo "Extracting translations..."
go run cmd/webfram-i18n/main.go -languages "en,fr,de,es,ja" -templates ./templates

echo "
Files updated:"
ls -lh locales/

echo "
Please review and translate new strings in locales/*.json"
echo "Look for entries where 'translation' field is empty"
```

#### Pre-commit Hook

Automate extraction before commits:

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Checking for untranslated strings..."
go run cmd/webfram-i18n/main.go -languages "en,fr" -templates ./templates > /dev/null

if git diff --name-only | grep -q "locales/"; then
    echo "Warning: Translation files have been updated"
    echo "Please review the changes in locales/ directory"
    exit 1
fi
```

#### Translation Status Report

Check translation completeness:

```bash
#!/bin/bash
# check_translations.sh

for file in locales/messages.*.json; do
    lang=$(basename "$file" .json | sed 's/messages.//')
    total=$(jq '.messages | length' "$file")
    untranslated=$(jq '[.messages[] | select(.translation == "")] | length' "$file")
    percent=$(( (total - untranslated) * 100 / total ))
    
    echo "$lang: $percent% complete ($untranslated/$total untranslated)"
done
```

## Troubleshooting

### Translations not appearing in application

**Symptoms**: Your app runs but doesn't show translated text.

**Solutions**:

1. Ensure catalogs are embedded:

```go
//go:embed locales
var assetsFS embed.FS
```

2. Check that i18n is configured:

```go
app.Configure(&app.Config{
    Assets: &app.Assets{
        FS: assetsFS,
        I18nMessages: &app.I18nMessages{Dir: "locales"},
    },
})
```

3. Verify the language tag matches the catalog filename:

```go
// If you have messages.fr-FR.json, use:
printer := app.GetI18nPrinter(language.MustParse("fr-FR"))
```

4. Check console output for loading errors:

```text
// Should see:
Loaded messages for language: en from locales/messages.en.json
Loaded messages for language: fr from locales/messages.fr.json
```

### Extraction tool not finding strings

**Symptoms**: The tool runs but doesn't extract expected strings.

**Solutions**:

1. Verify correct directories with flags:

```bash
# Check what directories you're scanning
go run cmd/webfram-i18n/main.go \
  -languages "en" \
  -code ./path/to/code \
  -templates ./path/to/templates
```

2. Ensure Go files use i18n printer methods, not `fmt` directly:

```go
// ✅ This will be extracted
printer := app.GetI18nPrinter(language.English)
msg := printer.Sprintf("Hello %s", name)

// ❌ This will NOT be extracted
msg := fmt.Sprintf("Hello %s", name)
```

3. Ensure template files use correct syntax:

```html
<!-- ✅ This will be extracted -->
<h1>{{T "Welcome to %s" .AppName}}</h1>

<!-- ❌ This will NOT be extracted -->
<h1>{{.Title}}</h1>
```

4. Review the extraction summary output:

```text
Found 5 translations in templates
Found 12 translations in Go code
Total unique translations: 15
```

### Placeholders not working correctly

**Symptoms**: Variables not showing in translated strings.

**Solutions**:

1. Check placeholder format verbs match argument types:

```go
// ✅ Correct
printer.Sprintf("You have %d messages", 5)        // %d for int
printer.Sprintf("Hello %s", "John")              // %s for string
printer.Sprintf("Price: $%.2f", 19.99)           // %f for float

// ❌ Incorrect
printer.Sprintf("You have %s messages", 5)        // Wrong: %s for int
printer.Sprintf("Price: $%d", 19.99)             // Wrong: %d for float
```

2. Verify placeholders in translations match source:

```json
{
  "id": "You have %d messages",
  "translation": "Vous avez %d messages"  // ✅ Same placeholder
}
```

3. Ensure argument order is preserved:

```json
{
  "id": "Welcome %s, you have %d messages",
  "translation": "Bienvenue %s, vous avez %d messages"  // ✅ Same order
}
```

### Invalid language codes

**Symptoms**: Error about invalid languages when running tool.

**Solutions**:

1. Use BCP 47 format (ISO 639-1 language + optional ISO 3166-1 region):

```bash
# ✅ Correct
-languages "en,fr,de,es,pt"
-languages "en-US,en-GB,fr-FR,pt-BR"

# ❌ Incorrect
-languages "english,french"           # Use codes, not names
-languages "en_US,fr_FR"             # Use hyphen, not underscore
```

2. Check for typos:

```bash
# Common typos:
-languages "eng"     # Should be: en
-languages "fra"     # Should be: fr
-languages "esp"     # Should be: es
```

3. Remove extra spaces:

```bash
# ✅ Correct (spaces are auto-trimmed, but better without)
-languages "en,fr,de"

# ⚠️ Acceptable but not recommended
-languages "en, fr, de"
```

### Missing language files

**Symptoms**: Expected language files aren't created.

**Solutions**:

1. Verify the `-languages` flag:

```bash
# Check spelling and format
go run cmd/webfram-i18n/main.go -languages "en,fr,de" -mode code
```

2. Check `-locales` directory path and permissions:

```bash
# Verify directory exists and is writable
ls -la ./locales
mkdir -p ./locales  # Create if doesn't exist
```

3. Review console output for errors:

```text
Error creating file: open ./locales/messages.fr.json: permission denied
```

4. Check disk space:

```bash
df -h .
```

### Duplicate message IDs

**Symptoms**: Warning about duplicate translations.

**Solutions**:

1. Use unique, descriptive message IDs:

```go
// ❌ Avoid generic messages
printer.Sprintf("Error")           // Too generic
printer.Sprintf("Submit")          // Too generic

// ✅ Use specific context
printer.Sprintf("Login error: Invalid credentials")
printer.Sprintf("Submit user registration form")
```

2. Add context to distinguish similar messages:

```go
// Different contexts
printer.Sprintf("Dashboard: Welcome %s", name)      // Dashboard greeting
printer.Sprintf("Email: Welcome %s", name)          // Email greeting
printer.Sprintf("Notification: Welcome %s", name)   // Notification greeting
```

### Performance issues with large codebases

**Symptoms**: Tool takes too long to extract translations.

**Solutions**:

1. Use specific directories instead of scanning entire project:

```bash
# ✅ Scan specific directories
go run cmd/webfram-i18n/main.go \
  -languages "en" \
  -code ./cmd ./internal ./pkg \
  -templates ./templates

# ❌ Don't scan everything
go run cmd/webfram-i18n/main.go \
  -languages "en" \
  -code ./
```

2. Exclude vendor and generated code:

```bash
# The tool automatically skips vendor/, but you can help by
# organizing code to avoid scanning unnecessary directories
```

3. Run for fewer languages during development:

```bash
# During development, only extract for base language
go run cmd/webfram-i18n/main.go -languages "en" -mode code

# Full extraction for production/release
go run cmd/webfram-i18n/main.go -languages "en,fr,de,es,ja,zh" -templates ./templates
```

### Translation file merge conflicts

**Symptoms**: Git merge conflicts in translation JSON files.

**Solutions**:

1. Always pull latest changes before running extraction:

```bash
git pull origin main
go run cmd/webfram-i18n/main.go -languages "en,fr" -templates ./templates
```

2. Use a merge strategy for JSON files in `.gitattributes`:

```text
locales/*.json merge=union
```

3. Manually resolve conflicts, then re-run extraction:

```bash
# Resolve conflicts in your editor
# Then re-run to ensure proper formatting
go run cmd/webfram-i18n/main.go -languages "en,fr" -templates ./templates
git add locales/
git commit -m "Resolve translation conflicts"
```

## Related Files

- [`cmd/webfram-i18n/main.go`](main.go) - Extraction tool source code
- [`cmd/web/main.go`](../web/main.go) - Example webfram application
- [`webfram/internal/i18n/i18n.go`](../../webfram/internal/i18n/i18n.go) - I18n implementation
- [`cmd/web/locales/`](../web/locales/) - Example message catalogs

## License

Part of the webfram project.

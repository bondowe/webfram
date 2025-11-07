# I18n Extraction Tool

A command-line tool for extracting translatable strings from Go code and templates, and managing translation catalogs in JSON format for webfram applications.

## Overview

This tool automatically extracts translatable strings from:

- **Go source files**: Detects i18n printer methods (`Sprintf`, `Printf`, `Fprintf`, etc.)
- **Go templates**: Detects `{{T "..." ...}}` template function calls

It generates and maintains JSON catalog files compatible with Go's `golang.org/x/text/message` package, preserving existing translations while adding new strings and removing obsolete ones.

## Features

- ✅ Extracts translations from both Go code and HTML/text templates
- ✅ Automatically detects placeholder types (`%s`, `%d`, `%v`, etc.)
- ✅ Supports plural forms for messages with integer placeholders
- ✅ Preserves existing translations when updating catalogs
- ✅ Maintains alphabetically sorted message entries
- ✅ Generates detailed extraction reports
- ✅ Configurable extraction modes and directories
- ✅ Customizable language list via command-line flag

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

```bash
go run cmd/webfram-i18n/main.go -languages "en,fr" -templates ./templates
```

#### Extract only from Go code

```bash
go run cmd/webfram-i18n/main.go -languages "en,fr,es" -mode code
```

#### Extract from custom directories

```bash
go run cmd/webfram-i18n/main.go \
  -languages "en,fr" \
  -mode both \
  -code ./src \
  -templates ./views \
  -locales ./i18n
```

#### Extract for multiple languages

```bash
go run cmd/webfram-i18n/main.go -languages "en,de,ja,zh" -templates ./templates
```

#### Extract for a single language

```bash
go run cmd/webfram-i18n/main.go -languages "en" -mode code
```

#### Extract from current directory code with custom locales

```bash
go run cmd/webfram-i18n/main.go \
  -languages "en,fr,de,es,it,pt" \
  -mode code \
  -locales ./translations
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

//go:embed locales/*.json
var i18nFS embed.FS

func main() {
    app.Configure(&app.Config{
        I18n: &app.I18nConfig{
            FS: i18nFS,
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

## Troubleshooting

### Translations not appearing in application

1. Ensure catalogs are embedded: `//go:embed locales/*.json`
2. Check that i18n is configured in `app.Configure()`
3. Verify the language tag matches the catalog filename
4. Check console output for loading errors

### Extraction tool not finding strings

1. Verify the correct directories are specified with flags
2. Check that Go files use i18n printer methods (not `fmt` package directly)
3. Ensure template files use `{{T "..." ...}}` syntax
4. Review the extraction summary output for clues

### Placeholders not working correctly

1. Check that placeholder format verbs match the argument types
2. Verify placeholders in translations match the source message
3. Ensure argument order is preserved in translations

### Invalid language codes

If you see an error about invalid languages:

1. Ensure language codes follow BCP 47 format (e.g., `en`, `en-US`, `fr-CA`)
2. Check for typos in your `-languages` flag
3. Remove any extra spaces or invalid characters

### Missing language files

If expected language files aren't created:

1. Verify the `-languages` flag is set correctly
2. Check that the `-locales` directory path is correct and writable
3. Review the console output for any errors during file creation

## Related Files

- [`cmd/webfram-i18n/main.go`](main.go) - Extraction tool source code
- [`cmd/web/main.go`](../web/main.go) - Example webfram application
- [`webfram/internal/i18n/i18n.go`](../../webfram/internal/i18n/i18n.go) - I18n implementation
- [`cmd/web/locales/`](../web/locales/) - Example message catalogs

## License

Part of the webfram project.

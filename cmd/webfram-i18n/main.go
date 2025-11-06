package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// Placeholder represents a placeholder in a translation message
type Placeholder struct {
	ID             string `json:"id"`
	String         string `json:"string"`
	Type           string `json:"type"`
	UnderlyingType string `json:"underlyingType"`
	Expr           string `json:"expr"`
	ArgNum         int    `json:"argNum"`
}

// Message represents a translation message in gotext format with plural support
type Message struct {
	ID           string                 `json:"id"`
	Key          string                 `json:"key,omitempty"`
	Message      string                 `json:"message"`
	Translation  string                 `json:"translation,omitempty"`
	Placeholders map[string]Placeholder `json:"placeholders,omitempty"`
	// For plural support
	Zero  string `json:"zero,omitempty"`
	One   string `json:"one,omitempty"`
	Two   string `json:"two,omitempty"`
	Few   string `json:"few,omitempty"`
	Many  string `json:"many,omitempty"`
	Other string `json:"other,omitempty"`
}

// Catalog represents a gotext catalog file
type Catalog struct {
	Language string    `json:"language"`
	Messages []Message `json:"messages"`
}

// TranslationInfo holds information about a translation string
type TranslationInfo struct {
	MessageID    string
	Placeholders []PlaceholderInfo
}

type PlaceholderInfo struct {
	Type   string
	ArgNum int
}

func main() {
	// Define command-line flags
	mode := flag.String("mode", "both", "Extraction mode: templates, code, or both")
	codeDir := flag.String("code", ".", "Directory containing Go source files (default: current directory)")
	templatesDir := flag.String("templates", "", "Directory containing template files (required for 'templates' and 'both' modes)")
	localesDir := flag.String("locales", "./locales", "Directory for message files (input and output)")
	languagesFlag := flag.String("languages", "", "Comma-separated list of language codes (e.g., en,fr,es,de) - REQUIRED")
	flag.Parse()

	// Validate languages - required parameter
	if *languagesFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: -languages flag is required\n")
		fmt.Fprintf(os.Stderr, "Example: -languages \"en,fr,es\"\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Parse languages from comma-separated string
	languages := parseLanguages(*languagesFlag)
	if len(languages) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No valid languages specified\n")
		os.Exit(1)
	}

	// Validate templates directory for modes that need it
	if (*mode == "templates" || *mode == "both") && *templatesDir == "" {
		fmt.Fprintf(os.Stderr, "Error: -templates flag is required for mode '%s'\n", *mode)
		fmt.Fprintf(os.Stderr, "Example: -templates \"./templates\"\n\n")
		flag.Usage()
		os.Exit(1)
	}

	var allTranslations map[string]TranslationInfo

	switch *mode {
	case "templates":
		fmt.Println("=== Extracting Template Translations ===")
		translations, err := extractTranslationsFromTemplates(*templatesDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		allTranslations = translations

	case "code":
		fmt.Println("=== Extracting Code Translations ===")
		translations, err := extractTranslationsFromGoFiles(*codeDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		allTranslations = translations

	case "both":
		fmt.Println("=== Extracting Translations from Templates and Code ===")

		// Extract from templates
		templateTranslations, err := extractTranslationsFromTemplates(*templatesDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error extracting template translations: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Found %d translations in templates\n", len(templateTranslations))

		// Extract from Go code
		codeTranslations, err := extractTranslationsFromGoFiles(*codeDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error extracting code translations: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Found %d translations in Go code (i18n printer calls, log calls, and validation errmsg tags)\n", len(codeTranslations))

		// Merge both
		allTranslations = mergeTranslations(templateTranslations, codeTranslations)
		fmt.Printf("Total unique translations: %d\n", len(allTranslations))

	default:
		fmt.Fprintf(os.Stderr, "Invalid mode: %s. Use 'templates', 'code', or 'both'\n", *mode)
		flag.Usage()
		os.Exit(1)
	}

	if len(allTranslations) == 0 {
		fmt.Println("No translations found")
		return
	}

	// Create locales directory if it doesn't exist
	if err := os.MkdirAll(*localesDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating locales directory: %v\n", err)
		os.Exit(1)
	}

	// Merge and update catalogs for each language
	fmt.Println("\n=== Updating Message Catalogs ===")
	for _, lang := range languages {
		if err := mergeAndUpdateCatalog(*localesDir, lang, allTranslations); err != nil {
			fmt.Fprintf(os.Stderr, "Error updating catalog for %s: %v\n", lang, err)
			os.Exit(1)
		}
	}

	printTranslationSummary(allTranslations)
	fmt.Println("\n✓ Extraction and merge completed successfully")
}

// parseLanguages splits a comma-separated string into a slice of language codes
func parseLanguages(input string) []string {
	if input == "" {
		return nil
	}

	var languages []string
	parts := strings.Split(input, ",")
	for _, part := range parts {
		lang := strings.TrimSpace(part)
		if lang != "" {
			languages = append(languages, lang)
		}
	}
	return languages
}

// mergeTranslations merges translations from multiple sources
func mergeTranslations(sources ...map[string]TranslationInfo) map[string]TranslationInfo {
	merged := make(map[string]TranslationInfo)

	for _, source := range sources {
		for msgID, info := range source {
			merged[msgID] = info
		}
	}

	return merged
}

// catalogsAreEqual checks if two catalogs are semantically equal (ignoring message order)
func catalogsAreEqual(catalog1, catalog2 *Catalog) bool {
	if catalog1.Language != catalog2.Language {
		return false
	}

	if len(catalog1.Messages) != len(catalog2.Messages) {
		return false
	}

	// Create maps for comparison
	messages1 := make(map[string]Message)
	for _, msg := range catalog1.Messages {
		messages1[msg.ID] = msg
	}

	messages2 := make(map[string]Message)
	for _, msg := range catalog2.Messages {
		messages2[msg.ID] = msg
	}

	// Check if all messages are equal
	for id, msg1 := range messages1 {
		msg2, exists := messages2[id]
		if !exists {
			return false
		}
		if !messagesAreEqual(msg1, msg2) {
			return false
		}
	}

	return true
}

// messagesAreEqual checks if two messages are equal
func messagesAreEqual(msg1, msg2 Message) bool {
	if msg1.ID != msg2.ID ||
		msg1.Key != msg2.Key ||
		msg1.Message != msg2.Message ||
		msg1.Translation != msg2.Translation ||
		msg1.Zero != msg2.Zero ||
		msg1.One != msg2.One ||
		msg1.Two != msg2.Two ||
		msg1.Few != msg2.Few ||
		msg1.Many != msg2.Many ||
		msg1.Other != msg2.Other {
		return false
	}

	// Compare placeholders
	if len(msg1.Placeholders) != len(msg2.Placeholders) {
		return false
	}

	for key, ph1 := range msg1.Placeholders {
		ph2, exists := msg2.Placeholders[key]
		if !exists {
			return false
		}
		if ph1.ID != ph2.ID ||
			ph1.String != ph2.String ||
			ph1.Type != ph2.Type ||
			ph1.UnderlyingType != ph2.UnderlyingType ||
			ph1.ArgNum != ph2.ArgNum ||
			ph1.Expr != ph2.Expr {
			return false
		}
	}

	return true
}

// mergeAndUpdateCatalog merges new translations with existing catalog
func mergeAndUpdateCatalog(localesDir, lang string, newTranslations map[string]TranslationInfo) error {
	filename := filepath.Join(localesDir, fmt.Sprintf("messages.%s.json", lang))

	// Try to load existing catalog
	existingCatalog, err := loadExistingCatalog(filename)
	if err != nil {
		// If file doesn't exist, create a new catalog
		if os.IsNotExist(err) {
			fmt.Printf("Creating new catalog: %s\n", filename)
			return createNewCatalog(filename, lang, newTranslations)
		}
		return fmt.Errorf("error loading existing catalog: %w", err)
	}

	// Create a map of existing messages for quick lookup
	existingMessages := make(map[string]Message)
	for _, msg := range existingCatalog.Messages {
		existingMessages[msg.ID] = msg
	}

	// Build the merged catalog
	mergedCatalog := Catalog{
		Language: lang,
		Messages: []Message{},
	}

	addedCount := 0
	removedCount := 0

	// Sort message IDs alphabetically for consistent ordering
	sortedIDs := make([]string, 0, len(newTranslations))
	for msgID := range newTranslations {
		sortedIDs = append(sortedIDs, msgID)
	}
	sort.Strings(sortedIDs)

	// Process all new translations in sorted order
	for _, msgID := range sortedIDs {
		info := newTranslations[msgID]
		if existingMsg, exists := existingMessages[msgID]; exists {
			// Message exists - preserve translation and metadata
			updatedMsg := createMessage(msgID, info)
			updatedMsg.Translation = existingMsg.Translation

			// Preserve plural forms if they exist
			if existingMsg.Zero != "" {
				updatedMsg.Zero = existingMsg.Zero
			}
			if existingMsg.One != "" {
				updatedMsg.One = existingMsg.One
			}
			if existingMsg.Two != "" {
				updatedMsg.Two = existingMsg.Two
			}
			if existingMsg.Few != "" {
				updatedMsg.Few = existingMsg.Few
			}
			if existingMsg.Many != "" {
				updatedMsg.Many = existingMsg.Many
			}
			if existingMsg.Other != "" {
				updatedMsg.Other = existingMsg.Other
			}

			mergedCatalog.Messages = append(mergedCatalog.Messages, updatedMsg)
		} else {
			// New message
			addedCount++
			mergedCatalog.Messages = append(mergedCatalog.Messages, createMessage(msgID, info))
		}
	}

	// Count removed messages (exist in old but not in new)
	for msgID := range existingMessages {
		if _, exists := newTranslations[msgID]; !exists {
			removedCount++
		}
	}

	// Check if the catalogs are semantically equal (ignoring order)
	if catalogsAreEqual(existingCatalog, &mergedCatalog) {
		fmt.Printf("Skipped %s: no changes detected\n", filename)
		return nil
	}

	// Write the merged catalog
	if err := writeCatalog(filename, mergedCatalog); err != nil {
		return err
	}

	// Only report if there are actual changes
	if addedCount > 0 || removedCount > 0 {
		status := "Updated"
		details := []string{}
		if addedCount > 0 {
			details = append(details, fmt.Sprintf("+%d new", addedCount))
		}
		if removedCount > 0 {
			details = append(details, fmt.Sprintf("-%d removed", removedCount))
		}
		fmt.Printf("%s %s: %s\n", status, filename, strings.Join(details, ", "))
	} else {
		fmt.Printf("Updated %s: reordered entries\n", filename)
	}

	return nil
}

// loadExistingCatalog loads an existing catalog file
func loadExistingCatalog(filename string) (*Catalog, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var catalog Catalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("error parsing catalog: %w", err)
	}

	return &catalog, nil
}

// createNewCatalog creates a new catalog file
func createNewCatalog(filename, lang string, translations map[string]TranslationInfo) error {
	catalog := Catalog{
		Language: lang,
		Messages: []Message{},
	}

	// Sort message IDs alphabetically for consistent ordering
	sortedIDs := make([]string, 0, len(translations))
	for msgID := range translations {
		sortedIDs = append(sortedIDs, msgID)
	}
	sort.Strings(sortedIDs)

	for _, msgID := range sortedIDs {
		info := translations[msgID]
		catalog.Messages = append(catalog.Messages, createMessage(msgID, info))
	}

	return writeCatalog(filename, catalog)
}

// createMessage creates a Message from TranslationInfo
func createMessage(msgID string, info TranslationInfo) Message {
	msg := Message{
		ID:           msgID,
		Message:      msgID,
		Translation:  "", // Empty for new entries
		Placeholders: make(map[string]Placeholder),
	}

	// Add placeholders
	for i, ph := range info.Placeholders {
		placeholderID := fmt.Sprintf("arg_%d", i+1)
		msg.Placeholders[placeholderID] = Placeholder{
			ID:             placeholderID,
			String:         fmt.Sprintf("%%%s", getFormatSpecifier(ph.Type)),
			Type:           ph.Type,
			UnderlyingType: ph.Type,
			ArgNum:         ph.ArgNum,
			Expr:           fmt.Sprintf("arg%d", ph.ArgNum),
		}
	}

	// If this is a potential plural message (contains %d), add plural forms
	if containsIntegerPlaceholder(info.Placeholders) {
		msg.Zero = ""  // Optional: specific form for zero
		msg.One = ""   // Form for singular (1)
		msg.Two = ""   // Optional: specific form for two
		msg.Few = ""   // Optional: specific form for few
		msg.Many = ""  // Optional: specific form for many
		msg.Other = "" // Form for all other cases (default plural)
	}

	return msg
}

// extractTranslationsFromTemplates extracts translations from template files
func extractTranslationsFromTemplates(dir string) (map[string]TranslationInfo, error) {
	translations := make(map[string]TranslationInfo)

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Printf("Warning: templates directory %s does not exist, skipping template extraction\n", dir)
		return translations, nil
	}

	// Regular expression to match {{T "..." ...}} patterns
	tPattern := regexp.MustCompile(`\{\{T\s+"([^"]+)"(?:\s+([^}]+))?\}\}`)

	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Only process template files
		if !strings.HasSuffix(path, ".go.html") && !strings.HasSuffix(path, ".go.txt") {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", path, err)
		}

		// Find all T function calls
		matches := tPattern.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) > 1 {
				msgID := match[1]
				placeholders := extractPlaceholders(msgID)

				translations[msgID] = TranslationInfo{
					MessageID:    msgID,
					Placeholders: placeholders,
				}
			}
		}

		return nil
	})

	return translations, err
}

// extractTranslationsFromGoFiles extracts translations from Go source files
// Includes: i18n printer calls, log calls (fmt, log packages), and validation errmsg tags
func extractTranslationsFromGoFiles(dir string) (map[string]TranslationInfo, error) {
	translations := make(map[string]TranslationInfo)

	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Parse the Go file
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error parsing %s: %v\n", path, err)
			return nil // Continue processing other files
		}

		// Walk the AST to find:
		// 1. i18n printer calls (printer.Sprintf, etc.)
		// 2. Log calls (fmt.Printf, log.Printf, etc.)
		// 3. Struct field tags with errmsg
		ast.Inspect(node, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.CallExpr:
				// Handle function calls (i18n printer and log calls)
				handleCallExpr(node, translations)
			case *ast.StructType:
				// Handle struct field tags
				handleStructType(node, translations)
			}
			return true
		})

		return nil
	})

	return translations, err
}

// handleCallExpr processes function calls to extract translatable strings
func handleCallExpr(callExpr *ast.CallExpr, translations map[string]TranslationInfo) {
	var isTranslatable bool
	var funcName string

	switch fun := callExpr.Fun.(type) {
	case *ast.SelectorExpr:
		// Handle selector expressions (e.g., printer.Sprintf, fmt.Printf, log.Printf)
		funcName = fun.Sel.Name

		// Check if it's an identifier (package or variable name)
		if ident, ok := fun.X.(*ast.Ident); ok {
			pkgName := ident.Name

			// Check for i18n printer methods
			if isI18nMethod(funcName) {
				isTranslatable = true
			}

			// Check for log calls (fmt, log packages)
			if isLogPackage(pkgName) && isLogMethod(funcName) {
				isTranslatable = true
			}
		}

	case *ast.Ident:
		// Handle direct function calls (e.g., Printf, Sprintf)
		funcName = fun.Name
		if isLogMethod(funcName) {
			isTranslatable = true
		}
	}

	if !isTranslatable {
		return
	}

	// Extract the first argument (message string)
	if len(callExpr.Args) < 1 {
		return
	}

	lit, ok := callExpr.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return
	}

	// Remove quotes from string literal
	messageID := strings.Trim(lit.Value, "`\"")

	// Skip empty strings
	if messageID == "" {
		return
	}

	// Extract placeholders
	placeholders := extractPlaceholders(messageID)

	translations[messageID] = TranslationInfo{
		MessageID:    messageID,
		Placeholders: placeholders,
	}
}

// handleStructType processes struct types to extract errmsg tags
func handleStructType(structType *ast.StructType, translations map[string]TranslationInfo) {
	if structType.Fields == nil {
		return
	}

	for _, field := range structType.Fields.List {
		if field.Tag == nil {
			continue
		}

		// Parse the struct tag
		tagValue := strings.Trim(field.Tag.Value, "`")
		tag := reflect.StructTag(tagValue)

		// Get the errmsg tag value
		errmsgTag := tag.Get("errmsg")
		if errmsgTag == "" {
			continue
		}

		// Parse errmsg tag: "rule1=message1;rule2=message2"
		rules := strings.Split(errmsgTag, ";")
		for _, rule := range rules {
			parts := strings.SplitN(rule, "=", 2)
			if len(parts) != 2 {
				continue
			}

			messageID := strings.TrimSpace(parts[1])
			if messageID == "" {
				continue
			}

			// Extract placeholders
			placeholders := extractPlaceholders(messageID)

			translations[messageID] = TranslationInfo{
				MessageID:    messageID,
				Placeholders: placeholders,
			}
		}
	}
}

// isI18nMethod checks if a method name is an i18n method
func isI18nMethod(name string) bool {
	i18nMethods := []string{
		"Sprintf", "Printf", "Fprintf",
		"Sprint", "Print", "Fprint",
		"Sprintln", "Println", "Fprintln",
	}
	for _, method := range i18nMethods {
		if name == method {
			return true
		}
	}
	return false
}

// isLogPackage checks if a package name is a logging package
func isLogPackage(pkgName string) bool {
	logPackages := []string{
		"fmt",
		"log",
	}
	for _, pkg := range logPackages {
		if pkgName == pkg {
			return true
		}
	}
	return false
}

// isLogMethod checks if a method name is a logging method
func isLogMethod(name string) bool {
	logMethods := []string{
		"Printf", "Print", "Println",
		"Sprintf", "Sprint", "Sprintln",
		"Fprintf", "Fprint", "Fprintln",
		"Errorf", "Error", "Errorln",
		"Fatalf", "Fatal", "Fatalln",
		"Panicf", "Panic", "Panicln",
	}
	for _, method := range logMethods {
		if name == method {
			return true
		}
	}
	return false
}

// printTranslationSummary prints a summary of extracted translations
func printTranslationSummary(translations map[string]TranslationInfo) {
	fmt.Printf("\n=== Translation Summary ===\n")
	fmt.Printf("Total unique translation strings: %d\n", len(translations))

	if len(translations) == 0 {
		return
	}

	fmt.Println("\nTranslation strings found:")

	// Sort message IDs for consistent output
	sortedIDs := make([]string, 0, len(translations))
	for msgID := range translations {
		sortedIDs = append(sortedIDs, msgID)
	}
	sort.Strings(sortedIDs)

	for _, msgID := range sortedIDs {
		info := translations[msgID]
		fmt.Printf("  - %s", msgID)
		if len(info.Placeholders) > 0 {
			fmt.Printf(" (placeholders: ")
			for i, ph := range info.Placeholders {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%%%s", getFormatSpecifier(ph.Type))
			}
			fmt.Print(")")
		}
		fmt.Println()
	}
}

func extractPlaceholders(message string) []PlaceholderInfo {
	var placeholders []PlaceholderInfo

	// Pattern to match printf-style format specifiers
	formatPattern := regexp.MustCompile(`%([+\-#0 ]*)(\*|\d+)?(\.\*|\.\d+)?([vTtbcdoOqxXUeEfFgGsp%])`)

	matches := formatPattern.FindAllStringSubmatch(message, -1)
	for i, match := range matches {
		if len(match) > 4 {
			verb := match[4]
			if verb == "%" { // Skip escaped %
				continue
			}
			placeholderType := inferPlaceholderType(verb)
			placeholders = append(placeholders, PlaceholderInfo{
				Type:   placeholderType,
				ArgNum: i + 1,
			})
		}
	}

	return placeholders
}

func inferPlaceholderType(verb string) string {
	switch verb {
	case "d", "b", "c", "o", "O", "x", "X", "U":
		return "int"
	case "e", "E", "f", "F", "g", "G":
		return "float64"
	case "s", "q":
		return "string"
	case "t":
		return "bool"
	case "p":
		return "pointer"
	case "v", "T":
		return "interface{}"
	default:
		return "interface{}"
	}
}

func getFormatSpecifier(placeholderType string) string {
	switch placeholderType {
	case "int":
		return "d"
	case "float64":
		return "f"
	case "string":
		return "s"
	case "bool":
		return "t"
	case "pointer":
		return "p"
	default:
		return "v"
	}
}

func containsIntegerPlaceholder(placeholders []PlaceholderInfo) bool {
	for _, ph := range placeholders {
		if ph.Type == "int" {
			return true
		}
	}
	return false
}

func writeCatalog(filename string, catalog Catalog) error {
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling catalog: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	return nil
}

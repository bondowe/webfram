package template

import (
	"errors"
	"fmt"
	htmlTemplate "html/template"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	textTemplate "text/template"
)

type Config struct {
	FS                    fs.FS
	LayoutBaseName        string
	HTMLTemplateExtension string
	TextTemplateExtension string
	I18nFuncName          string
}

var (
	config              *Config
	htmlLayoutFileName  string
	textLayoutFileName  string
	templatesCache      sync.Map       // map[string][string, *template.Template]
	partialsCache       sync.Map       // map[string]*htmlTemplate.Template - key: "folder|partialFilename"
	layoutsCache        map[string]any = make(map[string]any)
	layoutPatternString string
	layoutPattern       *regexp.Regexp
	funcMap             = htmlTemplate.FuncMap{}
)

// Configure initializes the template system with the provided configuration.
// It sets up the filesystem, template extensions, i18n function name, and layouts,
// then caches all templates found in the filesystem.
// Panics if any required configuration value is missing or invalid.
func Configure(cfg *Config) {
	config = cfg

	htmlLayoutFileName = config.LayoutBaseName + config.HTMLTemplateExtension
	textLayoutFileName = config.LayoutBaseName + config.TextTemplateExtension
	layoutPatternString = fmt.Sprintf("^_?(?:%s|%s)$", htmlLayoutFileName, textLayoutFileName)
	layoutPattern = regexp.MustCompile(layoutPatternString)

	funcMap[config.I18nFuncName] = fmt.Sprintf

	htmlLayouts := make([]string, 0)
	textLayouts := make([]string, 0)

	cacheTemplates(config.FS, ".", htmlLayouts, textLayouts)
	// Keep layoutsCache for dynamic template parsing
	// layoutsCache = nil
}

// Configuration returns the current template configuration.
// Returns the config and true if templates are configured, or an empty config and false if not configured.
func Configuration() (Config, bool) {
	if config == nil {
		return Config{}, false
	}
	return *config, true
}

// LookupTemplate retrieves a cached template by path.
// If absolute is true, uses the path as-is. If false, prepends the configured base path.
// Returns the template and true if found, or nil and false if not found.
func LookupTemplate(path string, absolute bool) (*htmlTemplate.Template, bool) {
	if absolute {
		if nv, ok := templatesCache.Load(path); ok {
			return nv.([2]any)[1].(*htmlTemplate.Template), ok
		}
		return nil, false
	}

	// For relative paths, search for templates ending with the given path
	var foundTemplate *htmlTemplate.Template
	var found bool
	templatesCache.Range(func(key, value any) bool {
		keyStr := key.(string)
		if strings.HasSuffix(keyStr, path) {
			foundTemplate = value.([2]any)[1].(*htmlTemplate.Template)
			found = true
			return false
		}
		return true
	})

	return foundTemplate, found
}

// Must is a helper that panics if err is not nil, otherwise returns v.
// Useful for wrapping function calls during initialization.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}

func cacheTemplates(dir fs.FS, dirPath string, htmlLayouts, textLayouts []string) {
	var layoutFilePath string

	if layoutFileName, ok := getLayout(dir, htmlLayoutFileName); ok {
		layoutFilePath = dirPath + "/" + layoutFileName
		layoutFilePath = strings.TrimPrefix(layoutFilePath, "./")

		if layoutFileName[:1] == "_" {
			htmlLayouts = []string{layoutFilePath}
		} else {
			htmlLayouts = append(htmlLayouts, layoutFilePath)
		}
	}

	if layoutFileName, ok := getLayout(dir, textLayoutFileName); ok {
		layoutFilePath = dirPath + "/" + layoutFileName
		layoutFilePath = strings.TrimPrefix(layoutFilePath, "./")

		if layoutFileName[:1] == "_" {
			textLayouts = []string{layoutFilePath}
		} else {
			textLayouts = append(textLayouts, layoutFilePath)
		}
	}

	templates := Must(fs.ReadDir(dir, "."))

	for _, entry := range templates {
		if entry.IsDir() {
			entryFS := Must(fs.Sub(dir, entry.Name()))
			nestedDirPath := dirPath + "/" + entry.Name()

			cacheTemplates(entryFS, nestedDirPath, htmlLayouts, textLayouts)
			continue
		}
		isLayoutFile := layoutPattern.MatchString(entry.Name())
		isHTMLTemplateFile := strings.HasSuffix(entry.Name(), config.HTMLTemplateExtension)
		isTextTemplateFile := strings.HasSuffix(entry.Name(), config.TextTemplateExtension)

		if isLayoutFile || !isHTMLTemplateFile && !isTextTemplateFile {
			continue
		}

		htmlLayoutsClone := slices.Clone(htmlLayouts)
		textLayoutsClone := slices.Clone(textLayouts)

		if strings.HasPrefix(entry.Name(), "_") {
			htmlLayoutsClone = nil
			textLayoutsClone = nil
		}

		templatePath := dirPath + "/" + entry.Name()
		templatePath = strings.TrimPrefix(templatePath, "./")

		if isHTMLTemplateFile {
			name, template := parseHTMLTemplate(templatePath, htmlLayoutsClone)
			templatesCache.Store(templatePath, [2]any{name, template})
		}

		if isTextTemplateFile {
			name, template := parseTextTemplate(templatePath, textLayoutsClone)
			templatesCache.Store(templatePath, [2]any{name, template})
		}
	}
}

func getLayout(dir fs.FS, layoutName string) (string, bool) {
	standardLayoutExists := layoutExists(dir, layoutName)
	noInheritLayoutExists := layoutExists(dir, "_"+layoutName)

	if standardLayoutExists && noInheritLayoutExists {
		panic(fmt.Errorf("both layout and _layout exist, ambiguous"))
	}

	if standardLayoutExists {
		return layoutName, true
	} else if noInheritLayoutExists {
		return "_" + layoutName, true
	}

	return "", false
}

func layoutExists(dir fs.FS, layoutName string) bool {
	layoutStat, err := fs.Stat(dir, layoutName)

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false
		}
		panic(err)
	}

	if !layoutStat.Mode().IsRegular() {
		return false
	}

	return true
}

func lookUpPartial(folder, partialFilename string) *htmlTemplate.Template {
	// Create cache key from starting folder and partial filename
	cacheKey := folder + "|" + partialFilename

	// Check if we've already resolved this partial from this folder
	if cached, ok := partialsCache.Load(cacheKey); ok {
		if cached == nil {
			return nil
		}
		return cached.(*htmlTemplate.Template)
	}

	// Search up the directory tree
	currentFolder := folder
	for {
		var partialPath string
		if currentFolder == "" || currentFolder == "." {
			partialPath = partialFilename
		} else {
			partialPath = currentFolder + "/" + partialFilename
		}

		if tmpl, ok := LookupTemplate(partialPath, true); ok {
			// Cache the result for the original folder
			partialsCache.Store(cacheKey, tmpl)
			return tmpl
		}

		parentFolder := strings.ReplaceAll(filepath.Dir(currentFolder), "\\", "/")

		if parentFolder == "." || parentFolder == "/" || parentFolder == currentFolder {
			// Not found, cache nil to avoid repeated searches
			partialsCache.Store(cacheKey, nil)
			return nil
		}

		currentFolder = parentFolder
	}
}

func getPartialFunc(templatePath string) func(name string, data any) (htmlTemplate.HTML, error) {
	return func(name string, data any) (htmlTemplate.HTML, error) {
		var templateDir string
		if templatePath != "" {
			templateDir = strings.ReplaceAll(filepath.Dir(templatePath), "\\", "/")
		}

		partialFilename := "_" + name + config.HTMLTemplateExtension

		tmpl := lookUpPartial(templateDir, partialFilename)

		if tmpl != nil {
			var sb strings.Builder
			err := tmpl.Execute(&sb, data)
			return htmlTemplate.HTML(sb.String()), err
		}

		return "", fmt.Errorf("template not found: %s", name)
	}
}

func parseHTMLTemplate(templatePath string, layouts []string) (string, *htmlTemplate.Template) {
	var tmpl *htmlTemplate.Template
	var tmplName string

	funcMap["partial"] = getPartialFunc(templatePath)

	data := Must(fs.ReadFile(config.FS, templatePath))

	if len(layouts) > 0 {
		if v, ok := layoutsCache[templatePath]; ok {
			tmpl = v.(*htmlTemplate.Template)
		} else {
			tmpl = getOrCreateHTMLLayoutChain(layouts)
		}
		tmpl = htmlTemplate.Must(htmlTemplate.Must(tmpl.Clone()).Funcs(funcMap).Parse(string(data)))
		tmplName = tmpl.Name()
	} else {
		tmplName, _ = strings.CutSuffix(templatePath, config.HTMLTemplateExtension)
		tmpl = htmlTemplate.Must(htmlTemplate.New(tmplName).Funcs(funcMap).Parse(string(data)))
	}

	return tmplName, tmpl
}

func getOrCreateHTMLLayoutChain(layouts []string) *htmlTemplate.Template {
	var tmpl *htmlTemplate.Template

	for i := 0; i < len(layouts); i++ {
		if v, ok := layoutsCache[layouts[i]]; ok {
			tmpl = v.(*htmlTemplate.Template)
		} else {
			funcMap["partial"] = getPartialFunc("")
			if tmpl == nil {
				tmplName := filepath.Base(layouts[i])
				tmpl = htmlTemplate.Must(htmlTemplate.New(tmplName).Funcs(funcMap).ParseFS(config.FS, layouts[i]))
			} else {
				data := Must(fs.ReadFile(config.FS, layouts[i]))

				tmpl = Must(htmlTemplate.Must(tmpl.Clone()).Funcs(funcMap).Parse(string(data)))
			}
			layoutsCache[layouts[i]] = tmpl
		}
	}
	return tmpl
}

func parseTextTemplate(templatePath string, layouts []string) (string, *textTemplate.Template) {
	var tmpl *textTemplate.Template
	var tmplName string

	data := Must(fs.ReadFile(config.FS, templatePath))

	if len(layouts) > 0 {
		if v, ok := layoutsCache[templatePath]; ok {
			tmpl = v.(*textTemplate.Template)
		} else {
			tmpl = getOrCreateTextLayoutChain(layouts)
		}
		tmpl = textTemplate.Must(textTemplate.Must(tmpl.Clone()).Parse(string(data)))
		tmplName = tmpl.Name()
	} else {
		tmplName = filepath.Base(templatePath)
		tmpl = textTemplate.Must(textTemplate.New(tmplName).Parse(string(data)))
	}

	return tmplName, tmpl
}

func getOrCreateTextLayoutChain(layouts []string) *textTemplate.Template {
	var tmpl *textTemplate.Template

	for i := 0; i < len(layouts); i++ {
		if v, ok := layoutsCache[layouts[i]]; ok {
			tmpl = v.(*textTemplate.Template)
		} else {
			if tmpl == nil {
				tmplName := filepath.Base(layouts[i])
				tmpl = textTemplate.Must(textTemplate.New(tmplName).ParseFS(config.FS, layouts[i]))
			} else {
				data := Must(fs.ReadFile(config.FS, layouts[i]))

				tmpl = Must(textTemplate.Must(tmpl.Clone()).Parse(string(data)))
			}
			layoutsCache[layouts[i]] = tmpl
		}
	}
	return tmpl
}

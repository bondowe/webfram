---
layout: default
title: Templates
nav_order: 12
description: "Template system with layouts and partials"
---

# Templates

{: .no_toc }

WebFram provides a powerful template system with automatic caching, layout inheritance, and partials support.
{: .fs-6 .fw-300 }

## Table of contents

{: .no_toc .text-delta }

1. TOC
{:toc}

## Configuration

Templates must be provided via an embedded file system:

```go
// Project structure:
// assets/
//   └── templates/
//       ├── layout.go.html
//       └── index.go.html

//go:embed all:assets
var assetsFS embed.FS

app.Configure(&app.Config{
    Assets: &app.Assets{
        FS: assetsFS,
        Templates: &app.Templates{
            Dir:                   "assets/templates",
            LayoutBaseName:        "layout",
            HTMLTemplateExtension: ".go.html",
            TextTemplateExtension: ".go.txt",
        },
    },
})
```

## Template Structure

```text
assets/
├── templates/
│   ├── layout.go.html              # Root layout
│   ├── users/
│   │   ├── layout.go.html          # Users layout (inherits from root)
│   │   ├── list.go.html
│   │   ├── details.go.html
│   │   └── manage/
│   │       ├── update.go.html
│   │       └── delete.go.html
│   ├── _partOne.go.html            # Partial template
│   └── openapi.html
└── locales/
    └── messages.en.json
```

## Layout Files

Layouts are automatically detected and applied:

**Root layout** (`templates/layout.go.html`):

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{block "title" .}}Default Title{{end}}</title>
</head>
<body>
    {{block "content" .}}{{end}}
</body>
</html>
```

**Page template** (`templates/users/list.go.html`):

```html
{{define "title"}}Users List{{end}}

{{define "content"}}
<h1>Users</h1>
<ul>
    {{range .Users}}
    <li>{{.Name}}</li>
    {{end}}
</ul>
{{end}}
```

## Partials

Partials are reusable components with names starting with `_`:

**Partial** (`templates/_partOne.go.html`):

```html
<header>
    <h1>{{.Title}}</h1>
</header>
```

**Using partials:**

```html
{{define "content"}}
    <!-- Include a partial -->
    {{partial "partOne" .}}
    
    <div>Your main content here</div>
{{end}}
```

**Important:** Use `//go:embed all:assets` to include files starting with `_`.

## Rendering Templates

```go
mux.HandleFunc("GET /users", func(w app.ResponseWriter, r *app.Request) {
    data := map[string]interface{}{
        "Users": []User{
            {Name: "John", Email: "john@example.com"},
            {Name: "Jane", Email: "jane@example.com"},
        },
    }
    
    err := w.HTML(r.Context(), "users/list", data)
    if err != nil {
        w.Error(http.StatusInternalServerError, err.Error())
    }
})
```

## Layout Inheritance

WebFram supports nested layouts:

1. **Root layout** - `layout.go.html` in root templates directory
2. **Directory layout** - `layout.go.html` in subdirectories
3. **Child layouts inherit from parent layouts**

Example hierarchy:

```text
templates/
├── layout.go.html           # Root layout
└── admin/
    ├── layout.go.html       # Admin layout (extends root)
    └── dashboard.go.html    # Uses admin layout
```

## Template Functions

WebFram provides built-in template functions:

### Standard Functions

Go's standard template functions are available:

```html
{{/* Conditionals */}}
{{if .IsAdmin}}Admin Panel{{end}}
{{if .Count}}{{.Count}} items{{else}}No items{{end}}

{{/* Loops */}}
{{range .Items}}
    <div>{{.Name}}</div>
{{end}}

{{/* Variables */}}
{{$name := .User.Name}}
<p>Hello, {{$name}}</p>

{{/* Pipelines */}}
{{.Name | printf "%s is logged in"}}
```

### Partial Function

```html
{{partial "header" .}}
```

### i18n Function

```html
{{T "Welcome to %s!" .AppName}}
```

See [Internationalization](i18n.md) for details.

## Text Templates

For non-HTML content (emails, configuration files):

```go
mux.HandleFunc("GET /email", func(w app.ResponseWriter, r *app.Request) {
    data := map[string]string{"Name": "John"}
    err := w.Text(r.Context(), "email/welcome", data)
    if err != nil {
        w.Error(http.StatusInternalServerError, err.Error())
    }
})
```

**Email template** (`templates/email/welcome.go.txt`):

```text
Hello {{.Name}},

Welcome to our service!

Best regards,
The Team
```

## Inline Templates

Render templates from strings:

**HTML:**

```go
err := w.HTMLString("<h1>{{.Title}}</h1>", map[string]string{"Title": "Hello"})
```

**Text:**

```go
err := w.TextString("Hello {{.Name}}", map[string]string{"Name": "John"})
```

## Template Caching

Templates are automatically cached:

- **Production**: Templates cached on first use
- **Development**: Set environment variable for hot-reload

```go
// Enable template reloading (development only)
os.Setenv("TEMPLATE_RELOAD", "true")
```

## Error Handling

```go
err := w.HTML(r.Context(), "users/profile", data)
if err != nil {
    log.Printf("Template error: %v", err)
    w.Error(http.StatusInternalServerError, "Failed to render template")
    return
}
```

## Best Practices

1. **Use embedded filesystems** - Ensures portability
2. **Organize templates** - Group by feature/module
3. **Reuse partials** - DRY principle
4. **Escape data** - Templates automatically escape HTML
5. **Test templates** - Include in unit tests
6. **Layouts for consistency** - Maintain uniform design
7. **Name conventions** - Use `_` prefix for partials

## Complete Example

```go
//go:embed all:assets
var assetsFS embed.FS

func main() {
    app.Configure(&app.Config{
        Assets: &app.Assets{
            FS: assetsFS,
            Templates: &app.Templates{
                Dir: "assets/templates",
            },
        },
    })

    mux := app.NewServeMux()

    mux.HandleFunc("GET /", func(w app.ResponseWriter, r *app.Request) {
        data := map[string]interface{}{
            "Title": "Home",
            "User": User{
                Name:  "John Doe",
                Email: "john@example.com",
            },
        }
        
        err := w.HTML(r.Context(), "home/index", data)
        if err != nil {
            w.Error(http.StatusInternalServerError, err.Error())
        }
    })

    app.ListenAndServe(":8080", mux, nil)
}
```

**Template** (`templates/home/index.go.html`):

```html
{{define "title"}}{{.Title}}{{end}}

{{define "content"}}
<div class="container">
    {{partial "header" .}}
    
    <h2>Welcome, {{.User.Name}}!</h2>
    <p>Email: {{.User.Email}}</p>
</div>
{{end}}
```

**Partial** (`templates/_header.go.html`):

```html
<header>
    <nav>
        <a href="/">Home</a>
        <a href="/about">About</a>
    </nav>
</header>
```

## See Also

- [Internationalization](i18n.md)
- [Request & Response](request-response.html)
- [Configuration](configuration.html)

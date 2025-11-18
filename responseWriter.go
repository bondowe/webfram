package webfram

import (
	"bufio"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"io/fs"
	"net"
	"net/http"
	"path/filepath"
	"reflect"
	textTemplate "text/template"

	"github.com/bondowe/webfram/internal/i18n"
	"github.com/bondowe/webfram/internal/template"
	"golang.org/x/text/message"
	yaml "sigs.k8s.io/yaml/goyaml.v2"
)

type (
	// ResponseWriter wraps http.ResponseWriter with additional functionality.
	ResponseWriter struct {
		http.ResponseWriter

		statusCode *int // Pointer to allow mutation across value copies
	}

	// ServeFileOptions configures how files are served to clients.
	ServeFileOptions struct {
		Inline   bool   // If true, serves the file inline; otherwise as an attachment
		Filename string // Optional filename for Content-Disposition header
	}
)

const (
	jsonSeqRecordSeparator = '\x1E'
)

func i18nPrinterFunc(messagePrinter *message.Printer) func(str string, args ...any) string {
	return func(str string, args ...any) string {
		return messagePrinter.Sprintf(str, args...)
	}
}

// Error sends an error response with the specified HTTP status code and message.
// Uses http.Error to format the error message as plain text.
func (w *ResponseWriter) Error(statusCode int, message string) {
	http.Error(w.ResponseWriter, message, statusCode)
}

// Header returns the response header map for inspection and modification.
func (w *ResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// Write writes the data to the connection as part of an HTTP reply.
// Implements the io.Writer interface.
func (w *ResponseWriter) Write(b []byte) (int, error) {
	// If no status code has been set yet, default to 200 OK
	if w.statusCode != nil && *w.statusCode == 0 {
		*w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// WriteHeader sends an HTTP response header with the provided status code.
func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	if w.statusCode != nil {
		*w.statusCode = statusCode
	}
}

// Flush sends any buffered data to the client.
// If the underlying writer does not support flushing, this is a no-op.
func (w *ResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack takes over the connection from the HTTP server.
// Returns the connection, buffered reader/writer, and any error.
// After hijacking, the HTTP server will not do anything else with the connection.
func (w *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}

	return nil, nil, http.ErrNotSupported
}

// Push initiates an HTTP/2 server push for the specified target.
// Returns an error if the underlying connection does not support HTTP/2 push.
func (w *ResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}

	return http.ErrNotSupported
}

// ReadFrom reads data from src until EOF or error and writes it to the response.
// Implements the io.ReaderFrom interface for efficient data transfer.
func (w *ResponseWriter) ReadFrom(src io.Reader) (int64, error) {
	if rf, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return rf.ReadFrom(src)
	}

	return 0, http.ErrNotSupported
}

// Unwrap returns the underlying http.ResponseWriter.
func (w *ResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// StatusCode retrieves the HTTP response status code that was written.
// Returns the status code and true if it was set, or 0 and false if not set.
func (w *ResponseWriter) StatusCode() (int, bool) {
	if w.statusCode != nil && *w.statusCode != 0 {
		return *w.statusCode, true
	}
	return 0, false
}

// JSON marshals the provided data as JSON and writes it to the response.
// If a JSONP callback is present in the context, wraps the response in the callback function.
// Sets Content-Type header to "application/json" or "application/javascript" for JSONP.
// The ctx parameter is used to check for JSONP callback; pass request context or context.Background().
// Returns an error if marshaling or writing fails.
func (w *ResponseWriter) JSON(ctx context.Context, v any) error {
	jsonpCallback, ok := ctx.Value(jsonpCallbackMethodNameKey).(string)
	if ok && jsonpCallback != "" {
		w.Header().Set("Content-Type", "application/javascript")
		if _, writeErr := w.Write([]byte(jsonpCallback + "(")); writeErr != nil {
			return writeErr
		}
		bs, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, writeErr := w.Write(bs); writeErr != nil {
			return writeErr
		}
		if _, writeErr := w.Write([]byte(");")); writeErr != nil {
			return writeErr
		}
		return nil
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	return encoder.Encode(v)
}

// JSONSeq streams a sequence of JSON objects as per RFC 7464.
// Each JSON object is prefixed with the ASCII Record Separator character.
// Sets Content-Type header to "application/json-seq".
// Returns an error if items is not a slice, marshaling fails, or writing fails.
func (w *ResponseWriter) JSONSeq(_ context.Context, items any) error {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return errors.New("items must be a slice")
	}

	flusher, ok := w.ResponseWriter.(http.Flusher)
	if !ok {
		return errors.New("response writer does not support flushing")
	}

	w.Header().Set("Content-Type", "application/json-seq")

	encoder := json.NewEncoder(w)

	for i := range v.Len() {
		item := v.Index(i).Interface()

		_, writeErr := fmt.Fprintf(w, "%c", jsonSeqRecordSeparator)
		if writeErr != nil {
			return writeErr
		}

		if err := encoder.Encode(item); err != nil {
			return err
		}

		flusher.Flush()
	}

	return nil
}

// HTMLString parses an HTML template string and executes it with the provided data.
// Sets Content-Type header to "text/html".
// Returns an error if template parsing or execution fails.
func (w *ResponseWriter) HTMLString(s string, data any) error {
	w.Header().Set("Content-Type", "text/html")

	tmpl, err := htmlTemplate.New("inline").Parse(s)

	if err != nil {
		return err
	}

	return tmpl.Execute(w.ResponseWriter, data)
}

// HTML renders a cached HTML template with the provided data.
// The path is relative to the template directory and does not include the extension.
// Automatically adds i18n support if a message printer is in the context.
// Sets Content-Type header to "text/html".
// The ctx parameter is used for i18n support; pass request context or context.Background().
// Returns an error if templates are not configured, template is not found, or execution fails.
func (w *ResponseWriter) HTML(ctx context.Context, path string, data any) error {
	return w.renderTemplate(ctx, path, data, "text/html", true)
}

// TextString parses a plain text template string and executes it with the provided data.
// Sets Content-Type header to "text/plain".
// Returns an error if template parsing or execution fails.
func (w *ResponseWriter) TextString(s string, data any) error {
	w.Header().Set("Content-Type", "text/plain")

	tmpl, err := textTemplate.New("inline").Parse(s)
	if err != nil {
		return err
	}
	return tmpl.Execute(w.ResponseWriter, data)
}

// Text renders a cached text template with the provided data.
// The path is relative to the template directory and does not include the extension.
// Automatically adds i18n support if a message printer is in the context.
// Sets Content-Type header to "text/plain".
// The ctx parameter is used for i18n support; pass request context or context.Background().
// Returns an error if templates are not configured, template is not found, or execution fails.
func (w *ResponseWriter) Text(ctx context.Context, path string, data any) error {
	return w.renderTemplate(ctx, path, data, "text/plain", false)
}

// renderTemplate is a helper function that handles template rendering for both HTML and text templates.
func (w *ResponseWriter) renderTemplate(
	ctx context.Context,
	path string,
	data any,
	contentType string,
	isHTML bool,
) error {
	tmplConfig, ok := template.Configuration()
	if !ok {
		return errors.New("templates not configured")
	}

	w.Header().Set("Content-Type", contentType)

	var extension string
	if isHTML {
		extension = tmplConfig.HTMLTemplateExtension
	} else {
		extension = tmplConfig.TextTemplateExtension
	}

	if tmpl, tmplFound := template.LookupTemplate(path+extension, false); tmplFound {
		if msgPrinter, printerOk := i18n.PrinterFromContext(ctx); printerOk {
			if isHTML {
				i18nFunc := i18nPrinterFunc(msgPrinter)
				funcs := htmlTemplate.FuncMap{
					tmplConfig.I18nFuncName: i18nFunc,
					"partial":               template.GetPartialFuncWithI18n(path+extension, i18nFunc),
				}
				return template.Must(tmpl.Clone()).Funcs(funcs).Execute(w.ResponseWriter, data)
			}
			i18nFunc := i18nPrinterFunc(msgPrinter)
			funcs := textTemplate.FuncMap{
				tmplConfig.I18nFuncName: i18nFunc,
				"partial":               template.GetTextPartialFuncWithI18n(path+extension, i18nFunc),
			}
			return template.Must(tmpl.Clone()).Funcs(funcs).Execute(w.ResponseWriter, data)
		}
		return tmpl.Execute(w.ResponseWriter, data)
	}

	return fmt.Errorf("template not found in cache: %s", path)
}

// XML marshals the provided data as XML and writes it to the response.
// Sets Content-Type header to "application/xml".
// Returns an error if marshaling or writing fails.
func (w *ResponseWriter) XML(v any) error {
	w.Header().Set("Content-Type", "application/xml")

	bs, err := xml.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(bs)
	return err
}

// YAML marshals the provided data as YAML and writes it to the response.
// Sets Content-Type header to "text/x-yaml".
// Returns an error if marshaling or writing fails.
func (w *ResponseWriter) YAML(v any) error {
	w.Header().Set("Content-Type", "text/x-yaml")

	data, err := yaml.Marshal(v)

	if err != nil {
		http.Error(w.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	_, err = w.Write(data)
	return err
}

// Bytes writes raw byte data to the response with the specified content type.
// If contentType is empty, automatically detects the content type using http.DetectContentType.
// Returns an error if writing fails.
func (w *ResponseWriter) Bytes(bs []byte, contentType string) error {
	if contentType == "" {
		contentType = http.DetectContentType(bs)
	}
	w.Header().Set("Content-Type", contentType)

	_, err := w.Write(bs)
	return err
}

// NoContent sends a 204 No Content response with no body.
func (w *ResponseWriter) NoContent() {
	w.WriteHeader(http.StatusNoContent)
}

// Redirect replies to the request with a redirect to urlStr.
// The code should be a 3xx status code (e.g., http.StatusFound, http.StatusMovedPermanently).
func (w *ResponseWriter) Redirect(req *Request, urlStr string, code int) {
	http.Redirect(w.ResponseWriter, req.Request, urlStr, code)
}

// ServeFileFS serves a file from the specified fs.FS at the given path.
// The options parameter allows setting Content-Disposition headers for inline or attachment serving.
// If options is nil, defaults to attachment serving with the original filename.
// Uses http.ServeFileFS to handle file serving.
// The req parameter is the original request.
func (w *ResponseWriter) ServeFileFS(req *Request, fsys fs.FS, path string, options *ServeFileOptions) {
	var disposition string
	var filename string

	if options != nil && options.Inline {
		disposition = "inline"
	} else {
		disposition = "attachment"
	}

	if options != nil && options.Filename != "" {
		filename = options.Filename
	} else {
		filename = filepath.Base(path)
	}

	w.Header().Set("Content-Disposition", disposition+"; filename=\""+filepath.Base(filename)+"\"")
	http.ServeFileFS(w.ResponseWriter, req.Request, fsys, path)
}

// ServeFile serves a file from the local filesystem at the given path.
// The options parameter allows setting Content-Disposition headers for inline or attachment serving.
// If options is nil, defaults to attachment serving with the original filename.
// Uses http.ServeFile to handle file serving.
// The req parameter is the original request.
func (w *ResponseWriter) ServeFile(req *Request, path string, options *ServeFileOptions) {
	var disposition string
	var filename string

	if options != nil && options.Inline {
		disposition = "inline"
	} else {
		disposition = "attachment"
	}

	if options != nil && options.Filename != "" {
		filename = options.Filename
	} else {
		filename = filepath.Base(path)
	}

	w.Header().Set("Content-Disposition", disposition+"; filename=\""+filepath.Base(filename)+"\"")
	http.ServeFile(w.ResponseWriter, req.Request, path)
}

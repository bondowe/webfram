package webfram

import (
	"bufio"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"net"
	"net/http"
	"path/filepath"
	textTemplate "text/template"

	"github.com/bondowe/webfram/internal/i18n"
	"github.com/bondowe/webfram/internal/template"
	"golang.org/x/text/message"
	yaml "sigs.k8s.io/yaml/goyaml.v2"
)

type ResponseWriter struct {
	http.ResponseWriter
	context context.Context
}

func i18nPrinterFunc(messagePrinter *message.Printer) func(str string, args ...any) string {
	return func(str string, args ...any) string {
		return messagePrinter.Sprintf(str, args...)
	}
}

// Context returns the request context associated with this response writer.
func (w *ResponseWriter) Context() context.Context {
	return w.context
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
	return w.ResponseWriter.Write(b)
}

// WriteHeader sends an HTTP response header with the provided status code.
func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
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

// CloseNotify returns a channel that receives a single value when the client connection is closed.
// Deprecated: Use Request.Context() instead.
func (w *ResponseWriter) CloseNotify() <-chan bool {
	if cn, ok := w.ResponseWriter.(http.CloseNotifier); ok {
		return cn.CloseNotify()
	}

	return nil
}

// ReadFrom reads data from src until EOF or error and writes it to the response.
// Implements the io.ReaderFrom interface for efficient data transfer.
func (w *ResponseWriter) ReadFrom(src io.Reader) (n int64, err error) {
	if rf, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return rf.ReadFrom(src)
	}

	return 0, http.ErrNotSupported
}

// Unwrap returns the underlying http.ResponseWriter.
func (w *ResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// JSON marshals the provided data as JSON and writes it to the response.
// If a JSONP callback is present in the context, wraps the response in the callback function.
// Sets Content-Type header to "application/json" or "application/javascript" for JSONP.
// Returns an error if marshaling or writing fails.
func (w *ResponseWriter) JSON(v any) error {
	jsonpCallback, ok := w.context.Value(jsonpCallbackMethodNameKey).(string)
	if ok && jsonpCallback != "" {
		w.Header().Set("Content-Type", "application/javascript")
		if _, err := w.Write([]byte(jsonpCallback + "(")); err != nil {
			return err
		}
		bs, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := w.Write(bs); err != nil {
			return err
		}
		if _, err := w.Write([]byte(");")); err != nil {
			return err
		}
		return nil
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	return encoder.Encode(v)
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
// Returns an error if templates are not configured, template is not found, or execution fails.
func (w *ResponseWriter) HTML(path string, data any) error {
	tmplConfig, ok := template.Configuration()
	if !ok {
		return fmt.Errorf("templates not configured")
	}

	w.Header().Set("Content-Type", "text/html")

	if tmpl, ok := template.LookupTemplate(path+tmplConfig.HTMLTemplateExtension, false); ok {
		if msgPrinter, ok := i18n.I18nPrinterFromContext(w.context); ok {
			funcs := htmlTemplate.FuncMap{
				tmplConfig.I18nFuncName: i18nPrinterFunc(msgPrinter),
			}
			return template.Must(tmpl.Clone()).Funcs(funcs).Execute(w.ResponseWriter, data)
		}
		return tmpl.Execute(w.ResponseWriter, data)
	}

	return fmt.Errorf("template not found in cache: %s", path)
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
// Returns an error if templates are not configured, template is not found, or execution fails.
func (w *ResponseWriter) Text(path string, data any) error {
	tmplConfig, ok := template.Configuration()
	if !ok {
		return fmt.Errorf("templates not configured")
	}

	w.Header().Set("Content-Type", "text/plain")

	if tmpl, ok := template.LookupTemplate(path+tmplConfig.TextTemplateExtension, false); ok {
		if msgPrinter, ok := i18n.I18nPrinterFromContext(w.context); ok {
			funcs := textTemplate.FuncMap{

				tmplConfig.I18nFuncName: i18nPrinterFunc(msgPrinter),
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

// ServeFile replies to the request with the contents of the named file.
// The inline parameter controls the Content-Disposition header (inline or attachment).
// The file is served from the configured template filesystem.
func (w *ResponseWriter) ServeFile(req *Request, name string, inline bool) {
	tmplConfig, ok := template.Configuration()
	if !ok {
		http.Error(w.ResponseWriter, "templates not configured", http.StatusInternalServerError)
		return
	}

	var disposition string
	if inline {
		disposition = "inline"
	} else {
		disposition = "attachment"
	}

	w.Header().Set("Content-Disposition", disposition+"; filename=\""+filepath.Base(name)+"\"")
	http.ServeFileFS(w.ResponseWriter, req.Request, tmplConfig.FS, name)
}

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

func (w *ResponseWriter) Context() context.Context {
	return w.context
}

func (w *ResponseWriter) Error(statusCode int, message string) {
	http.Error(w.ResponseWriter, message, statusCode)
}

func (w *ResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *ResponseWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *ResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}

	return nil, nil, http.ErrNotSupported
}

func (w *ResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}

	return http.ErrNotSupported
}

func (w *ResponseWriter) CloseNotify() <-chan bool {
	if cn, ok := w.ResponseWriter.(http.CloseNotifier); ok {
		return cn.CloseNotify()
	}

	return nil
}

func (w *ResponseWriter) ReadFrom(r io.Reader) (n int64, err error) {
	if rf, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return rf.ReadFrom(r)
	}

	return 0, http.ErrNotSupported
}

func (w *ResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

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

func (w *ResponseWriter) HTMLString(s string, data any) error {
	w.Header().Set("Content-Type", "text/html")

	tmpl, err := htmlTemplate.New("inline").Parse(s)

	if err != nil {
		return err
	}

	return tmpl.Execute(w.ResponseWriter, data)
}

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

func (w *ResponseWriter) TextString(s string, data any) error {
	w.Header().Set("Content-Type", "text/plain")

	tmpl, err := textTemplate.New("inline").Parse(s)
	if err != nil {
		return err
	}
	return tmpl.Execute(w.ResponseWriter, data)
}

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

func (w *ResponseWriter) XML(v any) error {
	w.Header().Set("Content-Type", "application/xml")

	bs, err := xml.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(bs)
	return err
}

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

func (w *ResponseWriter) Bytes(bs []byte, contentType string) error {
	if contentType == "" {
		contentType = http.DetectContentType(bs)
	}
	w.Header().Set("Content-Type", contentType)

	_, err := w.Write(bs)
	return err
}

func (w *ResponseWriter) NoContent() {
	w.WriteHeader(http.StatusNoContent)
}

func (w *ResponseWriter) Redirect(req *Request, urlStr string, code int) {
	http.Redirect(w.ResponseWriter, req.Request, urlStr, code)
}

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

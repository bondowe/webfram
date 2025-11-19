package webfram

import (
	"context"
	"crypto/tls"
	_ "embed"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bondowe/webfram/internal/telemetry"
)

//go:embed openapi.go.html
var openapiTemplate string

// ServerConfig configures HTTP server settings.
type ServerConfig struct {
	ConnState                    func(net.Conn, http.ConnState)
	TLSConfig                    *tls.Config
	Protocols                    *http.Protocols
	HTTP2                        *http.HTTP2Config
	ConnContext                  func(ctx context.Context, c net.Conn) context.Context
	BaseContext                  func(net.Listener) context.Context
	ErrorLog                     *slog.Logger
	TLSNextProto                 map[string]func(*http.Server, *tls.Conn, http.Handler)
	ReadHeaderTimeout            time.Duration
	MaxHeaderBytes               int
	IdleTimeout                  time.Duration
	WriteTimeout                 time.Duration
	ReadTimeout                  time.Duration
	DisableGeneralOptionsHandler bool
}

const (
	readTimeout       = 15 * time.Second
	readHeaderTimeout = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	maxHeaderBytes    = http.DefaultMaxHeaderBytes
)

// setupOpenAPIEndpoints configures the OpenAPI endpoints if enabled.
func setupOpenAPIEndpoints(mux *ServeMux) {
	if openAPIConfig == nil || !openAPIConfig.Enabled {
		return
	}

	openAPIConfig.internalConfig.Self = openAPIConfig.URLPath

	for _, hc := range handlerConfigs {
		if hc.mux == mux && hc.operation != nil {
			configureOpenAPIOperation(hc.pathPattern, hc.operation)
		}
	}

	doc, err := openAPIConfig.internalConfig.MarshalJSON()

	if err != nil {
		panic(err)
	}
	mux.HandleFunc(openAPIConfig.URLPath, func(w ResponseWriter, _ *Request) {
		if jsonErr := w.Bytes(doc, "application/json"); jsonErr != nil {
			w.Error(http.StatusInternalServerError, jsonErr.Error())
		}
	})

	openAPIDocumentPath := strings.TrimPrefix(openAPIConfig.URLPath, "GET ")

	pageURL := strings.TrimSuffix(openAPIConfig.URLPath, "/")
	pageURL = strings.TrimSuffix(pageURL, ".json")
	pageURL += ".html"

	openapiTemplateData := struct {
		OpenAPIDocumentPath string
	}{
		OpenAPIDocumentPath: openAPIDocumentPath,
	}

	mux.HandleFunc(pageURL, func(w ResponseWriter, _ *Request) {
		if htmlErr := w.HTMLString(openapiTemplate, openapiTemplateData); htmlErr != nil {
			w.Error(http.StatusInternalServerError, htmlErr.Error())
		}
	})

	if os.Getenv("WEBFRAM_SILENT") == "" {
		slog.Info("OpenAPI docs: " + openAPIConfig.URLPath) //nolint:sloglint // Startup logging is acceptable
		slog.Info("OpenAPI UI: " + pageURL)                 //nolint:sloglint // Startup logging is acceptable
	}
}

// setupTelemetry configures telemetry endpoints and returns a telemetry server if configured separately.
func setupTelemetry(addr string, mux *ServeMux) (*http.Server, bool) {
	if telemetryConfig == nil || !telemetryConfig.Enabled {
		return nil, false
	}

	handler := telemetry.GetHTTPHandler(telemetryConfig.HandlerOpts)

	// Check if telemetry should run on a separate server
	if telemetryConfig.Addr != "" && telemetryConfig.Addr != addr {
		// Create separate telemetry server
		telemetryMux := NewServeMux()
		telemetryMux.Handle(telemetryConfig.URLPath, adaptHTTPHandler(handler))

		telemetryServer := &http.Server{
			Addr:              telemetryConfig.Addr,
			Handler:           telemetryMux,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
			MaxHeaderBytes:    maxHeaderBytes,
		}
		return telemetryServer, true
	}

	// Run telemetry on the main server
	mux.Handle(telemetryConfig.URLPath, adaptHTTPHandler(handler))
	return nil, false
}

// createHTTPServer creates and configures an HTTP server with the provided settings.
func createHTTPServer(addr string, handler http.Handler, cfg *ServerConfig) *http.Server {
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}

	if cfg != nil {
		server.DisableGeneralOptionsHandler = cfg.DisableGeneralOptionsHandler
		server.TLSConfig = cfg.TLSConfig
		server.ReadTimeout = getValueOrDefault(cfg.ReadTimeout, server.ReadTimeout)
		server.ReadHeaderTimeout = getValueOrDefault(cfg.ReadHeaderTimeout, server.ReadHeaderTimeout)
		server.WriteTimeout = getValueOrDefault(cfg.WriteTimeout, server.WriteTimeout)
		server.IdleTimeout = getValueOrDefault(cfg.IdleTimeout, server.IdleTimeout)
		server.MaxHeaderBytes = getValueOrDefault(cfg.MaxHeaderBytes, server.MaxHeaderBytes)
		server.TLSNextProto = cfg.TLSNextProto
		server.ConnState = cfg.ConnState
		server.BaseContext = cfg.BaseContext
		server.ConnContext = cfg.ConnContext
		server.HTTP2 = cfg.HTTP2
		server.Protocols = cfg.Protocols
	}

	return server
}

// startServer starts an HTTP server in a goroutine and reports errors to the provided channel.
func startServer(server *http.Server, serverType string, errorChan chan<- error) {
	go func() {
		slog.Info("Starting server", "type", serverType, "addr", server.Addr)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errorChan <- err
		}
	}()
}

// waitForShutdownSignal waits for either a server error or a shutdown signal.
// Returns true if a shutdown signal was received, panics if a server error occurred.
func waitForShutdownSignal(errorChan <-chan error) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errorChan:
		panic(err)
	case sig := <-stop:
		//nolint:sloglint // Global logger is appropriate here during server shutdown
		slog.Info("Received shutdown signal", "signal", sig)
	}
}

// shutdownServers gracefully shuts down the main server and optionally the telemetry server.
func shutdownServers(mainServer *http.Server, telemetryServer *http.Server, hasSeparateTelemetry bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second) //nolint:mnd // graceful shutdown timeout
	defer cancel()

	// Shutdown main server
	if err := mainServer.Shutdown(ctx); err != nil {
		panic(err)
	}
	//nolint:sloglint // Global logger is appropriate here after server shutdown
	slog.Info("Server stopped")

	// Shutdown telemetry server if running separately
	if hasSeparateTelemetry {
		if err := telemetryServer.Shutdown(ctx); err != nil {
			panic(err)
		}
		//nolint:sloglint // Global logger is appropriate here after server shutdown
		slog.Info("Telemetry server stopped")
	}
}

func registerHandlers(mux *ServeMux) {
	for _, hc := range handlerConfigs {
		if hc.mux != mux {
			continue
		}
		registerHandlerFunc(hc)
	}
}

// ListenAndServe starts an HTTP server on the specified address with the given multiplexer.
// It automatically sets up OpenAPI endpoint if configured, applies server configuration,
// and handles graceful shutdown on SIGINT or SIGTERM signals.
// If telemetry is configured with a separate address, starts an additional server for metrics.
// Blocks until the server is shut down. Panics if server startup or shutdown fails.
func ListenAndServe(addr string, mux *ServeMux, cfg *ServerConfig) {
	setupOpenAPIEndpoints(mux)
	registerHandlers(mux)
	telemetryServer, hasSeparateTelemetry := setupTelemetry(addr, mux)
	mainServer := createHTTPServer(addr, mux, cfg)

	//nolint:mnd // buffer size for main and telemetry servers
	serverError := make(chan error, 2)
	startServer(mainServer, "main", serverError)

	if hasSeparateTelemetry {
		startServer(telemetryServer, "telemetry", serverError)
	}

	waitForShutdownSignal(serverError)
	shutdownServers(mainServer, telemetryServer, hasSeparateTelemetry)
}

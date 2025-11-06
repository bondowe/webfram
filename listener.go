package webfram

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ServerConfig struct {
	ConnState                    func(net.Conn, http.ConnState)
	TLSConfig                    *tls.Config
	Protocols                    *http.Protocols
	HTTP2                        *http.HTTP2Config
	ConnContext                  func(ctx context.Context, c net.Conn) context.Context
	BaseContext                  func(net.Listener) context.Context
	ErrorLog                     *log.Logger
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

// ListenAndServe starts an HTTP server on the specified address with the given multiplexer.
// It automatically sets up OpenAPI endpoint if configured, applies server configuration,
// and handles graceful shutdown on SIGINT or SIGTERM signals.
// Blocks until the server is shut down. Panics if server startup or shutdown fails.
func ListenAndServe(addr string, mux *ServeMux, cfg *ServerConfig) {
	if openAPIConfig != nil && openAPIConfig.EndpointEnabled {
		doc, err := openAPIConfig.Config.MarshalJSON()
		if err != nil {
			panic(err)
		}
		mux.HandleFunc(openAPIConfig.URLPath, func(w ResponseWriter, r *Request) {
			_ = w.Bytes(doc, "application/openapi+json")
		})
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
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
		server.ErrorLog = cfg.ErrorLog
		server.BaseContext = cfg.BaseContext
		server.ConnContext = cfg.ConnContext
		server.HTTP2 = cfg.HTTP2
		server.Protocols = cfg.Protocols
	}

	serverError := make(chan error, 1)

	go func() {
		log.Printf("Starting server on %s", addr)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serverError <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverError:
		panic(err)
	case sig := <-stop:
		log.Printf("Received shutdown signal: %v", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		panic(err)
	}
	log.Println("Server stopped")
}

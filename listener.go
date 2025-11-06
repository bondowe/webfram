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
	DisableGeneralOptionsHandler bool
	TLSConfig                    *tls.Config
	ReadTimeout                  time.Duration
	ReadHeaderTimeout            time.Duration
	WriteTimeout                 time.Duration
	IdleTimeout                  time.Duration
	MaxHeaderBytes               int
	TLSNextProto                 map[string]func(*http.Server, *tls.Conn, http.Handler)
	ConnState                    func(net.Conn, http.ConnState)
	ErrorLog                     *log.Logger
	BaseContext                  func(net.Listener) context.Context
	ConnContext                  func(ctx context.Context, c net.Conn) context.Context
	HTTP2                        *http.HTTP2Config
	Protocols                    *http.Protocols
}

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
		Addr:    addr,
		Handler: mux,
	}

	serverConfig := NewServerConfig()

	if cfg != nil {
		serverConfig.DisableGeneralOptionsHandler = cfg.DisableGeneralOptionsHandler
		serverConfig.TLSConfig = cfg.TLSConfig
		serverConfig.ReadTimeout = getValueOrDefault(cfg.ReadTimeout, serverConfig.ReadTimeout)
		serverConfig.ReadHeaderTimeout = getValueOrDefault(cfg.ReadHeaderTimeout, serverConfig.ReadHeaderTimeout)
		serverConfig.WriteTimeout = getValueOrDefault(cfg.WriteTimeout, serverConfig.WriteTimeout)
		serverConfig.IdleTimeout = getValueOrDefault(cfg.IdleTimeout, serverConfig.IdleTimeout)
		serverConfig.MaxHeaderBytes = getValueOrDefault(cfg.MaxHeaderBytes, serverConfig.MaxHeaderBytes)
		serverConfig.TLSNextProto = cfg.TLSNextProto
		serverConfig.ConnState = cfg.ConnState
		serverConfig.ErrorLog = cfg.ErrorLog
		serverConfig.BaseContext = cfg.BaseContext
		serverConfig.ConnContext = cfg.ConnContext
		serverConfig.HTTP2 = cfg.HTTP2
		serverConfig.Protocols = cfg.Protocols
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

func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
	}
}

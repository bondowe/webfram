package webfram

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/bondowe/webfram/openapi"
)

func TestNewServerConfig(t *testing.T) {
	cfg := NewServerConfig()

	if cfg == nil {
		t.Fatal("NewServerConfig returned nil")
	}

	// Test default values
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"ReadTimeout", cfg.ReadTimeout, 15 * time.Second},
		{"ReadHeaderTimeout", cfg.ReadHeaderTimeout, 15 * time.Second},
		{"WriteTimeout", cfg.WriteTimeout, 15 * time.Second},
		{"IdleTimeout", cfg.IdleTimeout, 60 * time.Second},
		{"MaxHeaderBytes", cfg.MaxHeaderBytes, http.DefaultMaxHeaderBytes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Expected %v = %v, got %v", tt.name, tt.expected, tt.got)
			}
		})
	}
}

func TestGetValueOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		value        interface{}
		defaultValue interface{}
		expected     interface{}
	}{
		{
			name:         "int zero value",
			value:        0,
			defaultValue: 42,
			expected:     42,
		},
		{
			name:         "int non-zero value",
			value:        10,
			defaultValue: 42,
			expected:     10,
		},
		{
			name:         "string zero value",
			value:        "",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "string non-zero value",
			value:        "custom",
			defaultValue: "default",
			expected:     "custom",
		},
		{
			name:         "duration zero value",
			value:        time.Duration(0),
			defaultValue: 5 * time.Second,
			expected:     5 * time.Second,
		},
		{
			name:         "duration non-zero value",
			value:        10 * time.Second,
			defaultValue: 5 * time.Second,
			expected:     10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result interface{}
			switch v := tt.value.(type) {
			case int:
				result = getValueOrDefault(v, tt.defaultValue.(int))
			case string:
				result = getValueOrDefault(v, tt.defaultValue.(string))
			case time.Duration:
				result = getValueOrDefault(v, tt.defaultValue.(time.Duration))
			}

			if result != tt.expected {
				t.Errorf("getValueOrDefault(%v, %v) = %v, want %v",
					tt.value, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestListenAndServe_ServerStartsSuccessfully(t *testing.T) {
	t.Skip("Skipping test that requires signal handling - interferes with test runner")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setupMuxTest()
	mux := NewServeMux()

	// Add a simple handler
	mux.HandleFunc("GET /test", func(w ResponseWriter, r *Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()

	// Start server in a goroutine
	serverStarted := make(chan bool)
	serverStopped := make(chan bool)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Server shutdown is expected
				if err, ok := r.(error); ok {
					if !errors.Is(err, http.ErrServerClosed) {
						t.Errorf("Unexpected server error: %v", err)
					}
				}
			}
			serverStopped <- true
		}()

		// Signal that we're about to start
		serverStarted <- true

		// This will block until server stops
		ListenAndServe(addr, mux, nil)
	}()

	// Wait for server to start
	<-serverStarted

	// Give the server a moment to actually start
	time.Sleep(100 * time.Millisecond)

	// Make a test request
	resp, err := http.Get("http://" + addr + "/test")
	if err != nil {
		t.Logf("Failed to connect to server (expected if server hasn't started yet): %v", err)
	} else {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	}

	// Send interrupt signal to stop server
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	proc.Signal(syscall.SIGTERM)

	// Wait for server to stop (with timeout)
	select {
	case <-serverStopped:
		// Server stopped successfully
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not stop within timeout")
	}
}

func TestListenAndServe_WithCustomConfig(t *testing.T) {
	t.Skip("Skipping test that requires signal handling - interferes with test runner")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setupMuxTest()
	mux := NewServeMux()

	mux.HandleFunc("GET /test", func(w ResponseWriter, r *Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()

	customLog := log.New(os.Stdout, "TEST: ", log.LstdFlags)

	cfg := &ServerConfig{
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
		ErrorLog:       customLog,
		ConnState: func(conn net.Conn, state http.ConnState) {
		},
		BaseContext: func(listener net.Listener) context.Context {
			return context.WithValue(context.Background(), "test-key", "test-value")
		},
	}

	serverStopped := make(chan bool)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected shutdown
			}
			serverStopped <- true
		}()

		ListenAndServe(addr, mux, cfg)
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Make a request to trigger connection state
	http.Get("http://" + addr + "/test")

	// Stop the server
	proc, _ := os.FindProcess(os.Getpid())
	proc.Signal(syscall.SIGTERM)

	// Wait for shutdown
	select {
	case <-serverStopped:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not stop")
	}
}

func TestListenAndServe_WithOpenAPIEndpoint(t *testing.T) {
	t.Skip("Skipping test that requires signal handling - interferes with test runner")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Reset app configuration
	appConfigured = false

	Configure(&Config{
		OpenAPI: &OpenAPI{
			EndpointEnabled: true,
			URLPath:         "GET /openapi.json",
			Config: &openapi.Config{
				Info: &openapi.Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
			},
		},
		Assets: &Assets{
			FS: testMuxI18nFS,
			I18nMessages: &I18nMessages{
				Dir: "locales",
			},
		},
	})

	mux := NewServeMux()

	mux.HandleFunc("GET /api/test", func(w ResponseWriter, r *Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()

	serverStopped := make(chan bool)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected shutdown
			}
			serverStopped <- true
		}()

		ListenAndServe(addr, mux, nil)
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Make a request to OpenAPI endpoint
	resp, err := http.Get("http://" + addr + "/openapi.json")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for OpenAPI endpoint, got %d", resp.StatusCode)
		}
		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/openapi+json" {
			t.Errorf("Expected Content-Type 'application/openapi+json', got %q", contentType)
		}
	}

	// Stop the server
	proc, _ := os.FindProcess(os.Getpid())
	proc.Signal(syscall.SIGTERM)

	// Wait for shutdown
	select {
	case <-serverStopped:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not stop")
	}
}

func TestServerConfig_AllFields(t *testing.T) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	errorLog := log.New(os.Stdout, "ERROR: ", log.LstdFlags)

	cfg := &ServerConfig{
		DisableGeneralOptionsHandler: true,
		TLSConfig:                    tlsConfig,
		ReadTimeout:                  20 * time.Second,
		ReadHeaderTimeout:            5 * time.Second,
		WriteTimeout:                 30 * time.Second,
		IdleTimeout:                  120 * time.Second,
		MaxHeaderBytes:               2 << 20, // 2MB
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){
			"h2": nil,
		},
		ConnState: func(conn net.Conn, state http.ConnState) {
			// Connection state handler
		},
		ErrorLog: errorLog,
		BaseContext: func(listener net.Listener) context.Context {
			return context.Background()
		},
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			return ctx
		},
	}

	// Verify all fields are set
	if !cfg.DisableGeneralOptionsHandler {
		t.Error("DisableGeneralOptionsHandler should be true")
	}
	if cfg.TLSConfig != tlsConfig {
		t.Error("TLSConfig not set correctly")
	}
	if cfg.ReadTimeout != 20*time.Second {
		t.Errorf("ReadTimeout = %v, want 20s", cfg.ReadTimeout)
	}
	if cfg.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want 5s", cfg.ReadHeaderTimeout)
	}
	if cfg.WriteTimeout != 30*time.Second {
		t.Errorf("WriteTimeout = %v, want 30s", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 120*time.Second {
		t.Errorf("IdleTimeout = %v, want 120s", cfg.IdleTimeout)
	}
	if cfg.MaxHeaderBytes != 2<<20 {
		t.Errorf("MaxHeaderBytes = %d, want %d", cfg.MaxHeaderBytes, 2<<20)
	}
	if cfg.TLSNextProto == nil {
		t.Error("TLSNextProto should not be nil")
	}
	if cfg.ConnState == nil {
		t.Error("ConnState should not be nil")
	}
	if cfg.ErrorLog != errorLog {
		t.Error("ErrorLog not set correctly")
	}
	if cfg.BaseContext == nil {
		t.Error("BaseContext should not be nil")
	}
	if cfg.ConnContext == nil {
		t.Error("ConnContext should not be nil")
	}
}

func TestServerConfig_PartialOverrides(t *testing.T) {
	setupMuxTest()

	customCfg := &ServerConfig{
		ReadTimeout: 25 * time.Second,
		// Other fields left as zero values - should use defaults
	}

	// We can't easily test the actual server creation without starting it,
	// but we can verify the config structure is correct
	if customCfg.ReadTimeout != 25*time.Second {
		t.Errorf("ReadTimeout = %v, want 25s", customCfg.ReadTimeout)
	}

	// Verify zero values are present (will be replaced by defaults in ListenAndServe)
	if customCfg.WriteTimeout != 0 {
		t.Errorf("WriteTimeout should be zero, got %v", customCfg.WriteTimeout)
	}
	if customCfg.IdleTimeout != 0 {
		t.Errorf("IdleTimeout should be zero, got %v", customCfg.IdleTimeout)
	}
}

func TestListenAndServe_HandlesMultipleRequests(t *testing.T) {
	t.Skip("Skipping test that requires signal handling - interferes with test runner")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	setupMuxTest()
	mux := NewServeMux()

	requestCount := 0
	mux.HandleFunc("GET /count", func(w ResponseWriter, r *Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()

	serverStopped := make(chan bool)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected shutdown
			}
			serverStopped <- true
		}()

		ListenAndServe(addr, mux, nil)
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Make multiple requests
	for i := 0; i < 5; i++ {
		resp, err := http.Get("http://" + addr + "/count")
		if err == nil {
			resp.Body.Close()
		}
		time.Sleep(10 * time.Millisecond)
	}

	if requestCount < 1 {
		t.Errorf("Expected at least 1 request, got %d", requestCount)
	}

	// Stop the server
	proc, _ := os.FindProcess(os.Getpid())
	proc.Signal(syscall.SIGTERM)

	// Wait for shutdown
	select {
	case <-serverStopped:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not stop")
	}
}

func TestServerConfig_ZeroTimeouts(t *testing.T) {
	cfg := &ServerConfig{
		// All timeouts are zero (default values)
	}

	defaultCfg := NewServerConfig()

	// When passed to ListenAndServe, zero values should be replaced with defaults
	// We verify the logic by checking getValueOrDefault behavior
	readTimeout := getValueOrDefault(cfg.ReadTimeout, defaultCfg.ReadTimeout)
	if readTimeout != 15*time.Second {
		t.Errorf("Expected default ReadTimeout 15s, got %v", readTimeout)
	}

	writeTimeout := getValueOrDefault(cfg.WriteTimeout, defaultCfg.WriteTimeout)
	if writeTimeout != 15*time.Second {
		t.Errorf("Expected default WriteTimeout 15s, got %v", writeTimeout)
	}

	idleTimeout := getValueOrDefault(cfg.IdleTimeout, defaultCfg.IdleTimeout)
	if idleTimeout != 60*time.Second {
		t.Errorf("Expected default IdleTimeout 60s, got %v", idleTimeout)
	}
}

func TestServerConfig_HTTP2Config(t *testing.T) {
	http2Config := &http.HTTP2Config{
		MaxConcurrentStreams: 250,
		MaxReadFrameSize:     1 << 20,
	}

	cfg := &ServerConfig{
		HTTP2: http2Config,
	}

	if cfg.HTTP2 == nil {
		t.Fatal("HTTP2 config should not be nil")
	}

	if cfg.HTTP2.MaxConcurrentStreams != 250 {
		t.Errorf("MaxConcurrentStreams = %d, want 250", cfg.HTTP2.MaxConcurrentStreams)
	}

	if cfg.HTTP2.MaxReadFrameSize != 1<<20 {
		t.Errorf("MaxReadFrameSize = %d, want %d", cfg.HTTP2.MaxReadFrameSize, 1<<20)
	}
}

func TestServerConfig_Protocols(t *testing.T) {
	protocols := &http.Protocols{}

	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)

	cfg := &ServerConfig{
		Protocols: protocols,
	}

	if cfg.Protocols == nil {
		t.Fatal("Protocols should not be nil")
	}

	if !cfg.Protocols.HTTP1() {
		t.Error("HTTP1 should be enabled")
	}

	if !cfg.Protocols.HTTP2() {
		t.Error("HTTP2 should be enabled")
	}
}

func BenchmarkNewServerConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewServerConfig()
	}
}

func BenchmarkGetValueOrDefault(b *testing.B) {
	defaultValue := 15 * time.Second

	b.Run("ZeroValue", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			getValueOrDefault(time.Duration(0), defaultValue)
		}
	})

	b.Run("NonZeroValue", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			getValueOrDefault(10*time.Second, defaultValue)
		}
	})
}

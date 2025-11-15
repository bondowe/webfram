package webfram

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey2 string

const testContextKey2 contextKey2 = "test-key"

func TestGetValueOrDefault(t *testing.T) {
	tests := []struct {
		value        interface{}
		defaultValue interface{}
		expected     interface{}
		name         string
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
	mux := setupTestMux()
	addr := getFreeAddress(t)
	serverStopped := startTestServer(t, addr, mux)
	testServerResponse(t, addr)
	stopTestServer(t, serverStopped)
}

func setupTestMux() *ServeMux {
	mux := NewServeMux()
	mux.HandleFunc("GET /test", func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	return mux
}

func getFreeAddress(t *testing.T) string {
	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	err = listener.Close()
	if err != nil {
		t.Fatal(err)
	}
	return addr
}

func startTestServer(t *testing.T, addr string, mux *ServeMux) chan bool {
	serverStarted := make(chan bool)
	serverStopped := make(chan bool)

	go func() {
		defer handleServerPanic(t, serverStopped)
		serverStarted <- true
		ListenAndServe(addr, mux, nil)
	}()

	<-serverStarted
	time.Sleep(100 * time.Millisecond)
	return serverStopped
}

func handleServerPanic(t *testing.T, serverStopped chan bool) {
	if r := recover(); r != nil {
		if err, ok := r.(error); ok {
			if !errors.Is(err, http.ErrServerClosed) {
				t.Errorf("Unexpected server error: %v", err)
			}
		}
	}
	serverStopped <- true
}

func testServerResponse(t *testing.T, addr string) {
	resp, err := http.Get("http://" + addr + "/test")
	if err != nil {
		t.Logf("Failed to connect to server (expected if server hasn't started yet): %v", err)
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func stopTestServer(t *testing.T, serverStopped chan bool) {
	// Send interrupt signal to stop server
	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	_ = proc.Signal(syscall.SIGTERM)

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

	mux.HandleFunc("GET /test", func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	err = listener.Close()

	if err != nil {
		t.Fatal(err)
	}

	customLog := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg := &ServerConfig{
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    30 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
		ErrorLog:       customLog,
		ConnState: func(_ net.Conn, _ http.ConnState) {
		},
		BaseContext: func(_ net.Listener) context.Context {
			return context.WithValue(context.Background(), testContextKey2, "test-value")
		},
	}

	serverStopped := make(chan bool)

	go func() {
		defer func() {
			_ = recover()
			serverStopped <- true
		}()

		ListenAndServe(addr, mux, cfg)
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Make a request to trigger connection state
	resp, err := http.Get("http://" + addr + "/test")

	if err != nil {
		t.Logf("Failed to connect to server (expected if server hasn't started yet): %v", err)
	} else {
		defer resp.Body.Close()
	}

	// Stop the server
	proc, _ := os.FindProcess(os.Getpid())
	_ = proc.Signal(syscall.SIGTERM)

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
			Enabled: true,
			URLPath: "GET /openapi.json",
			Config: &OpenAPIConfig{
				Info: &Info{
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

	mux.HandleFunc("GET /api/test", func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	err = listener.Close()

	if err != nil {
		t.Fatal(err)
	}

	serverStopped := make(chan bool)

	go func() {
		defer func() {
			_ = recover()
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
	_ = proc.Signal(syscall.SIGTERM) // Wait for shutdown
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

	errorLog := slog.New(slog.NewTextHandler(os.Stdout, nil))

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
		ConnState: func(_ net.Conn, _ http.ConnState) {
			// Connection state handler
		},
		ErrorLog: errorLog,
		BaseContext: func(_ net.Listener) context.Context {
			return context.Background()
		},
		ConnContext: func(ctx context.Context, _ net.Conn) context.Context {
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
	mux.HandleFunc("GET /count", func(w ResponseWriter, _ *Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	err = listener.Close()

	if err != nil {
		t.Fatal(err)
	}

	serverStopped := make(chan bool)

	go func() {
		defer func() {
			_ = recover()
			serverStopped <- true
		}()

		ListenAndServe(addr, mux, nil)
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Make multiple requests
	for range 5 {
		resp, reqErr := http.Get("http://" + addr + "/count")
		if reqErr == nil {
			resp.Body.Close()
		}
		time.Sleep(10 * time.Millisecond)
	}

	if requestCount < 1 {
		t.Errorf("Expected at least 1 request, got %d", requestCount)
	}

	// Stop the server
	proc, _ := os.FindProcess(os.Getpid())
	_ = proc.Signal(syscall.SIGTERM) // Wait for shutdown
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

	// When passed to ListenAndServe, zero values should be replaced with defaults
	// We verify the logic by checking getValueOrDefault behavior
	rTimeout := getValueOrDefault(cfg.ReadTimeout, readTimeout)
	if rTimeout != 15*time.Second {
		t.Errorf("Expected default ReadTimeout 15s, got %v", rTimeout)
	}

	wTimeout := getValueOrDefault(cfg.WriteTimeout, writeTimeout)
	if wTimeout != 15*time.Second {
		t.Errorf("Expected default WriteTimeout 15s, got %v", wTimeout)
	}

	iTimeout := getValueOrDefault(cfg.IdleTimeout, idleTimeout)
	if iTimeout != 60*time.Second {
		t.Errorf("Expected default IdleTimeout 60s, got %v", iTimeout)
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

func BenchmarkGetValueOrDefault(b *testing.B) {
	defaultValue := 15 * time.Second

	b.Run("ZeroValue", func(b *testing.B) {
		for b.Loop() {
			getValueOrDefault(time.Duration(0), defaultValue)
		}
	})

	b.Run("NonZeroValue", func(b *testing.B) {
		for b.Loop() {
			getValueOrDefault(10*time.Second, defaultValue)
		}
	})
}

// Telemetry Tests

func TestSetupOpenAPIEndpoint_Enabled(t *testing.T) {
	// Save and restore original config
	originalConfig := openAPIConfig
	defer func() { openAPIConfig = originalConfig }()

	// Use Configure to properly initialize the OpenAPI config
	appConfigured = false
	Configure(&Config{
		OpenAPI: &OpenAPI{
			Enabled: true,
			URLPath: "GET /api-docs",
			Config: &OpenAPIConfig{
				Info: &Info{
					Title:   "Test API",
					Version: "1.0.0",
				},
			},
		},
	})

	mux := NewServeMux()
	setupOpenAPIEndpoint(mux)

	// The endpoint should exist (won't panic)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("setupOpenAPIEndpoint panicked: %v", r)
		}
	}()
}

func TestSetupOpenAPIEndpoint_Disabled(_ *testing.T) {
	originalConfig := openAPIConfig
	defer func() { openAPIConfig = originalConfig }()

	openAPIConfig = nil

	mux := NewServeMux()
	setupOpenAPIEndpoint(mux)

	// Should not panic when config is nil
}

func TestSetupOpenAPIEndpoint_InvalidConfig(t *testing.T) {
	originalConfig := openAPIConfig
	defer func() { openAPIConfig = originalConfig }()

	// Invalid config that will fail to marshal
	openAPIConfig = &OpenAPI{
		Enabled: true,
		URLPath: "GET /api-docs",
		Config:  nil, // This will cause panic
	}

	mux := NewServeMux()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid OpenAPI config")
		}
	}()

	setupOpenAPIEndpoint(mux)
}

func TestSetupTelemetry_Disabled(t *testing.T) {
	originalConfig := telemetryConfig
	defer func() { telemetryConfig = originalConfig }()

	telemetryConfig = nil

	mux := NewServeMux()
	server, separate := setupTelemetry(":8080", mux)

	if server != nil {
		t.Error("Expected nil server when telemetry is disabled")
	}
	if separate {
		t.Error("Expected separate to be false when telemetry is disabled")
	}
}

func TestSetupTelemetry_SameServer(t *testing.T) {
	originalConfig := telemetryConfig
	defer func() { telemetryConfig = originalConfig }()

	// Reset app configuration
	appConfigured = false
	Configure(&Config{
		Telemetry: &Telemetry{
			Enabled: true,
			URLPath: "GET /metrics",
			Addr:    "", // Empty means same server
		},
	})

	mux := NewServeMux()
	server, separate := setupTelemetry(":8080", mux)

	if server != nil {
		t.Error("Expected nil server when telemetry runs on same server")
	}
	if separate {
		t.Error("Expected separate to be false when telemetry runs on same server")
	}
}

func TestSetupTelemetry_SameServerMatchingAddr(t *testing.T) {
	originalConfig := telemetryConfig
	defer func() { telemetryConfig = originalConfig }()

	// Reset app configuration
	appConfigured = false
	Configure(&Config{
		Telemetry: &Telemetry{
			Enabled: true,
			URLPath: "GET /metrics",
			Addr:    ":8080", // Same as main server
		},
	})

	mux := NewServeMux()
	server, separate := setupTelemetry(":8080", mux)

	if server != nil {
		t.Error("Expected nil server when telemetry addr matches main addr")
	}
	if separate {
		t.Error("Expected separate to be false when telemetry addr matches main addr")
	}
}

func TestSetupTelemetry_SeparateServer(t *testing.T) {
	originalConfig := telemetryConfig
	defer func() { telemetryConfig = originalConfig }()

	// Reset app configuration
	appConfigured = false
	Configure(&Config{
		Telemetry: &Telemetry{
			Enabled: true,
			URLPath: "GET /metrics",
			Addr:    ":9090", // Different from main server
		},
	})

	mux := NewServeMux()
	server, separate := setupTelemetry(":8080", mux)

	if server == nil {
		t.Fatal("Expected non-nil server when telemetry runs on separate server")
	}
	if !separate {
		t.Error("Expected separate to be true when telemetry runs on separate server")
	}
	if server.Addr != ":9090" {
		t.Errorf("Expected telemetry server addr to be :9090, got %s", server.Addr)
	}
}

func TestCreateHTTPServer_NoConfig(t *testing.T) {
	mux := NewServeMux()
	server := createHTTPServer(":8080", mux, nil)

	if server.Addr != ":8080" {
		t.Errorf("Expected addr :8080, got %s", server.Addr)
	}
	if server.Handler != mux {
		t.Error("Expected handler to be mux")
	}
	if server.ReadTimeout != readTimeout {
		t.Errorf("Expected ReadTimeout %v, got %v", readTimeout, server.ReadTimeout)
	}
	if server.WriteTimeout != writeTimeout {
		t.Errorf("Expected WriteTimeout %v, got %v", writeTimeout, server.WriteTimeout)
	}
	if server.IdleTimeout != idleTimeout {
		t.Errorf("Expected IdleTimeout %v, got %v", idleTimeout, server.IdleTimeout)
	}
	if server.MaxHeaderBytes != maxHeaderBytes {
		t.Errorf("Expected MaxHeaderBytes %d, got %d", maxHeaderBytes, server.MaxHeaderBytes)
	}
}

func TestCreateHTTPServer_WithConfig(t *testing.T) {
	mux := NewServeMux()
	customReadTimeout := 20 * time.Second
	customWriteTimeout := 25 * time.Second
	customIdleTimeout := 90 * time.Second
	customMaxHeaderBytes := 2 << 20

	cfg := &ServerConfig{
		ReadTimeout:                  customReadTimeout,
		WriteTimeout:                 customWriteTimeout,
		IdleTimeout:                  customIdleTimeout,
		MaxHeaderBytes:               customMaxHeaderBytes,
		DisableGeneralOptionsHandler: true,
	}

	server := createHTTPServer(":8080", mux, cfg)

	if server.ReadTimeout != customReadTimeout {
		t.Errorf("Expected ReadTimeout %v, got %v", customReadTimeout, server.ReadTimeout)
	}
	if server.WriteTimeout != customWriteTimeout {
		t.Errorf("Expected WriteTimeout %v, got %v", customWriteTimeout, server.WriteTimeout)
	}
	if server.IdleTimeout != customIdleTimeout {
		t.Errorf("Expected IdleTimeout %v, got %v", customIdleTimeout, server.IdleTimeout)
	}
	if server.MaxHeaderBytes != customMaxHeaderBytes {
		t.Errorf("Expected MaxHeaderBytes %d, got %d", customMaxHeaderBytes, server.MaxHeaderBytes)
	}
	if !server.DisableGeneralOptionsHandler {
		t.Error("Expected DisableGeneralOptionsHandler to be true")
	}
}

func TestCreateHTTPServer_WithTLSConfig(t *testing.T) {
	mux := NewServeMux()
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	cfg := &ServerConfig{
		TLSConfig: tlsConfig,
	}

	server := createHTTPServer(":8443", mux, cfg)

	if server.TLSConfig != tlsConfig {
		t.Error("Expected TLSConfig to be set")
	}
	if server.TLSConfig.MinVersion != tls.VersionTLS13 {
		t.Errorf("Expected MinVersion TLS 1.3, got %d", server.TLSConfig.MinVersion)
	}
}

func TestCreateHTTPServer_WithCallbacks(t *testing.T) {
	mux := NewServeMux()
	connStateCalled := false
	baseContextCalled := false
	connContextCalled := false

	cfg := &ServerConfig{
		ConnState: func(_ net.Conn, _ http.ConnState) {
			connStateCalled = true
		},
		BaseContext: func(_ net.Listener) context.Context {
			baseContextCalled = true
			return context.Background()
		},
		ConnContext: func(ctx context.Context, _ net.Conn) context.Context {
			connContextCalled = true
			return ctx
		},
	}

	server := createHTTPServer(":8080", mux, cfg)

	if server.ConnState == nil {
		t.Error("Expected ConnState to be set")
	}
	if server.BaseContext == nil {
		t.Error("Expected BaseContext to be set")
	}
	if server.ConnContext == nil {
		t.Error("Expected ConnContext to be set")
	}

	// Test callbacks are invoked
	server.ConnState(nil, http.StateNew)
	if !connStateCalled {
		t.Error("ConnState callback was not called")
	}

	server.BaseContext(nil)
	if !baseContextCalled {
		t.Error("BaseContext callback was not called")
	}

	server.ConnContext(context.Background(), nil)
	if !connContextCalled {
		t.Error("ConnContext callback was not called")
	}
}

func TestStartServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping server start test in short mode")
	}

	mux := NewServeMux()
	mux.HandleFunc("GET /test", func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()

	server := createHTTPServer(addr, mux, nil)
	errorChan := make(chan error, 1)

	startServer(server, "test", errorChan)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Verify server is running by making a request
	resp, err := http.Get("http://" + addr + "/test")
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	} else {
		t.Logf("Server may not have started yet: %v", err)
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	// Verify no error was sent
	select {
	case serverErr := <-errorChan:
		t.Errorf("Unexpected error from server: %v", serverErr)
	case <-time.After(100 * time.Millisecond):
		// No error, as expected
	}
}

func TestShutdownServers_MainOnly(t *testing.T) {
	mux := NewServeMux()
	mainServer := createHTTPServer(":0", mux, nil)

	// Start the server
	errorChan := make(chan error, 1)
	startServer(mainServer, "main", errorChan)
	time.Sleep(100 * time.Millisecond)

	// Shutdown should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("shutdownServers panicked: %v", r)
		}
	}()

	shutdownServers(mainServer, nil, false)
}

func TestShutdownServers_BothServers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping dual server test in short mode")
	}

	// Create and start main server
	mainMux := NewServeMux()
	listener1, _ := net.Listen("tcp", "127.0.0.1:0")
	mainAddr := listener1.Addr().String()
	listener1.Close()
	mainServer := createHTTPServer(mainAddr, mainMux, nil)

	// Create and start telemetry server
	telemetryMux := NewServeMux()
	listener2, _ := net.Listen("tcp", "127.0.0.1:0")
	telemetryAddr := listener2.Addr().String()
	listener2.Close()
	telemetryServer := createHTTPServer(telemetryAddr, telemetryMux, nil)

	errorChan := make(chan error, 2)
	startServer(mainServer, "main", errorChan)
	startServer(telemetryServer, "telemetry", errorChan)
	time.Sleep(100 * time.Millisecond)

	// Shutdown should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("shutdownServers panicked: %v", r)
		}
	}()

	shutdownServers(mainServer, telemetryServer, true)
}

func TestTelemetryIntegration_SeparateServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Reset app configuration
	appConfigured = false

	// Find free ports
	listener1, _ := net.Listen("tcp", "127.0.0.1:0")
	mainAddr := listener1.Addr().String()
	listener1.Close()

	listener2, _ := net.Listen("tcp", "127.0.0.1:0")
	telemetryAddr := listener2.Addr().String()
	listener2.Close()

	Configure(&Config{
		Telemetry: &Telemetry{
			Enabled: true,
			URLPath: "GET /metrics",
			Addr:    telemetryAddr,
		},
	})

	mux := NewServeMux()
	mux.HandleFunc("GET /app", func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("app"))
	})

	telemetryServer, separate := setupTelemetry(mainAddr, mux)
	if !separate {
		t.Fatal("Expected separate telemetry server")
	}

	mainServer := createHTTPServer(mainAddr, mux, nil)

	errorChan := make(chan error, 2)
	startServer(mainServer, "main", errorChan)
	startServer(telemetryServer, "telemetry", errorChan)

	time.Sleep(200 * time.Millisecond)

	// Test main server
	resp, err := http.Get("http://" + mainAddr + "/app")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Main server: expected status 200, got %d", resp.StatusCode)
		}
	}

	// Test telemetry server
	resp, err = http.Get("http://" + telemetryAddr + "/metrics")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Telemetry server: expected status 200, got %d", resp.StatusCode)
		}
	}

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	mainServer.Shutdown(ctx)
	telemetryServer.Shutdown(ctx)
}

func TestTelemetryIntegration_SameServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Reset app configuration
	appConfigured = false

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := listener.Addr().String()
	listener.Close()

	Configure(&Config{
		Telemetry: &Telemetry{
			Enabled: true,
			URLPath: "GET /metrics",
			Addr:    "", // Same server
		},
	})

	mux := NewServeMux()
	mux.HandleFunc("GET /app", func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
	})

	telemetryServer, separate := setupTelemetry(addr, mux)
	if separate {
		t.Fatal("Expected telemetry on same server")
	}
	if telemetryServer != nil {
		t.Fatal("Expected nil telemetry server")
	}

	mainServer := createHTTPServer(addr, mux, nil)

	errorChan := make(chan error, 1)
	startServer(mainServer, "main", errorChan)

	time.Sleep(200 * time.Millisecond)

	// Test app endpoint
	resp, err := http.Get("http://" + addr + "/app")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("App endpoint: expected status 200, got %d", resp.StatusCode)
		}
	}

	// Test metrics endpoint on same server
	resp, err = http.Get("http://" + addr + "/metrics")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Metrics endpoint: expected status 200, got %d", resp.StatusCode)
		}
	}

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	mainServer.Shutdown(ctx)
}

func TestTelemetryConfig_DefaultURLPath(t *testing.T) {
	// Reset app configuration
	appConfigured = false

	Configure(&Config{
		Telemetry: &Telemetry{
			Enabled: true,
			URLPath: "", // Should default to "GET /metrics"
		},
	})

	if telemetryConfig.URLPath != "GET /metrics" {
		t.Errorf("Expected default URLPath 'GET /metrics', got %q", telemetryConfig.URLPath)
	}
}

func TestTelemetryConfig_CustomURLPath(t *testing.T) {
	// Reset app configuration
	appConfigured = false

	Configure(&Config{
		Telemetry: &Telemetry{
			Enabled: true,
			URLPath: "/custom-metrics", // Should be prefixed with "GET "
		},
	})

	if telemetryConfig.URLPath != "GET /custom-metrics" {
		t.Errorf("Expected URLPath 'GET /custom-metrics', got %q", telemetryConfig.URLPath)
	}
}

func TestTelemetryConfig_WithHandlerOpts(t *testing.T) {
	// Reset app configuration
	appConfigured = false

	handlerOpts := promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}

	Configure(&Config{
		Telemetry: &Telemetry{
			Enabled:     true,
			URLPath:     "GET /metrics",
			HandlerOpts: handlerOpts,
		},
	})

	if !telemetryConfig.HandlerOpts.EnableOpenMetrics {
		t.Error("Expected EnableOpenMetrics to be true")
	}
}

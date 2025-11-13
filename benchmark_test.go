package webfram

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkWebFram_Simple tests a simple route with WebFram.
func BenchmarkWebFram_Simple(b *testing.B) {
	mux := NewServeMux()
	mux.HandleFunc("GET /ping", func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		mux.ServeHTTP(w, req)
	}
}

// BenchmarkWebFram_PathParam tests route with path parameters.
func BenchmarkWebFram_PathParam(b *testing.B) {
	mux := NewServeMux()
	mux.HandleFunc("GET /user/{id}", func(w ResponseWriter, r *Request) {
		id := r.PathValue("id")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(id))
	})

	req := httptest.NewRequest(http.MethodGet, "/user/123", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		mux.ServeHTTP(w, req)
	}
}

// BenchmarkWebFram_JSON tests JSON response.
func BenchmarkWebFram_JSON(b *testing.B) {
	mux := NewServeMux()
	mux.HandleFunc("GET /json", func(w ResponseWriter, r *Request) {
		data := map[string]interface{}{
			"message": "Hello, World!",
			"status":  "success",
		}
		_ = w.JSON(r.Context(), data)
	})

	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		mux.ServeHTTP(w, req)
	}
}

// BenchmarkWebFram_5Routes tests routing with 5 routes.
func BenchmarkWebFram_5Routes(b *testing.B) {
	mux := NewServeMux()
	mux.HandleFunc("GET /ping", func(w ResponseWriter, _ *Request) {
		_, _ = w.Write([]byte("pong"))
	})
	mux.HandleFunc("GET /user/{id}", func(w ResponseWriter, r *Request) {
		_, _ = w.Write([]byte(r.PathValue("id")))
	})
	mux.HandleFunc("POST /user", func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("GET /users", func(w ResponseWriter, _ *Request) {
		_, _ = w.Write([]byte("users"))
	})
	mux.HandleFunc("DELETE /user/{id}", func(w ResponseWriter, _ *Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/user/123", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		mux.ServeHTTP(w, req)
	}
}

// BenchmarkWebFram_Middleware tests with middleware.
func BenchmarkWebFram_Middleware(b *testing.B) {
	mux := NewServeMux()

	// Add global middleware
	Use(func(next Handler) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			next.ServeHTTP(w, r)
		})
	})

	mux.HandleFunc("GET /ping", func(w ResponseWriter, _ *Request) {
		_, _ = w.Write([]byte("pong"))
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		mux.ServeHTTP(w, req)
	}
}

// BenchmarkStdLib_Simple tests standard library for comparison.
func BenchmarkStdLib_Simple(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		mux.ServeHTTP(w, req)
	}
}

// BenchmarkStdLib_PathParam tests standard library with path parameters.
func BenchmarkStdLib_PathParam(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /user/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(id))
	})

	req := httptest.NewRequest(http.MethodGet, "/user/123", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		mux.ServeHTTP(w, req)
	}
}

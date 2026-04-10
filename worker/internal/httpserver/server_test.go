package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer() *Server {
	return New(8088, "cert.pem", "key.pem", nil)
}

func TestHealthEndpoint(t *testing.T) {
	srv := newTestServer()
	mux := http.NewServeMux()
	srv.SetupRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp healthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != "ok" {
		t.Errorf("status: got %q, want ok", resp.Status)
	}
	if resp.Version == "" {
		t.Error("version is empty")
	}
}

func TestHealthMethodNotAllowed(t *testing.T) {
	srv := newTestServer()
	mux := http.NewServeMux()
	srv.SetupRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestDataEndpoint(t *testing.T) {
	srv := newTestServer()
	mux := http.NewServeMux()
	srv.SetupRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}
	var data map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &data)
	required := []string{"version", "hostname", "os", "bios", "cpu", "memory", "disks", "network", "current_user"}
	for _, f := range required {
		if _, ok := data[f]; !ok {
			t.Errorf("missing field: %s", f)
		}
	}
}

func TestLogsEndpoint(t *testing.T) {
	srv := newTestServer()
	mux := http.NewServeMux()
	srv.SetupRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLogsDownloadNoLogger(t *testing.T) {
	srv := newTestServer()
	mux := http.NewServeMux()
	srv.SetupRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/logs/download", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", w.Code)
	}
}

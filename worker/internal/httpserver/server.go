package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Zam83-AZE/zaur-test/worker/internal/collector"
	"github.com/Zam83-AZE/zaur-test/worker/pkg/version"
)

type Server struct {
	port      int
	startTime time.Time
	certFile  string
	keyFile   string
}

func New(port int, certFile, keyFile string) *Server {
	return &Server{
		port:      port,
		startTime: time.Now(),
		certFile:  certFile,
		keyFile:   keyFile,
	}
}

type healthResponse struct {
	Status        string `json:"status"`
	Version       string `json:"version"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	Timestamp     string `json:"timestamp"`
}

type logResponse struct {
	TotalLines    int      `json:"total_lines"`
	ReturnedLines int      `json:"returned_lines"`
	Logs          []string `json:"logs"`
}

func (s *Server) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/data", s.handleData)
	mux.HandleFunc("/logs", s.handleLogs)
	mux.HandleFunc("/logs/download", s.handleLogsDownload)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	resp := healthResponse{
		Status:        "ok",
		Version:       version.Version,
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
		Timestamp:     time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	info := collector.CollectAll()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Worker-Version", version.Version)
	w.Header().Set("Cache-Control", "no-cache")
	json.NewEncoder(w).Encode(info)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	resp := logResponse{
		TotalLines:    0,
		ReturnedLines: 0,
		Logs:          []string{},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleLogsDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

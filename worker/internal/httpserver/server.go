package httpserver

import (
        "encoding/json"
        "fmt"
        "net/http"
        "os"
        "strconv"
        "strings"
        "time"

        "github.com/Zam83-AZE/zaur-test/worker/internal/collector"
        "github.com/Zam83-AZE/zaur-test/worker/internal/logger"
        "github.com/Zam83-AZE/zaur-test/worker/pkg/version"
)

type Server struct {
        port       int
        startTime  time.Time
        certFile   string
        keyFile    string
        logger     *logger.Logger
}

func New(port int, certFile, keyFile string, log *logger.Logger) *Server {
        return &Server{
                port:      port,
                startTime: time.Now(),
                certFile:  certFile,
                keyFile:   keyFile,
                logger:    log,
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
        mux.HandleFunc("/health", s.corsMiddleware(s.logMiddleware(s.handleHealth)))
        mux.HandleFunc("/data", s.corsMiddleware(s.logMiddleware(s.handleData)))
        mux.HandleFunc("/logs", s.corsMiddleware(s.logMiddleware(s.handleLogs)))
        mux.HandleFunc("/logs/download", s.corsMiddleware(s.logMiddleware(s.handleLogsDownload)))
}

// corsMiddleware adds CORS headers to allow cross-origin requests from any origin
func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Access-Control-Allow-Origin", "*")
                w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
                w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
                w.Header().Set("Access-Control-Max-Age", "86400")

                // Handle preflight OPTIONS request
                if r.Method == http.MethodOptions {
                        w.WriteHeader(http.StatusNoContent)
                        return
                }

                next(w, r)
        }
}

// logMiddleware wraps a handler to log API access
func (s *Server) logMiddleware(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                start := time.Now()
                rec := &statusRecorder{ResponseWriter: w, statusCode: 200}

                next(rec, r)

                duration := time.Since(start)
                if s.logger != nil {
                        s.logger.Access(r.RemoteAddr, r.Method, r.URL.Path, strconv.Itoa(rec.statusCode), duration)
                }
        }
}

type statusRecorder struct {
        http.ResponseWriter
        statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
        r.statusCode = code
        r.ResponseWriter.WriteHeader(code)
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

        // Parse query params
        query := r.URL.Query()
        limit := 100 // default
        if l := query.Get("limit"); l != "" {
                if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 10000 {
                        limit = n
                }
        }
        logType := query.Get("type") // "access" or empty for main log

        var lines []string
        var err error

        if logType == "access" {
                if s.logger != nil {
                        lines = s.readAccessLogs(limit)
                }
        } else {
                if s.logger != nil {
                        lines, err = s.logger.ReadLastNLines(limit)
                        if err != nil {
                                s.sendError(w, "failed to read logs", http.StatusInternalServerError)
                                return
                        }
                }
        }

        resp := logResponse{
                TotalLines:    len(lines),
                ReturnedLines: len(lines),
                Logs:          lines,
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleLogsDownload(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
                return
        }

        if s.logger == nil {
                http.Error(w, "Logging not configured", http.StatusNotImplemented)
                return
        }

        query := r.URL.Query()
        logType := query.Get("type") // "access" or empty for main log

        var files []string
        if logType == "access" {
                files = s.logger.GetAccessLogFiles()
        } else {
                files = s.logger.GetLogFiles()
        }

        if len(files) == 0 {
                s.sendError(w, "no log files found", http.StatusNotFound)
                return
        }

        // Download the latest log file
        latestFile := files[len(files)-1]
        data, err := os.ReadFile(latestFile)
        if err != nil {
                s.sendError(w, "failed to read log file", http.StatusInternalServerError)
                return
        }

        filename := strings.TrimPrefix(latestFile, s.logger.GetLogDir()+"/")
        w.Header().Set("Content-Type", "text/plain; charset=utf-8")
        w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
        w.Header().Set("Content-Length", strconv.Itoa(len(data)))
        w.Write(data)
}

// readAccessLogs reads the last N lines from access log
func (s *Server) readAccessLogs(limit int) []string {
        files := s.logger.GetAccessLogFiles()
        if len(files) == 0 {
                return nil
        }

        data, err := os.ReadFile(files[len(files)-1])
        if err != nil {
                return nil
        }

        lines := strings.Split(strings.TrimSpace(string(data)), "\n")
        if len(lines) <= limit {
                return lines
        }
        return lines[len(lines)-limit:]
}

func (s *Server) sendError(w http.ResponseWriter, msg string, code int) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(code)
        json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

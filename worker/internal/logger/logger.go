package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Level represents log severity
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel converts string to Level
func ParseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// Logger provides leveled, rotating file logging
type Logger struct {
	mu        sync.Mutex
	level     Level
	logDir    string
	baseName  string
	maxSize   int64         // max bytes per file before rotation
	maxFiles  int           // max rotated files to keep
	maxDays   int           // max days to keep log files
	file      *os.File
	currentSize int64
	accessLog *log.Logger   // logs of API access
	logger    *log.Logger
}

// Config holds logger configuration
type Config struct {
	LogDir   string
	BaseName string
	Level    string
	MaxSizeMB int
	MaxFiles int
	MaxDays  int
}

// New creates a new rotating file logger
func New(cfg Config) (*Logger, error) {
	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create log directory %s: %w", cfg.LogDir, err)
	}

	if cfg.BaseName == "" {
		cfg.BaseName = "sysworker"
	}
	if cfg.MaxSizeMB <= 0 {
		cfg.MaxSizeMB = 10
	}
	if cfg.MaxFiles <= 0 {
		cfg.MaxFiles = 5
	}
	if cfg.MaxDays <= 0 {
		cfg.MaxDays = 30
	}

	l := &Logger{
		level:    ParseLevel(cfg.Level),
		logDir:   cfg.LogDir,
		baseName: cfg.BaseName,
		maxSize:  int64(cfg.MaxSizeMB) * 1024 * 1024,
		maxFiles: cfg.MaxFiles,
		maxDays:  cfg.MaxDays,
	}

	// Clean up old log files
	l.cleanup()

	// Open or create log file
	if err := l.openFile(); err != nil {
		return nil, err
	}

	return l, nil
}

func (l *Logger) openFile() error {
	today := time.Now().Format("2006-01-02")
	logPath := filepath.Join(l.logDir, l.baseName+"-"+today+".log")

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("cannot open log file %s: %w", logPath, err)
	}

	info, _ := f.Stat()
	l.currentSize = info.Size()
	l.file = f

	// Create standard logger writing to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, f)
	l.logger = log.New(multiWriter, "", 0)

	// Separate access log file
	accessPath := filepath.Join(l.logDir, l.baseName+"-access-"+today+".log")
	accessFile, err := os.OpenFile(accessPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Non-fatal
		l.accessLog = log.New(f, "", 0)
	} else {
		l.accessLog = log.New(accessFile, "", 0)
	}

	return nil
}

// logLine formats and writes a log line
func (l *Logger) logLine(level Level, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("[%s] %s %s\n", timestamp, level.String(), msg)

	l.logger.Print(line)

	// Check rotation
	l.currentSize += int64(len(line))
	if l.currentSize >= l.maxSize {
		l.rotate()
	}
}

// Debug logs a DEBUG message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.logLine(DEBUG, format, args...)
}

// Info logs an INFO message
func (l *Logger) Info(format string, args ...interface{}) {
	l.logLine(INFO, format, args...)
}

// Warn logs a WARN message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.logLine(WARN, format, args...)
}

// Error logs an ERROR message
func (l *Logger) Error(format string, args ...interface{}) {
	l.logLine(ERROR, format, args...)
}

// Access logs an API access entry
func (l *Logger) Access(remoteAddr, method, path, statusCode string, duration time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	line := fmt.Sprintf("[%s] ACCESS %s %s %s %s %s %dms\n",
		timestamp, remoteAddr, method, path, statusCode, time.Now().Format(time.RFC3339), duration.Milliseconds())

	l.accessLog.Print(line)
}

// rotate creates a new log file after the current one exceeds maxSize
func (l *Logger) rotate() {
	if l.file != nil {
		l.file.Close()
	}

	// Rename current file with timestamp suffix
	today := time.Now().Format("2006-01-02")
	currentPath := filepath.Join(l.logDir, l.baseName+"-"+today+".log")
	rotatedPath := filepath.Join(l.logDir, fmt.Sprintf("%s-%s-rotated-%s.log",
		l.baseName, today, time.Now().Format("150405")))

	os.Rename(currentPath, rotatedPath)

	l.openFile()
}

// cleanup removes log files older than maxDays and keeps only maxFiles rotated files
func (l *Logger) cleanup() {
	cutoff := time.Now().AddDate(0, 0, -l.maxDays)

	entries, err := os.ReadDir(l.logDir)
	if err != nil {
		return
	}

	// Delete files older than maxDays
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasPrefix(entry.Name(), l.baseName) || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(l.logDir, entry.Name()))
		}
	}

	// Keep only maxFiles rotated files per base name
	l.pruneRotatedFiles()
}

// pruneRotatedFiles keeps only the newest maxFiles rotated log files
func (l *Logger) pruneRotatedFiles() {
	entries, err := os.ReadDir(l.logDir)
	if err != nil {
		return
	}

	var rotatedFiles []os.FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, l.baseName) && strings.Contains(name, "-rotated-") && strings.HasSuffix(name, ".log") {
			info, _ := entry.Info()
			if info != nil {
				rotatedFiles = append(rotatedFiles, info)
			}
		}
	}

	// Sort by modification time (newest first)
	sort.Slice(rotatedFiles, func(i, j int) bool {
		return rotatedFiles[i].ModTime().After(rotatedFiles[j].ModTime())
	})

	// Remove oldest files beyond maxFiles limit
	for i := l.maxFiles; i < len(rotatedFiles); i++ {
		os.Remove(filepath.Join(l.logDir, rotatedFiles[i].Name()))
	}
}

// GetLogDir returns the log directory path
func (l *Logger) GetLogDir() string {
	return l.logDir
}

// GetLogFiles returns list of current log file paths
func (l *Logger) GetLogFiles() []string {
	entries, err := os.ReadDir(l.logDir)
	if err != nil {
		return nil
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, l.baseName) && strings.HasSuffix(name, ".log") && !strings.Contains(name, "-rotated-") {
			files = append(files, filepath.Join(l.logDir, name))
		}
	}
	sort.Strings(files)
	return files
}

// GetAccessLogFiles returns list of access log file paths
func (l *Logger) GetAccessLogFiles() []string {
	entries, err := os.ReadDir(l.logDir)
	if err != nil {
		return nil
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, l.baseName+"-access") && strings.HasSuffix(name, ".log") {
			files = append(files, filepath.Join(l.logDir, name))
		}
	}
	sort.Strings(files)
	return files
}

// Close closes the log files
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		l.file.Close()
	}
}

// ReadLastNLines reads the last N lines from the main log file
func (l *Logger) ReadLastNLines(n int) ([]string, error) {
	files := l.GetLogFiles()
	if len(files) == 0 {
		return nil, nil
	}

	// Read the latest log file
	data, err := os.ReadFile(files[len(files)-1])
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) <= n {
		return lines, nil
	}

	return lines[len(lines)-n:], nil
}

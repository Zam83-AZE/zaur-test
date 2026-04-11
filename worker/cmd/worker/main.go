package main

import (
        "context"
        "flag"
        "fmt"
        "net/http"
        "os"
        "os/signal"
        "runtime"
        "syscall"
        "time"

        "github.com/Zam83-AZE/zaur-test/worker/internal/certmanager"
        "github.com/Zam83-AZE/zaur-test/worker/internal/httpserver"
        "github.com/Zam83-AZE/zaur-test/worker/internal/logger"
        "github.com/Zam83-AZE/zaur-test/worker/pkg/version"
)

func main() {
        port := flag.Int("port", 8088, "HTTPS server port")
        certDir := flag.String("cert-dir", defaultCertDir(), "Certificate directory")
        logDir := flag.String("log-dir", defaultLogDir(), "Log directory")
        logLevel := flag.String("log-level", "INFO", "Log level: DEBUG, INFO, WARN, ERROR")
        showVersion := flag.Bool("version", false, "Show version and exit")
        flag.Parse()

        if *showVersion {
                fmt.Printf("System Worker %s (built: %s)\n", version.Version, version.BuildDate)
                os.Exit(0)
        }

        runInteractive(*port, *certDir, *logDir, *logLevel)
}

func runInteractive(port int, certDir, logDir, logLevel string) {
        log, err := logger.New(logger.Config{
                LogDir: logDir, BaseName: "sysworker", Level: logLevel,
                MaxSizeMB: 10, MaxFiles: 5, MaxDays: 30,
        })
        if err != nil {
                fmt.Fprintf(os.Stderr, "ERROR: Failed to initialize logger: %v\n", err)
                os.Exit(1)
        }
        defer log.Close()

        log.Info("System Worker %s starting...", version.Version)
        log.Info("  Port: %d", port)
        log.Info("  Cert Dir: %s", certDir)
        log.Info("  Log Dir: %s", logDir)
        log.Info("  Log Level: %s", logLevel)

        cm := certmanager.New(certDir)
        if err := cm.EnsureCertificates(); err != nil {
                log.Error("Failed to setup certificates: %v", err)
                os.Exit(1)
        }
        certFile, keyFile := cm.GetCertPaths()
        log.Info("TLS certificates ready")

        if err := cm.InstallToSystemTrustStore(); err != nil {
                log.Warn("Could not install certificate to system trust store: %v", err)
                log.Warn("Browsers may show security warnings. To trust manually, import %s", certFile)
        } else {
                log.Info("Certificate installed to system trust store")
        }

        srv := httpserver.New(port, certFile, keyFile, log)
        mux := http.NewServeMux()
        srv.SetupRoutes(mux)

        httpServer := &http.Server{
                Addr:         fmt.Sprintf(":%d", port),
                Handler:      mux,
                ReadTimeout:  10 * time.Second,
                WriteTimeout: 30 * time.Second,
                IdleTimeout:  60 * time.Second,
        }

        go func() {
                log.Info("Server listening on https://localhost:%d", port)
                if err := httpServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
                        log.Error("Server failed: %v", err)
                        os.Exit(1)
                }
        }()

        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
        sig := <-quit
        log.Info("Received signal %v, shutting down...", sig)

        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := httpServer.Shutdown(ctx); err != nil {
                log.Error("Shutdown failed: %v", err)
        }
        log.Info("Server stopped")
}

func defaultCertDir() string {
        if d := os.Getenv("SYSDATA_DIR"); d != "" {
                return d + "/cert"
        }
        if runtime.GOOS == "windows" {
                pd := os.Getenv("ProgramData")
                if pd == "" {
                        pd = `C:\ProgramData`
                }
                return pd + `\SysWorker\cert`
        }
        home, _ := os.UserHomeDir()
        if home == "" {
                return "/tmp/sysworker/cert"
        }
        return home + "/.sysworker/cert"
}

func defaultLogDir() string {
        if d := os.Getenv("SYSDATA_DIR"); d != "" {
                return d + "/logs"
        }
        if runtime.GOOS == "windows" {
                pd := os.Getenv("ProgramData")
                if pd == "" {
                        pd = `C:\ProgramData`
                }
                return pd + `\SysWorker\logs`
        }
        home, _ := os.UserHomeDir()
        if home == "" {
                return "/tmp/sysworker/logs"
        }
        return home + "/.sysworker/logs"
}

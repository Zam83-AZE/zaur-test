package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Zam83-AZE/zaur-test/worker/internal/certmanager"
	"github.com/Zam83-AZE/zaur-test/worker/internal/httpserver"
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

	fmt.Printf("System Worker %s starting...\n", version.Version)
	fmt.Printf("  Port: %d\n", *port)
	fmt.Printf("  Cert Dir: %s\n", *certDir)
	fmt.Printf("  Log Dir: %s\n", *logDir)
	fmt.Printf("  Log Level: %s\n", *logLevel)

	cm := certmanager.New(*certDir)
	if err := cm.EnsureCertificates(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to setup certificates: %v\n", err)
		os.Exit(1)
	}
	certFile, keyFile := cm.GetCertPaths()
	fmt.Println("  TLS certificates ready")

	srv := httpserver.New(*port, certFile, keyFile)
	mux := http.NewServeMux()
	srv.SetupRoutes(mux)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Printf("  Server listening on https://localhost:%d\n", *port)
		if err := httpServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "ERROR: Server failed: %v\n", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	fmt.Printf("\n  Received signal %v, shutting down...\n", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Shutdown failed: %v\n", err)
	}
	fmt.Println("  Server stopped")
}

func defaultCertDir() string {
	if d := os.Getenv("SYSDATA_DIR"); d != "" {
		return d + "/cert"
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
	home, _ := os.UserHomeDir()
	if home == "" {
		return "/tmp/sysworker/logs"
	}
	return home + "/.sysworker/logs"
}

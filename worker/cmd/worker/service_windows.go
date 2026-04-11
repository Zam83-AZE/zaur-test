//go:build windows

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Zam83-AZE/zaur-test/worker/internal/certmanager"
	"github.com/Zam83-AZE/zaur-test/worker/internal/httpserver"
	"github.com/Zam83-AZE/zaur-test/worker/internal/logger"
	"github.com/Zam83-AZE/zaur-test/worker/pkg/version"
	"golang.org/x/sys/windows/svc"
)

type windowsService struct {
	port     int
	certDir  string
	logDir   string
	logLevel string
}

func (ws *windowsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending, Accepts: cmdsAccepted}

	certFile, keyFile, httpServer, log, initErr := ws.initialize()
	if initErr != nil {
		fmt.Fprintf(os.Stderr, "Service init failed: %v\n", initErr)
		changes <- svc.Status{State: svc.Stopped}
		return false, 1
	}
	defer log.Close()

	go func() {
		log.Info("Server listening on https://localhost:%d", ws.port)
		if err := httpServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			log.Error("Server failed: %v", err)
		}
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for {
		c := <-r
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			httpServer.Shutdown(ctx)
			cancel()
			log.Info("Service stopped")
			changes <- svc.Status{State: svc.Stopped}
			return false, 0
		}
	}
}

func (ws *windowsService) initialize() (certFile, keyFile string, httpServer *http.Server, log *logger.Logger, err error) {
	log, err = logger.New(logger.Config{
		LogDir: ws.logDir, BaseName: "sysworker", Level: ws.logLevel,
		MaxSizeMB: 10, MaxFiles: 5, MaxDays: 30,
	})
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("logger init failed: %w", err)
	}

	log.Info("System Worker %s starting as Windows service...", version.Version)
	log.Info("  Port: %d", ws.port)
	log.Info("  Cert Dir: %s", ws.certDir)
	log.Info("  Log Dir: %s", ws.logDir)

	cm := certmanager.New(ws.certDir)
	if err := cm.EnsureCertificates(); err != nil {
		log.Error("Certificate setup failed: %v", err)
		return "", "", nil, log, err
	}
	certFile, keyFile = cm.GetCertPaths()
	log.Info("TLS certificates ready")

	// Skip trust store install in service mode - installer handles it
	log.Info("Skipping system trust store (handled by installer)")

	srv := httpserver.New(ws.port, certFile, keyFile, log)
	mux := http.NewServeMux()
	srv.SetupRoutes(mux)
	httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", ws.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return certFile, keyFile, httpServer, log, nil
}

// tryRunAsService tries svc.Run(). If not in a service context, returns false.
// This avoids calling svc.IsWindowsService() which also calls
// StartServiceCtrlDispatcher internally - Windows only allows ONE call per process.
func tryRunAsService(port int, certDir, logDir, logLevel string) bool {
	ws := &windowsService{port: port, certDir: certDir, logDir: logDir, logLevel: logLevel}
	err := svc.Run("sysworker", ws)
	if err != nil {
		return false // not a service context
	}
	return true // service stopped normally
}

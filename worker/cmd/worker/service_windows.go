//go:build windows

package main

// On Windows, the worker runs as a normal process managed by Task Scheduler.
// No SCM (Service Control Manager) integration needed.
// svc.Run() is intentionally NOT called — schtasks handles process lifecycle.
func tryRunAsService(port int, certDir, logDir, logLevel string) bool {
	return false
}

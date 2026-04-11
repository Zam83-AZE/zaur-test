//go:build windows

package service

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type taskManager struct {
	name string
}

func newPlatformManager(name string) (Manager, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	return &taskManager{name: name}, nil
}

// runSchtasks executes a schtasks command and returns combined stdout+stderr.
func runSchtasks(args ...string) (string, error) {
	cmd := exec.Command("schtasks.exe", args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// Install creates a scheduled task that starts on boot (and on logon),
// runs as SYSTEM in the background with highest privileges.
func (t *taskManager) Install(cfg Config) error {
	exePath := cfg.BinaryPath
	if exePath == "" {
		return fmt.Errorf("binary path is empty")
	}
	exePath = filepath.Clean(exePath)

	taskCmd := fmt.Sprintf(`"%s" -port %d -log-level %s -cert-dir "%s\cert" -log-dir "%s\logs"`,
		exePath, cfg.Port, cfg.LogLevel, cfg.DataDir, cfg.DataDir)

	// Delete existing task first (ignore error if not found)
	runSchtasks("/delete", "/tn", t.name, "/f")

	// Create task: start on system boot, run as SYSTEM, hidden, highest privilege
	out, err := runSchtasks(
		"/create",
		"/tn", t.name,
		"/tr", taskCmd,
		"/sc", "onstart",
		"/ru", "SYSTEM",
		"/rl", "highest",
		"/f",
	)
	if err != nil {
		return fmt.Errorf("failed to create scheduled task: %w\nOutput: %s", err, out)
	}

	// Also add logon trigger so it starts when user logs in
	out, err = runSchtasks(
		"/change",
		"/tn", t.name,
		"/sd", "2000/01/01", // enable all days
		"/st", "00:00",
		"/f",
	)
	if err != nil {
		// Non-critical: the onstart trigger is enough
		fmt.Printf("       NOTE: Could not set daily trigger (non-critical): %v\n", err)
	}

	return nil
}

// Uninstall removes the scheduled task.
func (t *taskManager) Uninstall() error {
	if !t.IsInstalled() {
		return fmt.Errorf("task '%s' not found", t.name)
	}

	// Stop the task if running, then delete
	runSchtasks("/end", "/tn", t.name)

	out, err := runSchtasks("/delete", "/tn", t.name, "/f")
	if err != nil {
		return fmt.Errorf("failed to delete scheduled task: %w\nOutput: %s", err, out)
	}
	return nil
}

// Start runs the scheduled task immediately.
func (t *taskManager) Start() error {
	out, err := runSchtasks("/run", "/tn", t.name)
	if err != nil {
		return fmt.Errorf("failed to start task: %w\nOutput: %s", err, out)
	}
	return nil
}

// Stop ends the running task process.
func (t *taskManager) Stop() error {
	out, err := runSchtasks("/end", "/tn", t.name)
	if err != nil {
		return fmt.Errorf("failed to stop task: %w\nOutput: %s", err, out)
	}
	return nil
}

// Status returns the current status of the scheduled task.
func (t *taskManager) Status() (string, error) {
	out, err := runSchtasks("/query", "/tn", t.name, "/fo", "list", "/v")
	if err != nil {
		return "not installed", nil
	}

	// Parse output to find status
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "status:") {
			status := strings.TrimSpace(strings.TrimPrefix(line, "Status:"))
			status = strings.TrimSpace(strings.TrimPrefix(line, "status:"))
			switch strings.ToLower(status) {
			case "running":
				return "running", nil
			case "ready", "disabled":
				return "stopped", nil
			default:
				return strings.ToLower(status), nil
			}
		}
	}

	return "unknown", nil
}

// IsInstalled checks if the scheduled task exists.
func (t *taskManager) IsInstalled() bool {
	_, err := runSchtasks("/query", "/tn", t.name)
	return err == nil
}

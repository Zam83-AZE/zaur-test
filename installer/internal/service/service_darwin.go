//go:build darwin

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

type launchdManager struct {
	name string
}

func newPlatformManager(name string) (Manager, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	return &launchdManager{name: name}, nil
}

const launchdPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
        <string>-port</string>
        <string>{{.Port}}</string>
        <string>-log-level</string>
        <string>{{.LogLevel}}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.DataDir}}/logs/stdout.log</string>
    <key>StandardErrorPath</key>
    <string>{{.DataDir}}/logs/stderr.log</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>SYSDATA_DIR</key>
        <string>{{.DataDir}}</string>
    </dict>
    <key>ProcessType</key>
    <string>Background</string>
</dict>
</plist>
`

type plistData struct {
	Label      string
	BinaryPath string
	Port       string
	LogLevel   string
	DataDir    string
}

func (l *launchdManager) Install(cfg Config) error {
	plistContent, err := l.renderPlist(cfg)
	if err != nil {
		return fmt.Errorf("failed to render launchd plist: %w", err)
	}

	// Determine plist location
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	launchAgentsDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	plistPath := filepath.Join(launchAgentsDir, l.name+".plist")

	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	// Ensure log directory exists
	logDir := filepath.Join(cfg.DataDir, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Unload if already loaded
	exec.Command("launchctl", "unload", plistPath).Run()

	// Load the plist
	if output, err := exec.Command("launchctl", "load", plistPath).CombinedOutput(); err != nil {
		return fmt.Errorf("failed to load launchd service: %w\n%s", err, string(output))
	}

	return nil
}

func (l *launchdManager) Uninstall() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	plistPath := filepath.Join(home, "Library", "LaunchAgents", l.name+".plist")

	// Unload if loaded
	_ = exec.Command("launchctl", "unload", plistPath).Run()

	// Remove plist file
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	return nil
}

func (l *launchdManager) Start() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	plistPath := filepath.Join(home, "Library", "LaunchAgents", l.name+".plist")

	cmd := exec.Command("launchctl", "load", plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start service: %w\n%s", err, string(output))
	}
	return nil
}

func (l *launchdManager) Stop() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	plistPath := filepath.Join(home, "Library", "LaunchAgents", l.name+".plist")

	cmd := exec.Command("launchctl", "unload", plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop service: %w\n%s", err, string(output))
	}
	return nil
}

func (l *launchdManager) Status() (string, error) {
	cmd := exec.Command("launchctl", "list", l.name)
	output, err := cmd.CombinedOutput()
	if err != nil || len(output) == 0 {
		return "not loaded", nil
	}

	str := string(output)
	if strings.Contains(str, "PID") {
		return "running", nil
	}
	return "loaded", nil
}

func (l *launchdManager) IsInstalled() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	plistPath := filepath.Join(home, "Library", "LaunchAgents", l.name+".plist")
	_, err = os.Stat(plistPath)
	return err == nil
}

func (l *launchdManager) renderPlist(cfg Config) (string, error) {
	tmpl, err := template.New("launchd").Parse(launchdPlistTemplate)
	if err != nil {
		return "", err
	}

	data := plistData{
		Label:      "com.sysworker." + l.name,
		BinaryPath: cfg.BinaryPath,
		Port:       fmt.Sprintf("%d", cfg.Port),
		LogLevel:   cfg.LogLevel,
		DataDir:    cfg.DataDir,
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

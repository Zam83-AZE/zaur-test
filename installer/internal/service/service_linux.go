//go:build linux

package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

type systemdManager struct {
	name string
}

func newPlatformManager(name string) (Manager, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	return &systemdManager{name: name}, nil
}

const systemdUnitTemplate = `[Unit]
Description=System Worker - System monitoring HTTPS service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart={{.BinaryPath}} -port {{.Port}} -log-level {{.LogLevel}}
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths={{.DataDir}}
PrivateTmp=true

{{- if .UserName}}
User={{.UserName}}
{{- end}}

[Install]
WantedBy=multi-user.target
`

func (s *systemdManager) Install(cfg Config) error {
	unitContent, err := s.renderUnit(cfg)
	if err != nil {
		return fmt.Errorf("failed to render systemd unit: %w", err)
	}

	unitPath := filepath.Join("/etc/systemd/system", s.name+".service")

	// Check if systemd is available
	if _, err := os.Stat("/run/systemd/system"); os.IsNotExist(err) {
		return fmt.Errorf("systemd is not available on this system")
	}

	// Write the unit file
	if err := os.WriteFile(unitPath, []byte(unitContent), 0644); err != nil {
		return fmt.Errorf("failed to write systemd unit file (need sudo): %w", err)
	}

	// Reload systemd daemon
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Enable the service
	if err := exec.Command("systemctl", "enable", s.name+".service").Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	return nil
}

func (s *systemdManager) Uninstall() error {
	unitPath := filepath.Join("/etc/systemd/system", s.name+".service")

	// Stop if running
	_ = exec.Command("systemctl", "stop", s.name+".service").Run()

	// Disable the service
	_ = exec.Command("systemctl", "disable", s.name+".service").Run()

	// Remove unit file
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove systemd unit file: %w", err)
	}

	// Reload systemd
	_ = exec.Command("systemctl", "daemon-reload").Run()

	return nil
}

func (s *systemdManager) Start() error {
	cmd := exec.Command("systemctl", "start", s.name+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start service: %w\n%s", err, string(output))
	}
	return nil
}

func (s *systemdManager) Stop() error {
	cmd := exec.Command("systemctl", "stop", s.name+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop service: %w\n%s", err, string(output))
	}
	return nil
}

func (s *systemdManager) Status() (string, error) {
	cmd := exec.Command("systemctl", "is-active", s.name+".service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "inactive", nil
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *systemdManager) IsInstalled() bool {
	unitPath := filepath.Join("/etc/systemd/system", s.name+".service")
	_, err := os.Stat(unitPath)
	return err == nil
}

func (s *systemdManager) renderUnit(cfg Config) (string, error) {
	tmpl, err := template.New("systemd").Parse(systemdUnitTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", err
	}

	return buf.String(), nil
}

//go:build !linux && !windows && !darwin

package service

import (
	"fmt"
)

type noopManager struct {
	name string
}

func newPlatformManager(name string) (Manager, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	return &noopManager{name: name}, nil
}

func (n *noopManager) Install(cfg Config) error {
	return fmt.Errorf("service management is not supported on this platform")
}

func (n *noopManager) Uninstall() error {
	return fmt.Errorf("service management is not supported on this platform")
}

func (n *noopManager) Start() error {
	return fmt.Errorf("service management is not supported on this platform")
}

func (n *noopManager) Stop() error {
	return fmt.Errorf("service management is not supported on this platform")
}

func (n *noopManager) Status() (string, error) {
	return "unsupported", fmt.Errorf("service management is not supported on this platform")
}

func (n *noopManager) IsInstalled() bool {
	return false
}

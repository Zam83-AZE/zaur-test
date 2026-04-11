package service

import "fmt"

// Config holds service configuration parameters.
type Config struct {
	BinaryPath string
	DataDir    string
	Port       int
	LogLevel   string
	UserName   string // For running service as specific user (Unix)
}

// Manager defines the interface for system service management.
type Manager interface {
	// Install creates and enables the system service.
	Install(cfg Config) error

	// Uninstall stops and removes the system service.
	Uninstall() error

	// Start starts the system service.
	Start() error

	// Stop stops the running system service.
	Stop() error

	// Status returns the current status of the service.
	Status() (string, error)

	// IsInstalled checks if the service is already installed.
	IsInstalled() bool
}

// NewManager creates the appropriate service manager for the current platform.
func NewManager(name string) (Manager, error) {
	return newPlatformManager(name)
}

// serviceName is the global service name.
func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	return nil
}

package detect

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Platform holds detected OS and architecture information.
type Platform struct {
	GOOS   string
	GOARCH string
	OS     string // Human-readable OS name
	Arch   string // Human-readable architecture name
}

// Detect returns the current platform information.
func Detect() Platform {
	p := Platform{
		GOOS:   runtime.GOOS,
		GOARCH: runtime.GOARCH,
	}

	switch p.GOOS {
	case "linux":
		p.OS = "linux"
	case "windows":
		p.OS = "windows"
	case "darwin":
		p.OS = "darwin"
	default:
		p.OS = p.GOOS
	}

	switch p.GOARCH {
	case "amd64":
		p.Arch = "amd64"
	case "arm64":
		p.Arch = "arm64"
	case "386":
		p.Arch = "386"
	default:
		p.Arch = p.GOARCH
	}

	return p
}

// BinaryName returns the expected worker binary name for this platform.
func (p Platform) BinaryName() string {
	name := fmt.Sprintf("sysworker-%s-%s", p.OS, p.Arch)
	if p.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// BinaryExt returns the file extension for executables on this platform.
func (p Platform) BinaryExt() string {
	if p.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// ServiceName returns the system service name.
func (p Platform) ServiceName() string {
	return "sysworker"
}

// DefaultInstallDir returns the default installation directory for the worker binary.
func (p Platform) DefaultInstallDir() string {
	switch p.GOOS {
	case "windows":
		return `C:\Program Files\SysWorker`
	case "darwin":
		home, _ := os.UserHomeDir()
		if home != "" {
			return home + "/.sysworker/bin"
		}
		return "/usr/local/bin"
	default: // linux
		return "/usr/local/bin"
	}
}

// DefaultDataDir returns the default data directory (for certs, logs, config).
func (p Platform) DefaultDataDir() string {
	switch p.GOOS {
	case "windows":
		return `C:\ProgramData\SysWorker`
	default:
		home, _ := os.UserHomeDir()
		if home != "" {
			return home + "/.sysworker"
		}
		return "/tmp/sysworker"
	}
}

// InstallPath returns the full path where the worker binary will be installed.
func (p Platform) InstallPath(installDir string) string {
	if p.GOOS == "windows" {
		return installDir + "\\" + p.ServiceName() + p.BinaryExt()
	}
	return installDir + "/" + p.ServiceName() + p.BinaryExt()
}

// String returns a human-readable platform description.
func (p Platform) String() string {
	return fmt.Sprintf("%s/%s", p.OS, p.Arch)
}

// NormalizeGOARCH returns the normalized architecture string for release asset matching.
func NormalizeGOARCH(goarch string) string {
	goarch = strings.ToLower(goarch)
	switch goarch {
	case "arm64", "aarch64":
		return "arm64"
	case "amd64", "x86_64":
		return "amd64"
	case "386", "i386", "i686":
		return "386"
	default:
		return goarch
	}
}

//go:build windows

package collector

import (
	"os"
	"runtime"
	"syscall"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func getDomain() string {
	if d := os.Getenv("USERDOMAIN"); d != "" {
		return d
	}
	if d := os.Getenv("COMPUTERNAME"); d != "" {
		return d
	}
	return ""
}

func CollectOS() models.OSInfo {
	info := models.OSInfo{Arch: runtime.GOARCH}

	k, err := syscall.OpenKey(syscall.HKEY_LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		syscall.KEY_READ)
	if err == nil {
		defer syscall.CloseHandle(k)
		if name, _, err := syscall.RegQueryStringValue(k, "ProductName"); err == nil {
			info.Name = name
		}
		if ver, _, err := syscall.RegQueryStringValue(k, "DisplayVersion"); err == nil {
			info.Version = ver
		}
		if build, _, err := syscall.RegQueryStringValue(k, "CurrentBuild"); err == nil {
			info.Build = build
		}
	}

	return info
}

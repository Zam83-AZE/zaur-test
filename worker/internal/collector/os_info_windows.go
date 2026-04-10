//go:build windows

package collector

import (
	"os"
	"runtime"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
	"golang.org/x/sys/windows/registry"
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

func collectOSWindows() models.OSInfo {
	info := models.OSInfo{Arch: runtime.GOARCH}

	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()

		if name, _, err := k.GetStringValue("ProductName"); err == nil {
			info.Name = name
		}
		if ver, _, err := k.GetStringValue("DisplayVersion"); err == nil {
			info.Version = ver
		} else if ver, _, err := k.GetStringValue("CurrentVersion"); err == nil {
			info.Version = ver
		}
		if build, _, err := k.GetStringValue("CurrentBuild"); err == nil {
			info.Build = build
			if ubr, _, err := k.GetStringValue("UBR"); err == nil {
				info.Build = build + "." + ubr
			}
		}
	}

	return info
}

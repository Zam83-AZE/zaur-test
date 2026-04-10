//go:build linux

package collector

import (
	"os"
	"strings"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectBIOS() models.BIOSInfo {
	info := models.BIOSInfo{}

	// Try /sys/class/dmi/id/
	dmiBase := "/sys/class/dmi/id/"

	if data, err := os.ReadFile(dmiBase + "bios_vendor"); err == nil {
		info.Vendor = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(dmiBase + "bios_version"); err == nil {
		info.Version = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(dmiBase + "bios_date"); err == nil {
		info.ReleaseDate = strings.TrimSpace(string(data))
	}

	// BIOS name: combine vendor + version if both exist
	if info.Vendor != "" && info.Version != "" {
		info.Name = info.Vendor + " " + info.Version
	} else if info.Vendor != "" {
		info.Name = info.Vendor
	} else if info.Version != "" {
		info.Name = info.Version
	}

	// Serial number requires root/sudo
	if data, err := os.ReadFile(dmiBase + "board_serial"); err == nil {
		info.SerialNumber = strings.TrimSpace(string(data))
	}

	return info
}

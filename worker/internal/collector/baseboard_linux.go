//go:build linux

package collector

import (
	"os"
	"strings"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectBaseboard() models.BaseboardInfo {
	info := models.BaseboardInfo{}
	dmiBase := "/sys/class/dmi/id/"

	if data, err := os.ReadFile(dmiBase + "board_vendor"); err == nil {
		info.Manufacturer = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(dmiBase + "board_name"); err == nil {
		info.Product = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(dmiBase + "board_serial"); err == nil {
		info.SerialNumber = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(dmiBase + "board_version"); err == nil {
		info.Version = strings.TrimSpace(string(data))
	}

	return info
}

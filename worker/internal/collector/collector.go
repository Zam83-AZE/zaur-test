package collector

import (
	"os"
	"time"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
	"github.com/Zam83-AZE/zaur-test/worker/pkg/version"
)

func CollectAll() *models.SystemInfo {
	info := &models.SystemInfo{
		Version:     version.Version,
		CollectedAt: time.Now().Format(time.RFC3339),
	}

	info.Hostname, _ = os.Hostname()
	info.Domain = getDomain()
	info.OS = CollectOS()
	info.BIOS = CollectBIOS()
	info.Baseboard = CollectBaseboard()
	info.CPU = CollectCPU()
	info.Memory = CollectMemory()
	info.Disks = CollectDisks()
	info.Network = CollectNetwork()
	info.GPU = CollectGPU()
	info.CurrentUser = CollectUser()

	return info
}

// getDomain is defined per-OS in os_info_*.go files

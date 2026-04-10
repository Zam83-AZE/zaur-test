//go:build darwin

package collector

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectMemory() models.MemoryInfo {
	info := models.MemoryInfo{}

	if output, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
		if total, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64); err == nil {
			info.TotalBytes = total
			info.TotalGB = float64(total) / (1024 * 1024 * 1024)
		}
	}

	if output, err := exec.Command("vm_stat").Output(); err == nil {
		text := string(output)
		var pageSize int64 = 4096
		var freePages int64

		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Pages free:") {
				val := strings.TrimSpace(strings.TrimPrefix(line, "Pages free:"))
				val = strings.TrimSuffix(val, ".")
				freePages, _ = strconv.ParseInt(val, 10, 64)
			}
			if strings.HasPrefix(line, "Pages active:") || strings.HasPrefix(line, "Pages inactive:") || strings.HasPrefix(line, "Pages speculative:") || strings.HasPrefix(line, "Purgeable pages:") || strings.HasPrefix(line, "Wired down:") || strings.HasPrefix(line, "File-backed pages:") {
				// Additional accounting could go here
			}
		}

		info.AvailableBytes = freePages * pageSize
		info.AvailableGB = float64(info.AvailableBytes) / (1024 * 1024 * 1024)
		if info.TotalBytes > 0 {
			info.UsedPercent = float64(info.TotalBytes-info.AvailableBytes) / float64(info.TotalBytes) * 100
			if info.UsedPercent < 0 {
				info.UsedPercent = 0
			}
		}
	}

	return info
}

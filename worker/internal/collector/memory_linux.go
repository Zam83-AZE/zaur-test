//go:build linux

package collector

import (
	"os"
	"strconv"
	"strings"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectMemory() models.MemoryInfo {
	info := models.MemoryInfo{}

	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return info
	}

	var totalKB, availableKB, memFreeKB, buffersKB, cachedKB int64

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "MemTotal:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "MemTotal:"))
			val = strings.TrimSuffix(val, " kB")
			totalKB, _ = strconv.ParseInt(val, 10, 64)
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "MemAvailable:"))
			val = strings.TrimSuffix(val, " kB")
			availableKB, _ = strconv.ParseInt(val, 10, 64)
		}
		if strings.HasPrefix(line, "MemFree:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "MemFree:"))
			val = strings.TrimSuffix(val, " kB")
			memFreeKB, _ = strconv.ParseInt(val, 10, 64)
		}
		if strings.HasPrefix(line, "Buffers:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Buffers:"))
			val = strings.TrimSuffix(val, " kB")
			buffersKB, _ = strconv.ParseInt(val, 10, 64)
		}
		if strings.HasPrefix(line, "Cached:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Cached:"))
			val = strings.TrimSuffix(val, " kB")
			cachedKB, _ = strconv.ParseInt(val, 10, 64)
		}
	}

	info.TotalBytes = totalKB * 1024
	info.TotalGB = float64(info.TotalBytes) / (1024 * 1024 * 1024)

	// MemAvailable is not always present in older kernels
	if availableKB > 0 {
		info.AvailableBytes = availableKB * 1024
	} else {
		info.AvailableBytes = (memFreeKB + buffersKB + cachedKB) * 1024
	}
	info.AvailableGB = float64(info.AvailableBytes) / (1024 * 1024 * 1024)

	if info.TotalBytes > 0 {
		info.UsedPercent = float64(info.TotalBytes-info.AvailableBytes) / float64(info.TotalBytes) * 100
		if info.UsedPercent < 0 {
			info.UsedPercent = 0
		}
	}

	return info
}

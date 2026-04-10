//go:build linux

package collector

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectCPU() models.CPUInfo {
	info := models.CPUInfo{
		CoresLogical: runtime.NumCPU(),
	}

	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		text := string(data)
		modelFound := false
		freqFound := false
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "model name") && !modelFound {
				info.Model = strings.TrimSpace(strings.TrimPrefix(line, "model name"))
				info.Model = strings.TrimPrefix(info.Model, ":")
				info.Model = strings.TrimSpace(info.Model)
				modelFound = true
			}
			if strings.HasPrefix(line, "cpu MHz") && !freqFound {
				val := strings.TrimSpace(strings.TrimPrefix(line, "cpu MHz"))
				val = strings.TrimPrefix(val, ":")
				val = strings.TrimSpace(val)
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					info.FrequencyMHz = int(f)
					freqFound = true
				}
			}
		}
	}

	// Get physical cores by counting unique core IDs
	coreIDs := make(map[string]bool)
	cpuDirs, _ := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*")
	for _, cpuDir := range cpuDirs {
		coreIDPath := filepath.Join(cpuDir, "topology", "core_id")
		if data, err := os.ReadFile(coreIDPath); err == nil {
			coreIDs[strings.TrimSpace(string(data))] = true
		}
	}
	if len(coreIDs) > 0 {
		info.CoresPhysical = len(coreIDs)
	} else {
		info.CoresPhysical = info.CoresLogical
	}

	return info
}

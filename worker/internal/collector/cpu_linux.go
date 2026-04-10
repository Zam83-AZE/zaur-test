//go:build linux

package collector

import (
	"os"
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
			if strings.HasPrefix(line, "physical id") {
				break
			}
		}
	}

	// Get physical cores from CPU topology
	if data, err := os.ReadFile("/sys/devices/system/cpu/cpu0/topology/core_siblings_list"); err == nil {
		rangeStr := strings.TrimSpace(string(data))
		count := 1
		if strings.Contains(rangeStr, "-") {
			parts := strings.Split(rangeStr, "-")
			if len(parts) == 2 {
				start, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
				end, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				count = end - start + 1
			}
		}
		info.CoresPhysical = count
	} else if info.CoresLogical > 0 {
		info.CoresPhysical = info.CoresLogical
	}

	return info
}

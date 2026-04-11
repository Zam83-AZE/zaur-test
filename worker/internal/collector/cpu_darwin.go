//go:build darwin

package collector

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectCPU() models.CPUInfo {
	info := models.CPUInfo{
		CoresLogical: runtime.NumCPU(),
	}

	if output, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
		info.Model = strings.TrimSpace(string(output))
	}

	if output, err := exec.Command("sysctl", "-n", "hw.cpufrequency").Output(); err == nil {
		if freq, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64); err == nil {
			info.FrequencyMHz = int(freq / 1000000)
		}
	}

	if output, err := exec.Command("sysctl", "-n", "hw.physicalcpu").Output(); err == nil {
		if cores, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil {
			info.CoresPhysical = cores
		}
	}

	return info
}

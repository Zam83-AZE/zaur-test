//go:build darwin

package collector

import (
	"os/exec"
	"strings"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectBIOS() models.BIOSInfo {
	info := models.BIOSInfo{}

	if output, err := exec.Command("system_profiler", "SPHardwareDataType").Output(); err == nil {
		text := string(output)
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Boot ROM Version:") {
				info.Version = strings.TrimSpace(strings.TrimPrefix(line, "Boot ROM Version:"))
			}
			if strings.HasPrefix(line, "SMC Version (system):") {
				_ = strings.TrimSpace(strings.TrimPrefix(line, "SMC Version (system):"))
			}
		}
	}

	if output, err := exec.Command("ioreg", "-l", "-p", "IOService", "-w", "0").Output(); err == nil {
		text := string(output)
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, `"IOPlatformSerialNumber"`) {
				parts := strings.Split(line, `"`)
				for i, p := range parts {
					if p == "IOPlatformSerialNumber" && i+3 < len(parts) {
						info.SerialNumber = parts[i+3]
					}
				}
			}
		}
	}

	return info
}

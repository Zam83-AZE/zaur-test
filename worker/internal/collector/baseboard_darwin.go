//go:build darwin

package collector

import (
	"os/exec"
	"strings"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectBaseboard() models.BaseboardInfo {
	info := models.BaseboardInfo{}

	if output, err := exec.Command("system_profiler", "SPHardwareDataType").Output(); err == nil {
		text := string(output)
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Model Name:") {
				info.Product = strings.TrimSpace(strings.TrimPrefix(line, "Model Name:"))
			}
			if strings.HasPrefix(line, "Model Identifier:") {
				info.Version = strings.TrimSpace(strings.TrimPrefix(line, "Model Identifier:"))
			}
			if strings.HasPrefix(line, "Manufacturer:") {
				info.Manufacturer = strings.TrimSpace(strings.TrimPrefix(line, "Manufacturer:"))
			}
		}
	}

	return info
}

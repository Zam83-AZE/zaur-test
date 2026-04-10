//go:build darwin

package collector

import (
	"os/exec"
	"strings"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectGPU() []models.GPUInfo {
	var gpus []models.GPUInfo

	output, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		return gpus
	}

	text := string(output)
	var currentGPU models.GPUInfo

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Chipset Model:") {
			if currentGPU.Name != "" {
				gpus = append(gpus, currentGPU)
			}
			currentGPU = models.GPUInfo{
				Name: strings.TrimSpace(strings.TrimPrefix(line, "Chipset Model:")),
			}
		}
		if strings.HasPrefix(line, "Metal:") {
			currentGPU.DriverVersion = strings.TrimSpace(strings.TrimPrefix(line, "Metal:"))
		}
	}

	if currentGPU.Name != "" {
		gpus = append(gpus, currentGPU)
	}

	return gpus
}

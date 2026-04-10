//go:build windows

package collector

import (
	"golang.org/x/sys/windows/registry"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectGPU() []models.GPUInfo {
	var gpus []models.GPUInfo

	// Try enumerating GPU from registry
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}`,
		registry.READ|registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return gpus
	}
	defer k.Close()

	subkeys, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return gpus
	}

	for _, subkey := range subkeys {
		if subkey == "Properties" {
			continue
		}

		sk, err := registry.OpenKey(k, subkey, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		defer sk.Close()

		gpu := models.GPUInfo{}

		if desc, _, err := sk.GetStringValue("DriverDesc"); err == nil {
			gpu.Name = desc
		}
		if ver, _, err := sk.GetStringValue("DriverVersion"); err == nil {
			gpu.DriverVersion = ver
		}
		if mem, _, err := sk.GetIntegerValue("HardwareInformation.qwMemorySize"); err == nil {
			gpu.MemoryMB = int(mem / (1024 * 1024))
		}

		if gpu.Name != "" {
			gpus = append(gpus, gpu)
		}
	}

	return gpus
}

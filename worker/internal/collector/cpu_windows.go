//go:build windows

package collector

import (
	"runtime"
	"golang.org/x/sys/windows/registry"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectCPU() models.CPUInfo {
	info := models.CPUInfo{
		CoresLogical: runtime.NumCPU(),
	}

	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\CentralProcessor\0`,
		registry.QUERY_VALUE)
	if err != nil {
		return info
	}
	defer k.Close()

	if name, _, err := k.GetStringValue("ProcessorNameString"); err == nil {
		info.Model = name
	}
	if mhz, _, err := k.GetIntegerValue("~MHz"); err == nil {
		info.FrequencyMHz = int(mhz)
	}

	// Physical cores from environment or default to logical
	info.CoresPhysical = info.CoresLogical

	return info
}

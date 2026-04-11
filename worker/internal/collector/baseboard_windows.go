//go:build windows

package collector

import (
	"golang.org/x/sys/windows/registry"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectBaseboard() models.BaseboardInfo {
	info := models.BaseboardInfo{}

	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\SystemInformation`,
		registry.QUERY_VALUE)
	if err != nil {
		return models.BaseboardInfo{Error: "cannot read registry: " + err.Error()}
	}
	defer k.Close()

	if mfr, _, err := k.GetStringValue("SystemManufacturer"); err == nil {
		info.Manufacturer = mfr
	}
	if prod, _, err := k.GetStringValue("SystemProductName"); err == nil {
		info.Product = prod
	}
	if ver, _, err := k.GetStringValue("SystemVersion"); err == nil {
		info.Version = ver
	}

	return info
}

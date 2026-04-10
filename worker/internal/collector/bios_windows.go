//go:build windows

package collector

import (
	"golang.org/x/sys/windows/registry"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectBIOS() models.BIOSInfo {
	info := models.BIOSInfo{}

	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\SystemInformation`,
		registry.QUERY_VALUE)
	if err != nil {
		return models.BIOSInfo{Error: "cannot read registry: " + err.Error()}
	}
	defer k.Close()

	if vendor, _, err := k.GetStringValue("SystemManufacturer"); err == nil {
		info.Vendor = vendor
	}
	if name, _, err := k.GetStringValue("SystemProductName"); err == nil {
		info.Name = name
	}
	if ver, _, err := k.GetStringValue("SystemBIOSVersion"); err == nil {
		info.Version = ver
	}
	if serial, _, err := k.GetStringValue("SystemProductName"); err == nil {
		_ = serial
	}

	// BIOS serial from BIOS subkey
	k2, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\BIOS`,
		registry.QUERY_VALUE)
	if err == nil {
		defer k2.Close()
		if serial, _, err := k2.GetStringValue("BIOSVersion"); err == nil && info.Version == "" {
			info.Version = serial
		}
		if date, _, err := k2.GetStringValue("BIOSReleaseDate"); err == nil {
			info.ReleaseDate = date
		}
		if vendor, _, err := k2.GetStringValue("BIOSVendor"); err == nil && info.Vendor == "" {
			info.Vendor = vendor
		}
	}

	return info
}

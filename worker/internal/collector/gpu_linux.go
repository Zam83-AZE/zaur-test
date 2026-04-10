//go:build linux

package collector

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectGPU() []models.GPUInfo {
	var gpus []models.GPUInfo

	// Method 1: Use lspci for display controllers (most reliable)
	output, err := exec.Command("lspci", "-mm", "-d", "::0300").Output()
	if err == nil && len(output) > 0 {
		for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				gpu := models.GPUInfo{
					Name: strings.Trim(fields[5], "\""),
				}
				// Clean up: fields[4] is vendor, fields[5] is device name
				if len(fields) >= 5 {
					vendor := strings.Trim(fields[4], "\"")
					deviceName := strings.Trim(fields[5], "\"")
					// Combine vendor + device if device name doesn't include vendor
					if !strings.Contains(strings.ToLower(deviceName), strings.ToLower(vendor)) {
						gpu.Name = vendor + " " + deviceName
					} else {
						gpu.Name = deviceName
					}
				}
				gpus = append(gpus, gpu)
			}
		}
	}

	// Method 2: Also check VGA compatible controllers (class 0x0300 may miss some)
	vgaOutput, err := exec.Command("lspci", "-mm").Output()
	if err == nil {
		seen := make(map[string]bool)
		// Mark GPUs from method 1 as seen
		for _, gpu := range gpus {
			seen[strings.ToLower(gpu.Name)] = true
		}
		for _, line := range strings.Split(strings.TrimSpace(string(vgaOutput)), "\n") {
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				// VGA compatible controller
				if strings.Contains(line, "0300") {
					deviceName := strings.Trim(fields[5], "\"")
					if !seen[strings.ToLower(deviceName)] {
						seen[strings.ToLower(deviceName)] = true
						vendor := strings.Trim(fields[4], "\"")
						if !strings.Contains(strings.ToLower(deviceName), strings.ToLower(vendor)) {
							gpus = append(gpus, models.GPUInfo{Name: vendor + " " + deviceName})
						} else {
							gpus = append(gpus, models.GPUInfo{Name: deviceName})
						}
					}
				}
			}
		}
	}

	// Method 3: Fallback to /sys/class/drm if lspci not available
	if len(gpus) == 0 {
		cardPaths, _ := filepath.Glob("/sys/class/drm/card[0-9]")
		if cardPaths != nil {
			seen := make(map[string]bool)
			for _, cardPath := range cardPaths {
				devicePath := cardPath + "/device"
				gpu := models.GPUInfo{}

				if devLink, err := os.Readlink(devicePath); err == nil {
					pciDev := filepath.Base(devLink)
					pciPath := "/sys/bus/pci/devices/" + pciDev

					if vendorData, err := os.ReadFile(pciPath + "/vendor"); err == nil {
						vendorID := strings.TrimSpace(string(vendorData))
						if vendorID == "0x10de" {
							gpu.Name = "NVIDIA"
						} else if vendorID == "0x1002" {
							gpu.Name = "AMD"
						} else if vendorID == "0x8086" {
							gpu.Name = "Intel"
						}
					}
					if deviceData, err := os.ReadFile(pciPath + "/device"); err == nil {
						gpu.Name += " " + strings.TrimSpace(string(deviceData))
					}
				}

				if gpu.Name == "" {
					gpu.Name = "Unknown GPU"
				}

				if !seen[gpu.Name] {
					seen[gpu.Name] = true
					gpus = append(gpus, gpu)
				}
			}
		}
	}

	return gpus
}

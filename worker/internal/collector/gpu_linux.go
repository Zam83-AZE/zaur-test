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

	// Method 1: /sys/class/drm/card*/device
	cardPaths, _ := filepath.Glob("/sys/class/drm/card[0-9]")
	if cardPaths != nil {
		seen := make(map[string]bool)
		for _, cardPath := range cardPaths {
			devicePath := cardPath + "/device"

			gpu := models.GPUInfo{}

			// Get vendor name
			if vendorLink, err := os.Readlink(devicePath + "/vendor"); err == nil {
				vendorID := filepath.Base(vendorLink)
				if vendorID == "0x10de" {
					gpu.Name = "NVIDIA"
				} else if vendorID == "0x1002" {
					gpu.Name = "AMD"
				} else if vendorID == "0x8086" {
					gpu.Name = "Intel"
				}
			}

			// Get device name from label
			if nameData, err := os.ReadFile(filepath.Join(devicePath, "label")); err == nil {
				gpu.Name = strings.TrimSpace(string(nameData))
			}

			// Try reading from /sys/bus/pci/devices
			if gpu.Name == "" || gpu.Name == "NVIDIA" || gpu.Name == "AMD" || gpu.Name == "Intel" {
				if devLink, err := os.Readlink(devicePath); err == nil {
					deviceIDFile := filepath.Join("/sys/bus/pci/devices", filepath.Base(devLink), "device")
					if deviceData, err := os.ReadFile(deviceIDFile); err == nil {
						deviceID := strings.TrimSpace(string(deviceData))
						if gpu.Name != "" {
							gpu.Name = gpu.Name + " (" + deviceID + ")"
						} else {
							gpu.Name = "GPU " + deviceID
						}
					}
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

	// Method 2: Try lspci if no GPUs found via /sys
	if len(gpus) == 0 {
		cmd := exec.Command("lspci", "-mm", "-d", "::0300")
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
				if line == "" {
					continue
				}
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					gpu := models.GPUInfo{
						Name: strings.Trim(fields[4], "\""),
					}
					gpus = append(gpus, gpu)
				}
			}
		}
	}

	return gpus
}

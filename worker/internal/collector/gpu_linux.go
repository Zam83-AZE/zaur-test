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

        // Method 1: Use lspci human-readable output (most reliable for names)
        // Matches VGA compatible controllers (0300) and 3D controllers (0302)
        for _, classFilter := range []string{"::0300", "::0302"} {
                output, err := exec.Command("lspci", "-d", classFilter).Output()
                if err != nil {
                        continue
                }
                for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
                        if line == "" {
                                continue
                        }
                        // Format: "00:02.0 VGA compatible controller: Intel Corporation CoffeeLake-H GT2 [UHD Graphics 630] (rev 02)"
                        colonIdx := strings.Index(line, ": ")
                        if colonIdx < 0 {
                                continue
                        }
                        description := strings.TrimSpace(line[colonIdx+2:])
                        // Remove revision info at the end like "(rev 02)"
                        if parenIdx := strings.LastIndex(description, " ("); parenIdx > 0 {
                                description = description[:parenIdx]
                        }
                        description = strings.TrimSpace(description)

                        if description == "" {
                                continue
                        }

                        gpus = append(gpus, models.GPUInfo{Name: description})
                }
        }

        // Deduplicate (some systems list same GPU under both classes)
        seen := make(map[string]bool)
        var deduped []models.GPUInfo
        for _, gpu := range gpus {
                // Normalize for dedup: lowercase comparison
                key := strings.ToLower(gpu.Name)
                // Also check if one name contains the other
                isDup := false
                if seen[key] {
                        isDup = true
                } else {
                        for existing := range seen {
                                if strings.Contains(key, existing) || strings.Contains(existing, key) {
                                        isDup = true
                                        break
                                }
                        }
                }
                if !isDup {
                        seen[key] = true
                        deduped = append(deduped, gpu)
                }
        }
        gpus = deduped

        // Method 2: Fallback to /sys/class/drm if lspci not available
        if len(gpus) == 0 {
                cardPaths, _ := filepath.Glob("/sys/class/drm/card[0-9]")
                seenSys := make(map[string]bool)
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

                        if !seenSys[gpu.Name] {
                                seenSys[gpu.Name] = true
                                gpus = append(gpus, gpu)
                        }
                }
        }

        return gpus
}

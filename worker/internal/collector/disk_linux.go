//go:build linux

package collector

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectDisks() []models.DiskInfo {
	var disks []models.DiskInfo

	// Read mounted filesystems from /proc/mounts
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return disks
	}

	seen := make(map[string]bool)

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		device := fields[0]
		mountpoint := fields[1]
		fstype := fields[2]

		// Skip virtual/dev filesystems
		if strings.HasPrefix(device, "none") ||
			strings.HasPrefix(device, "tmpfs") ||
			strings.HasPrefix(device, "devtmpfs") ||
			strings.HasPrefix(device, "cgroup") ||
			strings.HasPrefix(device, "proc") ||
			strings.HasPrefix(device, "sys") ||
			strings.HasPrefix(device, "run") ||
			strings.HasPrefix(device, "udev") {
			continue
		}

		if seen[device] {
			continue
		}
		seen[device] = true

		disk := models.DiskInfo{
			Name:       device,
			Filesystem: fstype,
		}

		// Get size info via syscall.Statfs
		var stat syscall.Statfs_t
		if err := syscall.Statfs(mountpoint, &stat); err == nil {
			disk.SizeBytes = int64(stat.Blocks) * int64(stat.Bsize)
			disk.SizeGB = float64(disk.SizeBytes) / (1024 * 1024 * 1024)
			disk.FreeBytes = int64(stat.Bfree) * int64(stat.Bsize)
			disk.FreeGB = float64(disk.FreeBytes) / (1024 * 1024 * 1024)
		}

		// Get model and serial from /sys/block/
		if strings.HasPrefix(device, "/dev/") {
			devName := strings.TrimPrefix(device, "/dev/")
			// Handle partition names like sda1 -> sda
			for _, suffix := range []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"} {
				if strings.HasSuffix(devName, suffix) && len(devName) > len(suffix) {
					devName = devName[:len(devName)-len(suffix)]
					break
				}
			}
			// Also handle nvme0n1p1 -> nvme0n1
			if idx := strings.LastIndex(devName, "p"); idx > 0 {
				candidate := devName[:idx]
				if _, err := os.Stat("/sys/block/" + candidate); err == nil {
					devName = candidate
				}
			}

			blockPath := "/sys/block/" + devName
			if modelData, err := os.ReadFile(blockPath + "/device/model"); err == nil {
				disk.Model = strings.TrimSpace(string(modelData))
			}
			if serialData, err := os.ReadFile(filepath.Join(blockPath, "device", "serial")); err == nil {
				disk.SerialNumber = strings.TrimSpace(string(serialData))
			}

			// Determine type
			if strings.HasPrefix(devName, "nvme") || strings.HasPrefix(devName, "mmcblk") {
				disk.Type = "NVMe"
			} else if _, err := os.Stat(blockPath + "/device/rotational"); err == nil {
				if rotData, err := os.ReadFile(blockPath + "/device/rotational"); err == nil {
					if strings.TrimSpace(string(rotData)) == "1" {
						disk.Type = "HDD"
					} else {
						disk.Type = "SSD"
					}
				}
			}
		}

		disks = append(disks, disk)
	}

	return disks
}

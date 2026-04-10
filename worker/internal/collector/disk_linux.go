//go:build linux

package collector

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

// virtualFSTypes are filesystem types that are not real disk partitions
var virtualFSTypes = map[string]bool{
	"devpts":    true,
	"efivarfs":  true,
	"securityfs": true,
	"pstore":    true,
	"bpf":       true,
	"debugfs":   true,
	"hugetlbfs": true,
	"mqueue":    true,
	"tracefs":   true,
	"fusectl":   true,
	"configfs":  true,
	"binfmt_misc": true,
	"overlay":   true,
	"nsfs":      true,
	"proc":      true,
	"sysfs":     true,
	"tmpfs":     true,
	"devtmpfs":  true,
	"cgroup":    true,
	"cgroup2":   true,
	"none":      true,
}

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

		// Skip virtual filesystems by type
		if virtualFSTypes[fstype] {
			continue
		}

		// Skip fuse.* filesystems (gvfsd-fuse, portal, etc.)
		if strings.HasPrefix(fstype, "fuse.") {
			continue
		}

		// Skip virtual device names
		if strings.HasPrefix(device, "none") ||
			strings.HasPrefix(device, "tmpfs") ||
			strings.HasPrefix(device, "cgroup") ||
			strings.HasPrefix(device, "proc") ||
			strings.HasPrefix(device, "sys") ||
			strings.HasPrefix(device, "run") ||
			strings.HasPrefix(device, "udev") {
			continue
		}

		// Skip devices that don't start with /dev/
		if !strings.HasPrefix(device, "/dev/") {
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
		devName := strings.TrimPrefix(device, "/dev/")
		// Handle partition names like sda1 -> sda, nvme0n1p1 -> nvme0n1
		trimmed := trimPartitionSuffix(devName)
		if trimmed != devName {
			blockPath := "/sys/block/" + trimmed
			if modelData, err := os.ReadFile(blockPath + "/device/model"); err == nil {
				disk.Model = strings.TrimSpace(string(modelData))
			}
			if serialData, err := os.ReadFile(filepath.Join(blockPath, "device", "serial")); err == nil {
				disk.SerialNumber = strings.TrimSpace(string(serialData))
			}

			// Determine type
			if strings.HasPrefix(trimmed, "nvme") || strings.HasPrefix(trimmed, "mmcblk") {
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

func trimPartitionSuffix(devName string) string {
	// Handle nvme0n1p1 -> nvme0n1
	if idx := strings.LastIndex(devName, "p"); idx > 0 {
		candidate := devName[:idx]
		if _, err := os.Stat("/sys/block/" + candidate); err == nil {
			return candidate
		}
	}
	// Handle sda1 -> sda, mmcblk0p1 -> mmcblk0
	for i := len(devName) - 1; i >= 0; i-- {
		if devName[i] < '0' || devName[i] > '9' {
			candidate := devName[:i+1]
			if _, err := os.Stat("/sys/block/" + candidate); err == nil {
				return candidate
			}
			break
		}
	}
	return devName
}

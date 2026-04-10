//go:build darwin

package collector

import (
        "os/exec"
        "strconv"
        "strings"

        "github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectDisks() []models.DiskInfo {
        var disks []models.DiskInfo

        if output, err := exec.Command("df", "-k").Output(); err != nil {
                return disks
        }

        lines := strings.Split(strings.TrimSpace(string(output)), "\n")
        for _, line := range lines[1:] {
                fields := strings.Fields(line)
                if len(fields) < 9 {
                        continue
                }

                name := fields[0]
                // Skip virtual filesystems
                if strings.HasPrefix(name, "/dev/") == false {
                        continue
                }

                totalKB, _ := strconv.ParseInt(fields[1], 10, 64)
                availKB, _ := strconv.ParseInt(fields[3], 10, 64)

                disk := models.DiskInfo{
                        Name:       name,
                        Filesystem: fields[7],
                        SizeBytes:  totalKB * 1024,
                        SizeGB:     float64(totalKB*1024) / (1024 * 1024 * 1024),
                        FreeBytes:  availKB * 1024,
                        FreeGB:     float64(availKB*1024) / (1024 * 1024 * 1024),
                }

                disks = append(disks, disk)
        }

        return disks
}

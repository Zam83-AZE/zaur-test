//go:build linux

package collector

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func getDomain() string {
	cmd := exec.Command("hostname", "-f")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.TrimSpace(string(output)), ".")
	if len(parts) > 1 {
		return strings.Join(parts[1:], ".")
	}
	return ""
}

func CollectOS() models.OSInfo {
	info := models.OSInfo{Arch: runtime.GOARCH}

	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				info.Name = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			}
			if strings.HasPrefix(line, "VERSION_ID=") {
				info.Version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
			}
			if strings.HasPrefix(line, "BUILD_ID=") {
				info.Build = strings.Trim(strings.TrimPrefix(line, "BUILD_ID="), "\"")
			}
		}
	}

	if output, err := exec.Command("uname", "-r").Output(); err == nil {
		info.KernelVersion = strings.TrimSpace(string(output))
	}

	return info
}

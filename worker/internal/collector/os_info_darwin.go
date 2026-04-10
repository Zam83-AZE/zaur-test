//go:build darwin

package collector

import (
	"os/exec"
	"runtime"
	"strings"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func getDomain() string {
	cmd := exec.Command("scutil", "--get", "LocalHostName")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func CollectOS() models.OSInfo {
	info := models.OSInfo{Arch: runtime.GOARCH}

	if output, err := exec.Command("sw_vers", "-productName").Output(); err == nil {
		name := strings.TrimSpace(string(output))
		if ver, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
			info.Name = name + " " + strings.TrimSpace(string(ver))
			info.Version = strings.TrimSpace(string(ver))
		} else {
			info.Name = name
		}
	}
	if build, err := exec.Command("sw_vers", "-buildVersion").Output(); err == nil {
		info.Build = strings.TrimSpace(string(build))
	}
	if output, err := exec.Command("uname", "-r").Output(); err == nil {
		info.KernelVersion = strings.TrimSpace(string(output))
	}

	return info
}

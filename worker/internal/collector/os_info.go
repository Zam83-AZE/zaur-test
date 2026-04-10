//go:build !linux && !windows && !darwin

package collector

import (
	"runtime"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func getDomain() string { return "" }

func CollectOS() models.OSInfo {
	return models.OSInfo{Arch: runtime.GOARCH}
}

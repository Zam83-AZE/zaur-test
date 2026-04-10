//go:build windows

package collector

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

type memoryStatusEx struct {
	dwLength     uint32
	dwMemoryLoad uint32
	ullTotalPhys uint64
	ullAvailPhys uint64
	_            [6]uint64
}

func CollectMemory() models.MemoryInfo {
	info := models.MemoryInfo{}

	modkernel32 := windows.NewLazySystemDLL("kernel32.dll")
	procGlobalMem := modkernel32.NewProc("GlobalMemoryStatusEx")

	status := memoryStatusEx{dwLength: 64}
	_, _, err := procGlobalMem.Call(uintptr(unsafe.Pointer(&status)))
	if err != nil && err != windows.Errno(0) {
		return info
	}

	info.TotalBytes = int64(status.ullTotalPhys)
	info.TotalGB = float64(info.TotalBytes) / (1024 * 1024 * 1024)
	info.AvailableBytes = int64(status.ullAvailPhys)
	info.AvailableGB = float64(info.AvailableBytes) / (1024 * 1024 * 1024)
	info.UsedPercent = float64(status.dwMemoryLoad)

	return info
}

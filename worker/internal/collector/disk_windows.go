//go:build windows

package collector

import (
	"unsafe"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
	"golang.org/x/sys/windows"
)

func CollectDisks() []models.DiskInfo {
	var disks []models.DiskInfo

	modkernel32 := windows.NewLazySystemDLL("kernel32.dll")
	procGetDisk := modkernel32.NewProc("GetDiskFreeSpaceExW")

	driveLetters := []string{"C:", "D:", "E:", "F:", "G:", "H:", "I:", "J:", "K:", "L:", "M:", "N:", "O:", "P:", "Q:", "R:", "S:", "T:", "U:", "V:", "W:", "X:", "Y:", "Z:"}

	for _, letter := range driveLetters {
		path := letter + `\`
		var freeBytesAvailable, totalBytes, totalFreeBytes uint64

		ret, _, _ := procGetDisk.Call(
			uintptr(unsafe.Pointer(windows.StringToUTF16Ptr(path))),
			uintptr(unsafe.Pointer(&freeBytesAvailable)),
			uintptr(unsafe.Pointer(&totalBytes)),
			uintptr(unsafe.Pointer(&totalFreeBytes)),
		)

		if ret == 0 {
			continue
		}

		disk := models.DiskInfo{
			Name:       letter,
			SizeBytes:  int64(totalBytes),
			SizeGB:     float64(totalBytes) / (1024 * 1024 * 1024),
			FreeBytes:  int64(totalFreeBytes),
			FreeGB:     float64(totalFreeBytes) / (1024 * 1024 * 1024),
			Filesystem: "NTFS",
		}

		disks = append(disks, disk)
	}

	return disks
}

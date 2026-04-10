package models

type SystemInfo struct {
	Version     string             `json:"version"`
	Hostname    string             `json:"hostname"`
	Domain      string             `json:"domain"`
	CollectedAt string             `json:"collected_at"`
	OS          OSInfo             `json:"os"`
	BIOS        BIOSInfo           `json:"bios"`
	Baseboard   BaseboardInfo      `json:"baseboard"`
	CPU         CPUInfo            `json:"cpu"`
	Memory      MemoryInfo         `json:"memory"`
	Disks       []DiskInfo         `json:"disks"`
	Network     []NetworkInterface `json:"network"`
	GPU         []GPUInfo          `json:"gpu"`
	CurrentUser UserInfo           `json:"current_user"`
}

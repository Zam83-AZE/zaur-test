package models

type GPUInfo struct {
	Name          string `json:"name,omitempty"`
	DriverVersion string `json:"driver_version,omitempty"`
	MemoryMB      int    `json:"memory_mb,omitempty"`
	SerialNumber  string `json:"serial_number,omitempty"`
}

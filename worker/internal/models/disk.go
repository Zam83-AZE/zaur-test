package models

type DiskInfo struct {
	Name         string  `json:"name,omitempty"`
	Model        string  `json:"model,omitempty"`
	SerialNumber string  `json:"serial_number,omitempty"`
	Type         string  `json:"type,omitempty"`
	SizeBytes    int64   `json:"size_bytes"`
	SizeGB       float64 `json:"size_gb"`
	FreeBytes    int64   `json:"free_bytes"`
	FreeGB       float64 `json:"free_gb"`
	Filesystem   string  `json:"filesystem,omitempty"`
}

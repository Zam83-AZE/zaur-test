package models

type BIOSInfo struct {
	Vendor       string `json:"vendor,omitempty"`
	Name         string `json:"name,omitempty"`
	Version      string `json:"version,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	ReleaseDate  string `json:"release_date,omitempty"`
	Error        string `json:"error,omitempty"`
}

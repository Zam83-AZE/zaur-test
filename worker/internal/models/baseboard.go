package models

type BaseboardInfo struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	Product      string `json:"product,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	Version      string `json:"version,omitempty"`
	Error        string `json:"error,omitempty"`
}

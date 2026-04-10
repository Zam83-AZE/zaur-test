package models

type MemoryInfo struct {
	TotalBytes     int64   `json:"total_bytes"`
	TotalGB        float64 `json:"total_gb"`
	AvailableBytes int64   `json:"available_bytes"`
	AvailableGB    float64 `json:"available_gb"`
	UsedPercent    float64 `json:"used_percent"`
}

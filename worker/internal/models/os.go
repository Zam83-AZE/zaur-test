package models

type OSInfo struct {
	Name          string `json:"name,omitempty"`
	Version       string `json:"version,omitempty"`
	Build         string `json:"build,omitempty"`
	Arch          string `json:"arch,omitempty"`
	KernelVersion string `json:"kernel_version,omitempty"`
}

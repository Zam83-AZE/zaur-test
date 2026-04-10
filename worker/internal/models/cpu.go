package models

type CPUInfo struct {
	Model         string `json:"model,omitempty"`
	CoresPhysical int    `json:"cores_physical,omitempty"`
	CoresLogical  int    `json:"cores_logical,omitempty"`
	FrequencyMHz  int    `json:"frequency_mhz,omitempty"`
	Socket        string `json:"socket,omitempty"`
}

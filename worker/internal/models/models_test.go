package models

import (
	"encoding/json"
	"testing"
)

func CreateTestSystemInfo() *SystemInfo {
	return &SystemInfo{
		Version:     "1.0.0",
		Hostname:    "DESKTOP-TEST01",
		Domain:      "WORKGROUP",
		CollectedAt: "2026-04-10T14:30:00+04:00",
		OS: OSInfo{
			Name:          "Windows 11 Pro",
			Version:       "10.0.22631",
			Build:         "22631",
			Arch:          "amd64",
			KernelVersion: "",
		},
		BIOS: BIOSInfo{
			Vendor:       "Dell Inc.",
			Name:         "BIOS Date: 01/15/2024",
			Version:      "2.8.1",
			SerialNumber: "ABCDEF123456",
			ReleaseDate:  "2024-01-15",
		},
		Baseboard: BaseboardInfo{
			Manufacturer: "Dell Inc.",
			Product:      "OptiPlex 7090",
			SerialNumber: "GHIJKL789012",
			Version:      "A00",
		},
		CPU: CPUInfo{
			Model:         "Intel(R) Core(TM) i7-12700K",
			CoresPhysical: 12,
			CoresLogical:  20,
			FrequencyMHz:  3600,
			Socket:        "LGA1700",
		},
		Memory: MemoryInfo{
			TotalBytes:     34359738368,
			TotalGB:        32.0,
			AvailableBytes: 19818094592,
			AvailableGB:    18.46,
			UsedPercent:    42.3,
		},
		Disks: []DiskInfo{
			{
				Name:         "C:",
				Model:        "Samsung SSD 970 EVO Plus 500GB",
				SerialNumber: "S6Z2NF0X123456",
				Type:         "SSD",
				SizeBytes:    512110190592,
				SizeGB:       476.94,
				FreeBytes:    251258376192,
				FreeGB:       234.0,
				Filesystem:   "NTFS",
			},
		},
		Network: []NetworkInterface{
			{
				Name:          "Ethernet0",
				MACAddress:    "AA:BB:CC:DD:EE:FF",
				IPAddresses:   []string{"192.168.1.100"},
				Gateway:       "192.168.1.1",
				DNSServers:    []string{"8.8.8.8", "8.8.4.4"},
				InterfaceType: "Ethernet",
			},
		},
		GPU: []GPUInfo{
			{
				Name:          "NVIDIA GeForce RTX 4070",
				DriverVersion: "551.86",
				MemoryMB:      12288,
			},
		},
		CurrentUser: UserInfo{
			Username: "testuser",
			Domain:   "WORKGROUP",
			SID:      "S-1-5-21-TEST",
		},
	}
}

func TestSystemInfoSerialization(t *testing.T) {
	info := CreateTestSystemInfo()
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded SystemInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Hostname != info.Hostname {
		t.Errorf("Hostname: got %q, want %q", decoded.Hostname, info.Hostname)
	}
	if decoded.BIOS.SerialNumber != "ABCDEF123456" {
		t.Errorf("BIOS SerialNumber: got %q, want %q", decoded.BIOS.SerialNumber, "ABCDEF123456")
	}
	if decoded.Baseboard.SerialNumber != "GHIJKL789012" {
		t.Errorf("Baseboard SerialNumber: got %q", decoded.Baseboard.SerialNumber)
	}
	if len(decoded.Disks) != 1 || decoded.Disks[0].SerialNumber != "S6Z2NF0X123456" {
		t.Errorf("Disk SerialNumber mismatch")
	}
	if len(decoded.Network) != 1 || decoded.Network[0].MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("Network MAC mismatch")
	}
	if decoded.Memory.TotalGB != 32.0 {
		t.Errorf("Memory TotalGB: got %f, want 32.0", decoded.Memory.TotalGB)
	}
}

func TestNetworkInterface(t *testing.T) {
	nic := NetworkInterface{
		Name:        "eth0",
		MACAddress:  "AA:BB:CC:DD:EE:FF",
		IPAddresses: []string{"10.0.0.1"},
		DNSServers:  []string{"8.8.8.8"},
	}
	data, _ := json.Marshal(nic)
	var decoded NetworkInterface
	json.Unmarshal(data, &decoded)
	if decoded.MACAddress != nic.MACAddress {
		t.Errorf("MAC mismatch: got %q", decoded.MACAddress)
	}
	if len(decoded.DNSServers) != 1 {
		t.Errorf("DNS count: got %d", len(decoded.DNSServers))
	}
}

func TestGPUInfoEmptySerial(t *testing.T) {
	gpu := GPUInfo{Name: "Test GPU", SerialNumber: ""}
	data, _ := json.Marshal(gpu)
	s := string(data)
	t.Logf("GPU JSON: %s", s)
	// Empty serial should be omitted due to omitempty
}

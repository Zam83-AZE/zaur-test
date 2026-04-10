package models

type NetworkInterface struct {
	Name          string   `json:"name,omitempty"`
	MACAddress    string   `json:"mac_address,omitempty"`
	IPAddresses   []string `json:"ip_addresses,omitempty"`
	Gateway       string   `json:"gateway,omitempty"`
	DNSServers    []string `json:"dns_servers,omitempty"`
	InterfaceType string   `json:"interface_type,omitempty"`
}

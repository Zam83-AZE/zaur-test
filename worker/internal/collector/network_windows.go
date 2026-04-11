//go:build windows

package collector

import (
	"net"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectNetwork() []models.NetworkInterface {
	var interfaces []models.NetworkInterface

	ifaces, err := net.Interfaces()
	if err != nil {
		return interfaces
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if len(iface.HardwareAddr) == 0 {
			continue
		}

		ni := models.NetworkInterface{
			Name:       iface.Name,
			MACAddress: iface.HardwareAddr.String(),
		}

		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok {
					if !ipnet.IP.IsLoopback() && !ipnet.IP.IsLinkLocalUnicast() {
						ni.IPAddresses = append(ni.IPAddresses, ipnet.IP.String())
					}
				}
			}
		}

		ni.InterfaceType = "Ethernet"

		if len(ni.IPAddresses) > 0 {
			interfaces = append(interfaces, ni)
		}
	}

	return interfaces
}

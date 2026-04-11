//go:build darwin

package collector

import (
	"net"
	"strings"

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

		// macOS interface naming
		if strings.HasPrefix(iface.Name, "en") {
			ni.InterfaceType = "Ethernet"
		} else if strings.HasPrefix(iface.Name, "wi") || strings.HasPrefix(iface.Name, "awdl") {
			ni.InterfaceType = "WiFi"
		}

		if len(ni.IPAddresses) > 0 || ni.MACAddress != "" {
			interfaces = append(interfaces, ni)
		}
	}

	return interfaces
}

//go:build linux

package collector

import (
	"net"
	"os"
	"path/filepath"
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
		// Skip loopback
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		ni := models.NetworkInterface{
			Name:       iface.Name,
			MACAddress: iface.HardwareAddr.String(),
		}

		// Get IP addresses
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

		// Determine interface type
		ni.InterfaceType = "Unknown"
		driverPath := "/sys/class/net/" + iface.Name + "/device/driver"
		if link, err := os.Readlink(driverPath); err == nil {
			driverName := filepath.Base(link)
			if strings.Contains(strings.ToLower(driverName), "wireless") ||
				strings.Contains(strings.ToLower(driverName), "wifi") {
				ni.InterfaceType = "WiFi"
			} else if strings.Contains(strings.ToLower(driverName), "eth") ||
				strings.Contains(strings.ToLower(driverName), "r8169") ||
				strings.Contains(strings.ToLower(driverName), "e1000") {
				ni.InterfaceType = "Ethernet"
			} else {
				ni.InterfaceType = "Ethernet"
			}
		}

		// Try to get wifi type from wireless symlink
		if _, err := os.Stat("/sys/class/net/" + iface.Name + "/wireless"); err == nil {
			ni.InterfaceType = "WiFi"
		}

		// Skip interfaces with no IPs
		if len(ni.IPAddresses) > 0 || ni.MACAddress != "" {
			interfaces = append(interfaces, ni)
		}
	}

	return interfaces
}

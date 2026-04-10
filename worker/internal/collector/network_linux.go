//go:build linux

package collector

import (
        "net"
        "os"
        "strings"

        "github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

// Docker/container interface patterns
var virtualPrefixes = []string{
        "br-",    // Docker bridge
        "veth",   // Docker veth
        "docker", // Docker
        "virbr",  // Virtual bridge (libvirt)
        "vnet",   // libvirt
}

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

                // Check for WiFi
                if _, err := os.Stat("/sys/class/net/" + iface.Name + "/wireless"); err == nil {
                        ni.InterfaceType = "WiFi"
                } else {
                        // Check for virtual/container interfaces
                        isVirtual := false
                        for _, prefix := range virtualPrefixes {
                                if strings.HasPrefix(iface.Name, prefix) {
                                        isVirtual = true
                                        break
                                }
                        }
                        if isVirtual {
                                ni.InterfaceType = "Virtual"
                        } else {
                                ni.InterfaceType = "Ethernet"
                        }
                }

                // Skip interfaces with no useful info
                if len(ni.IPAddresses) > 0 || ni.MACAddress != "" {
                        interfaces = append(interfaces, ni)
                }
        }

        return interfaces
}

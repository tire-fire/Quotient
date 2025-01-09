package ipmanager

import (
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"strings"
	"time"
)

// GetCurrentIP retrieves the current IP address of the specified interface.
func GetCurrentIP(interfaceName string) (string, error) {
	cmd := exec.Command("ip", "addr", "show", interfaceName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "inet") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				ip := strings.Split(fields[1], "/")[0]
				return ip, nil
			}
		}
	}
	return "", fmt.Errorf("no IP address found on interface %s", interfaceName)
}

// GenerateRandomIP creates a random IP address within the given subnet.
func GenerateRandomIP(subnet string) string {
	_, ipNet, _ := net.ParseCIDR(subnet)
	baseIP := ipNet.IP.To4()
	mask := ipNet.Mask

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < len(baseIP); i++ {
		baseIP[i] |= byte(rand.Intn(256)) & ^mask[i]
	}
	return baseIP.String()
}

// ResolveHostname resolves a hostname to an IP address.
func ResolveHostname(hostname string) (string, error) {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return "", err
	}
	return addrs[0], nil
}


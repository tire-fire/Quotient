package ipmanager

import (
	"fmt"
	"os/exec"
)

// AddRoutingRule adds a custom routing rule for the container.
func AddRoutingRule(ip, gateway string, tableID int) error {
	cmdRule := exec.Command("ip", "rule", "add", "from", ip, "table", fmt.Sprint(tableID))
	if output, err := cmdRule.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %v", output, err)
	}

	cmdRoute := exec.Command("ip", "route", "add", "default", "via", gateway, "table", fmt.Sprint(tableID))
	if output, err := cmdRoute.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %v", output, err)
	}
	return nil
}

// SetIP adds a new IP and updates the default route.
func SetIP(ip, gateway, interfaceName string, tableID int) error {
	cmdAddIP := exec.Command("ip", "addr", "add", fmt.Sprintf("%s/24", ip), "dev", interfaceName)
	if output, err := cmdAddIP.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %v", output, err)
	}

	cmdSetRoute := exec.Command("ip", "route", "replace", "default", "via", gateway, "src", ip, "dev", interfaceName, "table", fmt.Sprint(tableID))
	if output, err := cmdSetRoute.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %v", output, err)
	}
	return nil
}

// RemoveIP removes an old IP address.
func RemoveIP(ip, interfaceName string) error {
	cmd := exec.Command("ip", "addr", "del", fmt.Sprintf("%s/24", ip), "dev", interfaceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %v", output, err)
	}
	return nil
}

// AddStaticRoute adds a static route for internal services.
func AddStaticRoute(targetIP, interfaceName string) error {
	cmd := exec.Command("ip", "route", "add", targetIP, "dev", interfaceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %v", output, err)
	}
	return nil
}


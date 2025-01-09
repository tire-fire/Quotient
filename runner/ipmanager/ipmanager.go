package ipmanager

import (
	"fmt"
	"log"
	"os"
)

const (
	interfaceName = "eth0"
)

var (
	gateway string
	subnet string
	redisAddr string
)

func init() {
	// Set defaults
	subnet = os.Getenv("SUBNET")
	if subnet == "" {
		subnet = "192.168.1.0/24" // Default subnet
	}

	gateway = os.Getenv("GATEWAY")
	if gateway == "" {
		gateway = "192.168.1.1" // Default gateway
	}

	log.Printf("Using subnet: %s, gateway: %s", subnet, gateway)
}

// SetupRouting initializes routing for the container.
func SetupRouting() error {
	// Get the container's dynamically assigned IP
	containerIP, err := GetCurrentIP(interfaceName)
	if err != nil {
		return fmt.Errorf("failed to get container IP: %v", err)
	}
	log.Printf("Container IP: %s\n", containerIP)

	// Configure the custom routing table
	err = AddRoutingRule(containerIP, gateway, 100)
	if err != nil {
		return fmt.Errorf("failed to set up routing table: %v", err)
	}

	return nil
}

// RotateIP handles the rotation of the container's IP.
func RotateIP() error {
	currentIP, err := GetCurrentIP(interfaceName)
	if err != nil {
		return fmt.Errorf("failed to get current IP: %v", err)
	}

	newIP := GenerateRandomIP(subnet)
	log.Printf("Rotating IP: current=%s, new=%s\n", currentIP, newIP)

	err = SetIP(newIP, gateway, interfaceName, 100)
	if err != nil {
		return fmt.Errorf("failed to set new IP: %v", err)
	}

	err = RemoveIP(currentIP, interfaceName)
	if err != nil {
		return fmt.Errorf("failed to remove old IP %s: %v", currentIP, err)
	}

	return nil
}


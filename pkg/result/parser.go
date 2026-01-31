package result

import (
	"fmt"
	"net"

	"github.com/containernetworking/cni/pkg/types"
	types040 "github.com/containernetworking/cni/pkg/types/040"
	types100 "github.com/containernetworking/cni/pkg/types/100"
)

// ExtractPodIP extracts the first IPv4 address from a CNI Result
// Supports both CNI 0.4.0 and CNI 1.0.0 result formats
//
// Parameters:
//   - result: CNI Result interface (can be types100.Result or types040.Result)
//
// Returns:
//   - string: IPv4 address as a plain string (e.g., "10.200.1.5")
//   - error: Non-nil if result is nil, unsupported type, or contains no IPv4 addresses
//
// The function skips IPv6 addresses and returns only the first IPv4 address found
func ExtractPodIP(result types.Result) (string, error) {
	if result == nil {
		return "", fmt.Errorf("CNI result is nil")
	}

	// Try types100.Result first (CNI 1.0.0 format)
	if r100, ok := result.(*types100.Result); ok {
		return extractIPv4FromResult100(r100)
	}

	// Fallback to types040.Result (CNI 0.4.0 format)
	if r040, ok := result.(*types040.Result); ok {
		return extractIPv4FromResult040(r040)
	}

	// Unsupported result type
	return "", fmt.Errorf("unsupported CNI result type: %T", result)
}

// extractIPv4FromResult100 extracts IPv4 from CNI 1.0.0 Result
func extractIPv4FromResult100(result *types100.Result) (string, error) {
	if len(result.IPs) == 0 {
		return "", fmt.Errorf("CNI result contains no IP addresses")
	}

	// Iterate through IPs array, return first IPv4
	for _, ipConfig := range result.IPs {
		if ipConfig.Address.IP == nil {
			continue
		}

		// Check if IP is IPv4 (IP.To4() returns nil for IPv6)
		if ipConfig.Address.IP.To4() != nil {
			return ipConfig.Address.IP.String(), nil
		}
	}

	return "", fmt.Errorf("CNI result contains no IPv4 addresses (only IPv6)")
}

// extractIPv4FromResult040 extracts IPv4 from CNI 0.4.0 Result
func extractIPv4FromResult040(result *types040.Result) (string, error) {
	if len(result.IPs) == 0 {
		return "", fmt.Errorf("CNI result contains no IP addresses")
	}

	// Iterate through IPs array, return first IPv4
	for _, ipConfig := range result.IPs {
		if ipConfig.Address.IP == nil {
			continue
		}

		// Check if IP is IPv4
		if ipConfig.Address.IP.To4() != nil {
			return ipConfig.Address.IP.String(), nil
		}
	}

	return "", fmt.Errorf("CNI result contains no IPv4 addresses (only IPv6)")
}

// IsIPv4 checks if the given IP address is IPv4
// Helper function for validation or filtering
func IsIPv4(ip net.IP) bool {
	return ip != nil && ip.To4() != nil
}

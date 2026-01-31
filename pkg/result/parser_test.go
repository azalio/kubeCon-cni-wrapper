package result

import (
	"net"
	"strings"
	"testing"

	"github.com/containernetworking/cni/pkg/types"
	types040 "github.com/containernetworking/cni/pkg/types/040"
	types100 "github.com/containernetworking/cni/pkg/types/100"
)

// TestExtractPodIP_IPv4Only verifies extraction of IPv4 from CNI 1.0.0 Result
func TestExtractPodIP_IPv4Only(t *testing.T) {
	// Create CNI 1.0.0 Result with single IPv4 address
	result := &types100.Result{
		CNIVersion: "1.0.0",
		IPs: []*types100.IPConfig{
			{
				Address: net.IPNet{
					IP:   net.ParseIP("10.200.1.5"),
					Mask: net.CIDRMask(24, 32),
				},
			},
		},
	}

	ip, err := ExtractPodIP(result)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if ip != "10.200.1.5" {
		t.Errorf("Expected IP 10.200.1.5, got: %s", ip)
	}
}

// TestExtractPodIP_IPv6Only verifies error when Result contains only IPv6
func TestExtractPodIP_IPv6Only(t *testing.T) {
	// Create CNI 1.0.0 Result with only IPv6 address
	result := &types100.Result{
		CNIVersion: "1.0.0",
		IPs: []*types100.IPConfig{
			{
				Address: net.IPNet{
					IP:   net.ParseIP("2001:db8::1"),
					Mask: net.CIDRMask(64, 128),
				},
			},
		},
	}

	_, err := ExtractPodIP(result)
	if err == nil {
		t.Fatal("Expected error when Result contains only IPv6")
	}

	if !strings.Contains(err.Error(), "no IPv4 addresses") {
		t.Errorf("Expected 'no IPv4 addresses' error, got: %v", err)
	}
}

// TestExtractPodIP_MixedIPv4IPv6 verifies IPv4 is returned from mixed addresses
func TestExtractPodIP_MixedIPv4IPv6(t *testing.T) {
	// Create CNI 1.0.0 Result with IPv6 first, IPv4 second
	// Should skip IPv6 and return IPv4
	result := &types100.Result{
		CNIVersion: "1.0.0",
		IPs: []*types100.IPConfig{
			{
				Address: net.IPNet{
					IP:   net.ParseIP("2001:db8::1"),
					Mask: net.CIDRMask(64, 128),
				},
			},
			{
				Address: net.IPNet{
					IP:   net.ParseIP("10.200.2.10"),
					Mask: net.CIDRMask(24, 32),
				},
			},
		},
	}

	ip, err := ExtractPodIP(result)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if ip != "10.200.2.10" {
		t.Errorf("Expected IPv4 10.200.2.10, got: %s", ip)
	}
}

// TestExtractPodIP_MultipleIPv4 verifies first IPv4 is returned
func TestExtractPodIP_MultipleIPv4(t *testing.T) {
	// Create CNI 1.0.0 Result with multiple IPv4 addresses
	// Should return first one
	result := &types100.Result{
		CNIVersion: "1.0.0",
		IPs: []*types100.IPConfig{
			{
				Address: net.IPNet{
					IP:   net.ParseIP("10.200.1.5"),
					Mask: net.CIDRMask(24, 32),
				},
			},
			{
				Address: net.IPNet{
					IP:   net.ParseIP("192.168.1.10"),
					Mask: net.CIDRMask(24, 32),
				},
			},
		},
	}

	ip, err := ExtractPodIP(result)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if ip != "10.200.1.5" {
		t.Errorf("Expected first IPv4 10.200.1.5, got: %s", ip)
	}
}

// TestExtractPodIP_EmptyIPs verifies error when IPs array is empty
func TestExtractPodIP_EmptyIPs(t *testing.T) {
	// Create CNI 1.0.0 Result with empty IPs array
	result := &types100.Result{
		CNIVersion: "1.0.0",
		IPs:        []*types100.IPConfig{},
	}

	_, err := ExtractPodIP(result)
	if err == nil {
		t.Fatal("Expected error when IPs array is empty")
	}

	if !strings.Contains(err.Error(), "no IP addresses") {
		t.Errorf("Expected 'no IP addresses' error, got: %v", err)
	}
}

// TestExtractPodIP_NilResult verifies error when Result is nil
func TestExtractPodIP_NilResult(t *testing.T) {
	var result types.Result = nil

	_, err := ExtractPodIP(result)
	if err == nil {
		t.Fatal("Expected error when Result is nil")
	}

	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("Expected 'nil' error, got: %v", err)
	}
}

// TestExtractPodIP_CNI040Format verifies CNI 0.4.0 Result support
func TestExtractPodIP_CNI040Format(t *testing.T) {
	// Create CNI 0.4.0 Result with IPv4 address
	result := &types040.Result{
		CNIVersion: "0.4.0",
		IPs: []*types040.IPConfig{
			{
				Address: net.IPNet{
					IP:   net.ParseIP("10.100.5.20"),
					Mask: net.CIDRMask(24, 32),
				},
			},
		},
	}

	ip, err := ExtractPodIP(result)
	if err != nil {
		t.Fatalf("Expected success for CNI 0.4.0 Result, got error: %v", err)
	}

	if ip != "10.100.5.20" {
		t.Errorf("Expected IP 10.100.5.20, got: %s", ip)
	}
}

// TestExtractPodIP_CNI040IPv6Only verifies error for CNI 0.4.0 Result with only IPv6
func TestExtractPodIP_CNI040IPv6Only(t *testing.T) {
	// Create CNI 0.4.0 Result with only IPv6
	result := &types040.Result{
		CNIVersion: "0.4.0",
		IPs: []*types040.IPConfig{
			{
				Address: net.IPNet{
					IP:   net.ParseIP("fd00::1"),
					Mask: net.CIDRMask(64, 128),
				},
			},
		},
	}

	_, err := ExtractPodIP(result)
	if err == nil {
		t.Fatal("Expected error when CNI 0.4.0 Result contains only IPv6")
	}

	if !strings.Contains(err.Error(), "no IPv4 addresses") {
		t.Errorf("Expected 'no IPv4 addresses' error, got: %v", err)
	}
}

// TestExtractPodIP_NilIPInConfig verifies handling of nil IP in IPConfig
func TestExtractPodIP_NilIPInConfig(t *testing.T) {
	// Create CNI 1.0.0 Result with nil IP, followed by valid IPv4
	result := &types100.Result{
		CNIVersion: "1.0.0",
		IPs: []*types100.IPConfig{
			{
				Address: net.IPNet{
					IP:   nil, // Invalid entry
					Mask: net.CIDRMask(24, 32),
				},
			},
			{
				Address: net.IPNet{
					IP:   net.ParseIP("10.200.3.15"),
					Mask: net.CIDRMask(24, 32),
				},
			},
		},
	}

	ip, err := ExtractPodIP(result)
	if err != nil {
		t.Fatalf("Expected success (should skip nil IP), got error: %v", err)
	}

	if ip != "10.200.3.15" {
		t.Errorf("Expected IP 10.200.3.15 (second entry), got: %s", ip)
	}
}

// TestIsIPv4_Valid verifies IsIPv4 helper with valid IPv4
func TestIsIPv4_Valid(t *testing.T) {
	ip := net.ParseIP("192.168.1.1")
	if !IsIPv4(ip) {
		t.Error("Expected IsIPv4 to return true for 192.168.1.1")
	}
}

// TestIsIPv4_IPv6 verifies IsIPv4 helper with IPv6
func TestIsIPv4_IPv6(t *testing.T) {
	ip := net.ParseIP("2001:db8::1")
	if IsIPv4(ip) {
		t.Error("Expected IsIPv4 to return false for IPv6 address")
	}
}

// TestIsIPv4_Nil verifies IsIPv4 helper with nil IP
func TestIsIPv4_Nil(t *testing.T) {
	var ip net.IP = nil
	if IsIPv4(ip) {
		t.Error("Expected IsIPv4 to return false for nil IP")
	}
}

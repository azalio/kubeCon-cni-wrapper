package result_test

import (
	"fmt"
	"net"

	"github.com/azalio/kubeCon-cni-wrapper/pkg/result"
	types100 "github.com/containernetworking/cni/pkg/types/100"
)

// ExampleExtractPodIP demonstrates extracting IPv4 from CNI Result
func ExampleExtractPodIP() {
	// Simulate CNI Result returned by delegate plugin
	cniResult := &types100.Result{
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

	// Extract pod IP for use in routing/fwmark rules
	podIP, err := result.ExtractPodIP(cniResult)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Pod IP: %s\n", podIP)
	// Output: Pod IP: 10.200.1.5
}

// ExampleExtractPodIP_mixedAddresses demonstrates IPv4 extraction from mixed IPv4/IPv6
func ExampleExtractPodIP_mixedAddresses() {
	// CNI Result with both IPv6 and IPv4 addresses
	cniResult := &types100.Result{
		CNIVersion: "1.0.0",
		IPs: []*types100.IPConfig{
			{
				Address: net.IPNet{
					IP:   net.ParseIP("2001:db8::1"), // IPv6 - will be skipped
					Mask: net.CIDRMask(64, 128),
				},
			},
			{
				Address: net.IPNet{
					IP:   net.ParseIP("10.200.2.10"), // IPv4 - will be returned
					Mask: net.CIDRMask(24, 32),
				},
			},
		},
	}

	podIP, err := result.ExtractPodIP(cniResult)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Pod IPv4: %s\n", podIP)
	// Output: Pod IPv4: 10.200.2.10
}

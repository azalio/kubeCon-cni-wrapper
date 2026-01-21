package iptables_test

import (
	"fmt"

	"github.com/azalio/kubeCon-cni-wrapper/pkg/iptables"
)

// ExampleAddMarkRule_invalidFwmark demonstrates fwmark validation
func ExampleAddMarkRule_invalidFwmark() {
	// Attempt to use invalid fwmark (prevents Cilium conflicts)
	err := iptables.AddMarkRule("10.200.1.5", "0x99")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	// Output: Error: invalid fwmark "0x99": must be 0x10 (Tenant A) or 0x20 (Tenant B) to avoid Cilium conflicts
}

// ExampleAddMarkRule_emptyIP demonstrates IP validation
func ExampleAddMarkRule_emptyIP() {
	// Attempt to add rule with empty IP
	err := iptables.AddMarkRule("", "0x10")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	// Output: Error: podIP cannot be empty
}

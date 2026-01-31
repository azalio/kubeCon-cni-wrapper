package delegate_test

import (
	"encoding/json"
	"fmt"
)

// Example_delegationUsage demonstrates the typical pattern for CNI delegation
// This is a documentation example that doesn't execute (would require actual CNI plugins)
func Example_delegationUsage() {
	// In production CNI wrapper plugin, delegation follows this pattern:
	//
	// 1. Parse wrapper configuration to extract delegate config
	_ = json.RawMessage(`{
		"type": "ptp",
		"cniVersion": "1.0.0",
		"ipam": {
			"type": "host-local",
			"subnet": "10.200.0.0/16"
		}
	}`)

	// 2. Before delegation, wrapper performs its work:
	//    - Read pod annotations from Kubernetes API
	//    - Set fwmark based on tenant annotation
	//    - Configure policy routing rules
	fmt.Println("Step 1: Wrapper configures fwmark and routing")

	// 3. Delegate to next CNI plugin (in real code):
	//    result, err := delegate.DelegateAdd(delegateConfig, stdin)
	//    This executes the ptp plugin which:
	//    - Creates veth pair
	//    - Calls IPAM plugin (host-local) to allocate IP
	//    - Returns CNI Result with interface and IP configuration
	fmt.Println("Step 2: Delegate to ptp plugin for interface creation")

	// 4. Wrapper returns the delegate's result unchanged
	//    The CNI contract requires wrapper to pass through the Result
	fmt.Println("Step 3: Return delegate result to container runtime")

	// Output:
	// Step 1: Wrapper configures fwmark and routing
	// Step 2: Delegate to ptp plugin for interface creation
	// Step 3: Return delegate result to container runtime
}

// Example_errorHandling demonstrates error handling patterns in CNI delegation
func Example_errorHandling() {
	// CNI plugins must handle errors from delegate plugins carefully:
	//
	// 1. Plugin not found in CNI_PATH
	fmt.Println("Error case 1: Plugin not found - check CNI_PATH")

	// 2. Delegate plugin fails (returns non-zero exit code)
	//    The error message from delegate should be preserved
	fmt.Println("Error case 2: Delegate fails - preserve error message")

	// 3. Timeout (plugin doesn't respond within ExecutionTimeout)
	fmt.Println("Error case 3: Timeout - abort after 30 seconds")

	// 4. Invalid JSON from delegate
	//    This is a critical error - delegate violated CNI spec
	fmt.Println("Error case 4: Invalid result JSON - delegate protocol violation")

	// Output:
	// Error case 1: Plugin not found - check CNI_PATH
	// Error case 2: Delegate fails - preserve error message
	// Error case 3: Timeout - abort after 30 seconds
	// Error case 4: Invalid result JSON - delegate protocol violation
}

// Example_cniChainOrder demonstrates the execution order in CNI chaining
func Example_cniChainOrder() {
	// CNI chaining with tenant-routing-wrapper → ptp → cilium-cni
	//
	// ADD command flow:
	fmt.Println("ADD: tenant-routing-wrapper (set fwmark)")
	fmt.Println("ADD: ptp (create veth, run IPAM)")
	fmt.Println("ADD: cilium-cni (attach to Cilium datapath)")

	fmt.Println("")

	// DEL command flow (reverse order):
	fmt.Println("DEL: cilium-cni (detach from Cilium)")
	fmt.Println("DEL: ptp (remove veth, release IP)")
	fmt.Println("DEL: tenant-routing-wrapper (remove fwmark rules)")

	// Output:
	// ADD: tenant-routing-wrapper (set fwmark)
	// ADD: ptp (create veth, run IPAM)
	// ADD: cilium-cni (attach to Cilium datapath)
	//
	// DEL: cilium-cni (detach from Cilium)
	// DEL: ptp (remove veth, release IP)
	// DEL: tenant-routing-wrapper (remove fwmark rules)
}

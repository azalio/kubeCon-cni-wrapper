// Package main implements a CNI meta-plugin for tenant-aware routing.
//
// This plugin operates as part of a CNI chain and adds host-level iptables MARK rules
// based on Kubernetes pod/namespace annotations (tenant.routing/fwmark).
//
// Plugin chain position: ptp → tenant-routing-wrapper → cilium-cni
//
// Responsibilities:
//   - Delegates network setup to next CNI plugin (ptp)
//   - Fetches fwmark annotation from pod or namespace
//   - Adds iptables mangle/PREROUTING rule: -s <pod-ip> -j MARK --set-mark <fwmark>
//   - Returns delegate CNI Result unchanged (transparent wrapper)
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/version"

	"github.com/azalio/kubeCon-cni-wrapper/pkg/config"
	"github.com/azalio/kubeCon-cni-wrapper/pkg/delegate"
	"github.com/azalio/kubeCon-cni-wrapper/pkg/iptables"
	"github.com/azalio/kubeCon-cni-wrapper/pkg/k8s"
	"github.com/azalio/kubeCon-cni-wrapper/pkg/result"
)

// Version information - injected at build time via ldflags
var (
	// version is the semantic version or git tag
	versionStr = "dev"
	// commit is the git commit short hash
	commit = "unknown"
	// date is the build timestamp
	date = "unknown"
)

// parseCNIArgs extracts K8S_POD_NAME and K8S_POD_NAMESPACE from CNI_ARGS
// CNI_ARGS format: "K8S_POD_NAME=foo;K8S_POD_NAMESPACE=bar;..."
func parseCNIArgs(cniArgs string) (podName, podNamespace string, err error) {
	if cniArgs == "" {
		return "", "", fmt.Errorf("CNI_ARGS is empty")
	}

	pairs := strings.Split(cniArgs, ";")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "K8S_POD_NAME":
			podName = kv[1]
		case "K8S_POD_NAMESPACE":
			podNamespace = kv[1]
		}
	}

	if podName == "" {
		return "", "", fmt.Errorf("K8S_POD_NAME not found in CNI_ARGS")
	}
	if podNamespace == "" {
		return "", "", fmt.Errorf("K8S_POD_NAMESPACE not found in CNI_ARGS")
	}

	return podName, podNamespace, nil
}

// cmdAdd handles CNI ADD command
// Called when a container is created and network configuration is required
//
// Flow:
// 1. Parse CNI config
// 2. Extract pod name/namespace from CNI_ARGS
// 3. Delegate to next CNI plugin (get pod IP)
// 4. Fetch fwmark annotation from pod or namespace
// 5. Add iptables MARK rule if fwmark annotation present
// 6. Return delegate Result unchanged
func cmdAdd(args *skel.CmdArgs) error {
	// Step 1: Parse CNI configuration
	pluginConf, err := config.ParseConfig(args.StdinData)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Step 2: Extract pod name/namespace from CNI_ARGS
	// Required BEFORE delegation to validate input early
	podName, podNamespace, err := parseCNIArgs(args.Args)
	if err != nil {
		return fmt.Errorf("failed to parse CNI_ARGS: %w", err)
	}

	// Step 3: Delegate to next CNI plugin
	// This creates the veth pair and assigns IP via IPAM
	// Pass network name from parent config - required by CNI spec
	delegateResult, err := delegate.DelegateAdd(pluginConf.Delegate, pluginConf.Name, args.StdinData)
	if err != nil {
		// Delegation failure is fatal - pod cannot start without network
		return fmt.Errorf("delegation failed: %w", err)
	}

	// Step 4: Extract pod IP from delegate result
	podIP, err := result.ExtractPodIP(delegateResult)
	if err != nil {
		return fmt.Errorf("failed to extract pod IP from delegate result: %w", err)
	}

	// Step 5: Create Kubernetes client and fetch fwmark annotation
	clientset, err := k8s.NewClient(pluginConf.Kubeconfig)
	if err != nil {
		// Log warning but don't fail pod creation
		// This allows pods to start even if K8s API is temporarily unavailable
		log.Printf("WARNING: failed to create K8s client, skipping fwmark setup: %v", err)
		return types.PrintResult(delegateResult, pluginConf.CNIVersion)
	}

	fwmark, err := k8s.GetFwmark(clientset, podName, podNamespace, pluginConf.AnnotationKey)
	if err != nil {
		// Log warning but don't fail pod creation
		log.Printf("WARNING: failed to get fwmark annotation for %s/%s: %v", podNamespace, podName, err)
		return types.PrintResult(delegateResult, pluginConf.CNIVersion)
	}

	// Step 6: Add iptables rule if fwmark annotation present
	if fwmark != "" {
		if err := iptables.AddMarkRule(podIP, fwmark); err != nil {
			// Log warning but don't fail pod creation
			// iptables failure is non-fatal to avoid blocking pod startup
			log.Printf("WARNING: failed to add iptables rule for pod %s/%s (IP: %s, fwmark: %s): %v",
				podNamespace, podName, podIP, fwmark, err)
		} else {
			log.Printf("INFO: added iptables MARK rule for pod %s/%s: -s %s -j MARK --set-mark %s",
				podNamespace, podName, podIP, fwmark)
		}
	}

	// Return delegate result unchanged
	// The CNI contract requires we pass through the Result from delegate
	return types.PrintResult(delegateResult, pluginConf.CNIVersion)
}

// cmdDel handles CNI DEL command
// Called when a container is deleted and network configuration should be cleaned up
//
// Flow:
// 1. Parse CNI config (including prevResult from ADD)
// 2. Extract pod IP from prevResult
// 3. Delegate DEL to next CNI plugin
// 4. Remove iptables MARK rule if we have fwmark annotation
//
// DEL operations MUST be idempotent - multiple calls with same args should succeed
func cmdDel(args *skel.CmdArgs) error {
	// Parse CNI configuration
	pluginConf, err := config.ParseConfig(args.StdinData)
	if err != nil {
		// Log error but don't fail - DEL should be tolerant
		log.Printf("WARNING: failed to parse config in DEL: %v", err)
		return nil
	}

	// Extract pod info from CNI_ARGS
	podName, podNamespace, err := parseCNIArgs(args.Args)
	if err != nil {
		// CNI_ARGS might be missing during cleanup - not fatal
		log.Printf("WARNING: failed to parse CNI_ARGS in DEL: %v", err)
	}

	// Try to extract pod IP from prevResult (the result saved from ADD operation)
	// CNI spec requires container runtimes to pass prevResult during DEL
	var podIP string
	if pluginConf.PrevResult != nil {
		// PrevResult is already a types.Result interface, can be used directly
		podIP, err = result.ExtractPodIP(pluginConf.PrevResult)
		if err != nil {
			log.Printf("WARNING: failed to extract pod IP from prevResult: %v", err)
		}
	}

	// Delegate DEL to next plugin first
	// Must happen regardless of iptables cleanup success
	// Pass network name from parent config - required by CNI spec
	if err := delegate.DelegateDel(pluginConf.Delegate, pluginConf.Name, args.StdinData); err != nil {
		log.Printf("WARNING: delegate DEL failed: %v", err)
	}

	// Clean up iptables rule if we have both pod IP and fwmark annotation
	if podIP != "" && podName != "" && podNamespace != "" {
		clientset, err := k8s.NewClient(pluginConf.Kubeconfig)
		if err != nil {
			log.Printf("WARNING: failed to create K8s client for cleanup: %v", err)
			return nil
		}

		fwmark, err := k8s.GetFwmark(clientset, podName, podNamespace, pluginConf.AnnotationKey)
		if err != nil {
			// Pod might already be deleted - this is expected during cleanup
			log.Printf("INFO: could not get fwmark for cleanup (pod may be deleted): %v", err)
			// Try to clean up both possible fwmark values since we don't know which one was used
			cleanupIptablesRules(podIP)
			return nil
		}

		if fwmark != "" {
			if err := iptables.DeleteMarkRule(podIP, fwmark); err != nil {
				log.Printf("WARNING: failed to delete iptables rule for pod %s/%s (IP: %s, fwmark: %s): %v",
					podNamespace, podName, podIP, fwmark, err)
			} else {
				log.Printf("INFO: deleted iptables MARK rule for pod %s/%s: -s %s -j MARK --set-mark %s",
					podNamespace, podName, podIP, fwmark)
			}
		}
	} else if podIP != "" {
		// We have IP but no pod info - try to clean up any rules for this IP
		log.Printf("INFO: cleaning up any iptables rules for IP %s (pod info unavailable)", podIP)
		cleanupIptablesRules(podIP)
	}

	return nil
}

// cleanupIptablesRules attempts to clean up iptables rules for a given IP
// Tries both valid fwmark values since we might not know which one was used
func cleanupIptablesRules(podIP string) {
	for fwmark := range k8s.ValidFwmarkValues {
		if err := iptables.DeleteMarkRule(podIP, fwmark); err != nil {
			// Log at debug level - rule might not exist
			log.Printf("DEBUG: DeleteMarkRule(%s, %s) failed: %v", podIP, fwmark, err)
		}
	}
}

// cmdCheck handles CNI CHECK command
// Called to verify that the container's network is configured as expected
//
// Flow:
// 1. Parse CNI config
// 2. Delegate CHECK to next CNI plugin
// 3. If fwmark annotation present, verify iptables rule exists
// 4. Return error if configuration drift detected (annotation present but rule missing)
func cmdCheck(args *skel.CmdArgs) error {
	// Parse CNI configuration
	pluginConf, err := config.ParseConfig(args.StdinData)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Delegate CHECK to next plugin first
	// This verifies the underlying network configuration (veth, IP, routes)
	// Pass network name from parent config - required by CNI spec
	if err := delegate.DelegateCheck(pluginConf.Delegate, pluginConf.Name, args.StdinData); err != nil {
		return fmt.Errorf("delegate CHECK failed: %w", err)
	}

	// Extract pod info from CNI_ARGS
	podName, podNamespace, err := parseCNIArgs(args.Args)
	if err != nil {
		// Cannot verify iptables without pod info
		log.Printf("WARNING: CHECK cannot verify iptables - failed to parse CNI_ARGS: %v", err)
		return nil
	}

	// Extract pod IP from prevResult
	var podIP string
	if pluginConf.PrevResult != nil {
		podIP, err = result.ExtractPodIP(pluginConf.PrevResult)
		if err != nil {
			log.Printf("WARNING: CHECK cannot verify iptables - failed to extract pod IP: %v", err)
			return nil
		}
	} else {
		log.Printf("WARNING: CHECK cannot verify iptables - no prevResult available")
		return nil
	}

	// Create Kubernetes client and fetch fwmark annotation
	clientset, err := k8s.NewClient(pluginConf.Kubeconfig)
	if err != nil {
		log.Printf("WARNING: CHECK cannot verify iptables - failed to create K8s client: %v", err)
		return nil
	}

	fwmark, err := k8s.GetFwmark(clientset, podName, podNamespace, pluginConf.AnnotationKey)
	if err != nil {
		// Pod might be terminating - not a CHECK failure
		log.Printf("WARNING: CHECK cannot verify iptables - failed to get fwmark annotation: %v", err)
		return nil
	}

	// If fwmark annotation is present, verify iptables rule exists
	if fwmark != "" {
		exists, err := iptables.RuleExists(podIP, fwmark)
		if err != nil {
			// Cannot determine rule state - log warning but don't fail CHECK
			log.Printf("WARNING: CHECK cannot verify iptables rule existence: %v", err)
			return nil
		}

		if !exists {
			// Configuration drift detected: annotation says rule should exist, but it doesn't
			return fmt.Errorf("configuration drift detected: fwmark annotation %s present for pod %s/%s (IP: %s) but iptables rule missing",
				fwmark, podNamespace, podName, podIP)
		}

		log.Printf("INFO: CHECK verified iptables rule exists for pod %s/%s (IP: %s, fwmark: %s)",
			podNamespace, podName, podIP, fwmark)
	}

	return nil
}

// buildVersionString returns the full version string for CNI about
func buildVersionString() string {
	return fmt.Sprintf("tenant-routing-wrapper %s (commit: %s, built: %s)", versionStr, commit, date)
}

func main() {
	// Configure logging to stderr (CNI spec: stdout is for results, stderr for logs)
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// skel.PluginMain automatically:
	// 1. Reads CNI_COMMAND environment variable
	// 2. Routes to appropriate handler (cmdAdd/cmdDel/cmdCheck)
	// 3. Handles stdout/stderr formatting per CNI spec
	// 4. Sets appropriate exit codes on errors
	skel.PluginMain(
		cmdAdd,
		cmdCheck,
		cmdDel,
		version.All,
		buildVersionString(),
	)
}

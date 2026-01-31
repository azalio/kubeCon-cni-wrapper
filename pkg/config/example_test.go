package config_test

import (
	"fmt"
	"log"

	"github.com/azalio/kubeCon-cni-wrapper/pkg/config"
)

// ExampleParseConfig demonstrates how to use ParseConfig in the CNI wrapper
func ExampleParseConfig() {
	// Simulated stdin from kubelet when invoking CNI plugin
	stdinData := []byte(`{
		"cniVersion": "1.0.0",
		"name": "tenant-routing",
		"type": "tenant-routing-wrapper",
		"kubeconfig": "/etc/cni/net.d/tenant-routing.kubeconfig",
		"delegate": {
			"type": "ptp",
			"ipam": {
				"type": "host-local",
				"subnet": "10.200.0.0/16"
			}
		}
	}`)

	// Parse configuration
	conf, err := config.ParseConfig(stdinData)
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Access configuration fields
	fmt.Printf("CNI Version: %s\n", conf.CNIVersion)
	fmt.Printf("Plugin Name: %s\n", conf.Name)
	fmt.Printf("Kubeconfig: %s\n", conf.Kubeconfig)
	fmt.Printf("Annotation Key: %s\n", conf.AnnotationKey)

	// Get delegate config to pass to next plugin
	delegateConfig := conf.GetDelegateConfig()
	fmt.Printf("Delegate config length: %d bytes\n", len(delegateConfig))

	// Output:
	// CNI Version: 1.0.0
	// Plugin Name: tenant-routing
	// Kubeconfig: /etc/cni/net.d/tenant-routing.kubeconfig
	// Annotation Key: tenant.routing/fwmark
	// Delegate config length: 97 bytes
}

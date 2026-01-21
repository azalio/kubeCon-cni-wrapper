package k8s

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClient creates a Kubernetes clientset with support for both in-cluster and out-of-cluster configurations.
//
// When kubeconfigPath is empty, it attempts to use in-cluster configuration (service account tokens).
// This is the typical mode when running as a CNI plugin inside a Kubernetes cluster.
//
// When kubeconfigPath is provided, it loads the configuration from the specified kubeconfig file.
// This is useful for local development and testing outside the cluster.
//
// Security considerations:
//   - Validates kubeconfig file exists and is readable before attempting to load
//   - Uses clientcmd.BuildConfigFromFlags for secure config loading
//   - Returns descriptive errors to help diagnose authentication/connectivity issues
//
// Returns:
//   - *kubernetes.Clientset: Configured client ready for API operations
//   - error: Validation or configuration errors with context
func NewClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	if kubeconfigPath == "" {
		// In-cluster configuration: use service account tokens
		// This relies on:
		//   - /var/run/secrets/kubernetes.io/serviceaccount/token
		//   - /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
		//   - KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT env vars
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		// Out-of-cluster configuration: load from kubeconfig file
		// Security: Validate file exists and is readable before loading
		if _, err := os.Stat(kubeconfigPath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("kubeconfig file does not exist: %s", kubeconfigPath)
			}
			return nil, fmt.Errorf("kubeconfig file is not readable: %s: %w", kubeconfigPath, err)
		}

		// BuildConfigFromFlags handles:
		//   - Loading kubeconfig from file
		//   - Merging multiple contexts if present
		//   - Validating authentication credentials
		// The first parameter (masterUrl) is empty to use the server from kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig %s: %w", kubeconfigPath, err)
		}
	}

	// Create clientset from validated config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	return clientset, nil
}

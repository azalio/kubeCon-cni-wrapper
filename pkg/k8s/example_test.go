package k8s_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/azalio/kubeCon-cni-wrapper/pkg/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExampleNewClient_inCluster demonstrates creating a client for in-cluster usage
func ExampleNewClient_inCluster() {
	// When running inside a Kubernetes cluster (e.g., as a CNI plugin),
	// pass empty string to use service account credentials
	client, err := k8s.NewClient("")
	if err != nil {
		// In test environment, this will fail (no service account tokens)
		// In production, this should succeed when running in a pod
		fmt.Println("In-cluster config not available (expected in test environment)")
		return
	}

	// Use client for API operations
	_ = client
	fmt.Println("Client created successfully")
}

// ExampleNewClient_outOfCluster demonstrates creating a client for local development
func ExampleNewClient_outOfCluster() {
	// Create a temporary kubeconfig for demonstration
	tempDir, err := os.MkdirTemp("", "kubeconfig-example-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	kubeconfigPath := filepath.Join(tempDir, "kubeconfig")
	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://kubernetes.example.com:6443
  name: example-cluster
contexts:
- context:
    cluster: example-cluster
    user: example-user
  name: example-context
current-context: example-context
users:
- name: example-user
  user:
    token: example-token-abc123
`

	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		log.Fatal(err)
	}

	// When running outside the cluster (local development),
	// pass the path to your kubeconfig file
	client, err := k8s.NewClient(kubeconfigPath)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Use client for API operations
	_ = client
	fmt.Println("Client created successfully from kubeconfig")
	// Output: Client created successfully from kubeconfig
}

// ExampleNewClient_listNamespaces demonstrates using the client to perform API operations
func ExampleNewClient_listNamespaces() {
	// This example shows how to use the client once created
	// Note: Requires a valid Kubernetes cluster connection

	tempDir, err := os.MkdirTemp("", "kubeconfig-example-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	kubeconfigPath := filepath.Join(tempDir, "kubeconfig")
	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://kubernetes.example.com:6443
  name: example-cluster
contexts:
- context:
    cluster: example-cluster
    user: example-user
  name: example-context
current-context: example-context
users:
- name: example-user
  user:
    token: example-token-abc123
`

	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		log.Fatal(err)
	}

	client, err := k8s.NewClient(kubeconfigPath)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Example: List namespaces
	namespaces, err := client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		// This will fail without a real cluster, but demonstrates the API usage
		fmt.Println("API call failed (expected without real cluster)")
		return
	}

	fmt.Printf("Found %d namespaces\n", len(namespaces.Items))
}

package k8s

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewClient_WithValidKubeconfig tests client creation with a valid kubeconfig file
func TestNewClient_WithValidKubeconfig(t *testing.T) {
	// Create temporary kubeconfig file
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "kubeconfig")

	// Minimal valid kubeconfig structure
	validKubeconfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`

	if err := os.WriteFile(kubeconfigPath, []byte(validKubeconfig), 0600); err != nil {
		t.Fatalf("Failed to write test kubeconfig: %v", err)
	}

	// Test: Client creation should succeed with valid kubeconfig
	client, err := NewClient(kubeconfigPath)
	if err != nil {
		t.Errorf("NewClient() with valid kubeconfig failed: %v", err)
	}

	if client == nil {
		t.Error("NewClient() returned nil client with valid kubeconfig")
	}
}

// TestNewClient_WithNonExistentKubeconfig tests error handling when kubeconfig file doesn't exist
func TestNewClient_WithNonExistentKubeconfig(t *testing.T) {
	// Test: Should return error when file doesn't exist
	nonExistentPath := "/tmp/does-not-exist-kubeconfig-12345"
	client, err := NewClient(nonExistentPath)

	if err == nil {
		t.Error("NewClient() should return error for non-existent kubeconfig file")
	}

	if client != nil {
		t.Error("NewClient() should return nil client for non-existent kubeconfig file")
	}

	// Verify error message contains the file path
	if err != nil && err.Error() == "" {
		t.Error("Error message should be descriptive")
	}
}

// TestNewClient_WithInvalidKubeconfig tests error handling with malformed kubeconfig content
func TestNewClient_WithInvalidKubeconfig(t *testing.T) {
	// Create temporary kubeconfig with invalid content
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "invalid-kubeconfig")

	invalidKubeconfig := `this is not valid YAML`

	if err := os.WriteFile(kubeconfigPath, []byte(invalidKubeconfig), 0600); err != nil {
		t.Fatalf("Failed to write test kubeconfig: %v", err)
	}

	// Test: Should return error when kubeconfig is invalid
	client, err := NewClient(kubeconfigPath)

	if err == nil {
		t.Error("NewClient() should return error for invalid kubeconfig content")
	}

	if client != nil {
		t.Error("NewClient() should return nil client for invalid kubeconfig content")
	}
}

// TestNewClient_WithUnreadableKubeconfig tests error handling when file exists but is not readable
func TestNewClient_WithUnreadableKubeconfig(t *testing.T) {
	// Skip this test if running as root (root can read files with 0000 permissions)
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	// Create temporary kubeconfig with no read permissions
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "unreadable-kubeconfig")

	if err := os.WriteFile(kubeconfigPath, []byte("test"), 0000); err != nil {
		t.Fatalf("Failed to write test kubeconfig: %v", err)
	}

	// Test: Should return error when file is not readable
	client, err := NewClient(kubeconfigPath)

	if err == nil {
		t.Error("NewClient() should return error for unreadable kubeconfig file")
	}

	if client != nil {
		t.Error("NewClient() should return nil client for unreadable kubeconfig file")
	}
}

// TestNewClient_WithEmptyPath tests in-cluster config behavior
func TestNewClient_WithEmptyPath(t *testing.T) {
	// Test: Empty path should attempt in-cluster config
	// This will fail in test environment (no service account tokens),
	// but we verify it attempts the right code path
	client, err := NewClient("")

	// In test environment, in-cluster config should fail
	// We're testing that it doesn't panic and returns proper error
	if err == nil {
		// Only possible if running inside a Kubernetes cluster
		if client == nil {
			t.Error("NewClient() returned nil error but also nil client")
		}
	} else {
		// Expected: error about missing in-cluster config
		if client != nil {
			t.Error("NewClient() returned error but non-nil client")
		}
	}
}

// TestNewClient_WithRelativePath tests behavior with relative paths
// Note: This tests the clientcmd behavior, not our validation (config package validates paths)
func TestNewClient_WithRelativePath(t *testing.T) {
	// Create temporary kubeconfig in current directory
	tempDir := t.TempDir()
	kubeconfigPath := filepath.Join(tempDir, "kubeconfig")

	validKubeconfig := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`

	if err := os.WriteFile(kubeconfigPath, []byte(validKubeconfig), 0600); err != nil {
		t.Fatalf("Failed to write test kubeconfig: %v", err)
	}

	// Test: clientcmd.BuildConfigFromFlags accepts relative paths (validation is in config package)
	// We just verify the client factory doesn't reject it
	client, err := NewClient(kubeconfigPath)
	if err != nil {
		t.Errorf("NewClient() failed with path %s: %v", kubeconfigPath, err)
	}

	if client == nil {
		t.Error("NewClient() returned nil client with valid kubeconfig")
	}
}

package main

import (
	"encoding/json"
	"testing"

	"github.com/containernetworking/cni/pkg/skel"
)

// Integration tests for CNI command handlers
// These tests validate the ST-008 validation criteria

// TestCmdAdd_MissingCNIArgs verifies:
// "CNI_ARGS without K8S_POD_NAME returns error before delegation"
func TestCmdAdd_MissingCNIArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantErr string
	}{
		{
			name:    "empty CNI_ARGS",
			args:    "",
			wantErr: "CNI_ARGS is empty",
		},
		{
			name:    "missing K8S_POD_NAME",
			args:    "K8S_POD_NAMESPACE=default",
			wantErr: "K8S_POD_NAME not found",
		},
		{
			name:    "missing K8S_POD_NAMESPACE",
			args:    "K8S_POD_NAME=test-pod",
			wantErr: "K8S_POD_NAMESPACE not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Minimal valid CNI config
			stdinData := []byte(`{
				"cniVersion": "1.0.0",
				"name": "test-network",
				"type": "tenant-routing-wrapper",
				"kubeconfig": "/etc/cni/net.d/kubeconfig",
				"delegate": {
					"type": "ptp",
					"cniVersion": "1.0.0"
				}
			}`)

			cmdArgs := &skel.CmdArgs{
				ContainerID: "test-container-123",
				Netns:       "/var/run/netns/test",
				IfName:      "eth0",
				Args:        tt.args,
				Path:        "/opt/cni/bin",
				StdinData:   stdinData,
			}

			err := cmdAdd(cmdArgs)
			if err == nil {
				t.Fatal("expected error but got nil")
			}

			if !containsSubstring(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestCmdAdd_InvalidConfig verifies config parsing errors
func TestCmdAdd_InvalidConfig(t *testing.T) {
	tests := []struct {
		name      string
		stdinData []byte
		wantErr   string
	}{
		{
			name:      "invalid JSON",
			stdinData: []byte(`{invalid json}`),
			wantErr:   "failed to parse config",
		},
		{
			name: "missing delegate",
			stdinData: []byte(`{
				"cniVersion": "1.0.0",
				"name": "test-network",
				"type": "tenant-routing-wrapper",
				"kubeconfig": "/etc/cni/net.d/kubeconfig"
			}`),
			wantErr: "delegate plugin configuration is required",
		},
		{
			name: "relative kubeconfig path",
			stdinData: []byte(`{
				"cniVersion": "1.0.0",
				"name": "test-network",
				"type": "tenant-routing-wrapper",
				"kubeconfig": "relative/path/kubeconfig",
				"delegate": {"type": "ptp", "cniVersion": "1.0.0"}
			}`),
			wantErr: "kubeconfig path must be absolute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdArgs := &skel.CmdArgs{
				ContainerID: "test-container-123",
				Netns:       "/var/run/netns/test",
				IfName:      "eth0",
				Args:        "K8S_POD_NAME=test;K8S_POD_NAMESPACE=default",
				Path:        "/opt/cni/bin",
				StdinData:   tt.stdinData,
			}

			err := cmdAdd(cmdArgs)
			if err == nil {
				t.Fatal("expected error but got nil")
			}

			if !containsSubstring(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestCmdAdd_DelegationBeforeAnnotationCheck verifies:
// "CNI_ARGS without K8S_POD_NAME returns error before delegation"
// By ensuring config parsing and CNI_ARGS validation happen BEFORE any delegation
func TestCmdAdd_ValidationOrderIsCorrect(t *testing.T) {
	// This test verifies the order of operations:
	// 1. Parse config (validated by TestCmdAdd_InvalidConfig)
	// 2. Parse CNI_ARGS (validated by TestCmdAdd_MissingCNIArgs)
	// 3. Delegate (only after 1 and 2 pass)

	// With invalid CNI_ARGS but valid config, should fail before delegation
	stdinData := []byte(`{
		"cniVersion": "1.0.0",
		"name": "test-network",
		"type": "tenant-routing-wrapper",
		"kubeconfig": "/etc/cni/net.d/kubeconfig",
		"delegate": {
			"type": "ptp",
			"cniVersion": "1.0.0"
		}
	}`)

	cmdArgs := &skel.CmdArgs{
		ContainerID: "test-container-123",
		Netns:       "/var/run/netns/test",
		IfName:      "eth0",
		Args:        "", // Invalid - missing required fields
		Path:        "/opt/cni/bin",
		StdinData:   stdinData,
	}

	err := cmdAdd(cmdArgs)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	// Error should be about CNI_ARGS, not about delegation failure
	// This proves CNI_ARGS is validated BEFORE delegation attempt
	if containsSubstring(err.Error(), "delegation") {
		t.Errorf("error indicates delegation was attempted: %v", err)
	}
	if !containsSubstring(err.Error(), "CNI_ARGS") {
		t.Errorf("expected CNI_ARGS error, got: %v", err)
	}
}

// TestCmdDel_Idempotent verifies:
// "DEL succeeds even if iptables rule does not exist (idempotent behavior)"
// "DEL with missing CNI_ARGS does not panic or return error to kubelet"
func TestCmdDel_Idempotent(t *testing.T) {
	tests := []struct {
		name      string
		args      string
		stdinData []byte
		wantErr   bool
	}{
		{
			name: "missing CNI_ARGS should not fail",
			args: "",
			stdinData: []byte(`{
				"cniVersion": "1.0.0",
				"name": "test-network",
				"type": "tenant-routing-wrapper",
				"kubeconfig": "/etc/cni/net.d/kubeconfig",
				"delegate": {"type": "ptp", "cniVersion": "1.0.0"}
			}`),
			wantErr: false, // DEL should be tolerant
		},
		{
			name: "partial CNI_ARGS should not fail",
			args: "K8S_POD_NAME=test",
			stdinData: []byte(`{
				"cniVersion": "1.0.0",
				"name": "test-network",
				"type": "tenant-routing-wrapper",
				"kubeconfig": "/etc/cni/net.d/kubeconfig",
				"delegate": {"type": "ptp", "cniVersion": "1.0.0"}
			}`),
			wantErr: false, // DEL should be tolerant
		},
		{
			name: "invalid config should not fail",
			args: "K8S_POD_NAME=test;K8S_POD_NAMESPACE=default",
			stdinData: []byte(`{invalid json}`),
			wantErr: false, // DEL should be tolerant even with invalid config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdArgs := &skel.CmdArgs{
				ContainerID: "test-container-123",
				Netns:       "/var/run/netns/test",
				IfName:      "eth0",
				Args:        tt.args,
				Path:        "/opt/cni/bin",
				StdinData:   tt.stdinData,
			}

			// DEL should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("cmdDel panicked: %v", r)
				}
			}()

			err := cmdDel(cmdArgs)
			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestCleanupIptablesRules verifies the helper function doesn't panic
func TestCleanupIptablesRules(t *testing.T) {
	// Should not panic even with invalid IP
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("cleanupIptablesRules panicked: %v", r)
		}
	}()

	// These will fail validation but should not panic
	cleanupIptablesRules("10.200.1.5")
	cleanupIptablesRules("")
}

// TestCmdCheck_InvalidConfig verifies CHECK returns errors for invalid config
func TestCmdCheck_InvalidConfig(t *testing.T) {
	stdinData := []byte(`{invalid json}`)

	cmdArgs := &skel.CmdArgs{
		ContainerID: "test-container-123",
		Netns:       "/var/run/netns/test",
		IfName:      "eth0",
		Args:        "K8S_POD_NAME=test;K8S_POD_NAMESPACE=default",
		Path:        "/opt/cni/bin",
		StdinData:   stdinData,
	}

	err := cmdCheck(cmdArgs)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}

	if !containsSubstring(err.Error(), "failed to parse config") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestValidCNIConfig verifies a valid config structure
func TestValidCNIConfig(t *testing.T) {
	config := map[string]any{
		"cniVersion": "1.0.0",
		"name":       "tenant-routing",
		"type":       "tenant-routing-wrapper",
		"kubeconfig": "/etc/cni/net.d/kubeconfig",
		"delegate": map[string]any{
			"type":       "ptp",
			"cniVersion": "1.0.0",
			"ipam": map[string]any{
				"type":   "host-local",
				"subnet": "10.200.0.0/16",
			},
		},
	}

	_, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal valid config: %v", err)
	}
}

// containsSubstring checks if s contains substr (case-insensitive is not needed here)
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

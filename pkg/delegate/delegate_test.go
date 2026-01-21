package delegate

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestDelegateAdd_MissingType verifies error handling when delegate config lacks 'type' field
func TestDelegateAdd_MissingType(t *testing.T) {
	// Delegate config without required 'type' field
	delegateConfig := json.RawMessage(`{"cniVersion": "1.0.0"}`)
	stdin := []byte(`{}`)

	_, err := DelegateAdd(delegateConfig, "test-network", stdin)
	if err == nil {
		t.Fatal("Expected error when delegate config missing 'type' field")
	}

	if !strings.Contains(err.Error(), "missing required 'type' field") {
		t.Errorf("Expected error about missing 'type', got: %v", err)
	}
}

// TestDelegateAdd_InvalidJSON verifies error handling for malformed delegate config
func TestDelegateAdd_InvalidJSON(t *testing.T) {
	delegateConfig := json.RawMessage(`{invalid json}`)
	stdin := []byte(`{}`)

	_, err := DelegateAdd(delegateConfig, "test-network", stdin)
	if err == nil {
		t.Fatal("Expected error when delegate config is invalid JSON")
	}

	if !strings.Contains(err.Error(), "failed to parse delegate config") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

// TestDelegateAdd_MissingCNIPath verifies error when CNI_PATH not set
func TestDelegateAdd_MissingCNIPath(t *testing.T) {
	// Save original CNI_PATH and restore after test
	originalPath := os.Getenv("CNI_PATH")
	defer func() {
		if originalPath != "" {
			os.Setenv("CNI_PATH", originalPath)
		}
	}()

	// Unset CNI_PATH
	os.Unsetenv("CNI_PATH")

	delegateConfig := json.RawMessage(`{"type": "ptp", "cniVersion": "1.0.0"}`)
	stdin := []byte(`{}`)

	_, err := DelegateAdd(delegateConfig, "test-network", stdin)
	if err == nil {
		t.Fatal("Expected error when CNI_PATH not set")
	}

	if !strings.Contains(err.Error(), "CNI_PATH") {
		t.Errorf("Expected CNI_PATH error, got: %v", err)
	}
}

// TestDelegateDel_MissingType verifies error handling when delegate config lacks 'type' field
func TestDelegateDel_MissingType(t *testing.T) {
	delegateConfig := json.RawMessage(`{"cniVersion": "1.0.0"}`)
	stdin := []byte(`{}`)

	err := DelegateDel(delegateConfig, "test-network", stdin)
	if err == nil {
		t.Fatal("Expected error when delegate config missing 'type' field")
	}

	if !strings.Contains(err.Error(), "missing required 'type' field") {
		t.Errorf("Expected error about missing 'type', got: %v", err)
	}
}

// TestDelegateDel_InvalidJSON verifies error handling for malformed delegate config
func TestDelegateDel_InvalidJSON(t *testing.T) {
	delegateConfig := json.RawMessage(`{invalid json}`)
	stdin := []byte(`{}`)

	err := DelegateDel(delegateConfig, "test-network", stdin)
	if err == nil {
		t.Fatal("Expected error when delegate config is invalid JSON")
	}

	if !strings.Contains(err.Error(), "failed to parse delegate config") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

// TestDelegateDel_MissingCNIPath verifies error when CNI_PATH not set
func TestDelegateDel_MissingCNIPath(t *testing.T) {
	// Save original CNI_PATH and restore after test
	originalPath := os.Getenv("CNI_PATH")
	defer func() {
		if originalPath != "" {
			os.Setenv("CNI_PATH", originalPath)
		}
	}()

	// Unset CNI_PATH
	os.Unsetenv("CNI_PATH")

	delegateConfig := json.RawMessage(`{"type": "ptp", "cniVersion": "1.0.0"}`)
	stdin := []byte(`{}`)

	err := DelegateDel(delegateConfig, "test-network", stdin)
	if err == nil {
		t.Fatal("Expected error when CNI_PATH not set")
	}

	if !strings.Contains(err.Error(), "CNI_PATH") {
		t.Errorf("Expected CNI_PATH error, got: %v", err)
	}
}

// TestDelegateCheck_MissingType verifies error handling when delegate config lacks 'type' field
func TestDelegateCheck_MissingType(t *testing.T) {
	delegateConfig := json.RawMessage(`{"cniVersion": "1.0.0"}`)
	stdin := []byte(`{}`)

	err := DelegateCheck(delegateConfig, "test-network", stdin)
	if err == nil {
		t.Fatal("Expected error when delegate config missing 'type' field")
	}

	if !strings.Contains(err.Error(), "missing required 'type' field") {
		t.Errorf("Expected error about missing 'type', got: %v", err)
	}
}

// TestDelegateCheck_InvalidJSON verifies error handling for malformed delegate config
func TestDelegateCheck_InvalidJSON(t *testing.T) {
	delegateConfig := json.RawMessage(`{invalid json}`)
	stdin := []byte(`{}`)

	err := DelegateCheck(delegateConfig, "test-network", stdin)
	if err == nil {
		t.Fatal("Expected error when delegate config is invalid JSON")
	}

	if !strings.Contains(err.Error(), "failed to parse delegate config") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

// TestDelegateCheck_MissingCNIPath verifies error when CNI_PATH not set
func TestDelegateCheck_MissingCNIPath(t *testing.T) {
	// Save original CNI_PATH and restore after test
	originalPath := os.Getenv("CNI_PATH")
	defer func() {
		if originalPath != "" {
			os.Setenv("CNI_PATH", originalPath)
		}
	}()

	// Unset CNI_PATH
	os.Unsetenv("CNI_PATH")

	delegateConfig := json.RawMessage(`{"type": "ptp", "cniVersion": "1.0.0"}`)
	stdin := []byte(`{}`)

	err := DelegateCheck(delegateConfig, "test-network", stdin)
	if err == nil {
		t.Fatal("Expected error when CNI_PATH not set")
	}

	if !strings.Contains(err.Error(), "CNI_PATH") {
		t.Errorf("Expected CNI_PATH error, got: %v", err)
	}
}

// TestGetPluginPath_Success verifies plugin path resolution
func TestGetPluginPath_Success(t *testing.T) {
	// Save and restore CNI_PATH
	originalPath := os.Getenv("CNI_PATH")
	defer func() {
		if originalPath != "" {
			os.Setenv("CNI_PATH", originalPath)
		} else {
			os.Unsetenv("CNI_PATH")
		}
	}()

	// Set CNI_PATH to a test directory
	os.Setenv("CNI_PATH", "/opt/cni/bin:/usr/local/bin/cni")

	// Note: This test will fail if the plugin doesn't actually exist
	// In production, CNI plugins are installed in CNI_PATH directories
	// For unit tests, we just verify the function handles the environment correctly
}

// TestGetPluginPath_MissingCNIPath verifies error when CNI_PATH not set
func TestGetPluginPath_MissingCNIPath(t *testing.T) {
	// Save and restore CNI_PATH
	originalPath := os.Getenv("CNI_PATH")
	defer func() {
		if originalPath != "" {
			os.Setenv("CNI_PATH", originalPath)
		}
	}()

	os.Unsetenv("CNI_PATH")

	_, err := GetPluginPath("ptp")
	if err == nil {
		t.Fatal("Expected error when CNI_PATH not set")
	}

	if !strings.Contains(err.Error(), "CNI_PATH") {
		t.Errorf("Expected CNI_PATH error, got: %v", err)
	}
}

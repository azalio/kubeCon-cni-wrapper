package delegate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/types"
)

// ExecutionTimeout is the maximum time allowed for delegate plugin execution
// Prevents hanging CNI operations that would block container creation
const ExecutionTimeout = 30 * time.Second

// DelegateAdd executes the delegate CNI plugin for ADD command
// Passes through all CNI environment variables and stdin unchanged
// Returns the delegate's CNI Result on success
//
// Parameters:
//   - delegateConfig: Raw JSON configuration for the delegate plugin (from PluginConf.Delegate)
//   - networkName: Name of the network (from parent config) - required by CNI spec
//   - stdin: Original CNI stdin data (may differ from delegateConfig if wrapper added fields)
//
// Environment variables propagated from current process:
//   - CNI_COMMAND (should be "ADD")
//   - CNI_CONTAINERID
//   - CNI_NETNS
//   - CNI_IFNAME
//   - CNI_ARGS
//   - CNI_PATH
//
// Returns:
//   - types.Result: Parsed CNI result from delegate plugin
//   - error: Non-nil if delegation fails or delegate returns error
func DelegateAdd(delegateConfig json.RawMessage, networkName string, stdin []byte) (types.Result, error) {
	// Parse delegate config to extract plugin type (required for execution)
	var delegateConf map[string]any
	if err := json.Unmarshal(delegateConfig, &delegateConf); err != nil {
		return nil, fmt.Errorf("failed to parse delegate config: %w", err)
	}

	pluginType, ok := delegateConf["type"].(string)
	if !ok || pluginType == "" {
		return nil, fmt.Errorf("delegate config missing required 'type' field")
	}

	// Inject network name into delegate config (required by CNI spec)
	// The name field must be present in the config passed to delegate plugins
	delegateConf["name"] = networkName

	// Re-marshal the config with injected name
	delegateConfigWithName, err := json.Marshal(delegateConf)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal delegate config: %w", err)
	}

	// Create execution context with timeout
	// Prevents indefinite hangs if delegate plugin is unresponsive
	ctx, cancel := context.WithTimeout(context.Background(), ExecutionTimeout)
	defer cancel()

	// Get CNI_PATH from environment (required for plugin discovery)
	// CNI plugins must be in directories listed in CNI_PATH
	if os.Getenv("CNI_PATH") == "" {
		return nil, fmt.Errorf("CNI_PATH environment variable not set")
	}

	// Create DefaultExec instance for plugin execution
	// DefaultExec implements invoke.Exec interface with actual command execution and version handling
	// Environment variables (CNI_COMMAND, CNI_CONTAINERID, etc.) are inherited from current process
	exec := &invoke.DefaultExec{
		RawExec: &invoke.RawExec{Stderr: os.Stderr},
	}

	// Execute delegate plugin using CNI invoke package
	// invoke.DelegateAdd handles:
	// - Finding plugin binary in CNI_PATH
	// - Executing with correct environment
	// - Passing delegateConfig as stdin
	// - Returning stdout as CNI Result
	// - Capturing stderr on failure
	result, err := invoke.DelegateAdd(ctx, pluginType, delegateConfigWithName, exec)

	if err != nil {
		// Preserve delegate error message exactly
		// Include delegate plugin name for debugging
		return nil, fmt.Errorf("delegate plugin %q failed: %w", pluginType, err)
	}

	// Result is already parsed by invoke.DelegateAdd
	return result, nil
}

// DelegateDel executes the delegate CNI plugin for DEL command
// Used to clean up network configuration when container is deleted
//
// Parameters:
//   - delegateConfig: Raw JSON configuration for the delegate plugin
//   - networkName: Name of the network (from parent config) - required by CNI spec
//   - stdin: Original CNI stdin data
//
// Returns:
//   - error: Non-nil if delegation fails (non-zero exit code or execution error)
//
// Note: DEL should be idempotent - multiple calls with same args should succeed
func DelegateDel(delegateConfig json.RawMessage, networkName string, stdin []byte) error {
	// Parse delegate config to extract plugin type
	var delegateConf map[string]any
	if err := json.Unmarshal(delegateConfig, &delegateConf); err != nil {
		return fmt.Errorf("failed to parse delegate config: %w", err)
	}

	pluginType, ok := delegateConf["type"].(string)
	if !ok || pluginType == "" {
		return fmt.Errorf("delegate config missing required 'type' field")
	}

	// Inject network name into delegate config
	delegateConf["name"] = networkName

	// Re-marshal the config with injected name
	delegateConfigWithName, err := json.Marshal(delegateConf)
	if err != nil {
		return fmt.Errorf("failed to marshal delegate config: %w", err)
	}

	// Create execution context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), ExecutionTimeout)
	defer cancel()

	// Get CNI_PATH from environment
	if os.Getenv("CNI_PATH") == "" {
		return fmt.Errorf("CNI_PATH environment variable not set")
	}

	// Create DefaultExec instance for plugin execution
	exec := &invoke.DefaultExec{
		RawExec: &invoke.RawExec{Stderr: os.Stderr},
	}

	// Execute delegate plugin DEL
	// DEL operations should clean up resources created by ADD
	err = invoke.DelegateDel(ctx, pluginType, delegateConfigWithName, exec)

	if err != nil {
		// Preserve delegate error message exactly
		return fmt.Errorf("delegate plugin %q DEL failed: %w", pluginType, err)
	}

	return nil
}

// DelegateCheck executes the delegate CNI plugin for CHECK command
// Verifies that network configuration is still valid
//
// Parameters:
//   - delegateConfig: Raw JSON configuration for the delegate plugin
//   - networkName: Name of the network (from parent config) - required by CNI spec
//   - stdin: Original CNI stdin data
//
// Returns:
//   - error: Non-nil if check fails (configuration not as expected)
func DelegateCheck(delegateConfig json.RawMessage, networkName string, stdin []byte) error {
	// Parse delegate config to extract plugin type
	var delegateConf map[string]any
	if err := json.Unmarshal(delegateConfig, &delegateConf); err != nil {
		return fmt.Errorf("failed to parse delegate config: %w", err)
	}

	pluginType, ok := delegateConf["type"].(string)
	if !ok || pluginType == "" {
		return fmt.Errorf("delegate config missing required 'type' field")
	}

	// Inject network name into delegate config
	delegateConf["name"] = networkName

	// Re-marshal the config with injected name
	delegateConfigWithName, err := json.Marshal(delegateConf)
	if err != nil {
		return fmt.Errorf("failed to marshal delegate config: %w", err)
	}

	// Create execution context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), ExecutionTimeout)
	defer cancel()

	// Get CNI_PATH from environment
	if os.Getenv("CNI_PATH") == "" {
		return fmt.Errorf("CNI_PATH environment variable not set")
	}

	// Create DefaultExec instance for plugin execution
	exec := &invoke.DefaultExec{
		RawExec: &invoke.RawExec{Stderr: os.Stderr},
	}

	// Execute delegate plugin CHECK
	// CHECK verifies configuration matches expected state
	err = invoke.DelegateCheck(ctx, pluginType, delegateConfigWithName, exec)

	if err != nil {
		// Preserve delegate error message exactly
		return fmt.Errorf("delegate plugin %q CHECK failed: %w", pluginType, err)
	}

	return nil
}

// GetPluginPath finds the full path to a CNI plugin binary
// Searches in directories specified by CNI_PATH environment variable
//
// This is useful for testing and debugging to verify plugin availability
func GetPluginPath(pluginType string) (string, error) {
	cniPath := os.Getenv("CNI_PATH")
	if cniPath == "" {
		return "", fmt.Errorf("CNI_PATH environment variable not set")
	}

	// Split CNI_PATH into individual directories
	paths := strings.Split(cniPath, ":")

	// Use RawExec to find plugin in path
	exec := &invoke.RawExec{}
	pluginPath, err := exec.FindInPath(pluginType, paths)
	if err != nil {
		return "", fmt.Errorf("plugin %q not found in CNI_PATH: %w", pluginType, err)
	}

	return pluginPath, nil
}

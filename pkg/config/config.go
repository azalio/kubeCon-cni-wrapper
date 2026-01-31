package config

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/containernetworking/cni/pkg/types"
)

const (
	// DefaultAnnotationKey is the default Kubernetes annotation key for fwmark values
	DefaultAnnotationKey = "tenant.routing/fwmark"
)

// PluginConf represents the CNI plugin configuration
// Extends standard NetConf with tenant routing specific fields
type PluginConf struct {
	types.NetConf

	// Kubeconfig path to Kubernetes API server credentials
	// MUST be an absolute path (security: prevent path traversal)
	Kubeconfig string `json:"kubeconfig"`

	// AnnotationKey specifies which pod annotation contains the fwmark value
	// Defaults to DefaultAnnotationKey if not specified
	AnnotationKey string `json:"annotationKey,omitempty"`

	// Delegate contains the configuration for the next CNI plugin in the chain
	// This is preserved as raw JSON to pass through unchanged
	Delegate json.RawMessage `json:"delegate"`
}

// ParseConfig parses CNI configuration from stdin data
// Validates required fields and security constraints
func ParseConfig(stdin []byte) (*PluginConf, error) {
	conf := &PluginConf{}

	// Parse JSON configuration
	if err := json.Unmarshal(stdin, conf); err != nil {
		return nil, fmt.Errorf("failed to parse network configuration: %w", err)
	}

	// Validate delegate configuration exists
	if len(conf.Delegate) == 0 {
		return nil, fmt.Errorf("delegate plugin configuration is required")
	}

	// Validate kubeconfig path is provided
	if conf.Kubeconfig == "" {
		return nil, fmt.Errorf("kubeconfig path is required")
	}

	// Security: Enforce absolute path to prevent path traversal attacks
	// Relative paths could be manipulated to access arbitrary files
	if !filepath.IsAbs(conf.Kubeconfig) {
		return nil, fmt.Errorf("kubeconfig path must be absolute, got: %s", conf.Kubeconfig)
	}

	// Security: Reject paths with '..' components (defense in depth)
	if strings.Contains(conf.Kubeconfig, "..") {
		return nil, fmt.Errorf("kubeconfig path cannot contain '..' components: %s", conf.Kubeconfig)
	}

	// Apply default annotation key if not specified
	if conf.AnnotationKey == "" {
		conf.AnnotationKey = DefaultAnnotationKey
	}

	return conf, nil
}

// GetDelegateConfig returns the delegate plugin configuration as raw JSON
// This allows the wrapper to pass the configuration unchanged to the next plugin
func (c *PluginConf) GetDelegateConfig() []byte {
	return c.Delegate
}

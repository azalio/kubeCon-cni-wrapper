package config

import (
	"encoding/json"
	"testing"
)

func TestParseConfig_ValidConfig(t *testing.T) {
	input := `{
		"cniVersion": "1.0.0",
		"name": "tenant-routing",
		"type": "tenant-routing-wrapper",
		"kubeconfig": "/etc/cni/net.d/tenant-routing.kubeconfig",
		"annotationKey": "custom.tenant/fwmark",
		"delegate": {
			"type": "ptp",
			"ipam": {
				"type": "host-local",
				"subnet": "10.200.0.0/16"
			}
		}
	}`

	conf, err := ParseConfig([]byte(input))
	if err != nil {
		t.Fatalf("Expected successful parse, got error: %v", err)
	}

	// Validate parsed fields
	if conf.CNIVersion != "1.0.0" {
		t.Errorf("Expected CNIVersion '1.0.0', got '%s'", conf.CNIVersion)
	}
	if conf.Name != "tenant-routing" {
		t.Errorf("Expected Name 'tenant-routing', got '%s'", conf.Name)
	}
	if conf.Type != "tenant-routing-wrapper" {
		t.Errorf("Expected Type 'tenant-routing-wrapper', got '%s'", conf.Type)
	}
	if conf.Kubeconfig != "/etc/cni/net.d/tenant-routing.kubeconfig" {
		t.Errorf("Expected Kubeconfig '/etc/cni/net.d/tenant-routing.kubeconfig', got '%s'", conf.Kubeconfig)
	}
	if conf.AnnotationKey != "custom.tenant/fwmark" {
		t.Errorf("Expected AnnotationKey 'custom.tenant/fwmark', got '%s'", conf.AnnotationKey)
	}

	// Validate delegate is preserved
	if len(conf.Delegate) == 0 {
		t.Error("Expected Delegate to be populated")
	}

	// Verify delegate can be parsed
	var delegateCheck map[string]interface{}
	if err := json.Unmarshal(conf.Delegate, &delegateCheck); err != nil {
		t.Errorf("Delegate should be valid JSON: %v", err)
	}
	if delegateCheck["type"] != "ptp" {
		t.Errorf("Expected delegate type 'ptp', got '%v'", delegateCheck["type"])
	}
}

func TestParseConfig_DefaultAnnotationKey(t *testing.T) {
	input := `{
		"cniVersion": "1.0.0",
		"name": "tenant-routing",
		"type": "tenant-routing-wrapper",
		"kubeconfig": "/etc/cni/net.d/tenant-routing.kubeconfig",
		"delegate": {
			"type": "ptp"
		}
	}`

	conf, err := ParseConfig([]byte(input))
	if err != nil {
		t.Fatalf("Expected successful parse, got error: %v", err)
	}

	// Verify default annotation key is applied
	if conf.AnnotationKey != DefaultAnnotationKey {
		t.Errorf("Expected default AnnotationKey '%s', got '%s'", DefaultAnnotationKey, conf.AnnotationKey)
	}
}

func TestParseConfig_MissingDelegate(t *testing.T) {
	input := `{
		"cniVersion": "1.0.0",
		"name": "tenant-routing",
		"type": "tenant-routing-wrapper",
		"kubeconfig": "/etc/cni/net.d/tenant-routing.kubeconfig"
	}`

	_, err := ParseConfig([]byte(input))
	if err == nil {
		t.Fatal("Expected error for missing delegate, got nil")
	}

	expected := "delegate plugin configuration is required"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%s'", expected, err.Error())
	}
}

func TestParseConfig_MissingKubeconfig(t *testing.T) {
	input := `{
		"cniVersion": "1.0.0",
		"name": "tenant-routing",
		"type": "tenant-routing-wrapper",
		"delegate": {
			"type": "ptp"
		}
	}`

	_, err := ParseConfig([]byte(input))
	if err == nil {
		t.Fatal("Expected error for missing kubeconfig, got nil")
	}

	expected := "kubeconfig path is required"
	if err.Error() != expected {
		t.Errorf("Expected error '%s', got '%s'", expected, err.Error())
	}
}

func TestParseConfig_RelativeKubeconfigPath(t *testing.T) {
	testCases := []struct {
		name string
		path string
	}{
		{"relative path", "etc/cni/kubeconfig"},
		{"dot relative", "./kubeconfig"},
		{"parent relative", "../kubeconfig"},
		{"home relative", "~/kubeconfig"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := `{
				"cniVersion": "1.0.0",
				"name": "tenant-routing",
				"type": "tenant-routing-wrapper",
				"kubeconfig": "` + tc.path + `",
				"delegate": {
					"type": "ptp"
				}
			}`

			_, err := ParseConfig([]byte(input))
			if err == nil {
				t.Fatalf("Expected error for relative kubeconfig path '%s', got nil", tc.path)
			}

			// Error should mention absolute path requirement
			if err.Error() == "" {
				t.Error("Expected non-empty error message")
			}
		})
	}
}

func TestParseConfig_InvalidJSON(t *testing.T) {
	input := `{
		"cniVersion": "1.0.0",
		"name": "tenant-routing",
		"type": "tenant-routing-wrapper",
		"kubeconfig": "/etc/cni/kubeconfig"
		// Missing closing brace
	`

	_, err := ParseConfig([]byte(input))
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
}

func TestParseConfig_EmptyDelegateObject(t *testing.T) {
	input := `{
		"cniVersion": "1.0.0",
		"name": "tenant-routing",
		"type": "tenant-routing-wrapper",
		"kubeconfig": "/etc/cni/net.d/tenant-routing.kubeconfig",
		"delegate": {}
	}`

	conf, err := ParseConfig([]byte(input))
	if err != nil {
		t.Fatalf("Expected successful parse for empty delegate object, got error: %v", err)
	}

	// Empty object is still valid JSON, just has no fields
	if len(conf.Delegate) == 0 {
		t.Error("Expected Delegate to contain empty object JSON")
	}
}

func TestGetDelegateConfig(t *testing.T) {
	input := `{
		"cniVersion": "1.0.0",
		"name": "tenant-routing",
		"type": "tenant-routing-wrapper",
		"kubeconfig": "/etc/cni/net.d/tenant-routing.kubeconfig",
		"delegate": {
			"type": "ptp",
			"ipMasq": true,
			"ipam": {
				"type": "host-local",
				"subnet": "10.200.0.0/16",
				"routes": [
					{"dst": "0.0.0.0/0"}
				]
			}
		}
	}`

	conf, err := ParseConfig([]byte(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	delegateConfig := conf.GetDelegateConfig()
	if len(delegateConfig) == 0 {
		t.Fatal("Expected non-empty delegate config")
	}

	// Verify delegate config is valid JSON
	var delegate map[string]interface{}
	if err := json.Unmarshal(delegateConfig, &delegate); err != nil {
		t.Fatalf("Delegate config should be valid JSON: %v", err)
	}

	// Verify nested structure is preserved
	if delegate["type"] != "ptp" {
		t.Errorf("Expected type 'ptp', got '%v'", delegate["type"])
	}
	if delegate["ipMasq"] != true {
		t.Errorf("Expected ipMasq true, got '%v'", delegate["ipMasq"])
	}

	// Verify IPAM section
	ipam, ok := delegate["ipam"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected ipam to be an object")
	}
	if ipam["type"] != "host-local" {
		t.Errorf("Expected ipam type 'host-local', got '%v'", ipam["type"])
	}
}

func TestParseConfig_SQLInjectionAttempt(t *testing.T) {
	// Test that malicious path values are rejected by absolute path check
	input := `{
		"cniVersion": "1.0.0",
		"name": "tenant-routing",
		"type": "tenant-routing-wrapper",
		"kubeconfig": "'; DROP TABLE pods; --",
		"delegate": {
			"type": "ptp"
		}
	}`

	_, err := ParseConfig([]byte(input))
	if err == nil {
		t.Fatal("Expected error for malicious kubeconfig path, got nil")
	}

	// Should fail on absolute path check, not SQL injection (we don't use SQL here)
	// but this demonstrates input validation catches malicious strings
}

func TestParseConfig_PathTraversalAttempt(t *testing.T) {
	// Test that path traversal attempts are caught
	input := `{
		"cniVersion": "1.0.0",
		"name": "tenant-routing",
		"type": "tenant-routing-wrapper",
		"kubeconfig": "../../../../../../etc/shadow",
		"delegate": {
			"type": "ptp"
		}
	}`

	_, err := ParseConfig([]byte(input))
	if err == nil {
		t.Fatal("Expected error for path traversal attempt, got nil")
	}

	// Should fail because path is not absolute
	expected := "kubeconfig path must be absolute"
	if err.Error()[:len(expected)] != expected {
		t.Errorf("Expected error starting with '%s', got '%s'", expected, err.Error())
	}
}

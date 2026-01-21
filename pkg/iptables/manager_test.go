package iptables

import (
	"testing"
)

// TestValidateFwmark tests fwmark validation logic
func TestValidateFwmark(t *testing.T) {
	tests := []struct {
		name    string
		fwmark  string
		wantErr bool
	}{
		{
			name:    "valid tenant A mark",
			fwmark:  "0x10",
			wantErr: false,
		},
		{
			name:    "valid tenant B mark",
			fwmark:  "0x20",
			wantErr: false,
		},
		{
			name:    "valid tenant A mark uppercase",
			fwmark:  "0X10",
			wantErr: false,
		},
		{
			name:    "valid tenant B mark with spaces",
			fwmark:  " 0x20 ",
			wantErr: false,
		},
		{
			name:    "invalid mark - cilium range",
			fwmark:  "0x0e00",
			wantErr: true,
		},
		{
			name:    "invalid mark - zero",
			fwmark:  "0x00",
			wantErr: true,
		},
		{
			name:    "invalid mark - empty",
			fwmark:  "",
			wantErr: true,
		},
		{
			name:    "invalid mark - arbitrary value",
			fwmark:  "0x99",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFwmark(tt.fwmark)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFwmark(%q) error = %v, wantErr %v", tt.fwmark, err, tt.wantErr)
			}
		})
	}
}

// TestAddMarkRule_Validation tests input validation for AddMarkRule
func TestAddMarkRule_Validation(t *testing.T) {
	tests := []struct {
		name    string
		podIP   string
		fwmark  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty pod IP",
			podIP:   "",
			fwmark:  "0x10",
			wantErr: true,
			errMsg:  "podIP cannot be empty",
		},
		{
			name:    "whitespace only pod IP",
			podIP:   "   ",
			fwmark:  "0x10",
			wantErr: true,
			errMsg:  "podIP cannot be empty",
		},
		{
			name:    "invalid IP format - not an IP",
			podIP:   "not-an-ip",
			fwmark:  "0x10",
			wantErr: true,
			errMsg:  "invalid IP address format",
		},
		{
			name:    "invalid IP format - out of range",
			podIP:   "999.999.999.999",
			fwmark:  "0x10",
			wantErr: true,
			errMsg:  "invalid IP address format",
		},
		{
			name:    "invalid IP format - command injection attempt",
			podIP:   "10.0.0.1; rm -rf /",
			fwmark:  "0x10",
			wantErr: true,
			errMsg:  "invalid IP address format",
		},
		{
			name:    "invalid fwmark",
			podIP:   "10.200.1.5",
			fwmark:  "0x99",
			wantErr: true,
			errMsg:  "invalid fwmark",
		},
		{
			name:    "cilium conflict fwmark",
			podIP:   "10.200.1.5",
			fwmark:  "0x0e00",
			wantErr: true,
			errMsg:  "avoid Cilium conflicts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AddMarkRule(tt.podIP, tt.fwmark)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddMarkRule(%q, %q) error = %v, wantErr %v", tt.podIP, tt.fwmark, err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				// Check error message contains expected substring
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("AddMarkRule(%q, %q) error message = %q, want substring %q",
						tt.podIP, tt.fwmark, err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestRuleExists_Validation tests input validation for RuleExists
func TestRuleExists_Validation(t *testing.T) {
	tests := []struct {
		name    string
		podIP   string
		fwmark  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty pod IP",
			podIP:   "",
			fwmark:  "0x10",
			wantErr: true,
			errMsg:  "podIP cannot be empty",
		},
		{
			name:    "invalid IP format",
			podIP:   "not-an-ip",
			fwmark:  "0x10",
			wantErr: true,
			errMsg:  "invalid IP address format",
		},
		{
			name:    "invalid fwmark",
			podIP:   "10.200.1.5",
			fwmark:  "0x99",
			wantErr: true,
			errMsg:  "invalid fwmark",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RuleExists(tt.podIP, tt.fwmark)
			if (err != nil) != tt.wantErr {
				t.Errorf("RuleExists(%q, %q) error = %v, wantErr %v", tt.podIP, tt.fwmark, err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("RuleExists(%q, %q) error message = %q, want substring %q",
						tt.podIP, tt.fwmark, err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestDeleteMarkRule_Validation tests input validation for DeleteMarkRule
func TestDeleteMarkRule_Validation(t *testing.T) {
	tests := []struct {
		name    string
		podIP   string
		fwmark  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty pod IP",
			podIP:   "",
			fwmark:  "0x10",
			wantErr: true,
			errMsg:  "podIP cannot be empty",
		},
		{
			name:    "whitespace only pod IP",
			podIP:   "   ",
			fwmark:  "0x20",
			wantErr: true,
			errMsg:  "podIP cannot be empty",
		},
		{
			name:    "invalid IP format - not an IP",
			podIP:   "not-an-ip",
			fwmark:  "0x10",
			wantErr: true,
			errMsg:  "invalid IP address format",
		},
		{
			name:    "invalid IP format - command injection attempt",
			podIP:   "10.0.0.1 && curl evil.com",
			fwmark:  "0x10",
			wantErr: true,
			errMsg:  "invalid IP address format",
		},
		{
			name:    "invalid fwmark",
			podIP:   "10.200.1.5",
			fwmark:  "0x99",
			wantErr: true,
			errMsg:  "invalid fwmark",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DeleteMarkRule(tt.podIP, tt.fwmark)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteMarkRule(%q, %q) error = %v, wantErr %v", tt.podIP, tt.fwmark, err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				// Check error message contains expected substring
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("DeleteMarkRule(%q, %q) error message = %q, want substring %q",
						tt.podIP, tt.fwmark, err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// contains checks if s contains substr (case-sensitive)
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Note: Integration tests for actual iptables operations require:
// 1. Root/CAP_NET_ADMIN permissions
// 2. Network namespace isolation
// 3. iptables binary installed
//
// These tests should be run separately in a controlled environment:
//
// Integration test pseudocode:
//
// func TestAddMarkRule_Integration(t *testing.T) {
//     if os.Getuid() != 0 {
//         t.Skip("Integration tests require root privileges")
//     }
//
//     // Create isolated network namespace
//     ns, err := netns.New()
//     defer ns.Close()
//
//     // Test 1: Add rule first time
//     err = AddMarkRule("10.200.1.5", "0x10")
//     if err != nil {
//         t.Fatalf("First AddMarkRule failed: %v", err)
//     }
//
//     // Verify rule exists using iptables -C
//     // iptables -t mangle -C PREROUTING -s 10.200.1.5 -j MARK --set-mark 0x10
//
//     // Test 2: Add same rule again (idempotency)
//     err = AddMarkRule("10.200.1.5", "0x10")
//     if err != nil {
//         t.Errorf("Second AddMarkRule (idempotent) failed: %v", err)
//     }
//
//     // Verify only one rule exists (no duplicates)
//     // iptables -t mangle -S PREROUTING | grep "10.200.1.5" | wc -l should be 1
//
//     // Test 3: Delete rule
//     err = DeleteMarkRule("10.200.1.5", "0x10")
//     if err != nil {
//         t.Errorf("DeleteMarkRule failed: %v", err)
//     }
//
//     // Verify rule is gone
//     // iptables -t mangle -C PREROUTING -s 10.200.1.5 -j MARK --set-mark 0x10 should fail
//
//     // Test 4: Delete non-existent rule (idempotency)
//     err = DeleteMarkRule("10.200.1.5", "0x10")
//     if err != nil {
//         t.Errorf("DeleteMarkRule (idempotent) failed: %v", err)
//     }
// }
//
// func TestMarkRule_TenantIsolation(t *testing.T) {
//     if os.Getuid() != 0 {
//         t.Skip("Integration tests require root privileges")
//     }
//
//     // Create namespace
//     ns, err := netns.New()
//     defer ns.Close()
//
//     // Add rules for both tenants
//     AddMarkRule("10.200.1.5", "0x10")  // Tenant A
//     AddMarkRule("10.200.1.6", "0x20")  // Tenant B
//
//     // Verify both rules exist independently
//     // iptables -t mangle -S PREROUTING should show both
//
//     // Delete Tenant A rule
//     DeleteMarkRule("10.200.1.5", "0x10")
//
//     // Verify Tenant B rule still exists
//     // iptables -t mangle -C PREROUTING -s 10.200.1.6 -j MARK --set-mark 0x20 should succeed
//
//     // Cleanup
//     DeleteMarkRule("10.200.1.6", "0x20")
// }

package iptables

import (
	"fmt"
	"net"
	"strings"

	"github.com/coreos/go-iptables/iptables"
)

const (
	// Allowed fwmark values to prevent Cilium conflicts
	// Cilium uses marks in ranges 0x0e00-0x0f00, 0x0200-0x0f00
	FwmarkTenantA = "0x10" // Tenant A routing mark
	FwmarkTenantB = "0x20" // Tenant B routing mark

	// iptables configuration
	tableNameMangle = "mangle"
	chainPrerouting = "PREROUTING"
)

// Manager handles iptables rules for tenant routing via fwmark
// Provides idempotent operations for adding and removing marking rules
type Manager struct {
	ipt *iptables.IPTables
}

// NewManager creates a new iptables manager instance
// Returns error if iptables initialization fails (requires root/CAP_NET_ADMIN)
func NewManager() (*Manager, error) {
	ipt, err := iptables.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize iptables: %w", err)
	}

	return &Manager{ipt: ipt}, nil
}

// validateFwmark ensures fwmark value is allowed (prevents Cilium conflicts)
// Only 0x10 (Tenant A) and 0x20 (Tenant B) are permitted
func validateFwmark(fwmark string) error {
	// Normalize to lowercase for comparison
	normalized := strings.ToLower(strings.TrimSpace(fwmark))

	if normalized != FwmarkTenantA && normalized != FwmarkTenantB {
		return fmt.Errorf("invalid fwmark %q: must be %s (Tenant A) or %s (Tenant B) to avoid Cilium conflicts",
			fwmark, FwmarkTenantA, FwmarkTenantB)
	}

	return nil
}

// AddMarkRule adds iptables rule to mark packets from podIP with fwmark
// Idempotent: succeeds if rule already exists
// Rule format: iptables -t mangle -A PREROUTING -s podIP -j MARK --set-mark fwmark
//
// Example:
//
//	err := mgr.AddMarkRule("10.200.1.5", "0x10")
//	// Creates: iptables -t mangle -A PREROUTING -s 10.200.1.5 -j MARK --set-mark 0x10
func AddMarkRule(podIP, fwmark string) error {
	// Validate pod IP is not empty (before iptables initialization)
	if strings.TrimSpace(podIP) == "" {
		return fmt.Errorf("podIP cannot be empty")
	}

	// Security: Validate IP format to prevent injection attacks
	if net.ParseIP(podIP) == nil {
		return fmt.Errorf("invalid IP address format: %s", podIP)
	}

	// Security: Validate fwmark to prevent conflicts with Cilium (before iptables initialization)
	if err := validateFwmark(fwmark); err != nil {
		return err
	}

	// Initialize iptables manager (requires iptables binary and CAP_NET_ADMIN)
	mgr, err := NewManager()
	if err != nil {
		return err
	}

	// Build rule specification
	rulespec := []string{
		"-s", podIP,
		"-j", "MARK",
		"--set-mark", fwmark,
	}

	// Check if rule already exists (idempotency)
	exists, err := mgr.ipt.Exists(tableNameMangle, chainPrerouting, rulespec...)
	if err != nil {
		return fmt.Errorf("failed to check if rule exists for podIP %s: %w", podIP, err)
	}

	if exists {
		// Rule already exists, success (idempotent behavior)
		return nil
	}

	// Add the rule
	if err := mgr.ipt.Append(tableNameMangle, chainPrerouting, rulespec...); err != nil {
		return fmt.Errorf("failed to add mark rule for podIP %s with fwmark %s: %w", podIP, fwmark, err)
	}

	return nil
}

// RuleExists checks if an iptables rule exists for the given podIP and fwmark
// Used during CHECK operations to verify expected state matches actual state
//
// Returns:
//   - true, nil: Rule exists
//   - false, nil: Rule does not exist
//   - false, err: Error checking rule existence
func RuleExists(podIP, fwmark string) (bool, error) {
	// Validate pod IP is not empty
	if strings.TrimSpace(podIP) == "" {
		return false, fmt.Errorf("podIP cannot be empty")
	}

	// Security: Validate IP format
	if net.ParseIP(podIP) == nil {
		return false, fmt.Errorf("invalid IP address format: %s", podIP)
	}

	// Security: Validate fwmark
	if err := validateFwmark(fwmark); err != nil {
		return false, err
	}

	// Initialize iptables manager
	mgr, err := NewManager()
	if err != nil {
		return false, err
	}

	// Build rule specification
	rulespec := []string{
		"-s", podIP,
		"-j", "MARK",
		"--set-mark", fwmark,
	}

	// Check if rule exists
	exists, err := mgr.ipt.Exists(tableNameMangle, chainPrerouting, rulespec...)
	if err != nil {
		return false, fmt.Errorf("failed to check if rule exists for podIP %s: %w", podIP, err)
	}

	return exists, nil
}

// DeleteMarkRule removes iptables rule that marks packets from podIP with fwmark
// Idempotent: succeeds even if rule does not exist
// Rule format: iptables -t mangle -D PREROUTING -s podIP -j MARK --set-mark fwmark
//
// Example:
//
//	err := mgr.DeleteMarkRule("10.200.1.5", "0x10")
//	// Removes: iptables -t mangle -D PREROUTING -s 10.200.1.5 -j MARK --set-mark 0x10
func DeleteMarkRule(podIP, fwmark string) error {
	// Validate pod IP is not empty (before iptables initialization)
	if strings.TrimSpace(podIP) == "" {
		return fmt.Errorf("podIP cannot be empty")
	}

	// Security: Validate IP format to prevent injection attacks
	if net.ParseIP(podIP) == nil {
		return fmt.Errorf("invalid IP address format: %s", podIP)
	}

	// Security: Validate fwmark to prevent accidental deletion of system rules (before iptables initialization)
	if err := validateFwmark(fwmark); err != nil {
		return err
	}

	// Initialize iptables manager (requires iptables binary and CAP_NET_ADMIN)
	mgr, err := NewManager()
	if err != nil {
		return err
	}

	// Build rule specification
	rulespec := []string{
		"-s", podIP,
		"-j", "MARK",
		"--set-mark", fwmark,
	}

	// Check if rule exists before attempting deletion
	exists, err := mgr.ipt.Exists(tableNameMangle, chainPrerouting, rulespec...)
	if err != nil {
		return fmt.Errorf("failed to check if rule exists for podIP %s: %w", podIP, err)
	}

	if !exists {
		// Rule doesn't exist, success (idempotent behavior)
		return nil
	}

	// Delete the rule
	if err := mgr.ipt.Delete(tableNameMangle, chainPrerouting, rulespec...); err != nil {
		return fmt.Errorf("failed to delete mark rule for podIP %s with fwmark %s: %w", podIP, fwmark, err)
	}

	return nil
}

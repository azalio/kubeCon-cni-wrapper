// Package result provides utilities for parsing and extracting information from CNI plugin results.
//
// This package is used in the tenant-routing-wrapper CNI plugin to extract pod IP addresses
// from delegate CNI plugin results (e.g., ptp, bridge, cilium-cni). The extracted IP is then
// used for setting up tenant-specific routing rules via fwmark and policy routing.
//
// Usage in CNI chaining workflow:
//
//  1. Wrapper CNI calls delegate plugin (ptp, bridge, etc.)
//  2. Delegate returns CNI Result with assigned IP addresses
//  3. ExtractPodIP() extracts the first IPv4 address
//  4. Wrapper uses this IP for iptables fwmark rules
//  5. Policy routing directs traffic to tenant-specific gateway
//
// Supported CNI Result versions:
//  - CNI 1.0.0 (types100.Result)
//  - CNI 0.4.0 (types040.Result)
package result

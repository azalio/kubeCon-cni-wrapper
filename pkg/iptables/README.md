# pkg/iptables

Idempotent iptables rule management for tenant-aware routing via fwmark.

## Overview

This package provides safe, idempotent operations for managing iptables mangle/PREROUTING rules that mark packets with fwmark values. Used by the CNI wrapper to implement tenant-specific routing.

## Features

- **Idempotent operations**: AddMarkRule and DeleteMarkRule can be called multiple times safely
- **Conflict prevention**: Validates fwmark values to avoid Cilium conflicts (only 0x10 and 0x20 allowed)
- **Error handling**: Comprehensive validation before iptables operations
- **Production-ready**: Uses coreos/go-iptables library for safe iptables interaction

## Usage

```go
import "github.com/azalio/bm.azalio.net/pkg/iptables"

// Add fwmark rule for Tenant A pod
err := iptables.AddMarkRule("10.200.1.5", "0x10")
if err != nil {
    log.Fatalf("Failed to add mark rule: %v", err)
}

// Delete fwmark rule when pod terminates
err = iptables.DeleteMarkRule("10.200.1.5", "0x10")
if err != nil {
    log.Fatalf("Failed to delete mark rule: %v", err)
}
```

## Tenant Routing Mapping

| Tenant | fwmark | Gateway | IP |
|--------|--------|---------|-----|
| Tenant A | 0x10 | router1 | 10.10.10.107 |
| Tenant B | 0x20 | router2 | 10.10.10.154 |

Routing tables are configured via `scripts/tenant-routing-setup.sh`:

```bash
# Tenant A (fwmark 0x10 → table 100 → router1)
ip rule add fwmark 0x10 table 100
ip route add default via 10.10.10.107 table 100

# Tenant B (fwmark 0x20 → table 200 → router2)
ip rule add fwmark 0x20 table 200
ip route add default via 10.10.10.154 table 200
```

## Requirements

### Runtime

- Linux kernel with iptables support
- `iptables` binary in PATH
- Root privileges or CAP_NET_ADMIN capability

### Testing

**Unit tests** (validation logic only):
```bash
go test ./pkg/iptables/
```

**Integration tests** (requires root and iptables):
```bash
sudo go test ./pkg/iptables/ -tags=integration
```

Integration tests require:
1. Root privileges (`sudo`)
2. iptables binary installed
3. Network namespace support (for isolation)

## Implementation Details

### Idempotency

Both `AddMarkRule` and `DeleteMarkRule` check if the rule exists before modifying iptables:

- **AddMarkRule**: Uses `iptables.Exists()` before `Append()`. Returns success if rule already exists.
- **DeleteMarkRule**: Uses `iptables.Exists()` before `Delete()`. Returns success if rule doesn't exist.

This ensures CNI ADD/DEL can be called multiple times safely (e.g., during kubelet restart, network re-initialization).

### Security

**fwmark validation**: Only 0x10 and 0x20 are allowed to prevent conflicts with Cilium's fwmark ranges:
- Cilium uses 0x0e00-0x0f00 for identity-based routing
- Cilium uses 0x0200-0x0f00 for various network policies

**Input validation**: Performed BEFORE iptables initialization to fail fast on invalid inputs.

### Rule Format

```
iptables -t mangle -A PREROUTING -s <podIP> -j MARK --set-mark <fwmark>
```

**Example:**
```
iptables -t mangle -A PREROUTING -s 10.200.1.5 -j MARK --set-mark 0x10
```

## Testing on KubeCon Demo Infrastructure

The integration tests can be run on the worker nodes:

```bash
# SSH to worker node
ssh -J root@bm.azalio.net ubuntu@10.10.10.181

# Install Go (if not present)
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Clone repository
git clone https://github.com/azalio/bm.azalio.net.git
cd bm.azalio.net

# Run integration tests
sudo go test ./pkg/iptables/ -tags=integration -v
```

## Troubleshooting

**Error: "executable file not found in $PATH"**
- Install iptables: `sudo apt-get install iptables` (Debian/Ubuntu)
- Verify: `which iptables`

**Error: "permission denied"**
- Run with sudo: `sudo go test ./pkg/iptables/`
- Or grant CAP_NET_ADMIN: `sudo setcap cap_net_admin+ep /path/to/test.binary`

**Error: "invalid fwmark"**
- Only 0x10 (Tenant A) and 0x20 (Tenant B) are allowed
- Check tenant annotation value in pod spec

## References

- [coreos/go-iptables](https://github.com/coreos/go-iptables) - Safe iptables library
- [Cilium fwmark usage](https://docs.cilium.io/en/stable/network/kubernetes/policy/#firewall-marks) - Conflict prevention
- [Linux policy routing](https://www.kernel.org/doc/html/latest/networking/policy-routing.html) - fwmark-based routing

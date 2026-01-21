# CNI Wrapper for Tenant-Aware Routing

A CNI meta-plugin that enables tenant-aware egress routing in Kubernetes by reading pod annotations and setting host-level iptables fwmark rules.

**Presented at:** KubeCon + CloudNativeCon India 2026
**Talk:** "Network Decisions Before Datapath: CNI Chaining for External IPAM and Tenant Routing"

## Architecture

```
    TENANT-AWARE ROUTING VIA CNI CHAINING

    Pod A (fwmark: 0x10)          Pod B (fwmark: 0x20)
           │                             │
           └──────────┬──────────────────┘
                      ▼
    ┌─────────────────────────────────────────┐
    │         CNI WRAPPER                     │
    │   • Reads annotation from K8s API       │
    │   • Delegates to ptp plugin             │
    │   • Sets iptables MARK rule on host     │
    └─────────────────────────────────────────┘
                      │
         ┌────────────┴────────────┐
         ▼                         ▼
    iptables MARK 0x10       iptables MARK 0x20
         │                         │
         ▼                         ▼
    ip rule → table 100      ip rule → table 200
         │                         │
         ▼                         ▼
    ┌──────────┐             ┌──────────┐
    │ Gateway A│             │ Gateway B│
    └──────────┘             └──────────┘

    TIMING: Kubernetes stops here ──▶ Linux takes over
                                CNI execution
```

## The Problem

Some network decisions must happen **before** CNI execution, not inside Kubernetes API:

1. **External IPAM**: When corporate IPAM or ISP delegation is the source of truth
2. **Tenant-aware routing**: Different customers require different egress gateways

Standard approaches fail:
- **Controllers**: Race with kubelet; reconcile after pod creation
- **Webhooks**: Modify pod specs, but CNI ignores most of it
- **Multus**: Parallel networks, not routing injection

## The Pattern

CNI chaining allows injecting routing decisions during CNI execution — the **only** moment when this is possible without racing kubelet.

The wrapper:
1. Receives CNI ADD request from kubelet
2. Delegates to underlying CNI (ptp) to create veth pair
3. Reads `tenant.routing/fwmark` annotation from pod (or namespace)
4. Sets iptables MARK rule on host: `-t mangle -A PREROUTING -s <pod-ip> -j MARK --set-mark <fwmark>`
5. Returns CNI result

Pre-configured `ip rule` entries on each node route marked traffic to tenant-specific gateways.

## Building

```bash
# Build the CNI plugin
go build -o bin/tenant-routing-wrapper ./cmd/tenant-routing-wrapper/

# Run tests
go test ./...
```

## Deployment

### 1. Set up routing tables on each node

```bash
# Run on each K8s node
sudo ./scripts/tenant-routing-setup.sh
```

This creates:
- Routing tables `tenant-a` (100) and `tenant-b` (200)
- IP rules: `fwmark 0x10 → table 100`, `fwmark 0x20 → table 200`
- Default routes via tenant gateways

### 2. Deploy CNI wrapper

Copy `tenant-routing-wrapper` binary to `/opt/cni/bin/` on each node.

### 3. Configure CNI chain

Example `/etc/cni/net.d/10-tenant-routing.conflist`:

```json
{
  "cniVersion": "1.0.0",
  "name": "tenant-routing",
  "plugins": [
    {
      "type": "tenant-routing-wrapper",
      "kubeconfig": "/etc/kubernetes/kubelet.conf",
      "annotationKey": "tenant.routing/fwmark",
      "delegate": {
        "type": "ptp",
        "ipam": {
          "type": "host-local",
          "subnet": "10.200.0.0/16"
        }
      }
    },
    {
      "type": "cilium-cni"
    }
  ]
}
```

### 4. Deploy test pods

```bash
kubectl apply -f scripts/manifests/tenant-a-pod.yaml
kubectl apply -f scripts/manifests/tenant-b-pod.yaml
```

## Demo

See [bm.azalio.net](https://github.com/azalio/bm.azalio.net) for full demo infrastructure with Terraform + Proxmox.

## Configuration

### Pod annotation

```yaml
metadata:
  annotations:
    tenant.routing/fwmark: "0x10"  # or "0x20"
```

### Supported fwmark values

- `0x10` — Tenant A (routes via Gateway A)
- `0x20` — Tenant B (routes via Gateway B)

Values chosen to avoid conflict with Cilium's fwmark range (0x200-0xf00).

## Trade-offs

**You gain:**
- Local control over routing decisions
- Works with off-the-shelf CNI plugins
- No fork or deep customization required
- Fast iteration on routing logic

**You pay:**
- Ownership of ~200 lines of glue code
- Debugging complexity (multiple plugin layers)
- Plugin order sensitivity
- On-call responsibility

## When NOT to use

- High pod churn (iptables rule management overhead)
- When standard CNI features are sufficient
- If you can't name the person who owns this code

## Author

Mikhail Petrov (@azalio)

## License

MIT

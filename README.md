# CNI Wrapper for Tenant-Aware Routing

A CNI meta-plugin that enables tenant-aware egress routing in Kubernetes by reading pod annotations and setting host-level iptables fwmark rules.

**Context:** Supporting code for a KubeCon + CloudNativeCon India 2026 CFP submission
**Talk title:** "Network Decisions Before Datapath: CNI Chaining for External IPAM and Tenant Routing"

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

Some networking decisions need to happen at CNI execution time (during `ADD`), not later via reconciliation:

1. External IPAM: when a separate system is the source of truth
2. Tenant-aware egress routing: when different workloads need different gateways

Common alternatives have drawbacks in this timing window:
- Controllers apply changes after pod creation and can race kubelet
- Admission webhooks can change pod specs, but most of that is not consumed by CNI
- Multus focuses on additional interfaces, not changing routing for the primary one

## The Pattern

CNI chaining provides a practical injection point during CNI execution.

The wrapper:
1. Receives a CNI `ADD` call from kubelet
2. Delegates to an underlying CNI (e.g. `ptp`) to create the interface and get the pod IP
3. Reads `tenant.routing/fwmark` from the pod (or namespace)
4. Installs an iptables `MARK` rule on the host: `-t mangle -A PREROUTING -s <pod-ip> -j MARK --set-mark <fwmark>`
5. Returns the delegated CNI result

Policy routing (`ip rule`/tables) is configured separately on each node to steer marked traffic via tenant-specific gateways.

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

A full lab (Terraform/Ansible/Proxmox) exists in a separate repo; it is not included here. This repo focuses on the CNI wrapper implementation and reproducible local primitives/scripts.

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

**Pros:**
- Works with off-the-shelf CNI plugins
- Keeps routing intent close to CNI execution

**Cons:**
- Extra moving parts (multiple plugin layers)
- Order sensitivity in plugin chains
- Operational ownership (debugging, lifecycle)

## When NOT to use

- High pod churn (iptables rule management overhead)
- When standard CNI features are sufficient
- When there is no clear owner/on-call for this glue

## Author

Mikhail Petrov (@azalio)

## License

MIT

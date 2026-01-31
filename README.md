# CNI Chaining for Tenant Routing

Demo repository for KubeCon India 2026 talk: **"Network Decisions Before Datapath: CNI Chaining for External IPAM and Tenant Routing"**

## The problem: routing can't wait

Most Kubernetes network tutorials show controllers reconciling state after pods exist. That works for many things. But egress routing is not one of them.

If your tenants need strict egress boundaries — different gateways, different NAT pools, compliance zones — the routing must be in place *before* the first packet leaves. A controller that reconciles seconds later is too late. The pod already sent traffic through the default gateway, bypassing your tenant isolation.

Admission webhooks can mutate pod specs, but CNI plugins don't read most of those fields. Multus adds extra interfaces, but doesn't change where the primary one routes traffic.

CNI chaining fixes this. You insert a small wrapper that runs during `CNI ADD`, reads an annotation, and sets up the routing path before the container starts. By the time the first packet hits the interface, policy routing is already in place.

This pattern isn't new — Istio CNI and GKE Dataplane v2 use the same mechanism — but it's rarely explained as something you can build yourself for your own use cases.

## How it works

This meta-plugin doesn't replace your CNI. It wraps it:

1. Calls your existing CNI (ptp, bridge, whatever) to create the interface and get an IP
2. Reads `tenant.routing/fwmark` annotation from the pod (falls back to namespace)
3. Adds an iptables MARK rule for that pod IP
4. Returns the original CNI result unchanged — your CNI does what it always did

The MARK integrates with standard Linux policy routing. Different marks hit different routing tables, different gateways.

```bash
Pod (fwmark 0x10) → iptables MARK 0x10 → table tenant-a → gateway A (10.10.10.151)
Pod (fwmark 0x20) → iptables MARK 0x20 → table tenant-b → gateway B (10.10.10.174)
```

## Quick start

Your CNI conflist must include `kubeconfig` pointing to a valid kubeconfig on the node (e.g. `/etc/kubernetes/kubelet.conf`). The wrapper needs API access to read pod annotations at `CNI ADD` time.

```bash
# Build static binary for Linux (cross-compile if building on macOS)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/tenant-routing-wrapper ./cmd/tenant-routing-wrapper/

# Set up routing tables on the node (configure GATEWAY_A/GATEWAY_B env vars for your routers)
sudo ./scripts/tenant-routing-setup.sh

# Copy binary to CNI directory
sudo cp bin/tenant-routing-wrapper /opt/cni/bin/

# Create test pods
kubectl apply -f scripts/manifests/
```

Run `traceroute` from the pods — they take completely different paths to the internet despite being on the same node.

## CNI config example

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
        "ipam": { "type": "host-local", "subnet": "10.200.0.0/16" }
      }
    }
  ]
}
```

`kubeconfig` is required — the wrapper needs API access to read pod annotations. Must be an absolute path.

## What's NOT in this repo

The lab environment with multiple routers, VMs, and policy routing topology lives in a separate repo. This one contains only the CNI plugin code that would run on a real cluster.

## Trade-offs

CNI chaining is powerful but it's another thing to debug. iptables rules don't always clean themselves up if a node crashes. Policy routing is unforgiving at scale.

But if you need routing decisions made at pod birth — not seconds later — this is where you make them. Not in a controller. Not in a webhook. In the CNI chain.

## Code structure

```bash
cmd/tenant-routing-wrapper/   # CNI entrypoint
pkg/config/                   # CNI config parsing and validation
pkg/delegate/                 # calls the underlying CNI
pkg/iptables/                 # MARK rule management
pkg/k8s/                      # annotation lookup (pod → namespace fallback)
pkg/result/                   # pod IP extraction from CNI result (0.4.0 + 1.0.0)
scripts/                      # node setup + test manifests
```

## Author

Mikhail Petrov ([@azalio](https://github.com/azalio))

## License

MIT

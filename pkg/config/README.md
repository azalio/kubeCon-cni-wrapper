# pkg/config

CNI configuration parsing and validation for the tenant-routing-wrapper plugin.

## Purpose

Parses CNI plugin configuration from stdin (JSON format) and validates security-critical fields before processing. This package ensures that:

1. Required fields are present (delegate plugin config, kubeconfig path)
2. Security constraints are enforced (absolute paths only to prevent path traversal)
3. Sensible defaults are applied (annotation key)
4. Delegate plugin configuration is preserved for chaining

## Usage

```go
import "github.com/azalio/bm.azalio.net/pkg/config"

// In your CNI plugin's cmdAdd/cmdDel/cmdCheck handlers
conf, err := config.ParseConfig(args.StdinData)
if err != nil {
    return fmt.Errorf("config parse failed: %w", err)
}

// Access configuration
kubeconfig := conf.Kubeconfig
annotationKey := conf.AnnotationKey

// Pass delegate config to next plugin
delegateConfig := conf.GetDelegateConfig()
```

## Configuration Schema

```json
{
  "cniVersion": "1.0.0",
  "name": "tenant-routing",
  "type": "tenant-routing-wrapper",
  "kubeconfig": "/etc/cni/net.d/tenant-routing.kubeconfig",
  "annotationKey": "tenant.routing/fwmark",
  "delegate": {
    "type": "ptp",
    "ipam": {
      "type": "host-local",
      "subnet": "10.200.0.0/16"
    }
  }
}
```

### Fields

- **kubeconfig** (required): Absolute path to kubeconfig file for Kubernetes API access
- **annotationKey** (optional): Pod annotation key containing fwmark value (default: `tenant.routing/fwmark`)
- **delegate** (required): Configuration for the next CNI plugin in the chain

## Security

- **Path Validation**: Kubeconfig path MUST be absolute (starts with `/`) to prevent path traversal attacks
- **Input Validation**: All JSON parsing errors are caught and returned with context
- **No Defaults for Paths**: Kubeconfig path must be explicitly provided, no default paths

## Testing

```bash
# Run all tests
go test ./pkg/config/

# Run with coverage
go test ./pkg/config/ -cover

# Run specific test
go test ./pkg/config/ -run TestParseConfig_ValidConfig -v
```

Current test coverage: 100%

# ZMS gRPC Plugins

This directory contains plugins migrated to the new gRPC-based plugin system using HashiCorp's go-plugin framework.

## Migration Overview

All plugins from the old shared library system ([plugins/](../plugins/)) have been migrated to the new gRPC-based system. The new system provides:

- **Better isolation**: Plugins run as separate processes
- **Version compatibility**: No Go version matching required
- **Crash resilience**: Plugin crashes don't affect the main application
- **Protocol Buffers**: Type-safe communication via gRPC

## Available Plugins

### 1. PostgreSQL (`psql`)
Stores Zabbix history data in PostgreSQL database.

**Features:**
- Saves history data to `performance.messages` table
- Connection pooling configuration
- Prometheus metrics for connection stats

**Configuration:**
```yaml
targets:
  - name: "postgres-target"
    type: "psql"
    connection: "postgres://user:password@localhost/dbname?sslmode=disable"
    options:
      max_conn: "10"
      max_idle: "5"
      max_conn_time: "1h"
      max_idle_time: "30m"
    exports:
      - "history"
```

### 2. Azure Table Storage (`azure_table`)
Stores Zabbix exports in Azure Table Storage.

**Features:**
- Saves history to `history` table
- Saves trends to `trends` table
- Uses itemID as partition key

**Configuration:**
```yaml
targets:
  - name: "azure-target"
    type: "azure_table"
    connection: "https://myaccount.table.core.windows.net/"
    exports:
      - "history"
      - "trends"
```

### 3. Prometheus Remote Write (`prometheus_remote_write`)
Writes Zabbix data to Prometheus via Remote Write protocol.

**Features:**
- Converts history to Prometheus time series
- Converts trends to separate metrics (min, max, avg, count)
- Timestamp ordering for compliance
- Only processes numeric values

**Configuration:**
```yaml
targets:
  - name: "prometheus-remote"
    type: "prometheus_remote_write"
    connection: "http://prometheus:9090/api/v1/write"
    exports:
      - "history"
      - "trends"
```

### 4. Print (`print`)
Outputs Zabbix data to stdout or stderr.

**Features:**
- Simple text output
- Configurable output destination
- Useful for debugging

**Configuration:**
```yaml
targets:
  - name: "print-target"
    type: "print"
    connection: "stdout"  # or "stderr"
    exports:
      - "history"
      - "trends"
```

### 5. GCP Cloud Monitor (`gcp_cloud_monitor`)
Sends Zabbix metrics to Google Cloud Monitoring.

**Features:**
- Creates custom metrics in GCP
- Only processes numeric values (float/unsigned)
- Configurable credentials

**Configuration:**
```yaml
targets:
  - name: "gcp-target"
    type: "gcp_cloud_monitor"
    connection: ""  # Uses default credentials
    options:
      credentials_file: "/path/to/service-account.json"  # Optional
    exports:
      - "history"
```

### 6. Prometheus Pushgateway (`prometheus_pushgateway`)
Pushes Zabbix metrics to Prometheus Pushgateway.

**Features:**
- Creates Prometheus gauges for history and trends
- Configurable job name
- Per-host instance grouping

**Configuration:**
```yaml
targets:
  - name: "pushgateway-target"
    type: "prometheus_pushgateway"
    connection: "http://pushgateway:9091"
    options:
      job_name: "zabbix_export"  # Optional, defaults to "zms_export"
    exports:
      - "history"
      - "trends"
```

## Building Plugins

### Manual Build

Build each plugin as a standalone executable:

```bash
# PostgreSQL plugin
cd plugins_grpc/psql
go build -o psql .

# Azure Table plugin
cd plugins_grpc/azure_table
go build -o azure_table .

# And so on...
```

### Using Docker

Use the Docker plugin builder for consistent builds:

```bash
# Build the Docker image
make docker-plugin-builder

# Build all gRPC plugins
docker run -v $(pwd)/plugins_grpc:/plugins -v $(pwd)/bin:/output zms-plugin-builder:latest
```

## Plugin Structure

All plugins follow this structure:

```go
package main

import (
    "context"
    "log"

    "github.com/hashicorp/go-plugin"
    pluginPkg "szuro.net/zms/pkg/plugin"
    "szuro.net/zms/proto"
)

const PLUGIN_NAME = "my_plugin"

type MyPlugin struct {
    proto.UnimplementedObserverServiceServer
    pluginPkg.BaseObserverGRPC
    // Custom fields...
}

func NewMyPlugin() *MyPlugin {
    return &MyPlugin{
        BaseObserverGRPC: *pluginPkg.NewBaseObserverGRPC(),
    }
}

func (p *MyPlugin) Initialize(ctx context.Context, req *proto.InitializeRequest) (*proto.InitializeResponse, error) {
    resp, err := p.BaseObserverGRPC.Initialize(ctx, req)
    if err != nil {
        return resp, err
    }

    p.PluginName = PLUGIN_NAME

    // Custom initialization...

    return &proto.InitializeResponse{Success: true}, nil
}

func (p *MyPlugin) SaveHistory(ctx context.Context, req *proto.SaveHistoryRequest) (*proto.SaveResponse, error) {
    history := p.FilterHistory(req.History)

    // Process history...

    return &proto.SaveResponse{
        Success: true,
        RecordsProcessed: int64(len(history)),
    }, nil
}

func main() {
    impl := NewMyPlugin()

    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: pluginPkg.Handshake,
        Plugins: map[string]plugin.Plugin{
            "observer": &pluginPkg.ObserverPlugin{Impl: impl},
        },
        GRPCServer: plugin.DefaultGRPCServer,
    })
}
```

## Key Differences from Old Plugins

| Aspect | Old System | New System |
|--------|-----------|------------|
| Build | `-buildmode=plugin` to `.so` | Standalone executable |
| Loading | `plugin.Open()` | Process spawning via go-plugin |
| Communication | Direct function calls | gRPC |
| Interface | `plugin.Observer` | `proto.ObserverServiceServer` |
| Base | `plugin.BaseObserverImpl` | `pluginPkg.BaseObserverGRPC` |
| Isolation | Same process | Separate process |
| Crash handling | Affects main app | Isolated |

## Migration Checklist

When migrating a plugin:

- [x] Change package to `main`
- [x] Embed `proto.UnimplementedObserverServiceServer`
- [x] Embed `pluginPkg.BaseObserverGRPC` instead of `BaseObserverImpl`
- [x] Update Initialize signature to accept `context.Context` and `*proto.InitializeRequest`
- [x] Update Save* methods to accept context and proto requests, return `*proto.SaveResponse`
- [x] Use `FilterHistory()`, `FilterTrends()`, `FilterEvents()` helper methods
- [x] Convert from `plugin.Observer` to gRPC service
- [x] Add `main()` function with `plugin.Serve()`
- [x] Build as executable (not shared library)

## Testing Plugins

Test plugins with ZMS:

```bash
# Build plugin
cd plugins_grpc/print
go build -o print .

# Place in plugins directory
mkdir -p ../../bin/plugins
cp print ../../bin/plugins/

# Update zmsd.yaml to use the plugin
# Run ZMS
../../zmsd -c zmsd-test.yaml
```

## Troubleshooting

**Plugin not found:**
- Ensure the executable is in the `plugins_dir` configured in `zmsd.yaml`
- Verify the executable has execute permissions: `chmod +x plugin_name`

**Handshake errors:**
- Ensure the plugin uses the correct `pluginPkg.Handshake` config
- Rebuild the plugin against the same ZMS version

**Connection failures:**
- Check that the plugin implements all required gRPC methods
- Verify proto definitions match between plugin and ZMS

**Data not being processed:**
- Check that the correct export types are enabled in config
- Verify filter configuration allows the data through
- Check plugin logs for errors

## Development Resources

- Main documentation: [CLAUDE.md](../CLAUDE.md)
- Plugin interface: [pkg/plugin/grpc_plugin.go](../pkg/plugin/grpc_plugin.go)
- Base observer: [pkg/plugin/grpc_base_observer.go](../pkg/plugin/grpc_base_observer.go)
- Proto definitions: [proto/observer.proto](../proto/observer.proto)
- Example plugin: [examples/plugins/log_print_grpc/](../examples/plugins/log_print_grpc/)

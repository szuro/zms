# ZMS Plugins

This directory contains plugins for ZMS using HashiCorp's go-plugin framework.

## Plugin System

ZMS uses a gRPC-based plugin system that provides:

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

### Using Makefile

Build all plugins using the Makefile:

```bash
# Build all plugins
make build-plugins

# Build only specific plugins manually
cd plugins/psql
go build -o psql .

cd plugins/azure_table
go build -o azure_table .
```

## Plugin Structure

All plugins follow this structure:

```go
package main

import (
    "context"
    "log"

    "github.com/hashicorp/go-plugin"
    pluginPkg "zms.szuro.net/pkg/plugin"
    "zms.szuro.net/pkg/proto"
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

## Plugin Architecture

| Aspect | Implementation |
|--------|----------------|
| Build | Standalone executable binary |
| Loading | Process spawning via go-plugin |
| Communication | gRPC with Protocol Buffers |
| Interface | `proto.ObserverServiceServer` |
| Base | `pluginPkg.BaseObserverGRPC` |
| Isolation | Separate process |
| Crash handling | Isolated from main application |

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
- Proto definitions: [pkg/proto/zbx_exports.proto](../pkg/proto/zbx_exports.proto)
- Example plugins: [examples/plugins/](../examples/plugins/)

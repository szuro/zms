# ZMS Observer Plugins

This directory contains observer plugins for ZMS (Zabbix Metric Shipper).

## Overview

ZMS supports a plugin system for creating custom observer implementations. Plugins are compiled as shared libraries and loaded dynamically at runtime, allowing developers to create custom output targets without recompiling the main application.

## Plugin Structure

A plugin must be implemented as a Go package with the following requirements:

1. **Package Declaration**: Must be `package main`
2. **Required Exports**: Must export `PluginInfo` variable and `NewObserver()` function
3. **Interface Implementation**: Must implement the `plugin.Observer` interface
4. **Base Observer**: Should embed `plugin.BaseObserverImpl` for core functionality

## Plugin Template

```go
package main

import (
    "szuro.net/zms/pkg/plugin"
    zbxpkg "szuro.net/zms/pkg/zbx"
)

// Plugin metadata - REQUIRED
var PluginInfo = plugin.PluginInfo{
    Name:        "my-plugin",
    Version:     "1.0.0",
    Type:        "custom",
    Description: "Custom observer plugin",
    Author:      "Your Name",
}

// Plugin struct - embed BaseObserverImpl for core functionality
type MyPlugin struct {
    plugin.BaseObserverImpl
    // Add your custom fields here
}

// Factory function - REQUIRED
// This function is called by ZMS to create plugin instances
func NewObserver() plugin.Observer {
    return &MyPlugin{}
}

// Initialize your plugin - REQUIRED
// connection: connection string from config
// options: key-value options from config
func (p *MyPlugin) Initialize(connection string, options map[string]string) error {
    // Initialize your plugin here
    return nil
}

// Return plugin type - REQUIRED
func (p *MyPlugin) GetType() string {
    return PluginInfo.Type
}

// Implement data processing methods as needed
func (p *MyPlugin) SaveHistory(h []zbxpkg.History) bool {
    for _, history := range h {
        // Check filter (provided by BaseObserver)
        if !p.EvaluateFilter(history.Tags) {
            continue
        }

        // Process history data
        // Use p.Monitor.HistoryValuesSent.Inc() for success metrics
        // Use p.Monitor.HistoryValuesFailed.Inc() for error metrics
    }
    return true
}

func (p *MyPlugin) SaveTrends(t []zbxpkg.Trend) bool {
    // Implement trend processing if needed
    // If not implemented, this method will panic by default (from BaseObserver)
    return true
}

func (p *MyPlugin) SaveEvents(e []zbxpkg.Event) bool {
    // Implement event processing if needed
    // If not implemented, this method will panic by default (from BaseObserver)
    return true
}
```

## Building Plugins

Plugins must be compiled as shared libraries (`.so` files on Linux):

```bash
# Build plugin as shared library
go build -buildmode=plugin -o my-plugin.so ./path/to/plugin

# Place in plugins directory
mkdir -p plugins/
mv my-plugin.so plugins/
```

## Plugin Configuration

Configure plugins in your `zmsd.yaml`:

```yaml
targets:
  - name: "my-custom-target"
    type: "plugin"
    connection: "connection-string-here"
    options:
      key1: "value1"
      key2: "value2"
    exports:
      - "history"
      - "trends"
```

## Available Functionality

Plugins have access to core ZMS functionality through the embedded `BaseObserverImpl`:

- **Filtering**: Use `p.EvaluateFilter(tags)` to apply configured tag filters
- **Metrics**: Access Prometheus counters via `p.Monitor.*` fields
- **Buffering**: Automatic offline buffering (handled by base observer)
- **Configuration**: Access to connection string and options map

## Available Plugins

### Print Plugin
- **File**: `print/print.go`
- **Type**: `print`
- **Purpose**: Outputs data to stdout/stderr for debugging
- **Configuration**: Connection string can be "stdout" or "stderr"

### PostgreSQL Plugin
- **File**: `psql/psql.go`
- **Type**: `postgresql`
- **Purpose**: Stores data in PostgreSQL database
- **Configuration**: Connection string is PostgreSQL URL

### Azure Table Storage Plugin
- **File**: `azure_table/azure_table.go`
- **Type**: `azure_table`
- **Purpose**: Stores data in Azure Table Storage
- **Configuration**: Connection string is Azure Storage URL

### GCP Cloud Monitoring Plugin
- **File**: `gcp_cloud_monitor/gcp_cloud_monitor.go`
- **Type**: `gcp_cloud_monitor`
- **Purpose**: Sends metrics to Google Cloud Monitoring
- **Configuration**: Uses default GCP credentials

### Prometheus Pushgateway Plugin
- **File**: `prometheus_pushgateway/prometheus_pushgateway.go`
- **Type**: `prometheus_pushgateway`
- **Purpose**: Pushes metrics to Prometheus Pushgateway
- **Configuration**: Connection string is Pushgateway URL

## Docker Plugin Builder

A Docker image is available for building plugins in a consistent environment:

```bash
# Build the Docker image
docker build -f Dockerfile.plugin-builder -t zms-plugin-builder .

# Build plugins from host directory
docker run --rm -v $(pwd)/my-plugins:/plugin-src -v $(pwd)/built-plugins:/plugins zms-plugin-builder

# Interactive development
docker run --rm -it -v $(pwd):/workspace -v $(pwd)/plugins:/plugins zms-plugin-builder bash
```

## Development Workflow

1. **Create Plugin**: Write your plugin following the template above
2. **Test Locally**: Build and test with local ZMS instance
3. **Use Docker Builder**: Leverage the Docker plugin builder for consistent builds
4. **Deploy**: Place compiled `.so` file in plugins directory
5. **Configure**: Add plugin target to ZMS configuration

## Best Practices

- **Error Handling**: Always handle errors gracefully and update metrics accordingly
- **Resource Management**: Clean up resources in plugin destructor if needed
- **Filter Usage**: Always check `p.EvaluateFilter()` before processing data
- **Metrics**: Update appropriate counters for monitoring and observability
- **Thread Safety**: Ensure your plugin is thread-safe as it may be called concurrently
- **Testing**: Test plugins thoroughly before deployment

## Troubleshooting

- **Plugin Load Errors**: Check Go version compatibility and shared library format
- **Missing Symbols**: Ensure `PluginInfo` and `NewObserver` are properly exported
- **Runtime Panics**: Implement all required interface methods or they will panic
- **Filter Issues**: Verify tag filtering logic using `EvaluateFilter()`

## Dependencies

Plugins can import:
- `szuro.net/zms/pkg/zbx` - Zabbix data types
- `szuro.net/zms/pkg/plugin` - Plugin interfaces
- Standard Go libraries
- External packages (ensure they're available at runtime)

Avoid importing internal packages as they may change between versions.
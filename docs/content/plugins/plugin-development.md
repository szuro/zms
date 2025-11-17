---
title: "Plugin Development Guide"
description: "Learn how to create custom plugins for ZMS"
weight: 1
toc: true
---

# Plugin Development Guide

ZMS uses HashiCorp's go-plugin framework for a robust plugin system. Plugins run as separate processes and communicate with the main application via gRPC. This provides better isolation, version compatibility, and crash resilience compared to shared libraries.

## Architecture Overview

- **Main Process**: Runs ZMS core application and manages plugin lifecycle
- **Plugin Processes**: Independent executables that implement observer functionality
- **Communication**: gRPC-based with Protocol Buffers for data serialization
- **Configuration**: Sent from main process to plugin via gRPC during initialization

## Plugin Structure

A plugin must be implemented as a standalone Go application with the following requirements:

1. **Package Declaration**: Must be `package main`
2. **Main Function**: Plugin binary entry point that serves the gRPC interface
3. **Interface Implementation**: Must implement `plugin.ObserverGRPC` interface
4. **Base Observer**: Should embed `plugin.BaseObserverGRPC` for core functionality

## Plugin Template

Here's a complete template for creating a new plugin:

```go
package main

import (
    "context"
    "log"

    "github.com/hashicorp/go-plugin"
    pluginPkg "zms.szuro.net/pkg/plugin"
    zbxpkg "zms.szuro.net/pkg/zbx"
    "zms.szuro.net/pkg/proto"
)

// MyPlugin implements the gRPC observer interface
type MyPlugin struct {
    pluginPkg.BaseObserverGRPC
    // Add your custom fields here
}

// NewMyPlugin creates a new plugin instance
func NewMyPlugin() *MyPlugin {
    return &MyPlugin{
        BaseObserverGRPC: *pluginPkg.NewBaseObserverGRPC(),
    }
}

// Initialize configures the plugin with settings from main application
func (p *MyPlugin) Initialize(ctx context.Context, req *proto.InitializeRequest) (*proto.InitializeResponse, error) {
    // Call base initialization to handle common setup
    resp, err := p.BaseObserverGRPC.Initialize(ctx, req)
    if err != nil {
        return resp, err
    }

    // Add your custom initialization here
    // req.Connection contains the connection string
    // req.Options contains key-value configuration options
    p.Logger.Info("Plugin initialized", "connection", req.Connection)

    return &proto.InitializeResponse{Success: true}, nil
}

// SaveHistory processes history data
func (p *MyPlugin) SaveHistory(ctx context.Context, req *proto.SaveHistoryRequest) (*proto.SaveResponse, error) {
    // Filter and convert proto history to zbx types
    history := p.FilterHistory(req.History)

    for _, h := range history {
        // Process history data
        p.Logger.Info("Processing history", "itemid", h.ItemID, "value", h.Value)
    }

    return &proto.SaveResponse{Success: true}, nil
}

// SaveTrends processes trend data (optional - can return success with no-op)
func (p *MyPlugin) SaveTrends(ctx context.Context, req *proto.SaveTrendsRequest) (*proto.SaveResponse, error) {
    return &proto.SaveResponse{Success: true}, nil
}

// SaveEvents processes event data (optional - can return success with no-op)
func (p *MyPlugin) SaveEvents(ctx context.Context, req *proto.SaveEventsRequest) (*proto.SaveResponse, error) {
    return &proto.SaveResponse{Success: true}, nil
}

// main is the entry point for the plugin binary
func main() {
    impl := NewMyPlugin()

    // Serve the plugin using HashiCorp go-plugin
    plugin.Serve(&plugin.ServeConfig{
        HandshakeConfig: pluginPkg.Handshake,
        Plugins: map[string]plugin.Plugin{
            "observer": &pluginPkg.ObserverPlugin{Impl: impl},
        },
        GRPCServer: plugin.DefaultGRPCServer,
    })

    log.Println("Plugin exited")
}
```

## Building Plugins

Plugins are built as standalone executables (NOT shared libraries):

```bash
# Build plugin as executable
go build -o my-plugin ./path/to/plugin

# Place in plugins directory
mkdir -p plugins/
mv my-plugin plugins/
```

## Plugin Configuration

Configure plugins in your `zmsd.yaml`:

```yaml
# Configure plugin directory where ZMS looks for plugin executables
plugins_dir: "./plugins"

targets:
  - name: "my-custom-target"
    type: "my-plugin"  # Must match the plugin executable name
    connection: "stdout"  # Plugin-specific connection string
    options:
      key1: "value1"
      key2: "value2"
    exports:
      - "history"
      - "trends"
    filter:
      accepted:
        - "tag_pattern:value"
      rejected:
        - "ignore:true"
```

## Available Functionality

Plugins have access to core ZMS functionality through the embedded `BaseObserverGRPC`:

- **Filtering**: Use `p.FilterHistory()`, `p.FilterTrends()`, `p.FilterEvents()` helper methods
- **Configuration**: All settings passed via `InitializeRequest` proto message
- **Logging**: Use `p.Logger` for structured logging
- **Context**: All methods receive context for cancellation/timeout support

### Type Conversions

The plugin system uses Protocol Buffers for data serialization. Proto messages use enum types that need to be cast to int32 when working with zbx types:

#### Proto Enums

- `proto.ValueType` - Data type (FLOAT, CHARACTER, LOG, UNSIGNED, TEXT)
- `proto.EventValue` - Event type (RECOVERY, PROBLEM)
- `proto.Severity` - Severity level (NOT_CLASSIFIED, INFORMATION, WARNING, AVERAGE, HIGH, DISASTER)

#### Converting Proto Types to ZBX Types

The `BaseObserverGRPC.FilterHistory()`, `FilterTrends()`, and `FilterEvents()` helper methods automatically handle conversion from proto types to `zbx` types. When you need to work with raw proto data:

```go
// Proto enums are already int32 compatible with zbx types
history := zbx.History{
    Type: int32(protoHistory.ValueType),  // proto.ValueType to int32
}

event := zbx.Event{
    Value:    int32(protoEvent.Value),     // proto.EventValue to int32
    Severity: int32(protoEvent.Severity),  // proto.Severity to int32
}
```

The proto definitions in `pkg/proto/zbx_exports.proto` define the enum values to match Zabbix's internal constants.

## Custom Filters

Plugins can implement custom filtering logic by providing their own `filter.Filter` implementation. This allows plugins to filter data based on criteria beyond tag-based filtering.

### Example: LOG Type Filter

The `log_print` example plugin implements a custom filter that only accepts LOG-type history items:

```go
type LogFilter struct{}

func (lf *LogFilter) AcceptHistory(h zbxpkg.History) bool {
    return h.Type == zbxpkg.LOG
}

func (lf *LogFilter) AcceptTrend(t zbxpkg.Trend) bool {
    return false  // Not supported
}

func (lf *LogFilter) AcceptEvent(e zbxpkg.Event) bool {
    return false  // Not supported
}

func (lf *LogFilter) FilterHistory(h []zbxpkg.History) []zbxpkg.History {
    accepted := make([]zbxpkg.History, 0, len(h))
    for _, history := range h {
        if lf.AcceptHistory(history) {
            accepted = append(accepted, history)
        }
    }
    return accepted
}
```

To use a custom filter, assign it during plugin initialization:

```go
func (p *MyPlugin) Initialize(ctx context.Context, req *proto.InitializeRequest) (*proto.InitializeResponse, error) {
    resp, err := p.BaseObserverGRPC.Initialize(ctx, req)
    if err != nil {
        return resp, err
    }

    // Override with custom filter
    p.Filter = &LogFilter{}

    return &proto.InitializeResponse{Success: true}, nil
}
```

## Plugin Examples

The `examples/plugins/` directory contains working plugin examples:

- **log_print**: Simple plugin that outputs LOG-type history items to stdout/stderr. Demonstrates custom filtering and basic data processing.

## Development Workflow

1. **Create Plugin**: Write your plugin following the template above
2. **Build**: Compile as standalone executable using `go build`
3. **Test Locally**: Run plugin with ZMS to test functionality
4. **Deploy**: Place compiled executable in plugins directory
5. **Configure**: Add target configuration referencing plugin name

## Plugin Architecture Benefits

The gRPC-based plugin system provides:

- **Process Isolation**: Plugins run as separate processes
- **Version Compatibility**: No Go version matching required between plugin and main application
- **Crash Resilience**: Plugin failures don't affect the main ZMS process
- **Type Safety**: gRPC with Protocol Buffers ensures correct data serialization
- **Configuration Flexibility**: Settings sent via gRPC during initialization
- **Independent Updates**: Plugins can be updated without recompiling ZMS

## Plugin Best Practices

- **Error Handling**: Return errors via `SaveResponse` with error message
- **Resource Management**: Implement cleanup logic in `Cleanup()` method
- **Filter Usage**: Use built-in filter helpers (`FilterHistory`, etc.)
- **Logging**: Use the provided `Logger` for consistent logging
- **Context Awareness**: Respect context cancellation in long-running operations
- **Testing**: Test plugins independently before integrating with ZMS

## Troubleshooting

- **Plugin Load Errors**: Ensure plugin executable has execute permissions
- **Connection Failures**: Check that plugin implements required gRPC service correctly
- **Handshake Errors**: Verify `HandshakeConfig` matches between plugin and main app
- **Data Processing Issues**: Check proto conversion and filter logic
- **Missing Configuration**: Ensure all required fields in `InitializeRequest` are handled

## Building Multiple Plugins

Use the Makefile to build all plugins at once:

```bash
# Build all plugins in the plugins/ directory
make build-plugins

# Build only the main binary
make build-main

# Build everything (main + all plugins)
make build
```

The Makefile automatically discovers and builds all plugins that have a `main.go` file in subdirectories of the `plugins/` folder.

# ZMS Observer Plugins

This directory contains example observer plugins for ZMS (Zabbix Metric Shipper).

## Overview

ZMS supports dynamic loading of observer plugins, allowing developers to create custom output targets without recompiling the main application. Plugins are loaded as Go shared libraries (.so files) at application startup.

## Plugin Interface

All observer plugins must implement the `plugin.Observer` interface defined in `pkg/plugin/interface.go`:

```go
type Observer interface {
    // Core observer methods
    Cleanup()
    GetName() string
    SetName(name string)
    InitBuffer(path string, ttl int64)
    SaveHistory(h []zbx.History) bool
    SaveTrends(t []zbx.Trend) bool
    SaveEvents(e []zbx.Event) bool
    SetFilter(filter filter.Filter)
    PrepareMetrics(exports []string)
    
    // Plugin-specific initialization
    Initialize(connection string, options map[string]string) error
    GetType() string
}
```

## BaseObserver Access

Plugins can use the `plugin.BaseObserver` interface to access baseObserver functionality:

- Buffer management (offline buffering with BadgerDB)
- Prometheus metrics
- Generic save operations
- Filter evaluation

## Creating a Plugin

### 1. Implement the Observer Interface

```go
package main

import (
    "szuro.net/zms/pkg/plugin"
    "szuro.net/zms/pkg/zbx"
)

type MyObserver struct {
    baseObserver plugin.BaseObserver
    // Your custom fields
}

// Required plugin factory function
func NewObserver() plugin.Observer {
    return &MyObserver{}
}

// Optional plugin metadata
var PluginInfo = plugin.PluginInfo{
    Name:        "my_observer",
    Version:     "1.0.0",
    Type:        "custom",
    Description: "My custom observer",
    Author:      "Your Name",
}

func (m *MyObserver) Initialize(connection string, options map[string]string) error {
    // Initialize your observer with the provided connection string and options
    m.baseObserver = plugin.NewBaseObserver(m.GetName(), "my_observer")
    return nil
}

func (m *MyObserver) SaveHistory(h []zbx.History) bool {
    // Process history data
    // Use m.baseObserver for common functionality
    return true
}

// Implement other required methods...
```

### 2. Build as Plugin

```bash
go build -buildmode=plugin -o my_observer.so my_observer.go
```

### 3. Configure in ZMS

Add the plugin configuration to your `zmsd.yaml`:

```yaml
plugins_dir: "/path/to/plugins"

targets:
- name: my_custom_target
  type: plugin:my_observer  # plugin:filename_without_extension
  connection: "connection_string_for_your_plugin"
  options:
    custom_option1: "value1"
    custom_option2: "value2"
  source:
  - history
  - trends
```

## Example Plugins

### File Observer

The `file_observer.go` example demonstrates:
- Writing Zabbix exports to JSON files
- Using the BaseObserver functionality
- Proper error handling and logging
- Thread-safe file operations

Build and use:
```bash
cd examples/plugins
go build -buildmode=plugin -o file_observer.so file_observer.go

# Configure in zmsd.yaml:
targets:
- name: file_output
  type: plugin:file_observer
  connection: "/tmp/zms_output"  # output directory
  source: [history, trends, events]
```

## Best Practices

### 1. Error Handling
- Always return appropriate boolean values from Save methods
- Use the baseObserver for consistent error handling and metrics
- Log errors using the internal logger

### 2. Thread Safety
- Observers may be called concurrently
- Use mutexes or channels for thread-safe operations
- The baseObserver handles its own thread safety

### 3. Resource Management
- Implement proper cleanup in the `Cleanup()` method
- Close files, connections, and other resources
- The baseObserver will handle buffer cleanup

### 4. Filtering
- Use `baseObserver.EvaluateFilter()` to respect tag filters
- Filter data before processing to improve performance

### 5. Metrics
- The baseObserver automatically handles Prometheus metrics
- Use `PrepareMetrics()` to set up export-specific metrics

### 6. Buffer Operations
- Use baseObserver buffer methods for offline support
- Handle failed operations by saving to buffer
- Implement retry logic for reliability

## Plugin Loading

ZMS loads plugins at startup:

1. Reads plugin directory from configuration (`plugins_dir`)
2. Loads all `.so` files from the directory
3. Validates each plugin has a `NewObserver` function
4. Optionally reads `PluginInfo` for metadata
5. Creates plugin instances when targets are configured

## Debugging

To debug plugin issues:

1. Set log level to DEBUG in configuration
2. Check startup logs for plugin loading messages
3. Verify plugin exports the required symbols
4. Test plugin initialization with various connection strings
5. Use the baseObserver metrics to monitor plugin performance

## Dependencies

Plugins can import:
- `szuro.net/zms/pkg/zbx` - Zabbix data types
- `szuro.net/zms/pkg/plugin` - Plugin interfaces
- Standard Go libraries
- External packages (ensure they're available at runtime)

Avoid importing internal packages as they may change between versions.
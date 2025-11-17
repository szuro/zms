---
title: "Architecture"
description: "ZMS architecture and design principles"
weight: 5
toc: true
---

# Architecture

ZMS follows a modular architecture with clear separation of concerns and adheres to the Go standard project layout.

## Directory Structure

```
zms/
├── cmd/
│   └── zmsd/              # Main application entry point
├── internal/              # Private application code
│   ├── config/           # Configuration management
│   ├── input/            # Input layer (FileInput, HTTPInput)
│   ├── zbx/              # Zabbix integration
│   ├── logger/           # Logging utilities
│   └── plugin/           # Plugin loader and registry
├── pkg/                   # Public APIs (importable by external code)
│   ├── zbx/              # Public Zabbix types (History, Trend, Event)
│   ├── plugin/           # Plugin interface and base implementation
│   ├── filter/           # Public filter types and interfaces
│   └── proto/            # Protocol Buffers definitions (gRPC service)
├── plugins/               # Built-in observer plugins (all gRPC-based)
│   ├── psql/             # PostgreSQL plugin
│   ├── azure_table/      # Azure Table Storage plugin
│   ├── gcp_cloud_monitor/  # Google Cloud Monitoring plugin
│   ├── prometheus_remote_write/  # Prometheus Remote Write plugin
│   ├── prometheus_pushgateway/   # Prometheus Pushgateway plugin
│   └── print/            # Debug print plugin
├── examples/
│   └── plugins/          # Example plugins (log_print, etc.)
├── docs/                  # Hugo documentation site
└── configs/               # Configuration templates
```

## Core Components

### 1. Main Application (`cmd/zmsd/`)

The application entry point that:
- Parses CLI arguments (`-c` for config, `-v` for version)
- Initializes logging and configuration
- Manages signal handling (SIGTERM, SIGINT, SIGQUIT)
- Starts HTTP server for Prometheus metrics
- Coordinates input sources and observers

### 2. Configuration Layer (`internal/config/`)

Manages application configuration:
- **ZMSConf**: Main configuration structure
- **Target**: Output target configuration
- **HTTPConf**: HTTP server configuration
- Supports two modes: `FILE_MODE` and `HTTP_MODE`
- Validates configuration and sets defaults

### 3. Input Layer (`internal/input/`)

Handles data ingestion from different sources:

#### Inputer Interface
```go
type Inputer interface {
    Start()
    Stop()
    GetDataChannel() <-chan zbx.Export
}
```

#### FileInput
- Monitors Zabbix export files
- Parses NDJSON format
- Watches for file changes
- Supports history, trends, and events

#### HTTPInput
- Runs HTTP server to receive data
- Accepts POST requests with JSON payloads
- Validates incoming data
- Converts to internal format

#### baseInput
- Common functionality for all inputs
- Channel management
- Error handling

### 4. Observer/Output Layer (`plugins/`)

**All observers are now plugin-based**. ZMS no longer has built-in observers in `internal/observer/`. All output targets are implemented as gRPC plugins using HashiCorp's go-plugin framework.

#### Observer gRPC Interface (Protocol Buffers)
```protobuf
service ObserverService {
  rpc Initialize(InitializeRequest) returns (InitializeResponse);
  rpc SaveHistory(SaveHistoryRequest) returns (SaveResponse);
  rpc SaveTrends(SaveTrendsRequest) returns (SaveResponse);
  rpc SaveEvents(SaveEventsRequest) returns (SaveResponse);
  rpc Cleanup(CleanupRequest) returns (CleanupResponse);
}
```

#### Built-in Plugins
All located in `plugins/` directory:
- **psql**: PostgreSQL database storage
- **azure_table**: Azure Table Storage
- **gcp_cloud_monitor**: GCP Cloud Monitoring
- **prometheus_remote_write**: Prometheus Remote Write API
- **prometheus_pushgateway**: Prometheus Pushgateway
- **print**: Debug output to stdout/stderr

#### BaseObserverGRPC
Located in `pkg/plugin/grpc_base_observer.go`, provides:
- Tag-based filtering with Filter interface
- Group-based filtering support
- Custom filter implementation support
- Configuration handling (name, connection, options, exports)
- Structured logging with slog
- Helper methods: `FilterHistory()`, `FilterTrends()`, `FilterEvents()`
- Type conversion utilities (proto ↔ zbx types)
- No offline buffering (removed in plugin architecture)

### 5. Plugin System (`internal/plugin/` and `pkg/plugin/`)

HashiCorp go-plugin based architecture for robust plugin support:

#### Plugin Architecture
```
┌─────────────┐         gRPC          ┌─────────────┐
│             │◄──────────────────────►│             │
│  ZMS Core   │   Protocol Buffers    │   Plugin    │
│  Process    │                        │  Process    │
│             │◄──────────────────────►│             │
└─────────────┘                        └─────────────┘
     │                                      │
     │ Launches & Manages                   │
     │                                      │
     └──────────────────────────────────────┘
```

#### Key Features
- **Process Isolation**: Plugins run as independent processes
- **Version Compatibility**: No Go version matching required
- **Crash Resilience**: Plugin failures don't affect ZMS core
- **Type Safety**: Protocol Buffers ensure correct serialization
- **Dynamic Loading**: Load plugins at runtime without recompilation

#### Plugin Loader (`internal/plugin/grpc_loader.go`)
- **GRPCPluginRegistry**: Global plugin registry
- Discovers plugin executables in `plugins_dir`
- Launches plugin processes using `exec.Command`
- Establishes gRPC connections via HashiCorp go-plugin
- Manages plugin lifecycle (start, connect, cleanup)
- Implements plugin client creation for observers

#### Plugin Interface (`pkg/plugin/`)
- **ObserverPlugin**: HashiCorp go-plugin wrapper implementing `plugin.Plugin`
- **BaseObserverGRPC**: Base functionality for plugin implementations
  - Filtering (tag-based and custom)
  - Configuration handling
  - Structured logging
  - Helper methods for data conversion
- **Handshake**: Plugin handshake configuration for compatibility checking

#### Protocol Buffers (`pkg/proto/`)
- **Message Definitions**: History, Trend, Event, Host, Tag
- **Service Definition**: ObserverService with Initialize, SaveHistory, SaveTrends, SaveEvents, Cleanup
- **Enum Types**: ValueType, EventValue, Severity, ExportType, FilterType
- **Request/Response Types**: InitializeRequest/Response, SaveHistoryRequest, SaveResponse, etc.
- Data serialization format ensuring type safety across process boundaries

### 6. Filtering System (`pkg/filter/`)

Tag-based and group-based filtering of Zabbix data:

#### Filter Interface
```go
type Filter interface {
    AcceptHistory(h zbx.History) bool
    AcceptTrend(t zbx.Trend) bool
    AcceptEvent(e zbx.Event) bool
    FilterHistory(h []zbx.History) []zbx.History
    FilterTrends(t []zbx.Trend) []zbx.Trend
    FilterEvents(e []zbx.Event) []zbx.Event
}
```

#### Filter Types (all in `pkg/filter/`)
- **TagFilter** (`tag_filter.go`): Tag-based filtering with accept/reject patterns
- **GroupFilter** (`group_filter.go`): Host group-based filtering
- **EmptyFilter** (`empty_filter.go`): No-op filter that accepts everything

#### Filter Configuration (Protocol Buffers)
```protobuf
enum FilterType {
  TAG = 0;    // Filter based on item/event tags
  GROUP = 1;  // Filter based on host groups
  CUSTOM = 69; // Custom filter implementation
}

message Filter {
  FilterType type = 1;
  repeated string accepted = 2;  // Patterns to accept
  repeated string rejected = 3;  // Patterns to reject
}
```

#### Filter Logic
1. If only accepted patterns: whitelist mode
2. If only rejected patterns: blacklist mode
3. If both: accepted items minus rejected items
4. Plugins can implement custom Filter for advanced logic

### 7. Zabbix Integration (`internal/zbx/`)

Handles Zabbix-specific functionality:
- Parses `zabbix_server.conf`
- Discovers export files
- Monitors file changes
- Node status tracking

### 8. Public APIs (`pkg/`)

Importable packages for external use:

#### pkg/zbx
- **Export Interface**: Generic export type interface
- **History**: Historical data structure
- **Trend**: Trend data structure
- **Event**: Event data structure
- **Tag**: Tag structure
- Constants for export types and value types

#### pkg/plugin
- **ObserverGRPC**: Plugin interface
- **BaseObserverGRPC**: Base implementation
- **Handshake**: Plugin handshake configuration

#### pkg/filter
- **FilterConfig**: Filter configuration
- **Filter**: Filter interface

## Data Flow

```
┌─────────────────────────┐
│    Zabbix Export Files  │
│  (history, trends,      │
│   events NDJSON)        │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│   Input Layer           │
│   • FileInput (monitors │
│     export files)       │
│   • HTTPInput (receives │
│     HTTP POST)          │
└────────┬────────────────┘
         │
         │ Channel: zbx.Export
         │
         ▼
┌─────────────────────────┐
│  ZMS Core Process       │
│  • Routing to targets   │
│  • Config management    │
└────────┬────────────────┘
         │
         │ Launches & manages via
         │ HashiCorp go-plugin
         │
    ┌────┴─────┬──────────┬─────────┐
    ▼          ▼          ▼         ▼
┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
│ Plugin │ │ Plugin │ │ Plugin │ │ Plugin │
│Process │ │Process │ │Process │ │Process │
│   #1   │ │   #2   │ │   #3   │ │   #N   │
└───┬────┘ └───┬────┘ └───┬────┘ └───┬────┘
    │          │          │          │
    │ gRPC     │ gRPC     │ gRPC     │ gRPC
    │ Proto    │ Proto    │ Proto    │ Proto
    │ Buffers  │ Buffers  │ Buffers  │ Buffers
    │          │          │          │
    ▼          ▼          ▼          ▼
┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
│ Plugin │ │ Plugin │ │ Plugin │ │ Plugin │
│ Filter │ │ Filter │ │ Filter │ │ Filter │
│(Tag/Grp│ │(Tag/Grp│ │(Tag/Grp│ │(Custom)│
└───┬────┘ └───┬────┘ └───┬────┘ └───┬────┘
    │          │          │          │
    ▼          ▼          ▼          ▼
┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
│ Destin │ │ Destin │ │ Destin │ │ Destin │
│ -ation │ │ -ation │ │ -ation │ │ -ation │
│  PSQL  │ │ Azure  │ │Prometh │ │  GCP   │
└────────┘ └────────┘ └────────┘ └────────┘

Process Isolation: Each plugin runs independently
Crash Resilience: Plugin failures don't affect ZMS core
Type Safety: Protocol Buffers ensure correct serialization
```

## Design Principles

### 1. Separation of Concerns
- Input layer: data ingestion only
- Filtering: independent of input/output
- Observers: focus on destination logic

### 2. Plugin Architecture (HashiCorp go-plugin)
- **Process Isolation**: Plugins run as separate OS processes
- **Crash Resilience**: Plugin failures don't crash ZMS core
- **Version Independence**: No Go version matching required between ZMS and plugins
- **Extensibility**: Add new targets without recompiling ZMS
- **Type Safety**: Protocol Buffers ensure correct data serialization
- **gRPC Communication**: Fast, language-agnostic IPC

### 3. Filtering Flexibility
- **Tag-based filtering**: Accept/reject by item/event tags
- **Group-based filtering**: Filter by host groups
- **Custom filters**: Plugins can implement custom Filter logic
- **Per-target configuration**: Each plugin has its own filter settings

### 4. Standard Go Layout
- `cmd/`: Application entry points
- `internal/`: Private implementation (ZMS core only)
- `pkg/`: Public APIs (importable by plugins and external code)
- `plugins/`: Plugin implementations (separate executables)
- Clear import boundaries

### 5. Configurability
- YAML-based configuration
- Per-target settings
- Global defaults

### 6. Observability
- Prometheus metrics endpoint
- Structured logging (slog)
- Performance monitoring

## Concurrency Model

- **Input goroutines**: Read and parse files (FileInput) or HTTP requests (HTTPInput)
- **Plugin processes**: Each plugin runs as independent OS process
- **Channel-based communication**: Type-safe data passing within ZMS core
- **gRPC streaming**: Asynchronous communication between ZMS and plugins
- **Context-based cancellation**: Graceful shutdown coordination
- **Per-plugin concurrency**: Plugins manage their own goroutines independently

## Error Handling

- **Input errors**: Logged with structured logging, continue processing
- **Filter errors**: Invalid items skipped, logged for debugging
- **Plugin errors**: Isolated to plugin process, don't crash ZMS core
- **gRPC errors**: Automatic reconnection attempts by go-plugin framework
- **Plugin crashes**: Detected and logged, other plugins continue operating
- **Initialization errors**: Plugin fails to start, logged and skipped
- **Graceful shutdown**: Clean resource cleanup via Cleanup() RPC call

## Performance Considerations

- **Batch processing**: Data sent to plugins in batches via SaveHistoryRequest/SaveTrendsRequest
- **Early filtering**: Unwanted data rejected before sending to plugins
- **Parallel plugins**: Multiple plugins process data concurrently in separate processes
- **gRPC efficiency**: Binary Protocol Buffers serialization for fast IPC
- **Process isolation**: Plugin resource usage doesn't impact ZMS core
- **Channel buffering**: Internal channels buffer data between input and output
- **Filter optimization**: TagFilter and GroupFilter use efficient pattern matching

## Future Enhancements

Potential areas for expansion:
- **Additional input sources**: MQTT, Kafka, message queues
- **More plugins**: InfluxDB, TimescaleDB, Elasticsearch, MongoDB
- **Enhanced filtering**: Regex patterns, complex boolean logic
- **Plugin discovery**: Auto-discovery of plugins in directories
- **Health monitoring**: Plugin health checks and automatic restart
- **Metrics dashboard**: Web UI for monitoring plugin status and metrics
- **Plugin marketplace**: Repository of community-contributed plugins
- **Multi-language plugins**: Support for plugins written in Python, Rust, etc.
- **Buffering/retry**: Optional buffering layer for resilience (removed in current version)

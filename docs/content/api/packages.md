---
title: "Go Packages"
description: "Public Go packages provided by ZMS"
weight: 1
toc: true
---

# Go Packages

ZMS provides public Go packages that can be imported and used in your own applications.

## Installation

```bash
go get zms.szuro.net
```

## Package Structure

ZMS follows the standard Go project layout with public APIs in the `pkg/` directory:

- `zms.szuro.net/pkg/zbx` - Zabbix export data types
- `zms.szuro.net/pkg/plugin` - Plugin interface and utilities
- `zms.szuro.net/pkg/filter` - Filtering types and interfaces
- `zms.szuro.net/pkg/proto` - Protocol Buffer definitions for gRPC communication

## pkg/zbx

The `zbx` package provides types and interfaces for handling Zabbix export data.

### Import

```go
import "zms.szuro.net/pkg/zbx"
```

### Types

#### Export Interface

Generic interface that all Zabbix export types implement:

```go
type Export interface {
    ShowTags() []Tag
    GetExportName() string
    Hash() string
}
```

#### History

Individual metric values collected from monitored items:

```go
type History struct {
    ItemID    uint64  `json:"itemid"`
    Clock     int     `json:"clock"`
    Timestamp string  `json:"timestamp"`
    NS        int     `json:"ns"`
    Value     string  `json:"value"`
    Type      int     `json:"type"`
    TTL       int     `json:"ttl"`
    State     int     `json:"state"`
    Host      Host    `json:"host"`
    Name      string  `json:"name"`
    Key       string  `json:"key_"`
    Groups    []Group `json:"groups"`
    Tags      []Tag   `json:"tags"`
}
```

Methods:
- `IsNumeric() bool` - Check if value is numeric
- `ShowTags() []Tag` - Get tags
- `GetExportName() string` - Returns "history"
- `Hash() string` - Generate unique hash

#### Trend

Aggregated hourly statistics for numeric items:

```go
type Trend struct {
    ItemID    uint64  `json:"itemid"`
    Clock     int     `json:"clock"`
    Timestamp string  `json:"timestamp"`
    Num       int     `json:"num"`
    ValueMin  float64 `json:"value_min"`
    ValueAvg  float64 `json:"value_avg"`
    ValueMax  float64 `json:"value_max"`
    Host      Host    `json:"host"`
    Name      string  `json:"name"`
    Key       string  `json:"key_"`
    Groups    []Group `json:"groups"`
    Tags      []Tag   `json:"tags"`
}
```

Methods:
- `ShowTags() []Tag` - Get tags
- `GetExportName() string` - Returns "trends"
- `Hash() string` - Generate unique hash

#### Event

Problem and recovery events from Zabbix triggers:

```go
type Event struct {
    Type           string  `json:"type"`
    Clock          int     `json:"clock"`
    Timestamp      string  `json:"timestamp"`
    NS             int     `json:"ns"`
    Value          int     `json:"value"`
    Severity       int     `json:"severity"`
    Name           string  `json:"name"`
    EventID        uint64  `json:"eventid"`
    Acknowledged   int     `json:"acknowledged"`
    REventID       *uint64 `json:"r_eventid,omitempty"`
    CorrelationID  *uint64 `json:"correlationid,omitempty"`
    UserID         *uint64 `json:"userid,omitempty"`
    Host           Host    `json:"host"`
    Groups         []Group `json:"groups"`
    Tags           []Tag   `json:"tags"`
    SuppressionData any     `json:"suppression_data"`
}
```

Methods:
- `ShowTags() []Tag` - Get tags
- `GetExportName() string` - Returns "events"
- `Hash() string` - Generate unique hash

#### Tag

Key-value pair tag:

```go
type Tag struct {
    Tag   string `json:"tag"`
    Value string `json:"value"`
}
```

#### Host

Zabbix host information:

```go
type Host struct {
    Host string `json:"host"`
    Name string `json:"name"`
}
```

#### Group

Zabbix host group:

```go
type Group struct {
    ID   uint64 `json:"groupid"`
    Name string `json:"name"`
}
```

### Constants

#### Export Types

```go
const (
    EVENT   = "events"
    HISTORY = "history"
    TREND   = "trends"
)
```

#### Value Types

```go
const (
    FLOAT     = 0  // Numeric floating-point
    CHARACTER = 1  // Character/string
    LOG       = 2  // Log file entries
    UNSIGNED  = 3  // Numeric unsigned integer
    TEXT      = 4  // Text values
)
```

#### Trend Value Types

```go
const (
    TREND_AVG   = "avg"
    TREND_MIN   = "min"
    TREND_MAX   = "max"
    TREND_COUNT = "count"
)
```

#### File Naming Constants

```go
const (
    HISTORY_EXPORT   = "history-history-syncer-%d.ndjson"
    HISTORY_MAIN     = "history-main-process-0.ndjson"
    TRENDS_EXPORT    = "trends-history-syncer-%d.ndjson"
    TRENDS_MAIN      = "trends-main-process-0.ndjson"
    PROBLEMS_EXPORT  = "problems-history-syncer-%d.ndjson"
    PROBLEMS_MAIN    = "problems-main-process-0.ndjson"
    PROBLEMS_TASK    = "problems-task-manager-1.ndjson"
)
```

### Example Usage

```go
package main

import (
    "fmt"
    "zms.szuro.net/pkg/zbx"
)

// Process any export type generically
func processExport[T zbx.Export](exports []T) {
    for _, export := range exports {
        tags := export.ShowTags()
        exportType := export.GetExportName()
        hash := export.Hash()
        fmt.Printf("Processing %s with hash %s\n", exportType, hash)
    }
}

func main() {
    // Create history items
    history := []zbx.History{
        {
            ItemID: 12345,
            Value:  "42.5",
            Type:   zbx.FLOAT,
            Tags: []zbx.Tag{
                {Tag: "environment", Value: "production"},
            },
        },
    }

    // Check if numeric
    if history[0].IsNumeric() {
        fmt.Println("History item is numeric")
    }

    // Process with generic function
    processExport(history)
}
```

## pkg/plugin

The `plugin` package provides the plugin interface and base functionality for creating ZMS plugins using HashiCorp's go-plugin framework.

### Import

```go
import "zms.szuro.net/pkg/plugin"
```

### Types

#### ObserverPlugin

HashiCorp go-plugin wrapper that implements `plugin.Plugin` interface:

```go
type ObserverPlugin struct {
    plugin.Plugin
    Impl proto.ObserverServiceServer
}
```

This handles the gRPC server/client setup for plugin communication.

#### BaseObserverGRPC

Base implementation providing common functionality for plugin implementations:

```go
type BaseObserverGRPC struct {
    Name       string           // Observer instance name
    PluginName string           // Plugin type identifier
    Filter     filter.Filter    // Tag-based filtering
    Logger     *slog.Logger     // Structured logging
}
```

Methods:
- `Initialize(ctx context.Context, req *proto.InitializeRequest) (*proto.InitializeResponse, error)` - Common initialization
- `FilterHistory(history []*proto.History) []zbxpkg.History` - Filter and convert history data
- `FilterTrends(trends []*proto.Trend) []zbxpkg.Trend` - Filter and convert trend data
- `FilterEvents(events []*proto.Event) []zbxpkg.Event` - Filter and convert event data

#### PluginInfo

Metadata about a plugin:

```go
type PluginInfo struct {
    Name        string
    Version     string
    Description string
    Author      string
}
```

#### Handshake

Configuration for plugin compatibility checking:

```go
var Handshake = plugin.HandshakeConfig{
    ProtocolVersion:  1,
    MagicCookieKey:   "ZMS_PLUGIN",
    MagicCookieValue: "zabbix_metric_shipper",
}
```

### Example

See [Plugin Development Guide](../plugins/plugin-development/) for complete examples.

## pkg/proto

The `proto` package contains Protocol Buffer definitions for gRPC-based plugin communication. These definitions are generated from `pkg/proto/zbx_exports.proto`.

### Import

```go
import "zms.szuro.net/pkg/proto"
```

### Enums

#### ValueType

Represents Zabbix data types:

```protobuf
enum ValueType {
  FLOAT = 0;       // Numeric floating-point
  CHARACTER = 1;   // Character/string
  LOG = 2;         // Log file entries
  UNSIGNED = 3;    // Numeric unsigned integer
  TEXT = 4;        // Text values
}
```

#### EventValue

Indicates event type:

```protobuf
enum EventValue {
  RECOVERY = 0;    // Trigger went from PROBLEM to OK
  PROBLEM = 1;     // Trigger went from OK to PROBLEM
}
```

#### Severity

Severity level:

```protobuf
enum Severity {
  NOT_CLASSIFIED = 0;
  INFORMATION = 1;
  WARNING = 2;
  AVERAGE = 3;
  HIGH = 4;
  DISASTER = 5;
}
```

#### ExportType

Export type identifier:

```protobuf
enum ExportType {
  HISTORY = 0;
  TRENDS = 1;
  EVENTS = 2;
}
```

#### FilterType

Filter type identifier:

```protobuf
enum FilterType {
  TAG = 0;      // Tag-based filtering
  GROUP = 1;    // Group-based filtering
  CUSTOM = 69;  // Custom filtering (not implemented)
}
```

### Messages

#### History

History record with collected item value:

```protobuf
message History {
  Host host = 1;
  int64 itemid = 2;
  string name = 3;
  int64 clock = 4;
  repeated string groups = 5;
  int64 ns = 6;
  oneof value {
    double numeric_value = 7;
    string string_value = 8;
  }
  repeated Tag tags = 9;
  ValueType value_type = 10;

  // Log-specific fields
  int64 timestamp = 11;
  string source = 12;
  Severity severity = 13;
  int64 eventid = 14;
}
```

#### Trend

Aggregated hourly statistics:

```protobuf
message Trend {
  Host host = 1;
  int64 itemid = 2;
  string name = 3;
  int64 clock = 4;
  int64 count = 5;
  repeated string groups = 6;
  double min = 7;
  double max = 8;
  double avg = 9;
  repeated Tag tags = 10;
  ValueType value_type = 11;
}
```

#### Event

Problem or recovery event:

```protobuf
message Event {
  int64 clock = 1;
  int64 ns = 2;
  EventValue value = 3;
  int64 eventid = 4;
  int64 p_eventid = 5;
  string name = 6;
  Severity severity = 7;
  repeated Host hosts = 8;
  repeated string groups = 9;
  repeated Tag tags = 10;
}
```

#### Host

Host information:

```protobuf
message Host {
  string host = 1;  // Technical host name
  string name = 2;  // Display name
}
```

#### Tag

Key-value pair tag:

```protobuf
message Tag {
  string tag = 1;   // Tag name/key
  string value = 2; // Tag value
}
```

### Service Definition

#### ObserverService

gRPC service interface for observer plugins:

```protobuf
service ObserverService {
  rpc Initialize(InitializeRequest) returns (InitializeResponse);
  rpc SaveHistory(SaveHistoryRequest) returns (SaveResponse);
  rpc SaveTrends(SaveTrendsRequest) returns (SaveResponse);
  rpc SaveEvents(SaveEventsRequest) returns (SaveResponse);
  rpc Cleanup(CleanupRequest) returns (CleanupResponse);
}
```

### Request/Response Messages

#### InitializeRequest

Sent to initialize a plugin:

```protobuf
message InitializeRequest {
  string name = 1;
  string connection = 2;
  map<string, string> options = 3;
  repeated ExportType exports = 4;
  Filter filter = 5;
}
```

#### InitializeResponse

Returned after initialization:

```protobuf
message InitializeResponse {
  bool success = 1;
  string error = 2;
  PluginInfo plugin_info = 3;
}
```

#### SaveHistoryRequest / SaveTrendsRequest / SaveEventsRequest

Send data to plugins:

```protobuf
message SaveHistoryRequest {
  repeated History history = 1;
}
```

#### SaveResponse

Returned after processing data:

```protobuf
message SaveResponse {
  bool success = 1;
  string error = 2;
  int64 records_processed = 3;
  int64 records_failed = 4;
}
```

### Type Conversion

When working with proto types in plugins, enum values are int32 compatible with zbx types:

```go
// Converting proto types to zbx types
history := zbx.History{
    Type: int32(protoHistory.ValueType),  // proto.ValueType to int32
}

event := zbx.Event{
    Value:    int32(protoEvent.Value),     // proto.EventValue to int32
    Severity: int32(protoEvent.Severity),  // proto.Severity to int32
}
```

The `BaseObserverGRPC` helper methods handle these conversions automatically.

## pkg/filter

The `filter` package provides filtering types and interfaces.

### Import

```go
import "zms.szuro.net/pkg/filter"
```

### Types

#### FilterConfig

Configuration structure for filters:

```go
type FilterConfig struct {
    Type     string
    Accepted []string
    Rejected []string
}
```

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

## Go Documentation

For complete API documentation, use `go doc`:

```bash
# View package documentation
go doc zms.szuro.net/pkg/zbx

# View type documentation
go doc zms.szuro.net/pkg/zbx.History

# View method documentation
go doc zms.szuro.net/pkg/zbx.History.IsNumeric
```

Or visit [pkg.go.dev](https://pkg.go.dev/zms.szuro.net) for online documentation.

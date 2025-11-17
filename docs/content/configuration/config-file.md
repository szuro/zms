---
title: "Configuration File"
description: "Complete reference for ZMS configuration file"
weight: 1
toc: true
---

# Configuration File Reference

A sample configuration file for ZMS:

```yaml
# Zabbix server configuration (for FILE mode)
server_config: /etc/zabbix/zabbix_server.conf

# Buffer size for batch processing
buffer_size: 100

# Directory containing plugin executables (optional)
plugins_dir: /usr/lib/zms/plugins

# HTTP mode configuration (optional - alternative to FILE mode)
# http:
#   listen_address: localhost
#   listen_port: 2020

# Global tag filters (optional)
filter:
  accepted:
  - "tag_name:tag_value"
  - "environment:production"
  rejected:
  - "ignore:true"

# Output targets
targets:
- name: <unique_name>
  type: pushgateway|azuretable|print|psql|log_print
  connection: <connectionstring>

  # Plugin-specific options (optional)
  options:
    custom_option: value

  # Per-target tag filters (optional)
  filter:
    accepted:
    - "tag_name:tag_value"
    rejected:
    - "ignore:true"

  # Export types to process
  exports:
  - history
  - trends
  - events

# Example plugin configurations:

# Built-in targets
- name: metrics_gateway
  type: pushgateway
  connection: http://localhost:9091
  exports:
  - history

# Plugin example: log_print (filters LOG type history items)
- name: log_output
  type: log_print
  connection: stdout  # or "stderr"
  exports:
  - history

# Plugin example: PostgreSQL
- name: postgres_db
  type: psql
  connection: postgres://user:pass@localhost/zabbix
  exports:
  - history
  options:
    max_connections: "10"
```

## Configuration Parameters

### server_config

Absolute path to Zabbix Server config. Must be readable by ZMS. It is used to get the number of DBSyncers running and export configuration, thus getting the number of export files and their paths.

**Type:** String (file path)
**Required:** Yes (for FILE mode)
**Example:** `/etc/zabbix/zabbix_server.conf`

### buffer_size

Size of local in-memory buffer. It is shared between targets. Setting buffer to N will force ZMS to send N values in one batch request if possible (not all targets support batch operations).

**Type:** Integer
**Default:** 100
**Example:** `buffer_size: 100`

### plugins_dir

Optional path to directory containing plugin executables. ZMS will search this directory for plugin binaries when loading targets.

**Type:** String (directory path)
**Default:** `./plugins`
**Example:** `/usr/lib/zms/plugins`

Plugins are standalone executable binaries (not shared libraries) that implement the gRPC observer interface using HashiCorp's go-plugin framework. Each plugin runs as a separate process and communicates with ZMS via gRPC using Protocol Buffers.

### http

Optional HTTP mode configuration. When specified, ZMS runs an HTTP server to receive data instead of reading from export files.

**Type:** Object
**Required:** No

**Fields:**
- `listen_address` - Address to bind the HTTP server (e.g., `localhost`, `0.0.0.0`)
- `listen_port` - Port number for the HTTP server (default: 2020)

**Example:**
```yaml
http:
  listen_address: localhost
  listen_port: 2020
```

This mode is mutually exclusive with file-based processing.

### filter

Optional filtering based on Zabbix item tags. May be useful when presented with a significant amount of data. No filter means every value is accepted and sent to configured targets.

**Type:** Object with `accepted` and `rejected` arrays

#### Filter Format

Filters use the format `"tag_name:tag_value"` as strings in YAML arrays:

```yaml
filter:
  accepted:
  - "environment:production"
  - "application:web"
  rejected:
  - "debug:true"
  - "ignore:yes"
```

#### Filter Logic

- **only accepted provided** → only matching tags are allowed
- **only rejected specified** → everything is allowed except for matching tags
- **both accepted and rejected provided** → only accepted tags that were not rejected later are accepted

Tag names and values _must_ be exact. Currently regex or wildcards are not supported.

### targets

This describes the locations to send data to. This is an array of target configurations.

**Type:** Array of target objects
**Required:** Yes

#### Target Configuration

##### name

A unique identifier for a target. Only used internally for bookkeeping and logging.

**Type:** String
**Required:** Yes
**Example:** `name: "my_pushgateway"`

##### type

Target type, or destination. Currently supported built-in targets:
- `pushgateway` - Prometheus Pushgateway
- `azuretable` - Azure Table Storage
- `print` - Standard output (stdout/stderr)
- `psql` - PostgreSQL database

For plugins, use the plugin executable name (e.g., `log_print`).

It is possible to define multiple targets with the same type, given that their names are unique.

**Type:** String
**Required:** Yes

##### connection

Connection string specific to the target type. See [Target Overview](#target-overview) for details.

**Type:** String
**Required:** Yes

##### exports

Determines which type of exported data should be sent to this target. ZMS can only send what's exported by Zabbix. If there's a mismatch, there will be an error.

**Type:** Array of strings
**Required:** Yes

Supported values:
- `history` - Historical data (item values)
- `trends` - Trend data (aggregated statistics)
- `events` - Event data

Note that it is possible to send different exports to different targets.

##### options

Optional key-value pairs for plugin-specific configuration. Different plugins may support different options.

**Type:** Object (key-value pairs)
**Required:** No

**Example:**
```yaml
options:
  max_connections: "10"
  custom_setting: "value"
```

For example, the `psql` plugin supports `max_connections` to configure the database connection pool.

##### filter

Per-target tag filters. Same format and logic as global filters. These are applied in addition to global filters.

**Type:** Object with `accepted` and `rejected` arrays
**Required:** No

## Target Overview

Here's an overview of what's supported for each target along with the meaning of `connection`:

| Target            | History | Trends | Events | Offline buffer | Connection                                                                                       |
| ----------------- | ------- | ------ | ------ | -------------- | ------------------------------------------------------------------------------------------------ |
| azuretable        | yes     | no     | no     | yes            | Storage account SAS URL                                                                          |
| gcp_cloud_monitor | yes     | no     | no     | no             | Absolute path to file with access credentials. If empty, GOOGLE_APPLICATION_CREDENTIALS is used  |
| print             | yes     | yes    | no     | yes            | stdout/stderr                                                                                    |
| pushgateway       | yes     | no     | no     | no             | URL of Pushgateway. May contain user and password                                                |
| psql              | yes     | no     | no     | yes            | PostgreSQL connection string                                                                     |

## Configuration Examples

### Basic File Mode

```yaml
server_config: /etc/zabbix/zabbix_server.conf
buffer_size: 100

targets:
- name: prom_gateway
  type: pushgateway
  connection: http://localhost:9091
  exports:
  - history
```

### HTTP Mode

```yaml
buffer_size: 50
http:
  listen_address: 0.0.0.0
  listen_port: 2020

targets:
- name: postgres
  type: psql
  connection: postgres://user:pass@localhost/zabbix
  exports:
  - history
  - trends
```

### With Filters

```yaml
server_config: /etc/zabbix/zabbix_server.conf

filter:
  accepted:
  - "environment:production"

targets:
- name: prod_metrics
  type: pushgateway
  connection: http://prom.example.com:9091
  exports:
  - history
  filter:
    rejected:
    - "debug:true"
```

### Multiple Targets

```yaml
server_config: /etc/zabbix/zabbix_server.conf
plugins_dir: /usr/lib/zms/plugins

targets:
- name: pushgateway
  type: pushgateway
  connection: http://localhost:9091
  exports:
  - history

- name: postgres_archive
  type: psql
  connection: postgres://zabbix:password@db.example.com/zabbix
  exports:
  - history
  - trends

- name: log_output
  type: log_print
  connection: stdout
  exports:
  - history
```

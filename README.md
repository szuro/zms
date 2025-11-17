# Zabbix Metric Shipper
Make data exported by Zabbix fly!

This program is designed to parse files created by Zabbix export, filter values based on tags and send them to a configured destination.

Features:
- Autodiscovery od export files (requires read perms to zabbix_server.conf file)
- Global tag filters
- Tag filters per target
- Internal Prometheus metrics
- Configurable buffer

# CLI arguments

`-c` - Path to zms config file<br>
`-v` - Show version info

# Configuration

A sample configuration file may be found below:

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

The parameters have the following meaning:

## server_config

Absolute path to Zabbix Server config. Must be readable by ZMS. It is used to get the number of DBSyncers running and export configuration, thus getting the number of export files and their paths.

## buffer_size

Size of local in-memory buffer. It is shared between targets. Setting buffer to N will force ZMS to send N values one batch request if possible (not all targets support this).

## plugins_dir

Optional path to directory containing plugin executables. ZMS will search this directory for plugin binaries when loading targets.
Defaults to `./plugins` if not specified.

Plugins are standalone executables (not shared libraries) that implement the gRPC observer interface.

## http

Optional HTTP mode configuration. When specified, ZMS runs an HTTP server to receive data instead of reading from export files.

Fields:
- `listen_address` - Address to bind the HTTP server (e.g., `localhost`, `0.0.0.0`)
- `listen_port` - Port number for the HTTP server

This mode is mutually exclusive with file-based processing.

## filter

Optional filtering based on Zabbix item tags. May be useful when presented with a significant amount of data. No filter means every value is accepted and sent to configured targets.

### Filter Format

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

### Filter Logic

- **only accepted provided** → only matching tags are allowed
- **only rejected specified** → everything is allowed except for matching tags
- **both accepted and rejected provided** → only accepted tags that were not rejected later are accepted

Tag names and values _must_ be exact. Currently regex or wildcards are not supported.

## targets

This describes the location to send data to.
Currently only History and Trends exports are supported (and not in all targets). This will change in the future.

### name

A unique identifier for a target. Only used internally for bookkeeping and logging.

### type

Target type, or destination if you will.
Currently supported targets are:
- pushgateway
- gcp_cloud_monitor
- azuretable
- print

It is possible to define multiple targets with the same type, given that their names are unique.

### connection

Connection specific to the target type.

### exports

Determines which type of exported data should be sent to this target. ZMS can only send what's exported by Zabbix.
If there's a mismatch, there will be an error.
Note that it is possible to send different exports to different targets.

Supported values:
- `history` - Historical data (item values)
- `trends` - Trend data (aggregated statistics)
- `events` - Event data

### options

Optional key-value pairs for plugin-specific configuration. Different plugins may support different options.
For example, the `psql` plugin supports `max_connections` to configure the database connection pool.

# Target overview

Here's an overwiev of what's supported for each target along with the meaning of `connection`.

| Target            | History | Trends | Events | Offline buffer | Connection                                                                                       |
| ----------------- | ------- | ------ | ------ | -------------- | ------------------------------------------------------------------------------------------------ |
| azuretable        | yes     | no     | no     | yes            | Storage account SAS URL.                                                                         |
| gcp_cloud_monitor | yes     | no     | no     | no             | Absolute path to file with access credentials. If empty, GOOGLE_APPLICATION_CREDENTIALS is used. |
| print             | yes     | yes    | no     | yes            | stdout/stderr.                                                                                   |
| pushgateway       | yes     | no     | no     | no             | URL of Pushgateway. May contain user and password.                                               |
| psql              | yes     | no     | no     | yes            | PSQL connection string                                                                           |

# Running

It is fairly simple to run ZMS. Simply run `zmsd -c /etc/zmsd.yaml`. Of course the config file should exist.

For your convenience, a sample systemd service file is included in this repository: `zmsd.service`.

# Building

To build ZMS from source, you can use the included build PowerShell scrip.

`$ build.ps1`

When on linux you can use this oneliner:

# Contributing

All supported targets can be found in the `observer` directory.
To add your own, simply create a struct that satisfies the criteria:
- Embeds `baseObserver`
- Implements `Observer` interface

---
title: "Overview"
description: "Introduction to ZMS and its features"
weight: 1
---

# Zabbix Metric Shipper

Make data exported by Zabbix fly!

This program is designed to parse files created by Zabbix export, filter values based on tags and send them to a configured destination.

## Features

- **Autodiscovery of export files** - Requires read permissions to zabbix_server.conf file
- **Global tag filters** - Filter data at the application level
- **Per-target tag filters** - Apply different filters to different destinations
- **Internal Prometheus metrics** - Monitor ZMS performance
- **Configurable buffer** - Batch processing for efficiency
- **Plugin system** - Extend functionality with HashiCorp go-plugin based architecture
  - Process isolation for stability
  - gRPC communication with Protocol Buffers
  - No Go version matching required
- **Offline buffering** - Automatic retry with BadgerDB for failed deliveries
- **Multiple input modes** - File-based or HTTP server mode

## CLI Arguments

- `-c` - Path to ZMS config file
- `-v` - Show version info

## Quick Start

```bash
# Download and build
go install zms.szuro.net/cmd/zmsd@latest

# Run with config
zmsd -c /path/to/config.yaml
```

## Running

It is fairly simple to run ZMS. Simply run:

```bash
zmsd -c /etc/zmsd.yaml
```

Of course the config file should exist. For your convenience, a sample systemd service file is included in the repository: `zmsd.service`.

## Building from Source

To build ZMS from source, you can use the included build PowerShell script:

```powershell
./build.ps1
```

On Linux, you can use this one-liner:

```bash
go build -trimpath -ldflags="-X zms.szuro.net/internal/config.Version=0.5.1 -X zms.szuro.net/internal/config.Commit=$(git log -n 1 --pretty=format:'%H') -X zms.szuro.net/internal/config.BuildDate=$(date -u +'%Y-%m-%dT%H:%M:%S.%3NZ')" -o zmsd ./cmd/zmsd
```

## Architecture

ZMS follows a modular architecture with clear separation of concerns:

### Directory Structure

```
cmd/zmsd/           # Main application entry point
internal/
  config/           # Configuration management
  input/            # Input layer (FileInput, HTTPInput)
  observer/         # Output targets (built-in observers)
  zbx/              # Zabbix integration
  filter/           # Tag-based filtering system
  logger/           # Logging utilities
  plugin/           # Plugin loader and registry
pkg/
  zbx/              # Public Zabbix types
  plugin/           # Plugin interface
  filter/           # Public filter types
```

### Core Components

1. **Input Layer** - Reads data from files or HTTP
2. **Filtering** - Tag-based filtering at global and target levels
3. **Observer Layer** - Sends data to configured destinations
4. **Plugin System** - Extensible architecture for custom observers
5. **Offline Buffer** - BadgerDB-backed persistence for failed deliveries

For more detailed architecture information, see the [Architecture Documentation](../architecture/).

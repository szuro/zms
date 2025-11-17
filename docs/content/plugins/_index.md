---
title: "Plugins"
description: "ZMS plugin system documentation"
weight: 3
---

# Plugins

ZMS uses HashiCorp's go-plugin framework for a robust plugin system. Plugins are standalone executables that run as separate processes, providing process isolation, crash resilience, and version independence.

## Plugin Documentation

- **[Built-in Plugins](builtin-plugins)** - Documentation for all included observer plugins (PostgreSQL, Azure, Prometheus, GCP, etc.)
- **[Plugin Development](plugin-development.md)** - Guide to creating custom observer plugins

## Quick Links

### Using Built-in Plugins

ZMS includes several production-ready plugins:
- **PostgreSQL** - Relational database storage
- **Azure Table Storage** - Cloud-native Azure integration
- **Prometheus Remote Write** - Prometheus monitoring integration
- **GCP Cloud Monitor** - Google Cloud Platform metrics
- **Prometheus Pushgateway** - Push-based Prometheus metrics
- **Print** - Debug output to stdout/stderr

See [Built-in Plugins](builtin-plugins.md) for detailed configuration and usage.

### Creating Custom Plugins

Learn how to build your own observer plugins using the plugin development guide. All plugins implement the same gRPC-based interface using Protocol Buffers for type-safe communication.

See [Plugin Development](plugin-development.md) for implementation details and examples.

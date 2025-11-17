---
title: "Debugging Guide"
description: "Guide to debugging ZMS with VSCode and gRPC plugins"
weight: 6
toc: true
---

# Debugging Guide

This guide explains how to debug ZMS with gRPC plugins in VSCode.

## Available Launch Configurations

### 1. Launch Debug profile (Legacy)
Original debug configuration using debug tags.
- **Use case**: Basic debugging of ZMS core functionality
- **Config file**: `zmsd-test.yaml`
- **Build flags**: `-tags debug`

### 2. ZMS with Print Plugin (Recommended)
Launches ZMS with the print plugin pre-built and ready to use.
- **Use case**: Debugging ZMS with gRPC plugin integration
- **Config file**: `zmsd-test.yaml`
- **Pre-launch task**: Automatically builds the print plugin
- **Build flags**: Includes version info and ldflags
- **Environment**: `ZMS_LOG_LEVEL=debug` for verbose logging
- **Console**: Integrated terminal for better output visibility

**How to use:**
1. Press `F5` or select "ZMS with Print Plugin" from the debug dropdown
2. The print plugin will be built automatically before launch
3. ZMS will start with debug logging enabled
4. Plugin output will appear in the integrated terminal

### 3. Debug Print Plugin (Standalone)
Launches the print plugin as a standalone process for testing.
- **Use case**: Testing plugin in isolation without ZMS
- **Program**: `plugins/print/print.go`
- **Console**: Integrated terminal

**How to use:**
1. Select "Debug Print Plugin" from the debug dropdown
2. The plugin will run standalone (useful for testing plugin logic)
3. Note: This is mainly for plugin development, not normal usage

## Build Tasks

The following VSCode tasks are available (via `Ctrl+Shift+B` or Terminal > Run Build Task):

### build-print-plugin
Builds the print plugin executable.
- **Output**: `build/bin/plugins/print`
- **Source**: `plugins/print/print.go`

### build-log-print-plugin
Builds the log_print example plugin executable.
- **Output**: `build/bin/plugins/log_print`
- **Source**: `examples/plugins/log_print/log_print.go`

### build-all-plugins (Default)
Builds all plugins in parallel.
- **Dependencies**: build-print-plugin, build-log-print-plugin

### clean-plugins
Removes all built plugin executables.
- **Removes**: `build/bin/plugins` directory

## Configuration Files

### zmsd-test.yaml
Test configuration file used by debug launches.

**Key settings:**
```yaml
server_config: /home/szuro/repos/zms/zabbix_server.conf
buffer_size: 1
plugins_dir: ./build/bin/plugins  # Points to VSCode-built plugins
data_dir: /tmp
```

**Configured targets:**
- `print2` (type: `log_print`) - Filters LOG-type history items
- `print` (type: `print`) - Prints all history items to stderr

## Plugin Architecture

ZMS uses HashiCorp's go-plugin framework for gRPC-based plugins:

- **Main Process**: ZMS core (`cmd/zmsd/main.go`)
- **Plugin Processes**: Independent executables in `build/bin/plugins/`
- **Communication**: gRPC with Protocol Buffers
- **Isolation**: Plugins run as separate processes

## Debugging Workflow

### Standard Debugging Session

1. **Set breakpoints** in ZMS code or plugin code
2. **Launch** with "ZMS with Print Plugin" configuration
3. **Monitor** output in the integrated terminal
4. **Step through** code as ZMS processes data

### Plugin Development Workflow

1. **Edit** plugin code in `plugins/print/print.go`
2. **Rebuild** using task: `Ctrl+Shift+B` > "build-print-plugin"
3. **Restart** debugger to load new plugin binary
4. **Test** changes

### Multi-Process Debugging

To debug both ZMS and the plugin simultaneously:

1. **Launch** ZMS with "ZMS with Print Plugin"
2. **Attach** to the plugin process using "Debug Print Plugin"
3. **Note**: The plugin runs as a child process of ZMS

## Environment Variables

The following environment variables are set for debugging:

- `ZMS_LOG_LEVEL=debug` - Enables verbose logging

You can add more in the launch configuration's `env` section.

## Troubleshooting

### Plugin Not Found
**Error**: `Failed to load plugin: exec: "print": executable file not found`

**Solution**: Run the "build-print-plugin" task or "build-all-plugins"

### Plugin Handshake Failed
**Error**: `plugin handshake failed`

**Solution**: Ensure plugin was built with matching `pkg/plugin` version

### Permission Denied
**Error**: `permission denied` when launching plugin

**Solution**: Ensure plugin executable has execute permissions:
```bash
chmod +x build/bin/plugins/print
```

### Config File Not Found
**Error**: `config file not found`

**Solution**: Ensure `zmsd-test.yaml` exists and paths are correct

### Zabbix Server Config Missing
**Error**: `failed to parse server config`

**Solution**: Update `server_config` path in `zmsd-test.yaml` to point to valid Zabbix server config

## Tips

1. **Use Integrated Terminal**: Set `"console": "integratedTerminal"` to see plugin output clearly
2. **Rebuild Before Launch**: The preLaunchTask ensures plugins are always fresh
3. **Check Plugin Logs**: Plugins use structured logging - check stderr for plugin messages
4. **Version Info**: Build flags include version/commit/date for debugging builds
5. **Clean Builds**: Use "clean-plugins" task if you encounter caching issues

## Related Files

- `.vscode/launch.json` - Debug configurations
- `.vscode/tasks.json` - Build tasks
- `zmsd-test.yaml` - Test configuration
- `plugins/print/print.go` - Print plugin source
- `examples/plugins/log_print/log_print.go` - Log print plugin example

## References

- [Architecture](architecture.md) - ZMS architecture overview
- [CLAUDE.md](https://github.com/yourusername/zms/blob/master/CLAUDE.md) - ZMS development guide
- [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin) - Plugin framework documentation
- [VSCode Go Debugging](https://github.com/golang/vscode-go/wiki/debugging) - VSCode Go extension debugging guide

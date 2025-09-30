# ZMS Docker Plugin Builder

A complete build environment for ZMS plugins using Docker.

## Quick Start

### 1. Build the Docker Image

```bash
docker build -f Dockerfile.plugin-builder -t zms-build:latest .
```

### 2. Build Your Plugin

**Simple Usage (output to host):**
```bash
docker run -v /path/to/your-plugin:/plugins zms-build:latest
```

**With Custom Output Directory:**
```bash
docker run -v /path/to/your-plugin:/plugins -v /path/to/output:/output zms-build:latest
```

**Interactive Development:**
```bash
docker run --rm -it -v /path/to/your-plugin:/plugins zms-build:latest bash
```

## Plugin Directory Structure

Your plugin directory should contain:

```
my-plugin/
├── main.go          # Plugin implementation (required)
├── go.mod           # Go module file (optional, will be auto-generated)
└── other_files.go   # Additional Go files (optional)
```

## Example Plugin

Create a directory with a simple plugin:

```go
// main.go
package main

import (
    "fmt"
    "szuro.net/zms/pkg/plugin"
    zbxpkg "szuro.net/zms/pkg/zbx"
)

var PluginInfo = plugin.PluginInfo{
    Name:        "my-plugin",
    Version:     "1.0.0",
    Description: "My custom ZMS plugin",
    Author:      "Your Name",
}

type MyPlugin struct {
    plugin.BaseObserverImpl
}

func NewObserver() plugin.Observer {
    return &MyPlugin{}
}

func (p *MyPlugin) Initialize(connection string, options map[string]string) error {
    fmt.Printf("Plugin initialized with connection: %s\n", connection)
    return nil
}

func (p *MyPlugin) GetType() string {
    return "my-plugin"
}

func (p *MyPlugin) SaveHistory(h []zbxpkg.History) bool {
    fmt.Printf("Processing %d history items\n", len(h))
    return true
}
```

Then build it:

```bash
docker run -v /path/to/my-plugin:/plugins /path/to/output:/output zms-build:latest
```

## Features

- **Complete Build Environment**: Includes Go 1.25.1, gcc, and all dependencies
- **Automatic go.mod Management**: Creates and manages go.mod files automatically
- **ZMS Integration**: Pre-configured with ZMS dependencies and types
- **Smart Naming**: Automatically determines output plugin name
- **Cross-Platform**: Builds Linux plugins regardless of host OS
- **Verbose Output**: Shows build progress and usage instructions

## Output

The builder will create a `.so` file in the output directory (defaults to the same directory as the plugin source). The built plugin can then be used in your ZMS configuration:

```yaml
targets:
  - name: "my-custom-plugin"
    type: "my-plugin"
    connection: "your-connection-string"
    options:
      key1: "value1"
    exports:
      - "history"
```

## Troubleshooting

- **No .go files found**: Ensure your plugin directory contains Go source files
- **Build fails**: Check that your plugin implements the required plugin.Observer interface
- **Import errors**: The builder automatically handles ZMS dependencies with local replacements
- **Permission issues**: Ensure the mounted directories have correct permissions for Docker

## Advanced Usage

### Environment Variables

The Docker image sets these environment variables:

- `CGO_ENABLED=1`: Enables CGO for plugin building
- `GOOS=linux`: Builds for Linux target
- `GOARCH=amd64`: Builds for x86_64 architecture
- `GO111MODULE=on`: Enables Go modules

### Custom Build Commands

For advanced use cases, you can override the default build script:

```bash
docker run --rm -it -v /path/to/your-plugin:/plugins zms-build:latest bash
cd /plugins
go build -buildmode=plugin -o /output/custom-name.so .
```

### Multiple Plugins

To build multiple plugins at once:

```bash
for plugin in plugin1 plugin2 plugin3; do
    docker run -v /path/to/$plugin:/plugins -v /path/to/output:/output zms-build:latest
done
```
#!/bin/bash
set -e

PLUGIN_DIR="/plugins"
OUTPUT_DIR="/output"

# If /output directory is not mounted (empty), output to plugin directory
if [ ! "$(ls -A "$OUTPUT_DIR" 2>/dev/null)" ] && [ ! -w "$OUTPUT_DIR" ]; then
    OUTPUT_DIR="$PLUGIN_DIR"
fi

echo "ZMS Plugin Builder"
echo "=================="
echo "Plugin directory: $PLUGIN_DIR"
echo "Output directory: $OUTPUT_DIR"
echo ""

# Check if plugin directory exists and has content
if [ ! -d "$PLUGIN_DIR" ]; then
    echo "Error: Plugin directory $PLUGIN_DIR not found"
    echo "Make sure to mount your plugin directory with: -v /path/to/plugin:/plugins"
    exit 1
fi

# Find Go files
GO_FILES=$(find "$PLUGIN_DIR" -name "*.go" | head -1)
if [ -z "$GO_FILES" ]; then
    echo "Error: No .go files found in $PLUGIN_DIR"
    echo "Plugin directory should contain Go source files"
    exit 1
fi

echo "Found Go files in plugin directory"

# Change to plugin directory
cd "$PLUGIN_DIR"

# Initialize go.mod if it doesn't exist or fix problematic module name
if [ ! -f "go.mod" ]; then
    echo "Initializing go.mod for plugin..."
    go mod init zms-plugin
    echo "require szuro.net/zms v0.0.0" >> go.mod
    echo "replace szuro.net/zms => /workspace" >> go.mod
    echo ""
elif grep -q "^module plugin$" go.mod; then
    echo "Fixing conflicting module name in go.mod..."
    sed -i 's/^module plugin$/module zms-plugin/' go.mod
    echo ""
fi

# Ensure ZMS dependency is available
if ! grep -q "szuro.net/zms" go.mod; then
    echo "Adding ZMS dependency to go.mod..."
    echo "require szuro.net/zms v0.0.0" >> go.mod
    echo "replace szuro.net/zms => /workspace" >> go.mod
fi

# Update go.mod with local ZMS reference
if ! grep -q "replace szuro.net/zms" go.mod; then
    echo "replace szuro.net/zms => /workspace" >> go.mod
fi

# Download dependencies
echo "Downloading dependencies..."
go mod tidy
echo ""

# Determine plugin name and output file
PLUGIN_NAME=""
if [ -f "main.go" ]; then
    PLUGIN_NAME="plugin"
elif [ $(find . -name "*.go" | wc -l) -eq 1 ]; then
    # Single Go file, use its name without extension
    PLUGIN_NAME=$(basename $(find . -name "*.go") .go)
else
    # Multiple files, use directory name
    PLUGIN_NAME=$(basename "$PLUGIN_DIR")
fi

OUTPUT_FILE="$OUTPUT_DIR/${PLUGIN_NAME}.so"

echo "Building plugin: $PLUGIN_NAME"
echo "Output file: $OUTPUT_FILE"
echo ""

# Build the plugin
echo "Compiling plugin..."
go build -ldflags "-s -w" -buildmode=plugin -o "$OUTPUT_FILE" .

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Plugin built successfully: $OUTPUT_FILE"
    echo ""
    echo "Plugin info:"
    ls -la "$OUTPUT_FILE"
    echo ""
    echo "To use this plugin:"
    echo "1. Copy $OUTPUT_FILE to your ZMS plugins directory"
    echo "2. Configure the plugin in your zmsd.yaml:"
    echo "   targets:"
    echo "     - name: my-plugin"
    echo "       type: ${PLUGIN_NAME}.so"
    echo "       connection: your-connection-string"
    echo "       source: [\"history\", \"trends\", \"events\"]"
else
    echo ""
    echo "❌ Plugin build failed"
    exit 1
fi
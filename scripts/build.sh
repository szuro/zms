#!/bin/bash
set -e

echo "ZMS Builder"
echo "==========="
echo "Building ZMS application..."
echo ""

# Get version information
VERSION=${VERSION:-"0.5.1"}
COMMIT=$(git log -n 1 --pretty=format:"%H" 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ")

echo "Version: $VERSION"
echo "Commit: $COMMIT"
echo "Build Date: $BUILD_DATE"
echo ""

# Build ZMS
echo "Compiling ZMS..."
go build -trimpath \
    -ldflags="-w -s -X szuro.net/zms/internal/config.Version=$VERSION -X szuro.net/zms/internal/config.Commit=$COMMIT -X szuro.net/zms/internal/config.BuildDate=$BUILD_DATE" \
    -o /output/zmsd ./cmd/zmsd

echo ""
echo "✅ ZMS built successfully: /output/zmsd"
echo ""

# Show binary info
echo "Binary info:"
file /output/zmsd
ls -la /output/zmsd
echo ""

# Test the binary
echo "Testing binary..."
/output/zmsd --help 2>&1 | head -10 || echo "Binary test completed"

PLUGINS=$(find plugins -name '*.go')

for plugin in $PLUGINS; do 
    p=$(basename $plugin)
    echo "Building plugin: $p"; 
    go build -ldflags "-s -w" -buildmode=plugin -o /output/plugins/${p%.go}.so $plugin
done

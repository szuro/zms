#!/bin/bash
# Deployment script for ZMS documentation

set -e

echo "Building ZMS documentation..."

# Clean previous build
rm -rf public/

# Build with Hugo
hugo --minify

echo "Build complete! Site generated in public/"
echo ""
echo "To deploy:"
echo "  - Copy public/ to your web server"
echo "  - Ensure it's accessible at https://szuro.net/zms/"
echo ""
echo "Testing vanity URL:"
echo "  curl -H 'User-Agent: Go-http-client/1.1' https://szuro.net/zms/"

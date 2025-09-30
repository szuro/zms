#!/bin/bash
# Pre-removal script for ZMS RPM package

set -e

# Stop and disable service if running
if systemctl is-active --quiet zmsd 2>/dev/null; then
    echo "Stopping ZMS service..."
    systemctl stop zmsd
fi

if systemctl is-enabled --quiet zmsd 2>/dev/null; then
    echo "Disabling ZMS service..."
    systemctl disable zmsd
fi

exit 0
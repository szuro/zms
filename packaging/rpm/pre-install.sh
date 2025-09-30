#!/bin/bash
# Pre-installation script for ZMS RPM package

set -e

# Create zms user if it doesn't exist
if ! id -u zms >/dev/null 2>&1; then
    echo "Creating zms user..."
    useradd -r -s /bin/false -d /var/lib/zms -c "ZMS Service User" zms
fi

# Create directories
mkdir -p /var/lib/zms
mkdir -p /etc/zms
mkdir -p /usr/lib/zms/plugins

exit 0
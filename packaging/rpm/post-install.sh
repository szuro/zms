#!/bin/bash
# Post-installation script for ZMS RPM package

set -e

# Set ownership and permissions
chown -R zms:zms /var/lib/zms
chmod 755 /usr/bin/zmsd
chmod 644 /usr/lib/systemd/system/zmsd.service
chmod -R 755 /usr/lib/zms/plugins
chmod 640 /etc/zms/zmsd.yaml
chmod 640 /etc/sysconfig/zms
chown root:zms /etc/sysconfig/zms

# Set SELinux contexts if SELinux is enabled
if command -v semanage >/dev/null 2>&1 && getenforce >/dev/null 2>&1; then
    echo "Setting SELinux contexts..."
    restorecon -R /usr/bin/zmsd /usr/lib/zms /var/lib/zms /etc/zms >/dev/null 2>&1 || true
fi

# Reload systemd
systemctl daemon-reload

# Create sample config if none exists
if [ ! -f /etc/zms/zmsd.yaml ] && [ -f /etc/zms/zmsd.yaml.example ]; then
    cp /etc/zms/zmsd.yaml.example /etc/zms/zmsd.yaml
    echo "Sample configuration created at /etc/zms/zmsd.yaml"
fi

echo "ZMS installation completed successfully!"
echo ""
echo "Next steps:"
echo "1. Edit configuration: /etc/zms/zmsd.yaml"
echo "2. Enable service: systemctl enable zmsd"
echo "3. Start service: systemctl start zmsd"
echo "4. Check status: systemctl status zmsd"

exit 0
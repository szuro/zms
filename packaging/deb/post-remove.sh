#!/bin/bash
# Post-removal script for ZMS DEB package

set -e

# Reload systemd after service file removal
systemctl daemon-reload

# Note: We don't remove the zms user or /var/lib/zms directory
# to preserve any data or logs that might be important
echo "ZMS has been removed."
echo ""
echo "Note: The zms user and /var/lib/zms directory have been preserved."
echo "Remove manually if no longer needed:"
echo "  userdel zms"
echo "  rm -rf /var/lib/zms"

exit 0
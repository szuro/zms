# ZMS Packaging

This directory contains packaging scripts for creating RPM and DEB packages of ZMS (Zabbix Metric Shipper).

## Prerequisites

- **fpm**: The Effing Package Management tool
  ```bash
  gem install fpm
  ```
- **Go**: Version 1.25.1 or later
- **Git**: For version information

## Package Structure

### Installation Paths
- **Binary**: `/usr/bin/zmsd`
- **Plugins**: `/usr/lib/zms/plugins/`
- **Configuration**: `/etc/zms/zmsd.yaml`
- **Service File**: `/usr/lib/systemd/system/zmsd.service`
- **Data Directory**: `/var/lib/zms`

### User and Permissions
- **Service User**: `zms` (system user)
- **Service Group**: `zms`
- **Data Directory Owner**: `zms:zms`

## Building Packages

### Build All Packages
```bash
make package
```

### Build Specific Package Type
```bash
# RPM package
make package-rpm

# DEB package
make package-deb
```

### Custom Version
```bash
VERSION=1.2.3 make package
```

## Package Installation

### RPM (RHEL/CentOS/Fedora)
```bash
sudo rpm -ivh dist/zms-VERSION-1.x86_64.rpm
```

### DEB (Debian/Ubuntu)
```bash
sudo dpkg -i dist/zms_VERSION_amd64.deb
```

## Post-Installation

1. **Edit Configuration**:
   ```bash
   sudo vim /etc/zms/zmsd.yaml
   ```

2. **Enable Service**:
   ```bash
   sudo systemctl enable zmsd
   ```

3. **Start Service**:
   ```bash
   sudo systemctl start zmsd
   ```

4. **Check Status**:
   ```bash
   sudo systemctl status zmsd
   ```

## Package Scripts

### Pre-Installation
- Creates `zms` system user
- Creates required directories

### Post-Installation
- Sets proper ownership and permissions
- Reloads systemd daemon
- Creates sample configuration
- Sets SELinux contexts (RPM only)

### Pre-Removal
- Stops and disables service

### Post-Removal
- Reloads systemd daemon
- Preserves user and data directory

## Configuration

The package installs a sample configuration file at `/etc/zms/zmsd.yaml` with:
- Example plugin configurations
- Commented-out target examples
- Default settings for production use

## Security Features

The systemd service includes security hardening:
- Runs as non-root user (`zms`)
- Private temporary directory
- Protected system directories
- Restricted file system access
- Kernel protection enabled

## Troubleshooting

### Service Won't Start
1. Check configuration syntax:
   ```bash
   sudo zmsd -c /etc/zms/zmsd.yaml -validate
   ```

2. Check service logs:
   ```bash
   sudo journalctl -u zmsd -f
   ```

3. Verify file permissions:
   ```bash
   ls -la /etc/zms/zmsd.yaml
   ls -la /var/lib/zms
   ```

### Package Installation Issues
1. Check dependencies:
   ```bash
   # DEB systems
   sudo apt-get install -f

   # RPM systems
   sudo yum install systemd
   ```

2. Manual cleanup if needed:
   ```bash
   # Remove user and data (if desired)
   sudo userdel zms
   sudo rm -rf /var/lib/zms
   ```

## Development

For local development installation:
```bash
make install
```

This installs directly from the build directory without creating packages.
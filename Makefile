# ZMS Build and Package Makefile

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git log -n 1 --pretty=format:"%H" 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%S.%3NZ")

# Build settings
GO_VERSION = 1.25.1
BINARY_NAME = zmsd
MODULE_NAME = zms.szuro.net
CMD_DIR = ./cmd/zmsd
PLUGINS_DIR = ./plugins
BUILD_DIR = ./build
DIST_DIR = ./dist

# Package settings
PACKAGE_NAME = zms
PACKAGE_VERSION = $(VERSION)
PACKAGE_DESCRIPTION = Zabbix Metric Shipper - ships Zabbix exports to various targets
PACKAGE_URL = https://szuro.net/zms
PACKAGE_MAINTAINER = ZMS Team

# Installation paths
INSTALL_BIN_DIR = /usr/bin
INSTALL_PLUGINS_DIR = /usr/lib/zms/plugins
INSTALL_CONFIG_DIR = /etc/zms
INSTALL_SERVICE_DIR = /usr/lib/systemd/system
INSTALL_VAR_DIR = /var/lib/zms
INSTALL_SYSCONFIG_DIR = /etc/sysconfig
INSTALL_DEFAULT_DIR = /etc/default

# LDFLAGS for version injection
LDFLAGS = -trimpath -ldflags="-X $(MODULE_NAME)/internal/config.Version=$(VERSION) -X $(MODULE_NAME)/internal/config.Commit=$(COMMIT) -X $(MODULE_NAME)/internal/config.BuildDate=$(BUILD_DATE)"

# Default target
all: clean build

# Help target
help:
	@echo "ZMS Build and Package System"
	@echo ""
	@echo "Available targets:"
	@echo "  all                  - Clean and build everything (default)"
	@echo "  build                - Build main binary and all plugins"
	@echo "  build-main           - Build only the main binary"
	@echo "  build-plugins        - Build all plugins"
	@echo "  package              - Create both RPM and DEB packages"
	@echo "  package-rpm          - Create RPM package"
	@echo "  package-deb          - Create DEB package"
	@echo "  install              - Install locally (requires sudo)"
	@echo "  test                 - Run tests"
	@echo "  deps                 - Install build dependencies"
	@echo "  deps-rpm             - Check RPM build dependencies"
	@echo "  clean                - Clean build artifacts"
	@echo "  info                 - Show build information"
	@echo "  help                 - Show this help"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION       - Package version (default: git describe)"
	@echo "  COMMIT        - Git commit hash (default: git log)"
	@echo "  BUILD_DATE    - Build timestamp (default: current UTC)"

# Install build dependencies
deps:
	@echo "Installing build dependencies..."
	@command -v fpm >/dev/null 2>&1 || { echo "Installing fpm..."; gem install fpm; }
	@command -v go >/dev/null 2>&1 || { echo "Go $(GO_VERSION) is required"; exit 1; }
	@echo "Dependencies ready"

# Check RPM build dependencies
deps-rpm: deps
	@echo "Checking RPM build dependencies..."
	@command -v rpmbuild >/dev/null 2>&1 || { echo "RPM build tools required. Install with: sudo apt-get install rpm or sudo yum install rpm-build"; exit 1; }
	@echo "RPM dependencies ready"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f $(BINARY_NAME)
	@find $(PLUGINS_DIR) -maxdepth 1 -type f -executable ! -name "*.go" -delete 2>/dev/null || true
	@echo "Clean complete"

# Create build directories
mkdirs:
	@mkdir -p $(BUILD_DIR)/bin
	@mkdir -p $(BUILD_DIR)/config
	@mkdir -p $(BUILD_DIR)/sysconfig
	@mkdir -p $(BUILD_DIR)/default

$(DIST_DIR):
	@mkdir -p $(DIST_DIR)

# Build main binary
build-main: mkdirs
	@echo "Building main binary..."
	@go build $(LDFLAGS) -o $(BUILD_DIR)/bin/$(BINARY_NAME) $(CMD_DIR)
	@echo "Main binary built: $(BUILD_DIR)/bin/$(BINARY_NAME)"

# Build all plugins
build-plugins: mkdirs
	@echo "Building plugins..."
	@mkdir -p $(BUILD_DIR)/bin/plugins
	@for plugin in $(PLUGINS_DIR)/*/; do \
		if [ -f "$$plugin/main.go" ]; then \
			plugin_name=$$(basename $$plugin); \
			echo "Building plugin: $$plugin_name"; \
			go build -o $(BUILD_DIR)/bin/plugins/$$plugin_name $$plugin; \
		fi \
	done
	@echo "Plugins built in $(BUILD_DIR)/bin/plugins/"

# Build everything
build: build-main build-plugins
	@echo "Build complete"

# Run tests
test:
	@echo "Running tests..."
	@go test ./...
	@echo "Tests complete"

# Copy environment file for RPM
$(BUILD_DIR)/sysconfig/zms: $(BUILD_DIR) packaging/zms.env
	@echo "Copying RPM environment configuration file..."
	@cp packaging/zms.env $(BUILD_DIR)/sysconfig/zms


# Copy environment file for RPM
$(BUILD_DIR)/default/zms: $(BUILD_DIR) packaging/zms.env
	@echo "Copying DEB environment configuration file..."
	@cp packaging/zms.env $(BUILD_DIR)/sysconfig/zms


# Copy sample config
$(BUILD_DIR)/config/zmsd.yaml: $(BUILD_DIR) configs/zmsd.yaml
	@echo "Copying sample configuration..."
	@cp configs/zmsd.yaml $(BUILD_DIR)/config/zmsd.yaml

# Create RPM package
package-rpm: $(BUILD_DIR)/sysconfig/zms $(BUILD_DIR)/config/zmsd.yaml $(DIST_DIR) deps-rpm
	@echo "Creating RPM package..."
	@mkdir -p $(BUILD_DIR)/var/lib/zms $(BUILD_DIR)/usr/lib/zms/plugins-empty
	@fpm -s dir -t rpm \
		--name $(PACKAGE_NAME) \
		--version $(PACKAGE_VERSION) \
		--description "$(PACKAGE_DESCRIPTION)" \
		--url "$(PACKAGE_URL)" \
		--maintainer "$(PACKAGE_MAINTAINER)" \
		--license "MIT" \
		--architecture native \
		--depends "systemd" \
		--rpm-user zms \
		--rpm-group zms \
		--before-install packaging/rpm/pre-install.sh \
		--after-install packaging/rpm/post-install.sh \
		--before-remove packaging/rpm/pre-remove.sh \
		--after-remove packaging/rpm/post-remove.sh \
		--config-files $(INSTALL_CONFIG_DIR)/zmsd.yaml \
		--config-files $(INSTALL_SYSCONFIG_DIR)/zms \
		--package $(DIST_DIR)/$(PACKAGE_NAME)-$(PACKAGE_VERSION)-1.x86_64.rpm \
		$(BUILD_DIR)/bin/$(BINARY_NAME)=$(INSTALL_BIN_DIR)/$(BINARY_NAME) \
		$(BUILD_DIR)/bin/plugins/=$(INSTALL_PLUGINS_DIR)/ \
		$(BUILD_DIR)/config/zmsd.yaml=$(INSTALL_CONFIG_DIR)/zmsd.yaml \
		$(BUILD_DIR)/sysconfig/zms=$(INSTALL_SYSCONFIG_DIR)/zms \
		$(BUILD_DIR)/var/lib/zms=$(INSTALL_VAR_DIR) \
		packaging/zmsd.service=$(INSTALL_SERVICE_DIR)/zmsd.service
	@echo "RPM package created: $(DIST_DIR)/$(PACKAGE_NAME)-$(PACKAGE_VERSION)-1.x86_64.rpm"

# Create DEB package
package-deb: $(BUILD_DIR)/default/zms $(BUILD_DIR)/config/zmsd.yaml $(DIST_DIR) deps
	@echo "Creating DEB package..."
	@mkdir -p $(BUILD_DIR)/var/lib/zms $(BUILD_DIR)/usr/lib/zms/plugins-empty
	@fpm -s dir -t deb \
		--name $(PACKAGE_NAME) \
		--version $(PACKAGE_VERSION) \
		--description "$(PACKAGE_DESCRIPTION)" \
		--url "$(PACKAGE_URL)" \
		--maintainer "$(PACKAGE_MAINTAINER)" \
		--license "MIT" \
		--architecture native \
		--depends "systemd" \
		--deb-user zms \
		--deb-group zms \
		--before-install packaging/deb/pre-install.sh \
		--after-install packaging/deb/post-install.sh \
		--before-remove packaging/deb/pre-remove.sh \
		--after-remove packaging/deb/post-remove.sh \
		--config-files $(INSTALL_CONFIG_DIR)/zmsd.yaml \
		--config-files $(INSTALL_DEFAULT_DIR)/zms \
		--package $(DIST_DIR)/$(PACKAGE_NAME)_$(PACKAGE_VERSION)_amd64.deb \
		$(BUILD_DIR)/bin/$(BINARY_NAME)=$(INSTALL_BIN_DIR)/$(BINARY_NAME) \
		$(BUILD_DIR)/bin/plugins/=$(INSTALL_PLUGINS_DIR)/ \
		$(BUILD_DIR)/config/zmsd.yaml=$(INSTALL_CONFIG_DIR)/zmsd.yaml \
		$(BUILD_DIR)/default/zms=$(INSTALL_DEFAULT_DIR)/zms \
		$(BUILD_DIR)/var/lib/zms=$(INSTALL_VAR_DIR) \
		packaging/zmsd.service=$(INSTALL_SERVICE_DIR)/zmsd.service
	@echo "DEB package created: $(DIST_DIR)/$(PACKAGE_NAME)_$(PACKAGE_VERSION)_amd64.deb"

# Create both packages
package: build package-rpm package-deb

# Local installation (for development/testing)
install: build $(BUILD_DIR)/sysconfig/zms $(BUILD_DIR)/default/zms $(BUILD_DIR)/config/zmsd.yaml
	@echo "Installing ZMS locally..."
	@sudo mkdir -p $(INSTALL_BIN_DIR) $(INSTALL_PLUGINS_DIR) $(INSTALL_CONFIG_DIR) $(INSTALL_VAR_DIR) $(INSTALL_SYSCONFIG_DIR) $(INSTALL_DEFAULT_DIR)
	@sudo cp $(BUILD_DIR)/bin/$(BINARY_NAME) $(INSTALL_BIN_DIR)/
	@if [ -d "$(BUILD_DIR)/bin/plugins" ] && [ -n "$$(ls -A $(BUILD_DIR)/bin/plugins 2>/dev/null)" ]; then \
		sudo cp $(BUILD_DIR)/bin/plugins/* $(INSTALL_PLUGINS_DIR)/; \
	fi
	@sudo cp $(BUILD_DIR)/config/zmsd.yaml $(INSTALL_CONFIG_DIR)/zmsd.yaml.example
	@sudo cp $(BUILD_DIR)/sysconfig/zms $(INSTALL_SYSCONFIG_DIR)/zms
	@sudo cp $(BUILD_DIR)/default/zms $(INSTALL_DEFAULT_DIR)/zms
	@sudo chmod 640 $(INSTALL_SYSCONFIG_DIR)/zms $(INSTALL_DEFAULT_DIR)/zms
	@sudo chown root:zms $(INSTALL_SYSCONFIG_DIR)/zms $(INSTALL_DEFAULT_DIR)/zms
	@sudo cp packaging/zmsd.service $(INSTALL_SERVICE_DIR)/
	@sudo useradd -r -s /bin/false -d $(INSTALL_VAR_DIR) zms 2>/dev/null || true
	@sudo chown -R zms:zms $(INSTALL_VAR_DIR)
	@sudo systemctl daemon-reload
	@echo "ZMS installed. Enable with: sudo systemctl enable zmsd"
	@echo "Configure at: $(INSTALL_CONFIG_DIR)/zmsd.yaml"
	@echo "Environment at: $(INSTALL_SYSCONFIG_DIR)/zms or $(INSTALL_DEFAULT_DIR)/zms"

# Show build information
info:
	@echo "Build Information:"
	@echo "  Version:    $(VERSION)"
	@echo "  Commit:     $(COMMIT)"
	@echo "  Build Date: $(BUILD_DATE)"
	@echo "  Go Version: $(shell go version)"

.PHONY: all build build-main build-plugins test clean deps deps-rpm mkdirs \
	package package-rpm package-deb install info help
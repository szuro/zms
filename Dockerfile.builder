# ZMS Builder
# Complete build environment for ZMS application
#
# Usage:
#   podman build -f Dockerfile.builder -t zms-builder:latest .
#   podman run -v /output:/output zms-builder:latest
#
# This will create a statically linked ZMS binary in /output/zmsd

FROM docker.io/golang:1.25.1-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    git \
    ca-certificates \
    bash \
    make \
    gcc \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /workspace

# Copy Go modules first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY plugins/ .plugins/

# Set build environment variables
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# Create build script
COPY scripts/build.sh /usr/local/bin/

# Make the script executable
RUN chmod +x /usr/local/bin/build.sh

# Create output directory
RUN mkdir -p /output

# Set the default command
CMD ["/usr/local/bin/build.sh"]

# Add helpful labels
LABEL maintainer="ZMS Team"
LABEL description="Complete build environment for ZMS application"
LABEL version="1.0"

# Add usage instructions as environment variable for easy access
ENV USAGE="podman run -v /path/to/output:/output zms-builder:latest"
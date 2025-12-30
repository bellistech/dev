#!/bin/bash
#
# Setup script for Metrics Collection System
#
# This script installs dependencies, generates protobuf code, and builds binaries.

set -e

echo "==================================="
echo " Metrics Collection System Setup"
echo "==================================="
echo ""

# Check for Go
if ! command -v go &> /dev/null; then
    echo "ERROR: Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "Found Go version: $GO_VERSION"

# Check for protoc
if ! command -v protoc &> /dev/null; then
    echo "WARNING: protoc not found. Installing protocol buffers compiler..."
    
    # Detect OS
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64) ARCH="x86_64" ;;
        aarch64|arm64) ARCH="aarch_64" ;;
        *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    
    echo "Please install protoc manually:"
    echo "  - macOS: brew install protobuf"
    echo "  - Ubuntu: sudo apt-get install -y protobuf-compiler"
    echo "  - Or download from: https://github.com/protocolbuffers/protobuf/releases"
    echo ""
fi

# Install Go protoc plugins
echo "Installing protoc Go plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure GOPATH/bin is in PATH
export PATH="$PATH:$(go env GOPATH)/bin"

# Download Go dependencies
echo ""
echo "Downloading Go dependencies..."
go mod download
go mod tidy

# Generate protobuf code
echo ""
echo "Generating protobuf code..."
if command -v protoc &> /dev/null; then
    protoc --go_out=. --go_opt=paths=source_relative \
           --go-grpc_out=. --go-grpc_opt=paths=source_relative \
           api/metrics/v1/metrics.proto
    echo "Protobuf code generated successfully!"
else
    echo "WARNING: Skipping protobuf generation (protoc not found)"
fi

# Build binaries
echo ""
echo "Building binaries..."
mkdir -p bin

echo "  Building agent..."
go build -ldflags "-X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o bin/agent ./cmd/agent

echo "  Building server..."
go build -ldflags "-X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" \
    -o bin/server ./cmd/server

echo ""
echo "==================================="
echo " Setup Complete!"
echo "==================================="
echo ""
echo "Binaries built:"
echo "  - bin/agent"
echo "  - bin/server"
echo ""
echo "Next steps:"
echo "1. Set up PostgreSQL (see scripts/init-db.sql)"
echo "2. Edit configs/server.yaml with your database settings"
echo "3. Edit configs/agent.yaml with your server address"
echo "4. Run the server: ./bin/server -config configs/server.yaml"
echo "5. Run the agent: sudo ./bin/agent -config configs/agent.yaml"
echo ""
echo "Or use Docker Compose:"
echo "  cd deployments/docker && docker-compose up -d"
echo ""

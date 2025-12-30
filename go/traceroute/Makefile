.PHONY: build run clean test help

# Default destination for testing
DEST ?= google.com

# Binary name
BINARY = traceroute

help:
	@echo "╔═══════════════════════════════════════════════════════════════╗"
	@echo "║               🔍 TRACEROUTE - Build Commands                  ║"
	@echo "╚═══════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "  make build        - Compile the traceroute binary"
	@echo "  make run          - Build and run (traces to google.com)"
	@echo "  make run DEST=X   - Build and run with custom destination"
	@echo "  make clean        - Remove compiled binary"
	@echo ""
	@echo "Examples:"
	@echo "  make run                      # Trace to google.com"
	@echo "  make run DEST=amazon.com      # Trace to amazon.com"
	@echo "  make run DEST=8.8.8.8         # Trace to Google DNS"
	@echo ""
	@echo "Note: Requires sudo for raw socket access!"

build:
	@echo "🔨 Building traceroute..."
	go build -o $(BINARY) main.go
	@echo "✅ Built: ./$(BINARY)"

run: build
	@echo ""
	@echo "🚀 Running traceroute to $(DEST)..."
	@echo "   (requires sudo for raw sockets)"
	@echo ""
	sudo ./$(BINARY) $(DEST)

clean:
	@echo "🧹 Cleaning up..."
	rm -f $(BINARY)
	@echo "✅ Clean!"

# Quick test targets for common destinations
google: build
	sudo ./$(BINARY) google.com

cloudflare: build
	sudo ./$(BINARY) 1.1.1.1

amazon: build
	sudo ./$(BINARY) amazon.com

github: build
	sudo ./$(BINARY) github.com

.PHONY: help setup proto build test clean lint fmt docker-build run-server run-agent

# Variables
BINARY_AGENT=bin/agent
BINARY_SERVER=bin/server
PROTO_DIR=api/metrics/v1
GO=go
PROTOC=protoc
DOCKER_REGISTRY?=ghcr.io/bellistech
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Install dependencies and tools
	@echo "Installing protoc plugins..."
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Downloading Go dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Setup complete!"

proto: ## Generate protobuf code
	@echo "Generating protobuf code..."
	$(PROTOC) --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/metrics.proto
	@echo "Protobuf generation complete!"

build: proto build-agent build-server ## Build all binaries
	@echo "Build complete!"

build-agent: ## Build the agent binary
	@echo "Building agent..."
	@mkdir -p bin
	$(GO) build $(LDFLAGS) -o $(BINARY_AGENT) ./cmd/agent

build-server: ## Build the server binary
	@echo "Building server..."
	@mkdir -p bin
	$(GO) build $(LDFLAGS) -o $(BINARY_SERVER) ./cmd/server

test: ## Run tests
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	$(GO) fmt ./...
	gofmt -s -w .

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f $(PROTO_DIR)/*.pb.go

docker-build: ## Build Docker images
	@echo "Building Docker images..."
	docker build -f deployments/docker/agent.Dockerfile -t $(DOCKER_REGISTRY)/metrics-agent:$(VERSION) .
	docker build -f deployments/docker/server.Dockerfile -t $(DOCKER_REGISTRY)/metrics-server:$(VERSION) .

docker-push: docker-build ## Push Docker images
	docker push $(DOCKER_REGISTRY)/metrics-agent:$(VERSION)
	docker push $(DOCKER_REGISTRY)/metrics-server:$(VERSION)

run-server: build-server ## Run the server locally
	./$(BINARY_SERVER) -config configs/server.yaml

run-agent: build-agent ## Run the agent locally (requires sudo)
	sudo ./$(BINARY_AGENT) -config configs/agent.yaml

dev: ## Start development environment
	cd deployments/docker && docker-compose up -d

dev-down: ## Stop development environment
	cd deployments/docker && docker-compose down

install: build ## Install binaries to /usr/local/bin
	sudo cp $(BINARY_AGENT) /usr/local/bin/metrics-agent
	sudo cp $(BINARY_SERVER) /usr/local/bin/metrics-server

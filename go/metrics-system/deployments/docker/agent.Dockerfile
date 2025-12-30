# Agent Dockerfile
#
# Multi-stage build for the metrics agent

# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git make protobuf

WORKDIR /app

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the agent
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /agent ./cmd/agent

# Final stage
FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user (though agent needs root for some metrics)
RUN adduser -D -u 1000 metrics

WORKDIR /app

# Copy binary from builder
COPY --from=builder /agent /app/agent

# Copy default configuration
COPY configs/agent.yaml /app/configs/agent.yaml

# Set environment variable for config
ENV CONFIG_PATH=/app/configs/agent.yaml

# Run as root for /proc access
USER root

ENTRYPOINT ["/app/agent"]
CMD ["-config", "/app/configs/agent.yaml"]

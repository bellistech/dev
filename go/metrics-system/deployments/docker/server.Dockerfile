# Server Dockerfile
#
# Multi-stage build for the metrics server

# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git make protobuf

WORKDIR /app

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Final stage
FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -u 1000 metrics

WORKDIR /app

# Copy binary from builder
COPY --from=builder /server /app/server

# Copy default configuration
COPY configs/server.yaml /app/configs/server.yaml

# Set environment variable for config
ENV CONFIG_PATH=/app/configs/server.yaml

# Expose gRPC port
EXPOSE 9090

# Run as non-root user
USER metrics

ENTRYPOINT ["/app/server"]
CMD ["-config", "/app/configs/server.yaml"]

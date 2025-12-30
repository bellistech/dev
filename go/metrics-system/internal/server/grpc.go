// Package server provides the metrics collection server.
package server

import (
	"context"
	"fmt"
	"net"

	metricsv1 "github.com/bellistech/metrics-system/api/metrics/v1"
	"github.com/bellistech/metrics-system/internal/logger"
	"github.com/bellistech/metrics-system/internal/server/storage"
	"github.com/bellistech/metrics-system/pkg/metrics"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Version is the server version.
var Version = "1.0.0"

// GRPCServer implements the MetricsService gRPC server.
type GRPCServer struct {
	metricsv1.UnimplementedMetricsServiceServer
	storage storage.Storage
}

// NewGRPCServer creates a new gRPC server.
func NewGRPCServer(store storage.Storage) *GRPCServer {
	return &GRPCServer{
		storage: store,
	}
}

// Start starts the gRPC server on the specified port.
func (s *GRPCServer) Start(port int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(16*1024*1024), // 16MB max message size
	)
	metricsv1.RegisterMetricsServiceServer(grpcServer, s)

	logger.Info("Starting gRPC server on port %d", port)
	return grpcServer.Serve(listener)
}

// SendMetrics handles incoming metric batches.
func (s *GRPCServer) SendMetrics(ctx context.Context, req *metricsv1.MetricBatchRequest) (*metricsv1.MetricBatchResponse, error) {
	if req == nil || len(req.Metrics) == 0 {
		return &metricsv1.MetricBatchResponse{
			Success:         true,
			Message:         "No metrics to process",
			MetricsReceived: 0,
		}, nil
	}

	logger.Debug("Received %d metrics from %s (agent: %s)", len(req.Metrics), req.Hostname, req.AgentId)

	// Convert and store metrics
	converted := make([]metrics.Metric, 0, len(req.Metrics))
	for _, m := range req.Metrics {
		converted = append(converted, convertFromProto(m))
	}

	// Store metrics
	err := s.storage.Store(ctx, converted)
	if err != nil {
		logger.Error("Error storing metrics: %v", err)
		return &metricsv1.MetricBatchResponse{
			Success:         false,
			Message:         fmt.Sprintf("Failed to store metrics: %v", err),
			MetricsReceived: 0,
			MetricsFailed:   int32(len(req.Metrics)),
			ServerTimestamp: timestamppb.Now(),
		}, nil
	}

	logger.Debug("Stored %d metrics from %s", len(req.Metrics), req.Hostname)

	return &metricsv1.MetricBatchResponse{
		Success:         true,
		Message:         "Metrics stored successfully",
		MetricsReceived: int32(len(req.Metrics)),
		MetricsFailed:   0,
		ServerTimestamp: timestamppb.Now(),
	}, nil
}

// StreamMetrics handles streaming metrics from agents.
func (s *GRPCServer) StreamMetrics(stream metricsv1.MetricsService_StreamMetricsServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		// Process the batch
		resp, err := s.SendMetrics(stream.Context(), req)
		if err != nil {
			return err
		}

		// Send response
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}

// HealthCheck returns the server health status.
func (s *GRPCServer) HealthCheck(ctx context.Context, req *metricsv1.HealthCheckRequest) (*metricsv1.HealthCheckResponse, error) {
	// Check storage health
	healthy := true
	if err := s.storage.Ping(ctx); err != nil {
		healthy = false
		logger.Warn("Storage health check failed: %v", err)
	}

	logger.Debug("Health check from agent %s: healthy=%v", req.AgentId, healthy)

	return &metricsv1.HealthCheckResponse{
		Healthy:   healthy,
		Version:   Version,
		Timestamp: timestamppb.Now(),
	}, nil
}

// convertFromProto converts a protobuf metric to internal format.
func convertFromProto(m *metricsv1.Metric) metrics.Metric {
	return metrics.Metric{
		Name:      m.Name,
		Type:      convertProtoType(m.Type),
		Value:     m.Value,
		Timestamp: m.Timestamp.AsTime(),
		Labels:    m.Labels,
		Hostname:  m.Hostname,
		Unit:      m.Unit,
	}
}

// convertProtoType converts protobuf metric type to internal type.
func convertProtoType(t metricsv1.MetricType) metrics.MetricType {
	switch t {
	case metricsv1.MetricType_METRIC_TYPE_GAUGE:
		return metrics.MetricTypeGauge
	case metricsv1.MetricType_METRIC_TYPE_COUNTER:
		return metrics.MetricTypeCounter
	case metricsv1.MetricType_METRIC_TYPE_SUMMARY:
		return metrics.MetricTypeSummary
	case metricsv1.MetricType_METRIC_TYPE_HISTOGRAM:
		return metrics.MetricTypeHistogram
	default:
		return metrics.MetricTypeGauge
	}
}

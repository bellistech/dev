// Package agent provides the metrics collection agent client.
package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	metricsv1 "github.com/bellistech/metrics-system/api/metrics/v1"
	"github.com/bellistech/metrics-system/pkg/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Client represents a gRPC client for sending metrics.
type Client struct {
	conn     *grpc.ClientConn
	client   metricsv1.MetricsServiceClient
	hostname string
	agentID  string
}

// NewClient creates a new gRPC client.
func NewClient(address, hostname, agentID string) (*Client, error) {
	// Create connection with options
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return &Client{
		conn:     conn,
		client:   metricsv1.NewMetricsServiceClient(conn),
		hostname: hostname,
		agentID:  agentID,
	}, nil
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SendMetrics sends a batch of metrics to the server.
func (c *Client) SendMetrics(ctx context.Context, metricsList []metrics.Metric) error {
	if len(metricsList) == 0 {
		return nil
	}

	// Convert to protobuf format
	pbMetrics := make([]*metricsv1.Metric, 0, len(metricsList))
	for _, m := range metricsList {
		pbMetrics = append(pbMetrics, convertToProto(m))
	}

	req := &metricsv1.MetricBatchRequest{
		Hostname:  c.hostname,
		AgentId:   c.agentID,
		Timestamp: timestamppb.Now(),
		Metrics:   pbMetrics,
	}

	resp, err := c.client.SendMetrics(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("server rejected metrics: %s", resp.Message)
	}

	log.Printf("Sent %d metrics to server (received: %d)", len(metricsList), resp.MetricsReceived)
	return nil
}

// HealthCheck checks if the server is healthy.
func (c *Client) HealthCheck(ctx context.Context) error {
	req := &metricsv1.HealthCheckRequest{
		AgentId: c.agentID,
	}

	resp, err := c.client.HealthCheck(ctx, req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if !resp.Healthy {
		return fmt.Errorf("server unhealthy")
	}

	return nil
}

// convertToProto converts a metric to protobuf format.
func convertToProto(m metrics.Metric) *metricsv1.Metric {
	return &metricsv1.Metric{
		Name:      m.Name,
		Type:      convertMetricType(m.Type),
		Value:     m.Value,
		Timestamp: timestamppb.New(m.Timestamp),
		Labels:    m.Labels,
		Hostname:  m.Hostname,
		Unit:      m.Unit,
	}
}

// convertMetricType converts internal metric type to protobuf.
func convertMetricType(t metrics.MetricType) metricsv1.MetricType {
	switch t {
	case metrics.MetricTypeGauge:
		return metricsv1.MetricType_METRIC_TYPE_GAUGE
	case metrics.MetricTypeCounter:
		return metricsv1.MetricType_METRIC_TYPE_COUNTER
	case metrics.MetricTypeSummary:
		return metricsv1.MetricType_METRIC_TYPE_SUMMARY
	case metrics.MetricTypeHistogram:
		return metricsv1.MetricType_METRIC_TYPE_HISTOGRAM
	default:
		return metricsv1.MetricType_METRIC_TYPE_UNSPECIFIED
	}
}

// Package metrics provides shared types for the metrics collection system.
package metrics

import "time"

// MetricType represents the type of metric being collected.
type MetricType int

const (
	// MetricTypeGauge represents a point-in-time value that can go up or down.
	MetricTypeGauge MetricType = iota
	// MetricTypeCounter represents a monotonically increasing value.
	MetricTypeCounter
	// MetricTypeSummary represents a statistical summary of values.
	MetricTypeSummary
	// MetricTypeHistogram represents a distribution of values.
	MetricTypeHistogram
)

// String returns the string representation of a MetricType.
func (t MetricType) String() string {
	switch t {
	case MetricTypeGauge:
		return "gauge"
	case MetricTypeCounter:
		return "counter"
	case MetricTypeSummary:
		return "summary"
	case MetricTypeHistogram:
		return "histogram"
	default:
		return "unknown"
	}
}

// Metric represents a single metric data point.
type Metric struct {
	// Name is the metric name (e.g., "cpu_usage_percent")
	Name string
	// Type is the metric type
	Type MetricType
	// Value is the metric value
	Value float64
	// Timestamp is when the metric was collected
	Timestamp time.Time
	// Labels are additional key-value pairs for dimensions
	Labels map[string]string
	// Hostname is the source host
	Hostname string
	// Unit is the unit of measurement (e.g., "percent", "bytes")
	Unit string
}

// MetricBatch represents a collection of metrics.
type MetricBatch struct {
	// Hostname is the source hostname
	Hostname string
	// AgentID is the unique agent identifier
	AgentID string
	// Timestamp is when the batch was collected
	Timestamp time.Time
	// Metrics is the list of metrics in this batch
	Metrics []Metric
	// Labels are labels applied to all metrics in the batch
	Labels map[string]string
}

// NewMetric creates a new Metric with the current timestamp.
func NewMetric(name string, value float64, metricType MetricType, hostname string) Metric {
	return Metric{
		Name:      name,
		Type:      metricType,
		Value:     value,
		Timestamp: time.Now(),
		Labels:    make(map[string]string),
		Hostname:  hostname,
	}
}

// WithLabel adds a label to the metric and returns it for chaining.
func (m Metric) WithLabel(key, value string) Metric {
	if m.Labels == nil {
		m.Labels = make(map[string]string)
	}
	m.Labels[key] = value
	return m
}

// WithUnit sets the unit and returns the metric for chaining.
func (m Metric) WithUnit(unit string) Metric {
	m.Unit = unit
	return m
}

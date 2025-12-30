package collector

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bellistech/metrics-system/pkg/metrics"
)

// Register apache collector factory on package init
// This is ALL you need to do - no changes to main.go required!
func init() {
	RegisterFactory("apache", func(cfg CollectorConfig) Collector {
		// Get status URL from options, with default
		statusURL := "http://localhost/server-status?auto"
		if url, ok := cfg.Options["status_url"]; ok {
			statusURL = url
		}
		return NewApacheCollector(cfg.Hostname, statusURL)
	})
}

// ApacheCollector collects Apache HTTPD metrics from mod_status.
// Requires mod_status to be enabled with ExtendedStatus On.
//
// Example Apache config:
//   <Location /server-status>
//       SetHandler server-status
//       Require local
//   </Location>
//   ExtendedStatus On
type ApacheCollector struct {
	hostname  string
	statusURL string
	client    *http.Client
}

// NewApacheCollector creates a new Apache collector.
func NewApacheCollector(hostname, statusURL string) *ApacheCollector {
	return &ApacheCollector{
		hostname:  hostname,
		statusURL: statusURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Name returns the collector name.
func (c *ApacheCollector) Name() string {
	return "apache"
}

// Collect gathers Apache metrics from mod_status.
func (c *ApacheCollector) Collect(ctx context.Context) ([]metrics.Metric, error) {
	now := time.Now()

	// Fetch status page
	req, err := http.NewRequestWithContext(ctx, "GET", c.statusURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse the auto format response
	var result []metrics.Metric
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])

		var metricName string
		var value float64

		switch key {
		case "Total Accesses":
			metricName = "apache_requests_total"
			value, _ = strconv.ParseFloat(valueStr, 64)
		case "Total kBytes":
			metricName = "apache_sent_bytes_total"
			v, _ := strconv.ParseFloat(valueStr, 64)
			value = v * 1024 // Convert KB to bytes
		case "CPULoad":
			metricName = "apache_cpu_load"
			value, _ = strconv.ParseFloat(valueStr, 64)
		case "Uptime":
			metricName = "apache_uptime_seconds"
			value, _ = strconv.ParseFloat(valueStr, 64)
		case "ReqPerSec":
			metricName = "apache_requests_per_second"
			value, _ = strconv.ParseFloat(valueStr, 64)
		case "BytesPerSec":
			metricName = "apache_bytes_per_second"
			value, _ = strconv.ParseFloat(valueStr, 64)
		case "BytesPerReq":
			metricName = "apache_bytes_per_request"
			value, _ = strconv.ParseFloat(valueStr, 64)
		case "BusyWorkers":
			metricName = "apache_workers_busy"
			value, _ = strconv.ParseFloat(valueStr, 64)
		case "IdleWorkers":
			metricName = "apache_workers_idle"
			value, _ = strconv.ParseFloat(valueStr, 64)
		case "ConnsTotal":
			metricName = "apache_connections_total"
			value, _ = strconv.ParseFloat(valueStr, 64)
		case "ConnsAsyncWriting":
			metricName = "apache_connections_async_writing"
			value, _ = strconv.ParseFloat(valueStr, 64)
		case "ConnsAsyncKeepAlive":
			metricName = "apache_connections_async_keepalive"
			value, _ = strconv.ParseFloat(valueStr, 64)
		case "ConnsAsyncClosing":
			metricName = "apache_connections_async_closing"
			value, _ = strconv.ParseFloat(valueStr, 64)
		default:
			continue
		}

		if metricName != "" {
			metricType := metrics.MetricTypeGauge
			if strings.HasSuffix(metricName, "_total") {
				metricType = metrics.MetricTypeCounter
			}

			result = append(result, metrics.Metric{
				Name:      metricName,
				Type:      metricType,
				Value:     value,
				Timestamp: now,
				Hostname:  c.hostname,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("error reading response: %w", err)
	}

	return result, nil
}

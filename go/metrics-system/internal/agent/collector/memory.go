package collector

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bellistech/metrics-system/pkg/metrics"
)

// Register memory collector factory on package init
func init() {
	RegisterFactory("memory", func(cfg CollectorConfig) Collector {
		return NewMemoryCollector(cfg.Hostname)
	})
}

// MemoryCollector collects memory metrics from /proc/meminfo.
type MemoryCollector struct {
	hostname string
}

// NewMemoryCollector creates a new memory collector.
func NewMemoryCollector(hostname string) *MemoryCollector {
	return &MemoryCollector{
		hostname: hostname,
	}
}

// Name returns the collector name.
func (c *MemoryCollector) Name() string {
	return "memory"
}

// Collect gathers memory metrics.
func (c *MemoryCollector) Collect(ctx context.Context) ([]metrics.Metric, error) {
	now := time.Now()

	memInfo, err := c.readMemInfo()
	if err != nil {
		return nil, err
	}

	var result []metrics.Metric

	// Total memory
	if total, ok := memInfo["MemTotal"]; ok {
		result = append(result, metrics.Metric{
			Name:      "memory_total_bytes",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(total * 1024), // Convert KB to bytes
			Timestamp: now,
			Hostname:  c.hostname,
			Unit:      "bytes",
		})
	}

	// Free memory
	if free, ok := memInfo["MemFree"]; ok {
		result = append(result, metrics.Metric{
			Name:      "memory_free_bytes",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(free * 1024),
			Timestamp: now,
			Hostname:  c.hostname,
			Unit:      "bytes",
		})
	}

	// Available memory
	if available, ok := memInfo["MemAvailable"]; ok {
		result = append(result, metrics.Metric{
			Name:      "memory_available_bytes",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(available * 1024),
			Timestamp: now,
			Hostname:  c.hostname,
			Unit:      "bytes",
		})
	}

	// Buffers
	if buffers, ok := memInfo["Buffers"]; ok {
		result = append(result, metrics.Metric{
			Name:      "memory_buffers_bytes",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(buffers * 1024),
			Timestamp: now,
			Hostname:  c.hostname,
			Unit:      "bytes",
		})
	}

	// Cached
	if cached, ok := memInfo["Cached"]; ok {
		result = append(result, metrics.Metric{
			Name:      "memory_cached_bytes",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(cached * 1024),
			Timestamp: now,
			Hostname:  c.hostname,
			Unit:      "bytes",
		})
	}

	// Swap Total
	if swapTotal, ok := memInfo["SwapTotal"]; ok {
		result = append(result, metrics.Metric{
			Name:      "memory_swap_total_bytes",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(swapTotal * 1024),
			Timestamp: now,
			Hostname:  c.hostname,
			Unit:      "bytes",
		})
	}

	// Swap Free
	if swapFree, ok := memInfo["SwapFree"]; ok {
		result = append(result, metrics.Metric{
			Name:      "memory_swap_free_bytes",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(swapFree * 1024),
			Timestamp: now,
			Hostname:  c.hostname,
			Unit:      "bytes",
		})
	}

	// Calculate used memory and usage percentage
	if total, ok := memInfo["MemTotal"]; ok {
		if available, ok := memInfo["MemAvailable"]; ok {
			used := total - available
			result = append(result, metrics.Metric{
				Name:      "memory_used_bytes",
				Type:      metrics.MetricTypeGauge,
				Value:     float64(used * 1024),
				Timestamp: now,
				Hostname:  c.hostname,
				Unit:      "bytes",
			})

			if total > 0 {
				usedPercent := (float64(used) / float64(total)) * 100
				result = append(result, metrics.Metric{
					Name:      "memory_used_percent",
					Type:      metrics.MetricTypeGauge,
					Value:     usedPercent,
					Timestamp: now,
					Hostname:  c.hostname,
					Unit:      "percent",
				})
			}
		}
	}

	// Calculate swap usage
	if swapTotal, ok := memInfo["SwapTotal"]; ok && swapTotal > 0 {
		if swapFree, ok := memInfo["SwapFree"]; ok {
			swapUsed := swapTotal - swapFree
			result = append(result, metrics.Metric{
				Name:      "memory_swap_used_bytes",
				Type:      metrics.MetricTypeGauge,
				Value:     float64(swapUsed * 1024),
				Timestamp: now,
				Hostname:  c.hostname,
				Unit:      "bytes",
			})

			swapUsedPercent := (float64(swapUsed) / float64(swapTotal)) * 100
			result = append(result, metrics.Metric{
				Name:      "memory_swap_used_percent",
				Type:      metrics.MetricTypeGauge,
				Value:     swapUsedPercent,
				Timestamp: now,
				Hostname:  c.hostname,
				Unit:      "percent",
			})
		}
	}

	// Additional memory metrics
	if active, ok := memInfo["Active"]; ok {
		result = append(result, metrics.Metric{
			Name:      "memory_active_bytes",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(active * 1024),
			Timestamp: now,
			Hostname:  c.hostname,
			Unit:      "bytes",
		})
	}

	if inactive, ok := memInfo["Inactive"]; ok {
		result = append(result, metrics.Metric{
			Name:      "memory_inactive_bytes",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(inactive * 1024),
			Timestamp: now,
			Hostname:  c.hostname,
			Unit:      "bytes",
		})
	}

	if dirty, ok := memInfo["Dirty"]; ok {
		result = append(result, metrics.Metric{
			Name:      "memory_dirty_bytes",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(dirty * 1024),
			Timestamp: now,
			Hostname:  c.hostname,
			Unit:      "bytes",
		})
	}

	return result, nil
}

// readMemInfo reads /proc/meminfo and returns values in KB.
func (c *MemoryCollector) readMemInfo() (map[string]uint64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	memInfo := make(map[string]uint64)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		// Remove trailing colon from key
		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		memInfo[key] = value
	}

	return memInfo, scanner.Err()
}

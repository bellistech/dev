package collector

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bellistech/metrics-system/pkg/metrics"
)

// Register uptime collector factory on package init
func init() {
	RegisterFactory("uptime", func(cfg CollectorConfig) Collector {
		return NewUptimeCollector(cfg.Hostname)
	})
}

// UptimeCollector collects system uptime metrics.
type UptimeCollector struct {
	hostname string
}

// NewUptimeCollector creates a new uptime collector.
func NewUptimeCollector(hostname string) *UptimeCollector {
	return &UptimeCollector{
		hostname: hostname,
	}
}

// Name returns the collector name.
func (c *UptimeCollector) Name() string {
	return "uptime"
}

// Collect gathers uptime metrics.
func (c *UptimeCollector) Collect(ctx context.Context) ([]metrics.Metric, error) {
	now := time.Now()
	var result []metrics.Metric

	// Read uptime from /proc/uptime
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(string(data))
	if len(fields) >= 1 {
		uptime, _ := strconv.ParseFloat(fields[0], 64)
		result = append(result, metrics.Metric{
			Name:      "system_uptime_seconds",
			Type:      metrics.MetricTypeGauge,
			Value:     uptime,
			Timestamp: now,
			Hostname:  c.hostname,
			Unit:      "seconds",
		})

		// Calculate boot time
		bootTime := now.Add(-time.Duration(uptime) * time.Second)
		result = append(result, metrics.Metric{
			Name:      "system_boot_time_seconds",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(bootTime.Unix()),
			Timestamp: now,
			Hostname:  c.hostname,
			Unit:      "unix_timestamp",
		})
	}

	// Read idle time
	if len(fields) >= 2 {
		idle, _ := strconv.ParseFloat(fields[1], 64)
		result = append(result, metrics.Metric{
			Name:      "system_idle_seconds",
			Type:      metrics.MetricTypeGauge,
			Value:     idle,
			Timestamp: now,
			Hostname:  c.hostname,
			Unit:      "seconds",
		})
	}

	// Count processes
	procCount, err := c.countProcesses()
	if err == nil {
		result = append(result, metrics.Metric{
			Name:      "system_processes_total",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(procCount),
			Timestamp: now,
			Hostname:  c.hostname,
		})
	}

	// Count open file descriptors
	fdCount, fdMax, err := c.getFileDescriptorStats()
	if err == nil {
		result = append(result, metrics.Metric{
			Name:      "system_fd_allocated",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(fdCount),
			Timestamp: now,
			Hostname:  c.hostname,
		})
		result = append(result, metrics.Metric{
			Name:      "system_fd_max",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(fdMax),
			Timestamp: now,
			Hostname:  c.hostname,
		})
	}

	return result, nil
}

// countProcesses counts the number of running processes.
func (c *UptimeCollector) countProcesses() (int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Check if the directory name is a number (PID)
		if _, err := strconv.Atoi(entry.Name()); err == nil {
			count++
		}
	}

	return count, nil
}

// getFileDescriptorStats reads file descriptor statistics.
func (c *UptimeCollector) getFileDescriptorStats() (allocated, max uint64, err error) {
	data, err := os.ReadFile("/proc/sys/fs/file-nr")
	if err != nil {
		return 0, 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) >= 3 {
		allocated, _ = strconv.ParseUint(fields[0], 10, 64)
		max, _ = strconv.ParseUint(fields[2], 10, 64)
	}

	return allocated, max, nil
}

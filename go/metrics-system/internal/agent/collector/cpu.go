package collector

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bellistech/metrics-system/pkg/metrics"
)

// Register CPU collector factory on package init
func init() {
	RegisterFactory("cpu", func(cfg CollectorConfig) Collector {
		return NewCPUCollector(cfg.Hostname)
	})
}

// CPUCollector collects CPU metrics from /proc/stat.
type CPUCollector struct {
	hostname string
	mu       sync.Mutex
	prevStat *cpuStat
	prevTime time.Time
}

// cpuStat holds raw CPU statistics from /proc/stat.
type cpuStat struct {
	User      uint64
	Nice      uint64
	System    uint64
	Idle      uint64
	IOWait    uint64
	IRQ       uint64
	SoftIRQ   uint64
	Steal     uint64
	Guest     uint64
	GuestNice uint64
}

// NewCPUCollector creates a new CPU collector.
func NewCPUCollector(hostname string) *CPUCollector {
	return &CPUCollector{
		hostname: hostname,
	}
}

// Name returns the collector name.
func (c *CPUCollector) Name() string {
	return "cpu"
}

// Collect gathers CPU metrics.
func (c *CPUCollector) Collect(ctx context.Context) ([]metrics.Metric, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var result []metrics.Metric

	// Read current CPU stats
	stats, err := c.readCPUStats()
	if err != nil {
		return nil, fmt.Errorf("failed to read CPU stats: %w", err)
	}

	// Calculate usage if we have previous stats
	if c.prevStat != nil {
		elapsed := now.Sub(c.prevTime).Seconds()
		if elapsed > 0 {
			usageMetrics := c.calculateUsage(stats["cpu"], c.prevStat, elapsed, now)
			result = append(result, usageMetrics...)
		}
	}

	// Store current stats for next calculation
	if total, ok := stats["cpu"]; ok {
		c.prevStat = total
		c.prevTime = now
	}

	// Read load averages
	loadMetrics, err := c.readLoadAverage(now)
	if err == nil {
		result = append(result, loadMetrics...)
	}

	// Read context switches and process stats
	contextMetrics, err := c.readContextSwitches(now)
	if err == nil {
		result = append(result, contextMetrics...)
	}

	return result, nil
}

// readCPUStats reads CPU statistics from /proc/stat.
func (c *CPUCollector) readCPUStats() (map[string]*cpuStat, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stats := make(map[string]*cpuStat)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		name := fields[0]
		stat := &cpuStat{
			User:    parseUint64(fields[1]),
			Nice:    parseUint64(fields[2]),
			System:  parseUint64(fields[3]),
			Idle:    parseUint64(fields[4]),
			IOWait:  parseUint64(fields[5]),
			IRQ:     parseUint64(fields[6]),
			SoftIRQ: parseUint64(fields[7]),
		}

		if len(fields) > 8 {
			stat.Steal = parseUint64(fields[8])
		}
		if len(fields) > 9 {
			stat.Guest = parseUint64(fields[9])
		}
		if len(fields) > 10 {
			stat.GuestNice = parseUint64(fields[10])
		}

		stats[name] = stat
	}

	return stats, scanner.Err()
}

// calculateUsage calculates CPU usage percentages.
func (c *CPUCollector) calculateUsage(curr, prev *cpuStat, elapsed float64, ts time.Time) []metrics.Metric {
	// Calculate deltas
	userDelta := float64(curr.User - prev.User)
	niceDelta := float64(curr.Nice - prev.Nice)
	systemDelta := float64(curr.System - prev.System)
	idleDelta := float64(curr.Idle - prev.Idle)
	iowaitDelta := float64(curr.IOWait - prev.IOWait)
	irqDelta := float64(curr.IRQ - prev.IRQ)
	softirqDelta := float64(curr.SoftIRQ - prev.SoftIRQ)
	stealDelta := float64(curr.Steal - prev.Steal)

	total := userDelta + niceDelta + systemDelta + idleDelta + iowaitDelta + irqDelta + softirqDelta + stealDelta

	if total == 0 {
		return nil
	}

	return []metrics.Metric{
		{
			Name:      "cpu_usage_user_percent",
			Type:      metrics.MetricTypeGauge,
			Value:     (userDelta / total) * 100,
			Timestamp: ts,
			Hostname:  c.hostname,
			Unit:      "percent",
		},
		{
			Name:      "cpu_usage_system_percent",
			Type:      metrics.MetricTypeGauge,
			Value:     (systemDelta / total) * 100,
			Timestamp: ts,
			Hostname:  c.hostname,
			Unit:      "percent",
		},
		{
			Name:      "cpu_usage_idle_percent",
			Type:      metrics.MetricTypeGauge,
			Value:     (idleDelta / total) * 100,
			Timestamp: ts,
			Hostname:  c.hostname,
			Unit:      "percent",
		},
		{
			Name:      "cpu_usage_iowait_percent",
			Type:      metrics.MetricTypeGauge,
			Value:     (iowaitDelta / total) * 100,
			Timestamp: ts,
			Hostname:  c.hostname,
			Unit:      "percent",
		},
		{
			Name:      "cpu_usage_total_percent",
			Type:      metrics.MetricTypeGauge,
			Value:     ((total - idleDelta - iowaitDelta) / total) * 100,
			Timestamp: ts,
			Hostname:  c.hostname,
			Unit:      "percent",
		},
	}
}

// readLoadAverage reads load averages from /proc/loadavg.
func (c *CPUCollector) readLoadAverage(ts time.Time) ([]metrics.Metric, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return nil, fmt.Errorf("invalid loadavg format")
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)

	return []metrics.Metric{
		{
			Name:      "cpu_load_1m",
			Type:      metrics.MetricTypeGauge,
			Value:     load1,
			Timestamp: ts,
			Hostname:  c.hostname,
		},
		{
			Name:      "cpu_load_5m",
			Type:      metrics.MetricTypeGauge,
			Value:     load5,
			Timestamp: ts,
			Hostname:  c.hostname,
		},
		{
			Name:      "cpu_load_15m",
			Type:      metrics.MetricTypeGauge,
			Value:     load15,
			Timestamp: ts,
			Hostname:  c.hostname,
		},
	}, nil
}

// readContextSwitches reads context switches from /proc/stat.
func (c *CPUCollector) readContextSwitches(ts time.Time) ([]metrics.Metric, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []metrics.Metric
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "ctxt ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseFloat(fields[1], 64)
				result = append(result, metrics.Metric{
					Name:      "cpu_context_switches_total",
					Type:      metrics.MetricTypeCounter,
					Value:     val,
					Timestamp: ts,
					Hostname:  c.hostname,
				})
			}
		}

		if strings.HasPrefix(line, "processes ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseFloat(fields[1], 64)
				result = append(result, metrics.Metric{
					Name:      "cpu_processes_created_total",
					Type:      metrics.MetricTypeCounter,
					Value:     val,
					Timestamp: ts,
					Hostname:  c.hostname,
				})
			}
		}

		if strings.HasPrefix(line, "procs_running ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseFloat(fields[1], 64)
				result = append(result, metrics.Metric{
					Name:      "cpu_procs_running",
					Type:      metrics.MetricTypeGauge,
					Value:     val,
					Timestamp: ts,
					Hostname:  c.hostname,
				})
			}
		}

		if strings.HasPrefix(line, "procs_blocked ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseFloat(fields[1], 64)
				result = append(result, metrics.Metric{
					Name:      "cpu_procs_blocked",
					Type:      metrics.MetricTypeGauge,
					Value:     val,
					Timestamp: ts,
					Hostname:  c.hostname,
				})
			}
		}
	}

	return result, scanner.Err()
}

// parseUint64 safely parses a string to uint64.
func parseUint64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

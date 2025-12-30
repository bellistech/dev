package collector

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bellistech/metrics-system/pkg/metrics"
)

// Register disk collector factory on package init
func init() {
	RegisterFactory("disk", func(cfg CollectorConfig) Collector {
		mountPoints := cfg.MountPoints
		if len(mountPoints) == 0 {
			// Auto-detect mount points
			mountPoints, _ = GetMountPoints()
		}
		if len(mountPoints) == 0 {
			mountPoints = []string{"/"}
		}
		return NewDiskCollector(cfg.Hostname, mountPoints)
	})
}

// DiskCollector collects disk metrics from /proc and syscalls.
type DiskCollector struct {
	hostname    string
	mountPoints []string
	mu          sync.Mutex
	lastStats   map[string]*diskIOStat
	lastTime    time.Time
}

// diskIOStat holds raw disk I/O statistics from /proc/diskstats.
type diskIOStat struct {
	ReadsCompleted  uint64
	ReadsMerged     uint64
	SectorsRead     uint64
	TimeReading     uint64
	WritesCompleted uint64
	WritesMerged    uint64
	SectorsWritten  uint64
	TimeWriting     uint64
	IOInProgress    uint64
	TimeIO          uint64
	WeightedTimeIO  uint64
}

// NewDiskCollector creates a new disk collector.
func NewDiskCollector(hostname string, mountPoints []string) *DiskCollector {
	if len(mountPoints) == 0 {
		mountPoints = []string{"/"}
	}
	return &DiskCollector{
		hostname:    hostname,
		mountPoints: mountPoints,
		lastStats:   make(map[string]*diskIOStat),
	}
}

// Name returns the collector name.
func (c *DiskCollector) Name() string {
	return "disk"
}

// Collect gathers disk metrics.
func (c *DiskCollector) Collect(ctx context.Context) ([]metrics.Metric, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var result []metrics.Metric

	// Collect filesystem usage
	for _, mountPoint := range c.mountPoints {
		fsMetrics, err := c.collectFilesystemUsage(mountPoint, now)
		if err != nil {
			continue // Skip this mount point on error
		}
		result = append(result, fsMetrics...)
	}

	// Collect I/O stats
	ioStats, err := c.readDiskStats()
	if err == nil {
		for device, stat := range ioStats {
			// Skip loop devices, ram disks, etc.
			if strings.HasPrefix(device, "loop") ||
				strings.HasPrefix(device, "ram") ||
				strings.HasPrefix(device, "dm-") {
				continue
			}

			// Only process if we have previous stats
			if prevStat, ok := c.lastStats[device]; ok {
				elapsed := now.Sub(c.lastTime).Seconds()
				if elapsed > 0 {
					ioMetrics := c.calculateIOMetrics(device, stat, prevStat, elapsed, now)
					result = append(result, ioMetrics...)
				}
			}
		}

		// Update last stats
		c.lastStats = ioStats
		c.lastTime = now
	}

	return result, nil
}

// collectFilesystemUsage collects filesystem usage for a mount point.
func (c *DiskCollector) collectFilesystemUsage(mountPoint string, ts time.Time) ([]metrics.Metric, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(mountPoint, &stat); err != nil {
		return nil, err
	}

	// Calculate sizes in bytes
	blockSize := uint64(stat.Bsize)
	totalBytes := stat.Blocks * blockSize
	freeBytes := stat.Bfree * blockSize
	availBytes := stat.Bavail * blockSize
	usedBytes := totalBytes - freeBytes

	// Calculate percentages
	usedPercent := 0.0
	if totalBytes > 0 {
		usedPercent = (float64(usedBytes) / float64(totalBytes)) * 100
	}

	// Inode statistics
	totalInodes := stat.Files
	freeInodes := stat.Ffree
	usedInodes := totalInodes - freeInodes
	inodesUsedPercent := 0.0
	if totalInodes > 0 {
		inodesUsedPercent = (float64(usedInodes) / float64(totalInodes)) * 100
	}

	labels := map[string]string{"mountpoint": mountPoint}

	return []metrics.Metric{
		{
			Name:      "disk_total_bytes",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(totalBytes),
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
			Unit:      "bytes",
		},
		{
			Name:      "disk_free_bytes",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(freeBytes),
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
			Unit:      "bytes",
		},
		{
			Name:      "disk_available_bytes",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(availBytes),
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
			Unit:      "bytes",
		},
		{
			Name:      "disk_used_bytes",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(usedBytes),
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
			Unit:      "bytes",
		},
		{
			Name:      "disk_used_percent",
			Type:      metrics.MetricTypeGauge,
			Value:     usedPercent,
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
			Unit:      "percent",
		},
		{
			Name:      "disk_inodes_total",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(totalInodes),
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
		},
		{
			Name:      "disk_inodes_free",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(freeInodes),
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
		},
		{
			Name:      "disk_inodes_used_percent",
			Type:      metrics.MetricTypeGauge,
			Value:     inodesUsedPercent,
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
			Unit:      "percent",
		},
	}, nil
}

// readDiskStats reads disk I/O statistics from /proc/diskstats.
func (c *DiskCollector) readDiskStats() (map[string]*diskIOStat, error) {
	file, err := os.Open("/proc/diskstats")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stats := make(map[string]*diskIOStat)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 14 {
			continue
		}

		device := fields[2]
		stat := &diskIOStat{
			ReadsCompleted:  parseUint64(fields[3]),
			ReadsMerged:     parseUint64(fields[4]),
			SectorsRead:     parseUint64(fields[5]),
			TimeReading:     parseUint64(fields[6]),
			WritesCompleted: parseUint64(fields[7]),
			WritesMerged:    parseUint64(fields[8]),
			SectorsWritten:  parseUint64(fields[9]),
			TimeWriting:     parseUint64(fields[10]),
			IOInProgress:    parseUint64(fields[11]),
			TimeIO:          parseUint64(fields[12]),
			WeightedTimeIO:  parseUint64(fields[13]),
		}
		stats[device] = stat
	}

	return stats, scanner.Err()
}

// calculateIOMetrics calculates I/O metrics from stats delta.
func (c *DiskCollector) calculateIOMetrics(device string, curr, prev *diskIOStat, elapsed float64, ts time.Time) []metrics.Metric {
	labels := map[string]string{"device": device}
	sectorSize := float64(512) // Standard sector size

	// Calculate deltas
	readsDelta := float64(curr.ReadsCompleted - prev.ReadsCompleted)
	writesDelta := float64(curr.WritesCompleted - prev.WritesCompleted)
	sectorsReadDelta := float64(curr.SectorsRead - prev.SectorsRead)
	sectorsWrittenDelta := float64(curr.SectorsWritten - prev.SectorsWritten)
	timeReadingDelta := float64(curr.TimeReading - prev.TimeReading)
	timeWritingDelta := float64(curr.TimeWriting - prev.TimeWriting)
	timeIODelta := float64(curr.TimeIO - prev.TimeIO)

	// Calculate rates
	readsPerSec := readsDelta / elapsed
	writesPerSec := writesDelta / elapsed
	readBytesPerSec := (sectorsReadDelta * sectorSize) / elapsed
	writeBytesPerSec := (sectorsWrittenDelta * sectorSize) / elapsed

	// Calculate average latencies (in milliseconds)
	avgReadLatency := 0.0
	if readsDelta > 0 {
		avgReadLatency = timeReadingDelta / readsDelta
	}
	avgWriteLatency := 0.0
	if writesDelta > 0 {
		avgWriteLatency = timeWritingDelta / writesDelta
	}

	// IO utilization percentage
	ioUtil := (timeIODelta / (elapsed * 1000)) * 100
	if ioUtil > 100 {
		ioUtil = 100
	}

	return []metrics.Metric{
		{
			Name:      "disk_reads_per_sec",
			Type:      metrics.MetricTypeGauge,
			Value:     readsPerSec,
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
		},
		{
			Name:      "disk_writes_per_sec",
			Type:      metrics.MetricTypeGauge,
			Value:     writesPerSec,
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
		},
		{
			Name:      "disk_read_bytes_per_sec",
			Type:      metrics.MetricTypeGauge,
			Value:     readBytesPerSec,
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
			Unit:      "bytes/sec",
		},
		{
			Name:      "disk_write_bytes_per_sec",
			Type:      metrics.MetricTypeGauge,
			Value:     writeBytesPerSec,
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
			Unit:      "bytes/sec",
		},
		{
			Name:      "disk_read_latency_ms",
			Type:      metrics.MetricTypeGauge,
			Value:     avgReadLatency,
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
			Unit:      "milliseconds",
		},
		{
			Name:      "disk_write_latency_ms",
			Type:      metrics.MetricTypeGauge,
			Value:     avgWriteLatency,
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
			Unit:      "milliseconds",
		},
		{
			Name:      "disk_io_in_progress",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(curr.IOInProgress),
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
		},
		{
			Name:      "disk_io_util_percent",
			Type:      metrics.MetricTypeGauge,
			Value:     ioUtil,
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    labels,
			Unit:      "percent",
		},
	}
}

// GetMountPoints returns common mount points to monitor.
func GetMountPoints() ([]string, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/mounts: %w", err)
	}
	defer file.Close()

	var mountPoints []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}

		mountPoint := fields[1]
		fsType := fields[2]

		// Only include real filesystems
		if fsType == "ext4" || fsType == "ext3" || fsType == "xfs" ||
			fsType == "btrfs" || fsType == "zfs" || fsType == "vfat" {
			mountPoints = append(mountPoints, mountPoint)
		}
	}

	if len(mountPoints) == 0 {
		mountPoints = []string{"/"}
	}

	return mountPoints, scanner.Err()
}

package collector

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bellistech/metrics-system/pkg/metrics"
)

// Register network collector factory on package init
func init() {
	RegisterFactory("network", func(cfg CollectorConfig) Collector {
		return NewNetworkCollector(cfg.Hostname, cfg.Interfaces)
	})
}

// NetworkCollector collects network metrics from /proc/net.
type NetworkCollector struct {
	hostname   string
	mu         sync.Mutex
	lastStats  map[string]*netDevStat
	lastTime   time.Time
	interfaces []string
}

// netDevStat holds network device statistics.
type netDevStat struct {
	RxBytes      uint64
	RxPackets    uint64
	RxErrors     uint64
	RxDropped    uint64
	TxBytes      uint64
	TxPackets    uint64
	TxErrors     uint64
	TxDropped    uint64
}

// NewNetworkCollector creates a new network collector.
func NewNetworkCollector(hostname string, interfaces []string) *NetworkCollector {
	return &NetworkCollector{
		hostname:   hostname,
		interfaces: interfaces,
		lastStats:  make(map[string]*netDevStat),
	}
}

// Name returns the collector name.
func (c *NetworkCollector) Name() string {
	return "network"
}

// Collect gathers network metrics.
func (c *NetworkCollector) Collect(ctx context.Context) ([]metrics.Metric, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var result []metrics.Metric

	// Read network device stats
	stats, err := c.readNetDevStats()
	if err != nil {
		return nil, err
	}

	for iface, stat := range stats {
		// Skip loopback unless specifically requested
		if iface == "lo" && !c.shouldInclude("lo") {
			continue
		}

		// Skip if interfaces are specified and this isn't in the list
		if len(c.interfaces) > 0 && !c.shouldInclude(iface) {
			continue
		}

		labels := map[string]string{"interface": iface}

		// Add counter metrics (total values)
		result = append(result,
			metrics.Metric{
				Name:      "network_rx_bytes_total",
				Type:      metrics.MetricTypeCounter,
				Value:     float64(stat.RxBytes),
				Timestamp: now,
				Hostname:  c.hostname,
				Labels:    labels,
				Unit:      "bytes",
			},
			metrics.Metric{
				Name:      "network_tx_bytes_total",
				Type:      metrics.MetricTypeCounter,
				Value:     float64(stat.TxBytes),
				Timestamp: now,
				Hostname:  c.hostname,
				Labels:    labels,
				Unit:      "bytes",
			},
			metrics.Metric{
				Name:      "network_rx_packets_total",
				Type:      metrics.MetricTypeCounter,
				Value:     float64(stat.RxPackets),
				Timestamp: now,
				Hostname:  c.hostname,
				Labels:    labels,
			},
			metrics.Metric{
				Name:      "network_tx_packets_total",
				Type:      metrics.MetricTypeCounter,
				Value:     float64(stat.TxPackets),
				Timestamp: now,
				Hostname:  c.hostname,
				Labels:    labels,
			},
			metrics.Metric{
				Name:      "network_rx_errors_total",
				Type:      metrics.MetricTypeCounter,
				Value:     float64(stat.RxErrors),
				Timestamp: now,
				Hostname:  c.hostname,
				Labels:    labels,
			},
			metrics.Metric{
				Name:      "network_tx_errors_total",
				Type:      metrics.MetricTypeCounter,
				Value:     float64(stat.TxErrors),
				Timestamp: now,
				Hostname:  c.hostname,
				Labels:    labels,
			},
			metrics.Metric{
				Name:      "network_rx_dropped_total",
				Type:      metrics.MetricTypeCounter,
				Value:     float64(stat.RxDropped),
				Timestamp: now,
				Hostname:  c.hostname,
				Labels:    labels,
			},
			metrics.Metric{
				Name:      "network_tx_dropped_total",
				Type:      metrics.MetricTypeCounter,
				Value:     float64(stat.TxDropped),
				Timestamp: now,
				Hostname:  c.hostname,
				Labels:    labels,
			},
		)

		// Calculate rates if we have previous stats
		if prevStat, ok := c.lastStats[iface]; ok {
			elapsed := now.Sub(c.lastTime).Seconds()
			if elapsed > 0 {
				rxBytesPerSec := float64(stat.RxBytes-prevStat.RxBytes) / elapsed
				txBytesPerSec := float64(stat.TxBytes-prevStat.TxBytes) / elapsed
				rxPacketsPerSec := float64(stat.RxPackets-prevStat.RxPackets) / elapsed
				txPacketsPerSec := float64(stat.TxPackets-prevStat.TxPackets) / elapsed

				result = append(result,
					metrics.Metric{
						Name:      "network_rx_bytes_per_sec",
						Type:      metrics.MetricTypeGauge,
						Value:     rxBytesPerSec,
						Timestamp: now,
						Hostname:  c.hostname,
						Labels:    labels,
						Unit:      "bytes/sec",
					},
					metrics.Metric{
						Name:      "network_tx_bytes_per_sec",
						Type:      metrics.MetricTypeGauge,
						Value:     txBytesPerSec,
						Timestamp: now,
						Hostname:  c.hostname,
						Labels:    labels,
						Unit:      "bytes/sec",
					},
					metrics.Metric{
						Name:      "network_rx_packets_per_sec",
						Type:      metrics.MetricTypeGauge,
						Value:     rxPacketsPerSec,
						Timestamp: now,
						Hostname:  c.hostname,
						Labels:    labels,
					},
					metrics.Metric{
						Name:      "network_tx_packets_per_sec",
						Type:      metrics.MetricTypeGauge,
						Value:     txPacketsPerSec,
						Timestamp: now,
						Hostname:  c.hostname,
						Labels:    labels,
					},
				)
			}
		}
	}

	// Update last stats
	c.lastStats = stats
	c.lastTime = now

	// Read TCP connection states
	tcpMetrics, err := c.readTCPStats(now)
	if err == nil {
		result = append(result, tcpMetrics...)
	}

	// Read socket statistics
	sockMetrics, err := c.readSockStats(now)
	if err == nil {
		result = append(result, sockMetrics...)
	}

	return result, nil
}

// shouldInclude checks if an interface should be included.
func (c *NetworkCollector) shouldInclude(iface string) bool {
	if len(c.interfaces) == 0 {
		return true
	}
	for _, i := range c.interfaces {
		if i == iface {
			return true
		}
	}
	return false
}

// readNetDevStats reads network device statistics from /proc/net/dev.
func (c *NetworkCollector) readNetDevStats() (map[string]*netDevStat, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stats := make(map[string]*netDevStat)
	scanner := bufio.NewScanner(file)

	// Skip header lines
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			continue
		}

		line := scanner.Text()
		// Split on colon first to get interface name
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		iface := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}

		stats[iface] = &netDevStat{
			RxBytes:   parseUint64(fields[0]),
			RxPackets: parseUint64(fields[1]),
			RxErrors:  parseUint64(fields[2]),
			RxDropped: parseUint64(fields[3]),
			TxBytes:   parseUint64(fields[8]),
			TxPackets: parseUint64(fields[9]),
			TxErrors:  parseUint64(fields[10]),
			TxDropped: parseUint64(fields[11]),
		}
	}

	return stats, scanner.Err()
}

// readTCPStats reads TCP connection state counts.
func (c *NetworkCollector) readTCPStats(ts time.Time) ([]metrics.Metric, error) {
	states := make(map[string]int)

	// Read both IPv4 and IPv6 TCP connections
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		file, err := os.Open(path)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			if lineNum == 1 {
				continue // Skip header
			}

			fields := strings.Fields(scanner.Text())
			if len(fields) < 4 {
				continue
			}

			// State is in field 3 (hex)
			stateHex := fields[3]
			state := tcpStateFromHex(stateHex)
			states[state]++
		}
		file.Close()
	}

	var result []metrics.Metric
	for state, count := range states {
		result = append(result, metrics.Metric{
			Name:      "network_tcp_connections",
			Type:      metrics.MetricTypeGauge,
			Value:     float64(count),
			Timestamp: ts,
			Hostname:  c.hostname,
			Labels:    map[string]string{"state": state},
		})
	}

	return result, nil
}

// readSockStats reads socket statistics from /proc/net/sockstat.
func (c *NetworkCollector) readSockStats(ts time.Time) ([]metrics.Metric, error) {
	file, err := os.Open("/proc/net/sockstat")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []metrics.Metric
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		proto := strings.TrimSuffix(fields[0], ":")

		// Parse key-value pairs
		for i := 1; i < len(fields)-1; i += 2 {
			key := fields[i]
			val, _ := strconv.ParseFloat(fields[i+1], 64)

			result = append(result, metrics.Metric{
				Name:      "network_sockstat_" + strings.ToLower(proto) + "_" + strings.ToLower(key),
				Type:      metrics.MetricTypeGauge,
				Value:     val,
				Timestamp: ts,
				Hostname:  c.hostname,
			})
		}
	}

	return result, scanner.Err()
}

// tcpStateFromHex converts TCP state hex to string.
func tcpStateFromHex(hex string) string {
	states := map[string]string{
		"01": "ESTABLISHED",
		"02": "SYN_SENT",
		"03": "SYN_RECV",
		"04": "FIN_WAIT1",
		"05": "FIN_WAIT2",
		"06": "TIME_WAIT",
		"07": "CLOSE",
		"08": "CLOSE_WAIT",
		"09": "LAST_ACK",
		"0A": "LISTEN",
		"0B": "CLOSING",
	}
	if state, ok := states[strings.ToUpper(hex)]; ok {
		return state
	}
	return "UNKNOWN"
}

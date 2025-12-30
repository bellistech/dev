# Go Crash Course: From Hello World to Production Metrics System

A comprehensive guide for script creators and amateur programmers transitioning to Go, culminating in building a production-ready metrics collection system.

## Table of Contents

1. [Introduction](#introduction)
2. [Part 1: Go Fundamentals](#part-1-go-fundamentals)
3. [Part 2: Data Structures & Error Handling](#part-2-data-structures--error-handling)
4. [Part 3: Concurrency](#part-3-concurrency)
5. [Part 4: Reading Linux Metrics](#part-4-reading-linux-metrics)
6. [Part 5: gRPC & Protocol Buffers](#part-5-grpc--protocol-buffers)
7. [Part 6: Building the Metrics System](#part-6-building-the-metrics-system)
8. [Part 7: Production Patterns](#part-7-production-patterns)
9. [Part 8: Adding New Collectors](#part-8-adding-new-collectors)
10. [Summary & Next Steps](#summary--next-steps)

---

## Introduction

If you have experience with Ruby, Python, Bash, or other scripting languages, you'll find Go refreshingly straightforward. This guide takes you from "Hello World" to building a production metrics system like Prometheus.

**What Makes Go Different:**

- **Compiled** - Produces single static binaries
- **Strongly typed** - Catches errors at compile time
- **Built-in concurrency** - Goroutines and channels
- **Simple syntax** - Easy to read and maintain
- **Fast** - Near C performance with garbage collection

---

## Part 1: Go Fundamentals

### 1.1 Hello World

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
```

Run it:
```bash
go run main.go
```

Key concepts:
- Every Go file starts with `package`
- `main` package + `main` function = entry point
- No semicolons needed
- Imports use double quotes

### 1.2 Go Modules

```bash
mkdir myproject
cd myproject
go mod init github.com/bellistech/myproject
```

This creates `go.mod` - think of it like `Gemfile` or `requirements.txt`.

### 1.3 Variables & Types

```go
package main

import "fmt"

func main() {
    // Explicit type declaration
    var hostname string = "server01"
    var port int = 9090
    var enabled bool = true
    
    // Type inference (short declaration)
    cpuUsage := 45.2       // float64
    metricName := "cpu"    // string
    
    // Constants
    const maxRetries = 3
    
    fmt.Printf("Server: %s:%d (enabled: %t)\n", hostname, port, enabled)
    fmt.Printf("CPU: %.1f%%\n", cpuUsage)
}
```

### 1.4 Functions

```go
package main

import "fmt"

// Function with parameters and return value
func calculateUsage(used, total float64) float64 {
    if total == 0 {
        return 0
    }
    return (used / total) * 100
}

// Multiple return values
func divmod(a, b int) (int, int) {
    return a / b, a % b
}

// Named return values
func parseMetric(line string) (name string, value float64, ok bool) {
    // Return values are pre-declared
    name = "cpu"
    value = 45.2
    ok = true
    return  // Naked return
}

func main() {
    usage := calculateUsage(3.5, 8.0)
    fmt.Printf("Usage: %.1f%%\n", usage)
    
    q, r := divmod(17, 5)
    fmt.Printf("17 / 5 = %d remainder %d\n", q, r)
}
```

### 1.5 Control Flow

```go
package main

import "fmt"

func main() {
    // If statements
    cpuUsage := 85.5
    if cpuUsage > 90 {
        fmt.Println("CRITICAL: CPU usage above 90%")
    } else if cpuUsage > 70 {
        fmt.Println("WARNING: CPU usage above 70%")
    } else {
        fmt.Println("OK: CPU usage normal")
    }
    
    // For loop (the only loop in Go)
    for i := 0; i < 3; i++ {
        fmt.Printf("Iteration %d\n", i)
    }
    
    // While-style loop
    count := 0
    for count < 3 {
        fmt.Printf("Count: %d\n", count)
        count++
    }
    
    // Infinite loop
    // for {
    //     // Do forever
    // }
    
    // Switch
    status := "warning"
    switch status {
    case "ok":
        fmt.Println("All good")
    case "warning":
        fmt.Println("Check the logs")
    case "critical":
        fmt.Println("Page someone!")
    default:
        fmt.Println("Unknown status")
    }
}
```

---

## Part 2: Data Structures & Error Handling

### 2.1 Arrays & Slices

```go
package main

import "fmt"

func main() {
    // Array (fixed size)
    var hosts [3]string
    hosts[0] = "server01"
    hosts[1] = "server02"
    hosts[2] = "server03"
    
    // Slice (dynamic size) - much more common
    metrics := []string{"cpu", "memory", "disk"}
    metrics = append(metrics, "network")
    
    // Iterate with range
    for i, metric := range metrics {
        fmt.Printf("%d: %s\n", i, metric)
    }
    
    // Slice operations
    first := metrics[0]        // First element
    last := metrics[len(metrics)-1]  // Last element
    subset := metrics[1:3]     // Elements 1 and 2
    
    fmt.Printf("First: %s, Last: %s\n", first, last)
    fmt.Printf("Subset: %v\n", subset)
}
```

### 2.2 Maps

```go
package main

import "fmt"

func main() {
    // Create a map
    metrics := make(map[string]float64)
    metrics["cpu_usage"] = 45.2
    metrics["memory_usage"] = 78.5
    metrics["disk_usage"] = 34.1
    
    // Shorthand initialization
    labels := map[string]string{
        "host":     "server01",
        "region":   "us-west-2",
        "env":      "production",
    }
    
    // Access values
    cpu := metrics["cpu_usage"]
    fmt.Printf("CPU: %.1f%%\n", cpu)
    
    // Check if key exists
    if val, exists := metrics["network"]; exists {
        fmt.Printf("Network: %.1f\n", val)
    } else {
        fmt.Println("Network metric not found")
    }
    
    // Iterate over map
    for key, value := range labels {
        fmt.Printf("%s = %s\n", key, value)
    }
    
    // Delete a key
    delete(metrics, "cpu_usage")
}
```

### 2.3 Structs

```go
package main

import (
    "fmt"
    "time"
)

// Define a struct
type Metric struct {
    Name      string
    Value     float64
    Timestamp time.Time
    Labels    map[string]string
}

// Method on struct
func (m *Metric) String() string {
    return fmt.Sprintf("%s: %.2f", m.Name, m.Value)
}

func main() {
    // Create struct instance
    metric := Metric{
        Name:      "cpu_usage",
        Value:     45.2,
        Timestamp: time.Now(),
        Labels: map[string]string{
            "host": "server01",
        },
    }
    
    fmt.Println(metric.String())
    fmt.Printf("Collected at: %v\n", metric.Timestamp)
}
```

### 2.4 Error Handling

```go
package main

import (
    "errors"
    "fmt"
    "os"
)

// Custom error
var ErrMetricNotFound = errors.New("metric not found")

func readMetric(name string) (float64, error) {
    validMetrics := map[string]float64{
        "cpu":    45.2,
        "memory": 78.5,
    }
    
    if val, exists := validMetrics[name]; exists {
        return val, nil
    }
    return 0, ErrMetricNotFound
}

func main() {
    // Standard Go error handling pattern
    value, err := readMetric("cpu")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    fmt.Printf("CPU: %.1f%%\n", value)
    
    // Error checking
    _, err = readMetric("invalid")
    if errors.Is(err, ErrMetricNotFound) {
        fmt.Println("The metric doesn't exist")
    }
}
```

**Key insight:** Go doesn't have try/catch/except. Errors are values you check explicitly. This is Go's philosophy.

---

## Part 3: Concurrency

### 3.1 Goroutines

```go
package main

import (
    "fmt"
    "time"
)

func collectCPU() {
    time.Sleep(100 * time.Millisecond)
    fmt.Println("CPU: 45.2%")
}

func collectMemory() {
    time.Sleep(150 * time.Millisecond)
    fmt.Println("Memory: 78.5%")
}

func collectDisk() {
    time.Sleep(200 * time.Millisecond)
    fmt.Println("Disk: 34.1%")
}

func main() {
    start := time.Now()
    
    // Launch concurrent goroutines
    go collectCPU()
    go collectMemory()
    go collectDisk()
    
    // Wait for completion (crude method)
    time.Sleep(250 * time.Millisecond)
    
    fmt.Printf("Completed in %v\n", time.Since(start))
}
```

### 3.2 Channels

```go
package main

import (
    "fmt"
    "time"
)

func collectCPU(results chan<- string) {
    time.Sleep(100 * time.Millisecond)
    results <- "CPU: 45.2%"
}

func collectMemory(results chan<- string) {
    time.Sleep(150 * time.Millisecond)
    results <- "Memory: 78.5%"
}

func collectDisk(results chan<- string) {
    time.Sleep(200 * time.Millisecond)
    results <- "Disk: 34.1%"
}

func main() {
    results := make(chan string, 3)  // Buffered channel
    
    go collectCPU(results)
    go collectMemory(results)
    go collectDisk(results)
    
    // Collect results
    for i := 0; i < 3; i++ {
        metric := <-results
        fmt.Println(metric)
    }
    
    close(results)
}
```

### 3.3 Context for Cancellation

```go
package main

import (
    "context"
    "fmt"
    "time"
)

func longRunningTask(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            fmt.Println("Task cancelled!")
            return
        default:
            fmt.Println("Working...")
            time.Sleep(500 * time.Millisecond)
        }
    }
}

func main() {
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    go longRunningTask(ctx)
    
    // Wait for timeout
    <-ctx.Done()
    fmt.Println("Main: context done")
}
```

---

## Part 4: Reading Linux Metrics

### 4.1 Reading /proc Files

```go
package main

import (
    "bufio"
    "fmt"
    "os"
    "strconv"
    "strings"
)

type CPUStat struct {
    User   uint64
    Nice   uint64
    System uint64
    Idle   uint64
    IOWait uint64
}

func readCPUStat() (*CPUStat, error) {
    file, err := os.Open("/proc/stat")
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "cpu ") {
            fields := strings.Fields(line)
            if len(fields) < 6 {
                continue
            }
            
            return &CPUStat{
                User:   parseUint(fields[1]),
                Nice:   parseUint(fields[2]),
                System: parseUint(fields[3]),
                Idle:   parseUint(fields[4]),
                IOWait: parseUint(fields[5]),
            }, nil
        }
    }
    
    return nil, fmt.Errorf("cpu stats not found")
}

func parseUint(s string) uint64 {
    v, _ := strconv.ParseUint(s, 10, 64)
    return v
}

func main() {
    stat, err := readCPUStat()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    total := stat.User + stat.Nice + stat.System + stat.Idle + stat.IOWait
    used := stat.User + stat.Nice + stat.System
    usage := float64(used) / float64(total) * 100
    
    fmt.Printf("CPU Usage: %.1f%%\n", usage)
}
```

### 4.2 Memory from /proc/meminfo

```go
func readMemInfo() (map[string]uint64, error) {
    file, err := os.Open("/proc/meminfo")
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    memInfo := make(map[string]uint64)
    scanner := bufio.NewScanner(file)
    
    for scanner.Scan() {
        fields := strings.Fields(scanner.Text())
        if len(fields) < 2 {
            continue
        }
        
        key := strings.TrimSuffix(fields[0], ":")
        value, _ := strconv.ParseUint(fields[1], 10, 64)
        memInfo[key] = value
    }
    
    return memInfo, nil
}
```

---

## Part 5: gRPC & Protocol Buffers

### 5.1 Protocol Buffer Definition

```protobuf
syntax = "proto3";

package metrics.v1;

option go_package = "github.com/bellistech/metrics-system/api/metrics/v1;metricsv1";

import "google/protobuf/timestamp.proto";

service MetricsService {
    rpc SendMetrics(MetricBatchRequest) returns (MetricBatchResponse);
}

message Metric {
    string name = 1;
    double value = 2;
    google.protobuf.Timestamp timestamp = 3;
    map<string, string> labels = 4;
}

message MetricBatchRequest {
    string hostname = 1;
    repeated Metric metrics = 2;
}

message MetricBatchResponse {
    bool success = 1;
    string message = 2;
    int32 metrics_received = 3;
}
```

### 5.2 Generate Go Code

```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       api/metrics/v1/metrics.proto
```

### 5.3 gRPC Server

```go
type MetricsServer struct {
    pb.UnimplementedMetricsServiceServer
    storage Storage
}

func (s *MetricsServer) SendMetrics(ctx context.Context, req *pb.MetricBatchRequest) (*pb.MetricBatchResponse, error) {
    // Process metrics
    for _, m := range req.Metrics {
        // Store metric
    }
    
    return &pb.MetricBatchResponse{
        Success:         true,
        Message:         "Metrics stored",
        MetricsReceived: int32(len(req.Metrics)),
    }, nil
}
```

### 5.4 gRPC Client

```go
func sendMetrics(client pb.MetricsServiceClient, metrics []Metric) error {
    req := &pb.MetricBatchRequest{
        Hostname: hostname,
        Metrics:  convertToProto(metrics),
    }
    
    resp, err := client.SendMetrics(context.Background(), req)
    if err != nil {
        return err
    }
    
    if !resp.Success {
        return fmt.Errorf("server rejected: %s", resp.Message)
    }
    
    return nil
}
```

---

## Part 6: Building the Metrics System

### 6.1 Project Structure

```
metrics-system/
├── api/metrics/v1/
│   └── metrics.proto
├── cmd/
│   ├── agent/main.go
│   └── server/main.go
├── internal/
│   ├── agent/
│   │   ├── collector/
│   │   │   ├── collector.go    # Registry & factory pattern
│   │   │   ├── cpu.go
│   │   │   ├── memory.go
│   │   │   └── disk.go
│   │   └── client.go
│   ├── server/
│   │   ├── grpc.go
│   │   └── storage/
│   │       └── postgres.go
│   ├── config/
│   │   └── config.go
│   └── logger/
│       └── logger.go           # Leveled logging
├── pkg/metrics/
│   └── types.go
├── configs/
├── deployments/
├── go.mod
└── Makefile
```

### 6.2 Self-Registering Collector Pattern

The system uses a **factory pattern with self-registration** - collectors register themselves via `init()` functions, eliminating the need to modify `main.go` when adding new collectors:

```go
// collector.go - Factory registry
type CollectorFactory func(cfg CollectorConfig) Collector

var factories = make(map[string]CollectorFactory)

func RegisterFactory(name string, factory CollectorFactory) {
    factories[name] = factory
}

// cpu.go - Self-registers on import
func init() {
    RegisterFactory("cpu", func(cfg CollectorConfig) Collector {
        return NewCPUCollector(cfg.Hostname)
    })
}

// main.go - Config-driven, never needs modification
registry.RegisterFromConfig(cfg.Collection.Collectors, collectorCfg)
```

### 6.3 Leveled Logging

The system includes a simple leveled logger supporting debug, info, warn, and error levels:

```go
// Usage
logger.Debug("Detailed info: %v", data)    // Only shown with -debug
logger.Info("Normal operation: %s", msg)   // Default level
logger.Warn("Warning: %v", err)            // Warnings
logger.Error("Error: %v", err)             // Errors
logger.Fatal("Fatal: %v", err)             // Exits after logging

// Setting level
logger.SetLevel(logger.LevelDebug)
logger.SetLevelFromString("debug")
```

### 6.4 Agent Main Loop

```go
func main() {
    // Parse flags including -v, -debug, -log-level
    // Set log level from flags or config
    
    // Config-driven collector registration
    registry := collector.NewRegistry()
    registry.RegisterFromConfig(cfg.Collection.Collectors, collectorCfg)
    
    // Collection loop
    ticker := time.NewTicker(cfg.Collection.Interval)
    for range ticker.C {
        metrics, _ := registry.CollectAll(ctx)
        client.SendMetrics(ctx, metrics)
    }
}
```

---

## Part 7: Production Patterns

### 7.1 Graceful Shutdown

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        logger.Info("Shutting down...")
        cancel()
    }()
    
    // Run until cancelled
    runServer(ctx)
}
```

### 7.2 Configuration Management

```go
type Config struct {
    Server struct {
        Address string `yaml:"address"`
    } `yaml:"server"`
    Collection struct {
        Interval time.Duration `yaml:"interval"`
    } `yaml:"collection"`
    Logging struct {
        Level string `yaml:"level"`  // debug, info, warn, error
    } `yaml:"logging"`
}

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    
    return &cfg, nil
}
```

### 7.3 Command-Line Flags

```bash
# Agent flags
./bin/agent -config configs/agent.yaml    # Config file path
./bin/agent -v                            # Verbose (info level)
./bin/agent -debug                        # Debug (most verbose)
./bin/agent -log-level=warn               # Specific level
./bin/agent -list-collectors              # Show available collectors
./bin/agent -version                      # Show version

# Server flags
./bin/server -config configs/server.yaml
./bin/server -debug
./bin/server -log-level=error
```

---

## Part 8: Adding New Collectors

Adding a new collector is simple with the self-registering pattern:

### Step 1: Create the Collector File

```go
// internal/agent/collector/redis.go
package collector

import (
    "context"
    "time"
    "github.com/bellistech/metrics-system/pkg/metrics"
)

// Self-register on package init - NO main.go changes needed!
func init() {
    RegisterFactory("redis", func(cfg CollectorConfig) Collector {
        addr := "localhost:6379"
        if a, ok := cfg.Options["address"]; ok {
            addr = a
        }
        return NewRedisCollector(cfg.Hostname, addr)
    })
}

type RedisCollector struct {
    hostname string
    address  string
}

func NewRedisCollector(hostname, address string) *RedisCollector {
    return &RedisCollector{hostname: hostname, address: address}
}

func (c *RedisCollector) Name() string {
    return "redis"
}

func (c *RedisCollector) Collect(ctx context.Context) ([]metrics.Metric, error) {
    now := time.Now()
    var result []metrics.Metric
    
    // Connect to Redis and collect metrics...
    // result = append(result, metrics.Metric{...})
    
    return result, nil
}
```

### Step 2: Enable in Config

```yaml
# configs/agent.yaml
collection:
  collectors:
    - cpu
    - memory
    - redis    # Just add the name!
```

### Step 3: Verify

```bash
./bin/agent -list-collectors
# Output: Available collectors: [cpu memory disk network uptime redis]
```

**That's it!** No switch statements, no main.go modifications.

---

## Summary & Next Steps

You now understand:

1. **Go Fundamentals** - Variables, functions, control flow
2. **Data Structures** - Arrays, slices, maps, structs
3. **Error Handling** - Explicit error checking
4. **Concurrency** - Goroutines, channels, context
5. **File I/O** - Reading /proc filesystem
6. **gRPC** - Protocol buffers and service definitions
7. **Production Patterns** - Logging, graceful shutdown, configuration
8. **Extensible Design** - Self-registering factory pattern

### What This Project Includes

- **Agent** - Collects CPU, memory, disk, network, uptime from /proc
- **Server** - gRPC service with PostgreSQL storage
- **Logger** - Leveled logging with debug/info/warn/error
- **Docker** - Ready-to-run containers
- **Grafana** - Pre-configured dashboards

### Next Steps

1. Run the system with Docker Compose
2. Study the collector implementations
3. Add a new collector (Apache, MySQL, Redis)
4. Create custom Grafana dashboards
5. Explore TimescaleDB features

### Resources

- [Effective Go](https://golang.org/doc/effective_go)
- [Go by Example](https://gobyexample.com/)
- [gRPC Documentation](https://grpc.io/docs/languages/go/)
- [TimescaleDB](https://www.timescale.com/)

---

*Built with ❤️ for SREs, DevOps Engineers, and Go Learners*

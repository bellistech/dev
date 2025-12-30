// Metrics Collection Agent
//
// This agent collects system metrics from Linux hosts and sends them
// to a central metrics server via gRPC.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bellistech/metrics-system/internal/agent"
	"github.com/bellistech/metrics-system/internal/agent/collector"
	"github.com/bellistech/metrics-system/internal/config"
	"github.com/bellistech/metrics-system/internal/logger"
)

var Version = "dev"

func main() {
	// Parse command line flags
	configPath := flag.String("config", "configs/agent.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version and exit")
	listCollectors := flag.Bool("list-collectors", false, "List available collectors and exit")
	verbose := flag.Bool("v", false, "Enable verbose (info) logging")
	debug := flag.Bool("debug", false, "Enable debug logging (most verbose)")
	logLevel := flag.String("log-level", "", "Set log level: debug, info, warn, error")
	flag.Parse()

	if *showVersion {
		logger.Info("metrics-agent version %s", Version)
		os.Exit(0)
	}

	// List available collectors (useful for config reference)
	if *listCollectors {
		logger.Info("Available collectors: %v", collector.ListFactories())
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadAgentConfig(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration: %v", err)
	}

	// Set log level (precedence: flag > config > default)
	// debug flag > verbose flag > log-level flag > config file
	if *debug {
		logger.SetLevel(logger.LevelDebug)
	} else if *verbose {
		logger.SetLevel(logger.LevelInfo)
	} else if *logLevel != "" {
		logger.SetLevelFromString(*logLevel)
	} else {
		logger.SetLevelFromString(cfg.Logging.Level)
	}

	logger.Debug("Log level set to: %s", logger.GetLevel())

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Generate agent ID if not configured
	agentID := cfg.Agent.ID
	if agentID == "" {
		agentID = hostname
	}

	logger.Info("Starting metrics agent (hostname: %s, agent_id: %s)", hostname, agentID)
	logger.Info("Server address: %s", cfg.Server.Address)
	logger.Info("Collection interval: %s", cfg.Collection.Interval)
	logger.Debug("Available collectors: %v", collector.ListFactories())
	logger.Info("Enabled collectors: %v", cfg.Collection.Collectors)

	// Create gRPC client
	logger.Debug("Connecting to server at %s...", cfg.Server.Address)
	client, err := agent.NewClient(cfg.Server.Address, hostname, agentID)
	if err != nil {
		logger.Fatal("Failed to create client: %v", err)
	}
	defer client.Close()
	logger.Debug("Connected to server")

	// Create collector registry and register collectors from config
	// No switch statement needed - collectors self-register via init()
	registry := collector.NewRegistry()

	collectorCfg := collector.CollectorConfig{
		Hostname:    hostname,
		MountPoints: nil, // Will auto-detect
		Interfaces:  nil, // Will monitor all
	}

	logger.Debug("Registering collectors from config...")
	if err := registry.RegisterFromConfig(cfg.Collection.Collectors, collectorCfg); err != nil {
		logger.Warn("Some collectors failed to register: %v", err)
	}

	logger.Info("Registered %d collectors: %v", len(registry.List()), registry.List())

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start collection loop
	ticker := time.NewTicker(cfg.Collection.Interval)
	defer ticker.Stop()

	// Initial collection
	collect(ctx, registry, client, cfg.Collection.Collectors)

	logger.Info("Agent started. Press Ctrl+C to stop.")

	for {
		select {
		case <-ticker.C:
			collect(ctx, registry, client, cfg.Collection.Collectors)
		case sig := <-sigChan:
			logger.Info("Received signal %v, shutting down...", sig)
			cancel()
			return
		case <-ctx.Done():
			return
		}
	}
}

// collect performs a single collection cycle.
func collect(ctx context.Context, registry *collector.Registry, client *agent.Client, collectors []string) {
	// Create a timeout context for collection
	collectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	logger.Debug("Starting collection cycle...")

	// Collect metrics
	metrics, err := registry.CollectFrom(collectCtx, collectors)
	if err != nil {
		logger.Error("Collection error: %v", err)
	}

	if len(metrics) == 0 {
		logger.Warn("No metrics collected")
		return
	}

	logger.Info("Collected %d metrics", len(metrics))

	// Log individual metrics at debug level
	if logger.GetLevel() == logger.LevelDebug {
		for _, m := range metrics {
			logger.Debug("  %s = %.4f (%s)", m.Name, m.Value, m.Unit)
		}
	}

	// Send metrics to server
	logger.Debug("Sending metrics to server...")
	sendCtx, sendCancel := context.WithTimeout(ctx, 10*time.Second)
	defer sendCancel()

	if err := client.SendMetrics(sendCtx, metrics); err != nil {
		logger.Error("Failed to send metrics: %v", err)
	} else {
		logger.Debug("Metrics sent successfully")
	}
}

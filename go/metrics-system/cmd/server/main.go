// Metrics Collection Server
//
// This server receives metrics from agents via gRPC and stores them
// in PostgreSQL/TimescaleDB for querying and visualization.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/bellistech/metrics-system/internal/config"
	"github.com/bellistech/metrics-system/internal/logger"
	"github.com/bellistech/metrics-system/internal/server"
	"github.com/bellistech/metrics-system/internal/server/storage"
)

var Version = "dev"

func main() {
	// Parse command line flags
	configPath := flag.String("config", "configs/server.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version and exit")
	verbose := flag.Bool("v", false, "Enable verbose (info) logging")
	debug := flag.Bool("debug", false, "Enable debug logging (most verbose)")
	logLevel := flag.String("log-level", "", "Set log level: debug, info, warn, error")
	flag.Parse()

	if *showVersion {
		logger.Info("metrics-server version %s", Version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadServerConfig(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration: %v", err)
	}

	// Set log level (precedence: flag > config > default)
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

	logger.Info("Starting metrics server (version: %s)", Version)
	logger.Info("gRPC port: %d", cfg.GRPC.Port)
	logger.Debug("Database config: %s@%s:%d/%s", cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Database)

	// Connect to database
	logger.Debug("Connecting to database...")
	store, err := storage.NewPostgresStorage(cfg.Database.ConnectionString())
	if err != nil {
		logger.Fatal("Failed to connect to database: %v", err)
	}
	defer store.Close()

	logger.Info("Connected to database")

	// Create gRPC server
	grpcServer := server.NewGRPCServer(store)

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := grpcServer.Start(cfg.GRPC.Port); err != nil {
			logger.Fatal("Server failed: %v", err)
		}
	}()

	logger.Info("Server started. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		logger.Info("Received signal %v, shutting down...", sig)
		cancel()
	case <-ctx.Done():
	}

	logger.Info("Server stopped")
}

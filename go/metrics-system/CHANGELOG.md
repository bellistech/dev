# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-11-20

### Added

- Initial release of Metrics Collection System
- Agent for collecting system metrics:
  - CPU usage and load averages
  - Memory and swap usage
  - Disk usage and I/O statistics
  - Network traffic and connection states
  - System uptime and process counts
- gRPC-based client-server architecture
- PostgreSQL with TimescaleDB for time-series storage
- Configuration via YAML files
- Docker deployment support
- systemd service files for production deployment
- Grafana integration and datasource configuration
- Comprehensive documentation including Go crash course
- Example configurations and setup scripts

### Features

- Direct /proc filesystem reading for efficient metric collection
- Batch metric transmission for optimal network usage
- Automatic data compression and retention policies
- Health check endpoint for monitoring
- Graceful shutdown handling
- Configurable collection intervals
- Label support for multi-dimensional metrics

## [Unreleased]

### Planned

- Additional collectors (Apache, MySQL, Redis, Nginx)
- Web UI for server
- Alert manager integration
- Metric aggregation at agent level
- Support for custom metrics
- High availability server deployment
- Prometheus remote write compatibility

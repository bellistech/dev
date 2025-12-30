# Metrics Collection System

A lightweight, production-ready metrics collection system built in Go, featuring a gRPC-based client-server architecture for collecting and storing Linux system metrics.

## Features

- **Comprehensive Metrics Collection**: CPU, Memory, Disk, Network, and Uptime
- **gRPC Communication**: High-performance, type-safe client-server communication
- **Time Series Storage**: PostgreSQL/TimescaleDB with optimized queries
- **Grafana Integration**: Pre-configured dashboards and data source
- **Direct /proc Reading**: Efficient metric collection without spawning shell commands
- **Production Ready**: Proper logging, error handling, graceful shutdown

## Architecture

```
┌─────────────┐                    ┌─────────────┐
│   Agent     │                    │   Server    │
│  (Client)   │  ─── gRPC ───>    │             │
│             │                    │             │
└─────────────┘                    └──────┬──────┘
                                          │
                                          ▼
                                   ┌─────────────┐
                                   │ PostgreSQL  │
                                   │ (TimescaleDB)│
                                   └─────────────┘
                                          │
                                          ▼
                                   ┌─────────────┐
                                   │   Grafana   │
                                   └─────────────┘
```

## Project Structure

```
metrics-system/
├── api/
│   └── metrics/
│       └── v1/
│           └── metrics.proto          # Protocol buffer definitions
├── cmd/
│   ├── agent/
│   │   └── main.go                    # Agent entry point
│   └── server/
│       └── main.go                    # Server entry point
├── internal/
│   ├── agent/
│   │   ├── collector/                 # Metric collectors
│   │   │   ├── cpu.go
│   │   │   ├── memory.go
│   │   │   ├── disk.go
│   │   │   ├── network.go
│   │   │   ├── uptime.go
│   │   │   └── collector.go
│   │   └── client.go                  # gRPC client
│   ├── server/
│   │   ├── grpc.go                    # gRPC server
│   │   └── storage/
│   │       └── postgres.go            # PostgreSQL storage
│   └── config/
│       └── config.go                  # Configuration management
├── pkg/
│   └── metrics/                       # Shared metric types
│       └── types.go
├── deployments/
│   ├── docker/
│   │   ├── docker-compose.yml
│   │   ├── agent.Dockerfile
│   │   └── server.Dockerfile
│   └── systemd/
│       ├── metrics-agent.service
│       └── metrics-server.service
├── configs/
│   ├── agent.yaml
│   └── server.yaml
├── scripts/
│   ├── setup.sh
│   └── init-db.sql
├── grafana/
│   ├── dashboards/
│   └── datasources/
├── docs/
│   └── GO_CRASH_COURSE.md
├── .gitignore
├── .golangci.yml
├── go.mod
├── Makefile
├── LICENSE
├── CONTRIBUTING.md
├── CHANGELOG.md
└── README.md
```

## Quick Start

### Prerequisites

- Go 1.21 or later
- Protocol Buffers compiler (protoc)
- PostgreSQL 12+ (or Docker)
- Make

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/bellistech/metrics-system.git
   cd metrics-system
   ```

2. **Install dependencies:**
   ```bash
   make setup
   ```

3. **Generate protobuf code:**
   ```bash
   make proto
   ```

4. **Build the binaries:**
   ```bash
   make build
   ```

### Running with Docker Compose

```bash
cd deployments/docker
docker-compose up -d
```

This starts:
- PostgreSQL with TimescaleDB extension
- Metrics Server
- Metrics Agent
- Grafana (accessible at http://localhost:3000)

### Running Locally

1. **Start PostgreSQL:**
   ```bash
   docker run -d \
     --name metrics-postgres \
     -e POSTGRES_PASSWORD=metrics \
     -e POSTGRES_USER=metrics \
     -e POSTGRES_DB=metrics \
     -p 5432:5432 \
     timescale/timescaledb:latest-pg15
   
   psql -h localhost -U metrics -d metrics -f scripts/init-db.sql
   ```

2. **Start the server:**
   ```bash
   ./bin/server -config configs/server.yaml
   ```

3. **Start the agent:**
   ```bash
   sudo ./bin/agent -config configs/agent.yaml
   ```

## Configuration

### Agent Configuration (configs/agent.yaml)

```yaml
server:
  address: "localhost:9090"
  
collection:
  interval: 60s
  
collectors:
  - cpu
  - memory
  - disk
  - network
  - uptime
```

### Server Configuration (configs/server.yaml)

```yaml
grpc:
  port: 9090
  
database:
  host: localhost
  port: 5432
  user: metrics
  password: metrics
  database: metrics
  sslmode: disable
```

## Collected Metrics

| Category | Metrics |
|----------|---------|
| CPU | Usage per core, user/system/idle time, load averages, context switches |
| Memory | Total, free, available, swap usage, buffers/cache |
| Disk | Usage per filesystem, I/O ops, throughput, service time |
| Network | Bytes/packets sent/received, errors, TCP states |
| System | Uptime, process counts, open file descriptors |

## Development

```bash
make test      # Run tests
make lint      # Run linter
make fmt       # Format code
make proto     # Generate protobuf
make build     # Build binaries
make clean     # Clean build artifacts
```

## Production Deployment

### Using systemd

```bash
sudo cp deployments/systemd/*.service /etc/systemd/system/
sudo cp bin/agent /usr/local/bin/metrics-agent
sudo cp bin/server /usr/local/bin/metrics-server
sudo systemctl daemon-reload
sudo systemctl enable --now metrics-server
sudo systemctl enable --now metrics-agent
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - See [LICENSE](LICENSE) for details.

## Support

- GitHub Issues: https://github.com/bellistech/metrics-system/issues
- Documentation: See `docs/` folder

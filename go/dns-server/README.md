# Go DNS Server

An authoritative DNS server with IPv4/IPv6 dual-stack support, built from scratch in Go.

## Features

- **Dual-stack IPv4/IPv6** support
- **Record types**: A, AAAA, CNAME, MX, NS, TXT
- **BIND-style zone files**
- **Concurrent query handling**
- **Statistics tracking**
- **Graceful shutdown**

## Quick Start

```bash
# Build
go build -o dns-server ./cmd/dns-server

# Run (uses port 5353 by default to avoid needing root)
./dns-server -zone zones/example.com.zone

# Run on standard DNS port (requires root)
sudo ./dns-server -zone zones/example.com.zone -4 :53 -6 [::]:53
```

## Testing

```bash
# Query A record
dig @localhost -p 5353 example.com A

# Query AAAA record (IPv6)
dig @localhost -p 5353 example.com AAAA

# Query MX records
dig @localhost -p 5353 example.com MX

# Query TXT records
dig @localhost -p 5353 example.com TXT

# Query CNAME
dig @localhost -p 5353 ftp.example.com A

# Query non-existent domain (NXDOMAIN)
dig @localhost -p 5353 nonexistent.example.com A

# IPv6 query
dig @::1 -p 5353 example.com AAAA
```

## Command Line Options

```
-zone <file>  Zone file to load (required)
-4 <addr>     IPv4 listen address (default: :5353, empty to disable)
-6 <addr>     IPv6 listen address (default: [::]:5353, empty to disable)
```

## Zone File Format

BIND-style zone files are supported:

```
$ORIGIN example.com.
$TTL 3600

; A Records
@       IN  A       93.184.216.34
www     IN  A       93.184.216.34

; AAAA Records
@       IN  AAAA    2606:2800:220:1:248:1893:25c8:1946

; CNAME
ftp     IN  CNAME   www.example.com.

; MX Records
@       IN  MX  10  mail.example.com.

; TXT Records
@       IN  TXT     "v=spf1 mx -all"
```

## Project Structure

```
dns-server/
├── cmd/dns-server/main.go  # Server entry point
├── dns/
│   ├── types.go            # DNS types and constants
│   ├── parser.go           # DNS message parser
│   ├── builder.go          # DNS message builder
│   └── zone.go             # Zone file parser
└── zones/
    └── example.com.zone    # Example zone file
```

## Architecture

```
                    ┌─────────────────────────┐
   DNS Query ──────►│     UDP Listener       │
   (port 5353)      │   (IPv4 and/or IPv6)   │
                    └───────────┬─────────────┘
                                │
                    ┌───────────▼─────────────┐
                    │     Query Parser        │
                    │  (DNS wire format)      │
                    └───────────┬─────────────┘
                                │
                    ┌───────────▼─────────────┐
                    │     Zone Lookup         │
                    │  (thread-safe map)      │
                    └───────────┬─────────────┘
                                │
                    ┌───────────▼─────────────┐
                    │    Response Builder     │
                    │  (DNS wire format)      │
                    └───────────┬─────────────┘
                                │
   DNS Response ◄───────────────┘
```

## Learning Objectives

This project demonstrates:

1. **Binary protocol handling** - Parsing and building DNS wire format
2. **UDP networking** - Connectionless protocol handling
3. **Concurrent programming** - Goroutines for parallel queries
4. **IPv6 support** - Dual-stack networking
5. **File parsing** - Zone file format
6. **Production patterns** - Graceful shutdown, logging, statistics

## Extending

Ideas for extending this DNS server:

- Add TCP support (for large responses >512 bytes)
- Implement EDNS0 (extended DNS)
- Add DNSSEC signing
- Implement zone transfers (AXFR)
- Add caching/forwarding
- Add Prometheus metrics
- Containerize with Docker

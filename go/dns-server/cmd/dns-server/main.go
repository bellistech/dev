// DNS Server - An authoritative DNS server with IPv4/IPv6 dual-stack support
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/bellistech/dns-server/dns"
)

// Server represents the DNS server
type Server struct {
	zones   map[string]*dns.Zone
	mu      sync.RWMutex
	builder *dns.Builder

	udpConn4 *net.UDPConn
	udpConn6 *net.UDPConn

	// Statistics
	queries  uint64
	answers  uint64
	nxdomain uint64
	errors   uint64
}

// NewServer creates a new DNS server
func NewServer() *Server {
	return &Server{
		zones:   make(map[string]*dns.Zone),
		builder: dns.NewBuilder(),
	}
}

// LoadZone loads a zone file
func (s *Server) LoadZone(filename string) error {
	zone, err := dns.LoadZoneFile(filename)
	if err != nil {
		return fmt.Errorf("loading %s: %w", filename, err)
	}

	s.mu.Lock()
	s.zones[zone.Name] = zone
	s.mu.Unlock()

	log.Printf("Loaded zone: %s", zone.Name)
	return nil
}

// Start starts the DNS server
func (s *Server) Start(ctx context.Context, addr4, addr6 string) error {
	var wg sync.WaitGroup

	// Start IPv4 listener
	if addr4 != "" {
		udpAddr4, err := net.ResolveUDPAddr("udp4", addr4)
		if err != nil {
			return fmt.Errorf("resolve IPv4: %w", err)
		}

		s.udpConn4, err = net.ListenUDP("udp4", udpAddr4)
		if err != nil {
			return fmt.Errorf("listen IPv4: %w", err)
		}

		log.Printf("Listening on IPv4 %s", addr4)

		wg.Add(1)
		go func() {
			defer wg.Done()
			s.serveUDP(ctx, s.udpConn4)
		}()
	}

	// Start IPv6 listener
	if addr6 != "" {
		udpAddr6, err := net.ResolveUDPAddr("udp6", addr6)
		if err != nil {
			return fmt.Errorf("resolve IPv6: %w", err)
		}

		s.udpConn6, err = net.ListenUDP("udp6", udpAddr6)
		if err != nil {
			return fmt.Errorf("listen IPv6: %w", err)
		}

		log.Printf("Listening on IPv6 %s", addr6)

		wg.Add(1)
		go func() {
			defer wg.Done()
			s.serveUDP(ctx, s.udpConn6)
		}()
	}

	wg.Wait()
	return nil
}

func (s *Server) serveUDP(ctx context.Context, conn *net.UDPConn) {
	buffer := make([]byte, 512)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("Read error: %v", err)
				continue
			}
		}

		// Copy data for goroutine
		data := make([]byte, n)
		copy(data, buffer[:n])

		// Handle in goroutine for concurrency
		go s.handleQuery(conn, clientAddr, data)
	}
}

func (s *Server) handleQuery(conn *net.UDPConn, clientAddr *net.UDPAddr, data []byte) {
	atomic.AddUint64(&s.queries, 1)

	// Parse query
	parser := dns.NewParser(data)
	query, err := parser.Parse()
	if err != nil {
		log.Printf("Parse error from %s: %v", clientAddr, err)
		atomic.AddUint64(&s.errors, 1)
		return
	}

	if len(query.Questions) == 0 {
		return
	}

	q := query.Questions[0]
	log.Printf("Query from %s: %s %s", clientAddr, q.Name, dns.TypeToString(q.Type))

	// Find zone
	zone := s.findZone(q.Name)
	if zone == nil {
		// Not authoritative
		response := s.builder.BuildErrorResponse(query, dns.RcodeRefused)
		conn.WriteToUDP(response, clientAddr)
		return
	}

	// Lookup records
	records := zone.Lookup(q.Name, q.Type)

	if len(records) == 0 && !zone.HasName(q.Name) {
		// NXDOMAIN
		atomic.AddUint64(&s.nxdomain, 1)
		response := s.builder.BuildErrorResponse(query, dns.RcodeNameError)
		conn.WriteToUDP(response, clientAddr)
		log.Printf("  -> NXDOMAIN")
		return
	}

	// Build response
	atomic.AddUint64(&s.answers, 1)

	// Get NS records for authority section
	nsRecords := zone.Lookup(zone.Name, dns.TypeNS)

	response := s.builder.BuildResponse(query, records, nsRecords)
	conn.WriteToUDP(response, clientAddr)

	if len(records) > 0 {
		log.Printf("  -> %d record(s)", len(records))
	} else {
		log.Printf("  -> NODATA")
	}
}

func (s *Server) findZone(name string) *dns.Zone {
	s.mu.RLock()
	defer s.mu.RUnlock()

	name = strings.ToLower(name)

	// Find most specific zone
	labels := splitLabels(name)

	for i := 0; i < len(labels); i++ {
		zoneName := joinLabels(labels[i:])
		if zone, ok := s.zones[zoneName]; ok {
			return zone
		}
	}

	return nil
}

func splitLabels(name string) []string {
	name = strings.TrimSuffix(name, ".")
	if name == "" {
		return nil
	}
	return strings.Split(name, ".")
}

func joinLabels(labels []string) string {
	return strings.Join(labels, ".")
}

// Stop stops the server and prints statistics
func (s *Server) Stop() {
	if s.udpConn4 != nil {
		s.udpConn4.Close()
	}
	if s.udpConn6 != nil {
		s.udpConn6.Close()
	}

	log.Printf("Statistics: queries=%d, answers=%d, nxdomain=%d, errors=%d",
		atomic.LoadUint64(&s.queries),
		atomic.LoadUint64(&s.answers),
		atomic.LoadUint64(&s.nxdomain),
		atomic.LoadUint64(&s.errors))
}

func main() {
	addr4 := flag.String("4", ":5353", "IPv4 listen address (empty to disable)")
	addr6 := flag.String("6", "[::]:5353", "IPv6 listen address (empty to disable)")
	zoneFile := flag.String("zone", "", "Zone file to load (required)")
	flag.Parse()

	if *zoneFile == "" {
		fmt.Fprintln(os.Stderr, "Error: Zone file required (-zone)")
		fmt.Fprintln(os.Stderr, "Usage: dns-server -zone <zonefile> [-4 <addr>] [-6 <addr>]")
		fmt.Fprintln(os.Stderr, "\nExample:")
		fmt.Fprintln(os.Stderr, "  dns-server -zone zones/example.com.zone")
		fmt.Fprintln(os.Stderr, "  dns-server -zone zones/example.com.zone -4 :53 -6 \"\"")
		os.Exit(1)
	}

	server := NewServer()

	if err := server.LoadZone(*zoneFile); err != nil {
		log.Fatalf("Failed to load zone: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
		server.Stop()
	}()

	log.Println("DNS Server starting...")
	if err := server.Start(ctx, *addr4, *addr6); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// Package dns implements DNS message parsing and building.
package dns

import (
	"fmt"
	"net"
)

// DNS record types
const (
	TypeA     uint16 = 1
	TypeNS    uint16 = 2
	TypeCNAME uint16 = 5
	TypeSOA   uint16 = 6
	TypeMX    uint16 = 15
	TypeTXT   uint16 = 16
	TypeAAAA  uint16 = 28
)

// DNS classes
const (
	ClassIN uint16 = 1 // Internet
)

// DNS response codes
const (
	RcodeNoError        uint8 = 0
	RcodeFormatError    uint8 = 1
	RcodeServerFailure  uint8 = 2
	RcodeNameError      uint8 = 3 // NXDOMAIN
	RcodeNotImplemented uint8 = 4
	RcodeRefused        uint8 = 5
)

// DNS flags
const (
	FlagQR uint16 = 1 << 15 // Query/Response
	FlagAA uint16 = 1 << 10 // Authoritative Answer
	FlagTC uint16 = 1 << 9  // Truncated
	FlagRD uint16 = 1 << 8  // Recursion Desired
	FlagRA uint16 = 1 << 7  // Recursion Available
)

// Header represents a DNS message header
type Header struct {
	ID      uint16
	Flags   uint16
	QDCount uint16 // Question count
	ANCount uint16 // Answer count
	NSCount uint16 // Authority count
	ARCount uint16 // Additional count
}

// Question represents a DNS question
type Question struct {
	Name  string
	Type  uint16
	Class uint16
}

// ResourceRecord represents a DNS resource record
type ResourceRecord struct {
	Name     string
	Type     uint16
	Class    uint16
	TTL      uint32
	RDLength uint16
	RData    []byte

	// Parsed data (depending on type)
	Address  net.IP   // For A, AAAA
	Target   string   // For CNAME, NS, MX
	Priority uint16   // For MX
	Text     []string // For TXT
	SOAData  *SOA     // For SOA
}

// SOA represents Start of Authority data
type SOA struct {
	MName   string // Primary nameserver
	RName   string // Admin email (@ replaced with .)
	Serial  uint32
	Refresh uint32
	Retry   uint32
	Expire  uint32
	Minimum uint32
}

// Message represents a complete DNS message
type Message struct {
	Header     Header
	Questions  []Question
	Answers    []ResourceRecord
	Authority  []ResourceRecord
	Additional []ResourceRecord
}

// TypeToString converts record type to string
func TypeToString(t uint16) string {
	switch t {
	case TypeA:
		return "A"
	case TypeAAAA:
		return "AAAA"
	case TypeCNAME:
		return "CNAME"
	case TypeMX:
		return "MX"
	case TypeNS:
		return "NS"
	case TypeTXT:
		return "TXT"
	case TypeSOA:
		return "SOA"
	default:
		return fmt.Sprintf("TYPE%d", t)
	}
}

// StringToType converts string to record type
func StringToType(s string) uint16 {
	switch s {
	case "A":
		return TypeA
	case "AAAA":
		return TypeAAAA
	case "CNAME":
		return TypeCNAME
	case "MX":
		return TypeMX
	case "NS":
		return TypeNS
	case "TXT":
		return TypeTXT
	case "SOA":
		return TypeSOA
	default:
		return 0
	}
}

// NewARecord creates an A record
func NewARecord(name string, ttl uint32, ip net.IP) ResourceRecord {
	return ResourceRecord{
		Name:    name,
		Type:    TypeA,
		Class:   ClassIN,
		TTL:     ttl,
		Address: ip.To4(),
	}
}

// NewAAAARecord creates an AAAA record
func NewAAAARecord(name string, ttl uint32, ip net.IP) ResourceRecord {
	return ResourceRecord{
		Name:    name,
		Type:    TypeAAAA,
		Class:   ClassIN,
		TTL:     ttl,
		Address: ip.To16(),
	}
}

// NewCNAMERecord creates a CNAME record
func NewCNAMERecord(name string, ttl uint32, target string) ResourceRecord {
	return ResourceRecord{
		Name:   name,
		Type:   TypeCNAME,
		Class:  ClassIN,
		TTL:    ttl,
		Target: target,
	}
}

// NewMXRecord creates an MX record
func NewMXRecord(name string, ttl uint32, priority uint16, target string) ResourceRecord {
	return ResourceRecord{
		Name:     name,
		Type:     TypeMX,
		Class:    ClassIN,
		TTL:      ttl,
		Priority: priority,
		Target:   target,
	}
}

// NewTXTRecord creates a TXT record
func NewTXTRecord(name string, ttl uint32, texts ...string) ResourceRecord {
	return ResourceRecord{
		Name:  name,
		Type:  TypeTXT,
		Class: ClassIN,
		TTL:   ttl,
		Text:  texts,
	}
}

// NewNSRecord creates an NS record
func NewNSRecord(name string, ttl uint32, target string) ResourceRecord {
	return ResourceRecord{
		Name:   name,
		Type:   TypeNS,
		Class:  ClassIN,
		TTL:    ttl,
		Target: target,
	}
}

// NewSOARecord creates an SOA record
func NewSOARecord(name string, ttl uint32, soa *SOA) ResourceRecord {
	return ResourceRecord{
		Name:    name,
		Type:    TypeSOA,
		Class:   ClassIN,
		TTL:     ttl,
		SOAData: soa,
	}
}

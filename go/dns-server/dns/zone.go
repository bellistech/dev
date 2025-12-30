package dns

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Zone represents a DNS zone
type Zone struct {
	Name    string
	Records map[string][]ResourceRecord // Keyed by name+type
	SOA     *SOA
	mu      sync.RWMutex
}

// NewZone creates a new zone
func NewZone(name string) *Zone {
	return &Zone{
		Name:    strings.ToLower(name),
		Records: make(map[string][]ResourceRecord),
	}
}

// AddRecord adds a record to the zone
func (z *Zone) AddRecord(rr ResourceRecord) {
	z.mu.Lock()
	defer z.mu.Unlock()

	key := z.recordKey(rr.Name, rr.Type)
	z.Records[key] = append(z.Records[key], rr)

	if rr.Type == TypeSOA && rr.SOAData != nil {
		z.SOA = rr.SOAData
	}
}

// Lookup finds records matching name and type
func (z *Zone) Lookup(name string, qtype uint16) []ResourceRecord {
	z.mu.RLock()
	defer z.mu.RUnlock()

	name = strings.ToLower(name)

	// Direct match
	key := z.recordKey(name, qtype)
	if records, ok := z.Records[key]; ok {
		return records
	}

	// If looking for A/AAAA, check for CNAME
	if qtype == TypeA || qtype == TypeAAAA {
		cnameKey := z.recordKey(name, TypeCNAME)
		if cnames, ok := z.Records[cnameKey]; ok {
			return cnames
		}
	}

	return nil
}

// HasName checks if zone has any records for name
func (z *Zone) HasName(name string) bool {
	z.mu.RLock()
	defer z.mu.RUnlock()

	name = strings.ToLower(name)

	for key := range z.Records {
		if strings.HasPrefix(key, name+":") {
			return true
		}
	}
	return false
}

// IsAuthoritative checks if this zone is authoritative for the name
func (z *Zone) IsAuthoritative(name string) bool {
	name = strings.ToLower(name)
	zoneName := strings.ToLower(z.Name)

	return name == zoneName || strings.HasSuffix(name, "."+zoneName)
}

func (z *Zone) recordKey(name string, qtype uint16) string {
	return strings.ToLower(name) + ":" + strconv.Itoa(int(qtype))
}

// LoadZoneFile loads a zone from BIND-style zone file
func LoadZoneFile(filename string) (*Zone, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var zone *Zone
	var origin string
	var defaultTTL uint32 = 3600
	var currentName string

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		// Handle directives
		if strings.HasPrefix(line, "$ORIGIN") {
			origin = strings.TrimSpace(strings.TrimPrefix(line, "$ORIGIN"))
			origin = strings.TrimSuffix(origin, ".")
			if zone == nil {
				zone = NewZone(origin)
			}
			continue
		}

		if strings.HasPrefix(line, "$TTL") {
			ttlStr := strings.TrimSpace(strings.TrimPrefix(line, "$TTL"))
			ttl, err := parseTTL(ttlStr)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid TTL: %v", lineNum, err)
			}
			defaultTTL = ttl
			continue
		}

		// Skip multi-line SOA for now (simplified parser)
		if strings.Contains(line, "(") {
			// Read until closing paren
			for scanner.Scan() {
				lineNum++
				if strings.Contains(scanner.Text(), ")") {
					break
				}
			}
			continue
		}

		// Parse record
		rr, name, err := parseZoneLine(line, origin, currentName, defaultTTL)
		if err != nil {
			// Skip unparseable lines
			continue
		}

		if name != "" {
			currentName = name
		}

		if zone == nil {
			zone = NewZone(origin)
		}

		zone.AddRecord(rr)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return zone, nil
}

func parseZoneLine(line, origin, currentName string, defaultTTL uint32) (ResourceRecord, string, error) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return ResourceRecord{}, "", fmt.Errorf("too few fields")
	}

	var rr ResourceRecord
	var name string
	idx := 0

	// First field: name, TTL, class, or type
	field := fields[idx]

	// Check if first field is a name
	if !isClassOrType(field) && !isTTL(field) {
		if field == "@" {
			name = origin
		} else if !strings.HasSuffix(field, ".") {
			name = field + "." + origin
		} else {
			name = strings.TrimSuffix(field, ".")
		}
		idx++
	} else {
		name = currentName
	}

	rr.Name = name
	rr.TTL = defaultTTL
	rr.Class = ClassIN

	// Parse optional TTL
	if idx < len(fields) && isTTL(fields[idx]) {
		ttl, _ := parseTTL(fields[idx])
		rr.TTL = ttl
		idx++
	}

	// Parse optional class
	if idx < len(fields) && isClass(fields[idx]) {
		idx++ // Skip class, assume IN
	}

	// Parse type
	if idx >= len(fields) {
		return rr, name, fmt.Errorf("missing type")
	}

	rr.Type = StringToType(strings.ToUpper(fields[idx]))
	if rr.Type == 0 {
		return rr, name, fmt.Errorf("unknown type: %s", fields[idx])
	}
	idx++

	// Parse RDATA
	if idx >= len(fields) {
		return rr, name, fmt.Errorf("missing rdata")
	}

	rdata := strings.Join(fields[idx:], " ")

	switch rr.Type {
	case TypeA:
		ip := net.ParseIP(rdata)
		if ip == nil || ip.To4() == nil {
			return rr, name, fmt.Errorf("invalid IPv4: %s", rdata)
		}
		rr.Address = ip.To4()

	case TypeAAAA:
		ip := net.ParseIP(rdata)
		if ip == nil || ip.To16() == nil || ip.To4() != nil {
			return rr, name, fmt.Errorf("invalid IPv6: %s", rdata)
		}
		rr.Address = ip.To16()

	case TypeCNAME, TypeNS:
		target := fields[idx]
		if target == "@" {
			target = origin
		} else if !strings.HasSuffix(target, ".") {
			target = target + "." + origin
		} else {
			target = strings.TrimSuffix(target, ".")
		}
		rr.Target = target

	case TypeMX:
		if idx+1 >= len(fields) {
			return rr, name, fmt.Errorf("MX needs priority and target")
		}
		priority, err := strconv.ParseUint(fields[idx], 10, 16)
		if err != nil {
			return rr, name, fmt.Errorf("invalid MX priority: %v", err)
		}
		rr.Priority = uint16(priority)

		target := fields[idx+1]
		if !strings.HasSuffix(target, ".") {
			target = target + "." + origin
		} else {
			target = strings.TrimSuffix(target, ".")
		}
		rr.Target = target

	case TypeTXT:
		// Handle quoted strings
		text := strings.Trim(rdata, "\"")
		rr.Text = []string{text}

	case TypeSOA:
		// Simplified SOA handling
		if len(fields) >= idx+7 {
			soa := &SOA{}
			soa.MName = normalizeSOAName(fields[idx], origin)
			soa.RName = normalizeSOAName(fields[idx+1], origin)
			soa.Serial, _ = parseUint32(fields[idx+2])
			soa.Refresh, _ = parseTTL(fields[idx+3])
			soa.Retry, _ = parseTTL(fields[idx+4])
			soa.Expire, _ = parseTTL(fields[idx+5])
			soa.Minimum, _ = parseTTL(fields[idx+6])
			rr.SOAData = soa
		}
	}

	return rr, name, nil
}

func normalizeSOAName(name, origin string) string {
	if name == "@" {
		return origin
	}
	if !strings.HasSuffix(name, ".") {
		return name + "." + origin
	}
	return strings.TrimSuffix(name, ".")
}

func isClassOrType(s string) bool {
	return isClass(s) || StringToType(strings.ToUpper(s)) != 0
}

func isClass(s string) bool {
	return strings.ToUpper(s) == "IN" || strings.ToUpper(s) == "CH"
}

func isTTL(s string) bool {
	_, err := parseTTL(s)
	return err == nil
}

func parseTTL(s string) (uint32, error) {
	s = strings.ToLower(s)
	multiplier := uint32(1)

	if strings.HasSuffix(s, "w") {
		multiplier = 604800
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "d") {
		multiplier = 86400
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "h") {
		multiplier = 3600
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "m") {
		multiplier = 60
		s = s[:len(s)-1]
	} else if strings.HasSuffix(s, "s") {
		s = s[:len(s)-1]
	}

	val, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}

	return uint32(val) * multiplier, nil
}

func parseUint32(s string) (uint32, error) {
	val, err := strconv.ParseUint(s, 10, 32)
	return uint32(val), err
}

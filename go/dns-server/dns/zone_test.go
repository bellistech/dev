package dns

import (
	"os"
	"testing"
)

func TestZoneAddAndLookup(t *testing.T) {
	zone := NewZone("example.com")

	// Add A record
	zone.AddRecord(ResourceRecord{
		Name:    "example.com",
		Type:    TypeA,
		Class:   ClassIN,
		TTL:     3600,
		Address: []byte{93, 184, 216, 34},
	})

	// Lookup
	records := zone.Lookup("example.com", TypeA)
	if len(records) != 1 {
		t.Fatalf("Lookup returned %d records, want 1", len(records))
	}

	if records[0].TTL != 3600 {
		t.Errorf("TTL = %d, want 3600", records[0].TTL)
	}
}

func TestZoneLookupCNAME(t *testing.T) {
	zone := NewZone("example.com")

	// Add CNAME record
	zone.AddRecord(ResourceRecord{
		Name:   "www.example.com",
		Type:   TypeCNAME,
		Class:  ClassIN,
		TTL:    3600,
		Target: "example.com",
	})

	// Looking up A should return CNAME
	records := zone.Lookup("www.example.com", TypeA)
	if len(records) != 1 {
		t.Fatalf("Lookup returned %d records, want 1", len(records))
	}

	if records[0].Type != TypeCNAME {
		t.Errorf("Type = %d, want %d (CNAME)", records[0].Type, TypeCNAME)
	}
}

func TestZoneHasName(t *testing.T) {
	zone := NewZone("example.com")

	zone.AddRecord(ResourceRecord{
		Name:  "www.example.com",
		Type:  TypeA,
		Class: ClassIN,
		TTL:   3600,
	})

	if !zone.HasName("www.example.com") {
		t.Error("HasName(www.example.com) = false, want true")
	}

	if zone.HasName("ftp.example.com") {
		t.Error("HasName(ftp.example.com) = true, want false")
	}
}

func TestZoneCaseInsensitive(t *testing.T) {
	zone := NewZone("Example.COM")

	zone.AddRecord(ResourceRecord{
		Name:    "WWW.Example.COM",
		Type:    TypeA,
		Class:   ClassIN,
		TTL:     3600,
		Address: []byte{1, 2, 3, 4},
	})

	// Lookup with different case
	records := zone.Lookup("www.example.com", TypeA)
	if len(records) != 1 {
		t.Fatalf("Case-insensitive lookup failed: got %d records", len(records))
	}
}

func TestLoadZoneFile(t *testing.T) {
	// Create temporary zone file
	content := `$ORIGIN test.com.
$TTL 3600

@       IN  NS  ns1.test.com.
@       IN  A   192.0.2.1
www     IN  A   192.0.2.2
mail    IN  MX  10 mail.test.com.
`
	tmpfile, err := os.CreateTemp("", "zone-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Load zone
	zone, err := LoadZoneFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadZoneFile error: %v", err)
	}

	if zone.Name != "test.com" {
		t.Errorf("Zone name = %s, want test.com", zone.Name)
	}

	// Check NS record
	ns := zone.Lookup("test.com", TypeNS)
	if len(ns) != 1 {
		t.Errorf("NS records = %d, want 1", len(ns))
	}

	// Check A record for @
	a := zone.Lookup("test.com", TypeA)
	if len(a) != 1 {
		t.Errorf("A records for @ = %d, want 1", len(a))
	}

	// Check A record for www
	www := zone.Lookup("www.test.com", TypeA)
	if len(www) != 1 {
		t.Errorf("A records for www = %d, want 1", len(www))
	}

	// Check MX record
	mx := zone.Lookup("test.com", TypeMX)
	if len(mx) != 1 {
		t.Errorf("MX records = %d, want 1", len(mx))
	}
	if mx[0].Priority != 10 {
		t.Errorf("MX priority = %d, want 10", mx[0].Priority)
	}
}

func TestParseTTL(t *testing.T) {
	tests := []struct {
		input string
		want  uint32
	}{
		{"3600", 3600},
		{"1h", 3600},
		{"1d", 86400},
		{"1w", 604800},
		{"30m", 1800},
		{"60s", 60},
		{"2h", 7200},
		{"7d", 604800},
	}

	for _, tt := range tests {
		got, err := parseTTL(tt.input)
		if err != nil {
			t.Errorf("parseTTL(%s) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseTTL(%s) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestIsAuthoritative(t *testing.T) {
	zone := NewZone("example.com")

	tests := []struct {
		name string
		want bool
	}{
		{"example.com", true},
		{"www.example.com", true},
		{"sub.domain.example.com", true},
		{"other.com", false},
		{"exampleXcom", false},
	}

	for _, tt := range tests {
		got := zone.IsAuthoritative(tt.name)
		if got != tt.want {
			t.Errorf("IsAuthoritative(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

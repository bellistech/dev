package dns

import (
	"net"
	"testing"
)

func TestParseQuery(t *testing.T) {
	// DNS query for example.com A record
	query := []byte{
		0x12, 0x34, // ID
		0x01, 0x00, // Flags (standard query, RD=1)
		0x00, 0x01, // Questions: 1
		0x00, 0x00, // Answers: 0
		0x00, 0x00, // Authority: 0
		0x00, 0x00, // Additional: 0
		// Question: example.com A IN
		0x07, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
		0x03, 'c', 'o', 'm',
		0x00,       // End of name
		0x00, 0x01, // Type A
		0x00, 0x01, // Class IN
	}

	parser := NewParser(query)
	msg, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if msg.Header.ID != 0x1234 {
		t.Errorf("ID = %x, want 0x1234", msg.Header.ID)
	}

	if msg.Header.Flags&FlagRD == 0 {
		t.Error("RD flag not set")
	}

	if len(msg.Questions) != 1 {
		t.Fatalf("Questions = %d, want 1", len(msg.Questions))
	}

	q := msg.Questions[0]
	if q.Name != "example.com" {
		t.Errorf("Name = %s, want example.com", q.Name)
	}
	if q.Type != TypeA {
		t.Errorf("Type = %d, want %d", q.Type, TypeA)
	}
	if q.Class != ClassIN {
		t.Errorf("Class = %d, want %d", q.Class, ClassIN)
	}
}

func TestParseQueryAAAA(t *testing.T) {
	// DNS query for example.com AAAA record
	query := []byte{
		0xAB, 0xCD, // ID
		0x01, 0x00, // Flags
		0x00, 0x01, // Questions: 1
		0x00, 0x00, // Answers: 0
		0x00, 0x00, // Authority: 0
		0x00, 0x00, // Additional: 0
		// Question: example.com AAAA IN
		0x07, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
		0x03, 'c', 'o', 'm',
		0x00,       // End of name
		0x00, 0x1C, // Type AAAA (28)
		0x00, 0x01, // Class IN
	}

	parser := NewParser(query)
	msg, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if msg.Header.ID != 0xABCD {
		t.Errorf("ID = %x, want 0xABCD", msg.Header.ID)
	}

	q := msg.Questions[0]
	if q.Type != TypeAAAA {
		t.Errorf("Type = %d, want %d (AAAA)", q.Type, TypeAAAA)
	}
}

func TestParseSubdomain(t *testing.T) {
	// Query for www.example.com
	query := []byte{
		0x00, 0x01, // ID
		0x01, 0x00, // Flags
		0x00, 0x01, // Questions: 1
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// www.example.com
		0x03, 'w', 'w', 'w',
		0x07, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
		0x03, 'c', 'o', 'm',
		0x00,
		0x00, 0x01, // A
		0x00, 0x01, // IN
	}

	parser := NewParser(query)
	msg, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if msg.Questions[0].Name != "www.example.com" {
		t.Errorf("Name = %s, want www.example.com", msg.Questions[0].Name)
	}
}

func TestParseTooShort(t *testing.T) {
	// Too short to be valid
	query := []byte{0x12, 0x34, 0x01, 0x00}

	parser := NewParser(query)
	_, err := parser.Parse()
	if err == nil {
		t.Error("Expected error for short query")
	}
}

func TestBuildResponse(t *testing.T) {
	query := &Message{
		Header: Header{ID: 0x1234, QDCount: 1, Flags: FlagRD},
		Questions: []Question{
			{Name: "example.com", Type: TypeA, Class: ClassIN},
		},
	}

	answers := []ResourceRecord{
		NewARecord("example.com", 3600, net.ParseIP("93.184.216.34")),
	}

	builder := NewBuilder()
	response := builder.BuildResponse(query, answers, nil)

	// Parse response
	parser := NewParser(response)
	msg, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Check header
	if msg.Header.ID != 0x1234 {
		t.Errorf("Response ID = %x, want 0x1234", msg.Header.ID)
	}

	if msg.Header.Flags&FlagQR == 0 {
		t.Error("QR flag not set")
	}

	if msg.Header.Flags&FlagAA == 0 {
		t.Error("AA flag not set")
	}

	if msg.Header.Flags&FlagRD == 0 {
		t.Error("RD flag not preserved")
	}

	// Check answers
	if len(msg.Answers) != 1 {
		t.Fatalf("Answers = %d, want 1", len(msg.Answers))
	}

	a := msg.Answers[0]
	expected := net.ParseIP("93.184.216.34").To4()
	if !a.Address.Equal(expected) {
		t.Errorf("Address = %v, want %v", a.Address, expected)
	}

	if a.TTL != 3600 {
		t.Errorf("TTL = %d, want 3600", a.TTL)
	}
}

func TestBuildResponseAAAA(t *testing.T) {
	query := &Message{
		Header: Header{ID: 0x5678, QDCount: 1},
		Questions: []Question{
			{Name: "example.com", Type: TypeAAAA, Class: ClassIN},
		},
	}

	answers := []ResourceRecord{
		NewAAAARecord("example.com", 7200, net.ParseIP("2606:2800:220:1:248:1893:25c8:1946")),
	}

	builder := NewBuilder()
	response := builder.BuildResponse(query, answers, nil)

	parser := NewParser(response)
	msg, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(msg.Answers) != 1 {
		t.Fatalf("Answers = %d, want 1", len(msg.Answers))
	}

	a := msg.Answers[0]
	if a.Type != TypeAAAA {
		t.Errorf("Type = %d, want %d", a.Type, TypeAAAA)
	}

	expected := net.ParseIP("2606:2800:220:1:248:1893:25c8:1946")
	if !a.Address.Equal(expected) {
		t.Errorf("Address = %v, want %v", a.Address, expected)
	}
}

func TestBuildErrorResponse(t *testing.T) {
	query := &Message{
		Header: Header{ID: 0xABCD, QDCount: 1},
		Questions: []Question{
			{Name: "nonexistent.example.com", Type: TypeA, Class: ClassIN},
		},
	}

	builder := NewBuilder()
	response := builder.BuildErrorResponse(query, RcodeNameError)

	parser := NewParser(response)
	msg, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if msg.Header.ID != 0xABCD {
		t.Errorf("ID = %x, want 0xABCD", msg.Header.ID)
	}

	rcode := uint8(msg.Header.Flags & 0x000F)
	if rcode != RcodeNameError {
		t.Errorf("RCODE = %d, want %d (NXDOMAIN)", rcode, RcodeNameError)
	}

	if len(msg.Answers) != 0 {
		t.Errorf("Answers = %d, want 0", len(msg.Answers))
	}
}

func TestTypeToString(t *testing.T) {
	tests := []struct {
		typ  uint16
		want string
	}{
		{TypeA, "A"},
		{TypeAAAA, "AAAA"},
		{TypeCNAME, "CNAME"},
		{TypeMX, "MX"},
		{TypeNS, "NS"},
		{TypeTXT, "TXT"},
		{TypeSOA, "SOA"},
		{99, "TYPE99"},
	}

	for _, tt := range tests {
		got := TypeToString(tt.typ)
		if got != tt.want {
			t.Errorf("TypeToString(%d) = %s, want %s", tt.typ, got, tt.want)
		}
	}
}

func TestStringToType(t *testing.T) {
	tests := []struct {
		s    string
		want uint16
	}{
		{"A", TypeA},
		{"AAAA", TypeAAAA},
		{"CNAME", TypeCNAME},
		{"MX", TypeMX},
		{"NS", TypeNS},
		{"TXT", TypeTXT},
		{"SOA", TypeSOA},
		{"UNKNOWN", 0},
	}

	for _, tt := range tests {
		got := StringToType(tt.s)
		if got != tt.want {
			t.Errorf("StringToType(%s) = %d, want %d", tt.s, got, tt.want)
		}
	}
}

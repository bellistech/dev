package dns

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

// Parser handles DNS message parsing
type Parser struct {
	data []byte
	pos  int
}

// NewParser creates a new DNS parser
func NewParser(data []byte) *Parser {
	return &Parser{data: data, pos: 0}
}

// Parse parses a complete DNS message
func (p *Parser) Parse() (*Message, error) {
	msg := &Message{}

	// Parse header
	if err := p.parseHeader(&msg.Header); err != nil {
		return nil, fmt.Errorf("header: %w", err)
	}

	// Parse questions
	msg.Questions = make([]Question, msg.Header.QDCount)
	for i := 0; i < int(msg.Header.QDCount); i++ {
		if err := p.parseQuestion(&msg.Questions[i]); err != nil {
			return nil, fmt.Errorf("question %d: %w", i, err)
		}
	}

	// Parse answers
	msg.Answers = make([]ResourceRecord, msg.Header.ANCount)
	for i := 0; i < int(msg.Header.ANCount); i++ {
		if err := p.parseResourceRecord(&msg.Answers[i]); err != nil {
			return nil, fmt.Errorf("answer %d: %w", i, err)
		}
	}

	// Parse authority
	msg.Authority = make([]ResourceRecord, msg.Header.NSCount)
	for i := 0; i < int(msg.Header.NSCount); i++ {
		if err := p.parseResourceRecord(&msg.Authority[i]); err != nil {
			return nil, fmt.Errorf("authority %d: %w", i, err)
		}
	}

	// Parse additional
	msg.Additional = make([]ResourceRecord, msg.Header.ARCount)
	for i := 0; i < int(msg.Header.ARCount); i++ {
		if err := p.parseResourceRecord(&msg.Additional[i]); err != nil {
			return nil, fmt.Errorf("additional %d: %w", i, err)
		}
	}

	return msg, nil
}

func (p *Parser) parseHeader(h *Header) error {
	if len(p.data) < 12 {
		return fmt.Errorf("header too short")
	}

	h.ID = binary.BigEndian.Uint16(p.data[0:2])
	h.Flags = binary.BigEndian.Uint16(p.data[2:4])
	h.QDCount = binary.BigEndian.Uint16(p.data[4:6])
	h.ANCount = binary.BigEndian.Uint16(p.data[6:8])
	h.NSCount = binary.BigEndian.Uint16(p.data[8:10])
	h.ARCount = binary.BigEndian.Uint16(p.data[10:12])

	p.pos = 12
	return nil
}

func (p *Parser) parseQuestion(q *Question) error {
	name, err := p.parseName()
	if err != nil {
		return err
	}
	q.Name = name

	if p.pos+4 > len(p.data) {
		return fmt.Errorf("question too short")
	}

	q.Type = binary.BigEndian.Uint16(p.data[p.pos : p.pos+2])
	q.Class = binary.BigEndian.Uint16(p.data[p.pos+2 : p.pos+4])
	p.pos += 4

	return nil
}

func (p *Parser) parseResourceRecord(rr *ResourceRecord) error {
	name, err := p.parseName()
	if err != nil {
		return err
	}
	rr.Name = name

	if p.pos+10 > len(p.data) {
		return fmt.Errorf("resource record too short")
	}

	rr.Type = binary.BigEndian.Uint16(p.data[p.pos : p.pos+2])
	rr.Class = binary.BigEndian.Uint16(p.data[p.pos+2 : p.pos+4])
	rr.TTL = binary.BigEndian.Uint32(p.data[p.pos+4 : p.pos+8])
	rr.RDLength = binary.BigEndian.Uint16(p.data[p.pos+8 : p.pos+10])
	p.pos += 10

	if p.pos+int(rr.RDLength) > len(p.data) {
		return fmt.Errorf("rdata too short")
	}

	rr.RData = p.data[p.pos : p.pos+int(rr.RDLength)]

	// Parse type-specific data
	switch rr.Type {
	case TypeA:
		if rr.RDLength == 4 {
			rr.Address = net.IP(rr.RData)
		}
	case TypeAAAA:
		if rr.RDLength == 16 {
			rr.Address = net.IP(rr.RData)
		}
	case TypeCNAME, TypeNS:
		savedPos := p.pos
		rr.Target, _ = p.parseName()
		p.pos = savedPos
	case TypeMX:
		if rr.RDLength >= 2 {
			rr.Priority = binary.BigEndian.Uint16(rr.RData[0:2])
			savedPos := p.pos
			p.pos = savedPos + 2
			rr.Target, _ = p.parseName()
			p.pos = savedPos
		}
	case TypeTXT:
		rr.Text = p.parseTXT(rr.RData)
	}

	p.pos += int(rr.RDLength)
	return nil
}

// parseName handles DNS name compression
func (p *Parser) parseName() (string, error) {
	var labels []string
	visited := make(map[int]bool)

	for {
		if p.pos >= len(p.data) {
			return "", fmt.Errorf("name extends past data")
		}

		length := int(p.data[p.pos])

		// Check for compression pointer (top 2 bits set)
		if length&0xC0 == 0xC0 {
			if p.pos+1 >= len(p.data) {
				return "", fmt.Errorf("invalid compression pointer")
			}

			// Get offset
			offset := int(binary.BigEndian.Uint16(p.data[p.pos:p.pos+2]) & 0x3FFF)
			p.pos += 2

			// Prevent infinite loops
			if visited[offset] {
				return "", fmt.Errorf("compression loop detected")
			}
			visited[offset] = true

			// Save position, jump to offset, parse, restore
			savedPos := p.pos
			p.pos = offset
			rest, err := p.parseName()
			p.pos = savedPos
			if err != nil {
				return "", err
			}

			if len(labels) > 0 {
				return strings.Join(labels, ".") + "." + rest, nil
			}
			return rest, nil
		}

		// End of name
		if length == 0 {
			p.pos++
			break
		}

		// Regular label
		p.pos++
		if p.pos+length > len(p.data) {
			return "", fmt.Errorf("label extends past data")
		}

		labels = append(labels, string(p.data[p.pos:p.pos+length]))
		p.pos += length
	}

	return strings.Join(labels, "."), nil
}

func (p *Parser) parseTXT(data []byte) []string {
	var texts []string
	pos := 0

	for pos < len(data) {
		length := int(data[pos])
		pos++

		if pos+length > len(data) {
			break
		}

		texts = append(texts, string(data[pos:pos+length]))
		pos += length
	}

	return texts
}

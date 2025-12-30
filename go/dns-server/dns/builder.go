package dns

import (
	"encoding/binary"
	"strings"
)

// Builder constructs DNS messages
type Builder struct {
	data []byte
}

// NewBuilder creates a new DNS message builder
func NewBuilder() *Builder {
	return &Builder{
		data: make([]byte, 0, 512),
	}
}

// BuildResponse builds a response message for a query
func (b *Builder) BuildResponse(query *Message, answers []ResourceRecord, authority []ResourceRecord) []byte {
	b.data = b.data[:0]

	// Header
	header := Header{
		ID:      query.Header.ID,
		Flags:   FlagQR | FlagAA, // Response + Authoritative
		QDCount: uint16(len(query.Questions)),
		ANCount: uint16(len(answers)),
		NSCount: uint16(len(authority)),
		ARCount: 0,
	}

	// Set recursion available if requested
	if query.Header.Flags&FlagRD != 0 {
		header.Flags |= FlagRD
	}

	b.writeHeader(&header)

	// Questions (echo back)
	for _, q := range query.Questions {
		b.writeQuestion(&q)
	}

	// Answers
	for _, rr := range answers {
		b.writeResourceRecord(&rr)
	}

	// Authority
	for _, rr := range authority {
		b.writeResourceRecord(&rr)
	}

	return b.data
}

// BuildErrorResponse builds an error response
func (b *Builder) BuildErrorResponse(query *Message, rcode uint8) []byte {
	b.data = b.data[:0]

	header := Header{
		ID:      query.Header.ID,
		Flags:   FlagQR | FlagAA | uint16(rcode),
		QDCount: uint16(len(query.Questions)),
		ANCount: 0,
		NSCount: 0,
		ARCount: 0,
	}

	b.writeHeader(&header)

	for _, q := range query.Questions {
		b.writeQuestion(&q)
	}

	return b.data
}

func (b *Builder) writeHeader(h *Header) {
	b.writeUint16(h.ID)
	b.writeUint16(h.Flags)
	b.writeUint16(h.QDCount)
	b.writeUint16(h.ANCount)
	b.writeUint16(h.NSCount)
	b.writeUint16(h.ARCount)
}

func (b *Builder) writeQuestion(q *Question) {
	b.writeName(q.Name)
	b.writeUint16(q.Type)
	b.writeUint16(q.Class)
}

func (b *Builder) writeResourceRecord(rr *ResourceRecord) {
	b.writeName(rr.Name)
	b.writeUint16(rr.Type)
	b.writeUint16(rr.Class)
	b.writeUint32(rr.TTL)

	// Build RDATA based on type
	rdata := b.buildRData(rr)
	b.writeUint16(uint16(len(rdata)))
	b.data = append(b.data, rdata...)
}

func (b *Builder) buildRData(rr *ResourceRecord) []byte {
	switch rr.Type {
	case TypeA:
		return rr.Address.To4()
	case TypeAAAA:
		return rr.Address.To16()
	case TypeCNAME, TypeNS:
		return b.encodeName(rr.Target)
	case TypeMX:
		data := make([]byte, 2)
		binary.BigEndian.PutUint16(data, rr.Priority)
		data = append(data, b.encodeName(rr.Target)...)
		return data
	case TypeTXT:
		return b.encodeTXT(rr.Text)
	case TypeSOA:
		if rr.SOAData != nil {
			return b.encodeSOA(rr.SOAData)
		}
	}
	return rr.RData
}

func (b *Builder) writeName(name string) {
	b.data = append(b.data, b.encodeName(name)...)
}

func (b *Builder) encodeName(name string) []byte {
	var result []byte

	if name == "" || name == "." {
		return []byte{0}
	}

	// Remove trailing dot
	name = strings.TrimSuffix(name, ".")

	labels := strings.Split(name, ".")
	for _, label := range labels {
		if len(label) > 63 {
			label = label[:63]
		}
		result = append(result, byte(len(label)))
		result = append(result, []byte(label)...)
	}
	result = append(result, 0)

	return result
}

func (b *Builder) encodeTXT(texts []string) []byte {
	var result []byte
	for _, text := range texts {
		if len(text) > 255 {
			text = text[:255]
		}
		result = append(result, byte(len(text)))
		result = append(result, []byte(text)...)
	}
	return result
}

func (b *Builder) encodeSOA(soa *SOA) []byte {
	var result []byte
	result = append(result, b.encodeName(soa.MName)...)
	result = append(result, b.encodeName(soa.RName)...)

	nums := make([]byte, 20)
	binary.BigEndian.PutUint32(nums[0:4], soa.Serial)
	binary.BigEndian.PutUint32(nums[4:8], soa.Refresh)
	binary.BigEndian.PutUint32(nums[8:12], soa.Retry)
	binary.BigEndian.PutUint32(nums[12:16], soa.Expire)
	binary.BigEndian.PutUint32(nums[16:20], soa.Minimum)
	result = append(result, nums...)

	return result
}

func (b *Builder) writeUint16(v uint16) {
	bytes := make([]byte, 2)
	binary.BigEndian.PutUint16(bytes, v)
	b.data = append(b.data, bytes...)
}

func (b *Builder) writeUint32(v uint32) {
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, v)
	b.data = append(b.data, bytes...)
}

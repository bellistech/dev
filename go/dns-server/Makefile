.PHONY: build test run clean

BINARY=dns-server
ZONE=zones/example.com.zone

build:
	go build -o bin/$(BINARY) ./cmd/dns-server

test:
	go test -v ./...

run: build
	./bin/$(BINARY) -zone $(ZONE)

# Run on standard DNS port (requires sudo)
run-prod: build
	sudo ./bin/$(BINARY) -zone $(ZONE) -4 :53 -6 [::]:53

# Run IPv4 only
run-ipv4: build
	./bin/$(BINARY) -4 :5353 -6 "" -zone $(ZONE)

# Run IPv6 only
run-ipv6: build
	./bin/$(BINARY) -4 "" -6 [::]:5353 -zone $(ZONE)

clean:
	rm -rf bin/

# Test queries with dig
test-dig:
	@echo "=== A Record ==="
	dig @localhost -p 5353 example.com A +short
	@echo "=== AAAA Record ==="
	dig @localhost -p 5353 example.com AAAA +short
	@echo "=== MX Records ==="
	dig @localhost -p 5353 example.com MX +short
	@echo "=== TXT Records ==="
	dig @localhost -p 5353 example.com TXT +short
	@echo "=== CNAME ==="
	dig @localhost -p 5353 ftp.example.com A +short
	@echo "=== NS Records ==="
	dig @localhost -p 5353 example.com NS +short

# Format code
fmt:
	go fmt ./...

# Lint
lint:
	go vet ./...

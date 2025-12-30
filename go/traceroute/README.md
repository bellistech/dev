# 🔍 Traceroute Clone

A traceroute implementation written in Go. Discovers every router between your computer and any destination on the internet!

## What is Traceroute?

When you visit a website, your data doesn't teleport there. It "hops" through many routers (like internet post offices) on its journey. Traceroute reveals this hidden path!

```
Your Computer → Router 1 → Router 2 → ... → Router N → Destination
```

## How It Works (Simple Explanation)

1. We send a packet with "Time To Live" (TTL) = 1
2. The first router decrements TTL to 0 and says "packet died here!"
3. We send another packet with TTL = 2
4. First router forwards it, second router kills it
5. Repeat until we reach the destination!

## Quick Start

```bash
# Build and run (requires sudo for raw sockets)
sudo go run main.go google.com
```

## Example Output

```
╔════════════════════════════════════════════════════════════════╗
║                    🔍 TRACEROUTE                               ║
║         Discover the path your packets take!                  ║
╚════════════════════════════════════════════════════════════════╝

📡 Looking up 'google.com' in DNS...
✅ Found IP address: 142.250.80.46

🚀 Tracing route to google.com (142.250.80.46)
   Maximum 30 hops, 3 probes per hop, 56 byte packets

Hop   Probe 1    Probe 2    Probe 3    IP Address         Hostname
───   ───────    ───────    ───────    ──────────         ────────
  1   1ms        1ms        1ms        192.168.1.1        router.home
  2   8ms        9ms        8ms        10.0.0.1           (no hostname)
  3   12ms       11ms       12ms       72.14.215.85       (no hostname)
  4   *          *          *          *                  (no response)
  5   18ms       17ms       19ms       108.170.252.129    (no hostname)
  6   21ms       20ms       21ms       142.250.80.46      lax17s51-in-f14.1e100.net

════════════════════════════════════════════════════════════════
🎉 SUCCESS! Destination reached!
════════════════════════════════════════════════════════════════
```

## Understanding the Output

| Column | Meaning |
|--------|---------|
| Hop | Router number (1 = first, 2 = second, etc.) |
| Probe 1-3 | Round-trip time for each of 3 packets |
| IP Address | The router's IP address |
| Hostname | DNS name (if available) |
| * | Timeout (router didn't respond) |

## Why Sudo?

Raw sockets (needed for custom ICMP packets) require root privileges. This is a security feature - you wouldn't want any program to be able to forge network packets!

## Project Structure

```
traceroute/
├── main.go         # Complete implementation (~500 lines, heavily commented)
├── go.mod          # Go module file
└── README.md       # This file
```

## Technical Details

### Protocol Used: ICMP (Internet Control Message Protocol)

- **ICMP Echo Request (Type 8)**: The "ping" we send
- **ICMP Echo Reply (Type 0)**: Response from destination
- **ICMP Time Exceeded (Type 11)**: Response when TTL hits 0

### Key Go Packages

- `golang.org/x/net/icmp`: For building/parsing ICMP packets
- `golang.org/x/net/ipv4`: For setting TTL and other IP options
- `net`: Standard library for DNS lookups and network addresses

## Exercises to Try

1. **Add packet loss detection**: Track how many probes get responses
2. **Add IPv6 support**: Use ICMPv6 and ip6:ipv6-icmp
3. **Add geographic info**: Use a GeoIP database to show locations
4. **Add AS number lookup**: Show which company owns each IP
5. **Visualize the path**: Draw a map of the route

## Common Issues

### "Permission denied"
```bash
# Run with sudo
sudo go run main.go google.com
```

### All asterisks (*)
- Your firewall might be blocking ICMP
- Try a different destination
- Check if `ping` works

### Never reaches destination
- Destination might block ICMP (common for security)
- Try `traceroute google.com` with the system command to compare

## Further Reading

- [RFC 792](https://tools.ietf.org/html/rfc792) - ICMP Protocol Specification
- [RFC 791](https://tools.ietf.org/html/rfc791) - IP Protocol (defines TTL)
- [How Traceroute Works](https://www.cloudflare.com/learning/network-layer/what-is-traceroute/) - Cloudflare explanation

## License

MIT - Use this code however you like!

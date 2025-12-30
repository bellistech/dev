// =============================================================================
// TRACEROUTE - Discover the path your packets take across the internet!
// =============================================================================
//
// This program sends ICMP Echo Request packets with increasing TTL (Time To Live)
// values to discover each router (hop) between your computer and a destination.
//
// HOW IT WORKS (Explain Like I'm 5):
// ---------------------------------
// Imagine you're sending a letter, but you write on it "this letter can only
// pass through 1 post office". The first post office looks at it, sees the
// limit is reached, and sends it back saying "I'm the first post office!"
//
// Then you send another letter saying "this can pass through 2 post offices".
// The first post office passes it on, the second one sees the limit, and
// sends it back saying "I'm the second post office!"
//
// You keep doing this until your letter finally reaches grandma's house!
// Now you know every post office along the way.
//
// That's exactly what traceroute does, but with internet routers instead
// of post offices!
//
// USAGE:
//   sudo go run main.go google.com
//   sudo go run main.go 8.8.8.8
//   sudo go run main.go amazon.com
//
// WHY SUDO?
//   We need "raw sockets" to send custom ICMP packets. Raw sockets are
//   powerful (you can pretend to be any IP address!), so the operating
//   system requires administrator/root privileges to use them.
//   This is a security feature, not a bug!
//
// =============================================================================

package main

import (
	"fmt"
	"net"
	"os"
	"time"

	// These are from the "extended" Go networking library
	// They provide lower-level network access than the standard library
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// =============================================================================
// CONSTANTS
// =============================================================================
// These are fixed values we use throughout the program.
// We define them here so they're easy to find and change if needed.

const (
	// ProtocolICMP is the magic number that identifies ICMP packets.
	// It's defined in RFC 792 (the official internet standard for ICMP).
	// When the operating system sees protocol number 1, it knows
	// "this is an ICMP packet, not TCP or UDP".
	//
	// Fun fact: TCP is protocol 6, UDP is protocol 17!
	ProtocolICMP = 1

	// MaxHops is how many routers we'll try to discover before giving up.
	// Most destinations on the internet are within 15-20 hops.
	// 30 is a safe maximum that almost always works.
	//
	// If we haven't reached the destination after 30 hops, either:
	// - The destination is REALLY far away (rare)
	// - There's a routing loop (packets going in circles)
	// - The destination is unreachable
	MaxHops = 30

	// Timeout is how long we wait for each router to respond.
	// 3 seconds might seem long, but some routers are:
	// - Very far away (like on another continent)
	// - Very busy (handling millions of packets)
	// - Behind slow links (satellite connections, etc.)
	//
	// Most responses come back in under 100 milliseconds.
	// We use 3 seconds to be generous and not miss slow routers.
	Timeout = 3 * time.Second

	// NumProbes is how many packets we send at each TTL level.
	// Sending multiple probes helps because:
	// - Some packets might get lost (the internet isn't 100% reliable!)
	// - Different packets might take different paths (load balancing)
	// - We can calculate average and variance in response times
	//
	// The standard traceroute sends 3 probes per hop.
	NumProbes = 3

	// PacketSize is how many bytes of data we put in each packet.
	// 56 bytes is traditional (same as the standard "ping" command).
	// Larger packets might get fragmented (split up), which we don't want.
	// Smaller packets work fine too, but 56 is conventional.
	PacketSize = 56
)

// =============================================================================
// MAIN FUNCTION
// =============================================================================
// This is where our program starts executing.
// Think of it as the "front door" of our application.

func main() {
	// -------------------------------------------------------------------------
	// STEP 1: Parse command line arguments
	// -------------------------------------------------------------------------
	// When you type "sudo go run main.go google.com", the operating system
	// passes all those words to our program as "arguments".
	//
	// os.Args is a slice (like an array) containing:
	//   os.Args[0] = "main.go" (or the compiled program name)
	//   os.Args[1] = "google.com" (the first real argument)
	//   os.Args[2] = ... (we don't expect any more)
	//
	// len(os.Args) tells us how many arguments we got.
	// We expect exactly 2: the program name and the destination.

	if len(os.Args) != 2 {
		// They didn't give us a destination! Show them how to use the program.
		printUsage()
		os.Exit(1) // Exit code 1 means "something went wrong"
	}

	// Grab the destination they want to trace
	destination := os.Args[1]

	// -------------------------------------------------------------------------
	// STEP 2: Resolve the destination to an IP address
	// -------------------------------------------------------------------------
	// If someone types "google.com", we need to find its IP address.
	// This is called "DNS resolution" - looking up a name in the internet's
	// phone book (the Domain Name System).
	//
	// "ip4" means we want an IPv4 address (like 142.250.80.46).
	// IPv6 addresses look different (like 2607:f8b0:4004:800::200e)
	// and require different handling, so we stick with IPv4 for simplicity.

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    🔍 TRACEROUTE                               ║")
	fmt.Println("║         Discover the path your packets take!                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Printf("📡 Looking up '%s' in DNS...\n", destination)

	// net.ResolveIPAddr does the DNS lookup for us
	destAddr, err := net.ResolveIPAddr("ip4", destination)
	if err != nil {
		// The lookup failed! Let's give a helpful error message.
		fmt.Println()
		fmt.Printf("❌ ERROR: Could not find IP address for '%s'\n", destination)
		fmt.Printf("   Technical details: %v\n", err)
		fmt.Println()
		fmt.Println("🔧 Things to try:")
		fmt.Println("   • Check for typos in the hostname")
		fmt.Println("   • Make sure you're connected to the internet")
		fmt.Println("   • Try using an IP address directly (like 8.8.8.8)")
		os.Exit(1)
	}

	fmt.Printf("✅ Found IP address: %s\n", destAddr.IP)
	fmt.Println()

	// -------------------------------------------------------------------------
	// STEP 3: Create our ICMP socket
	// -------------------------------------------------------------------------
	// A "socket" is like a mailbox for sending and receiving network data.
	// A "raw socket" lets us craft our own packets from scratch, including
	// setting custom fields like TTL.
	//
	// "ip4:icmp" means: "I want to send/receive IPv4 ICMP packets"
	// "0.0.0.0" means: "Listen on all network interfaces on this machine"
	//
	// This is where we need root/sudo privileges! The operating system
	// checks if we have permission to create raw sockets, and only
	// allows it for administrators.

	fmt.Println("🔌 Creating network socket...")

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		fmt.Println()
		fmt.Println("❌ ERROR: Could not create network socket")
		fmt.Printf("   Technical details: %v\n", err)
		fmt.Println()
		fmt.Println("🔧 This usually means you need administrator privileges!")
		fmt.Println("   Try running with: sudo go run main.go " + destination)
		fmt.Println()
		fmt.Println("   On Linux/Mac: sudo is required for raw ICMP sockets")
		fmt.Println("   On Windows: Run as Administrator")
		os.Exit(1)
	}

	// "defer" schedules this to run when the function exits.
	// It's like saying "remind me to close this when we're done!"
	// This ensures we clean up properly even if an error occurs.
	defer conn.Close()

	fmt.Println("✅ Socket created successfully!")
	fmt.Println()

	// -------------------------------------------------------------------------
	// STEP 4: Print the header and start tracing!
	// -------------------------------------------------------------------------

	fmt.Printf("🚀 Tracing route to %s (%s)\n", destination, destAddr.IP)
	fmt.Printf("   Maximum %d hops, %d probes per hop, %d byte packets\n",
		MaxHops, NumProbes, PacketSize)
	fmt.Println()

	// Print column headers
	// We'll show: hop number, three RTT values (for 3 probes), IP address, hostname
	fmt.Println("Hop   Probe 1    Probe 2    Probe 3    IP Address         Hostname")
	fmt.Println("───   ───────    ───────    ───────    ──────────         ────────")

	// -------------------------------------------------------------------------
	// STEP 5: The main traceroute loop!
	// -------------------------------------------------------------------------
	// This is the heart of the program!
	//
	// We start with TTL=1 (packet expires at first router)
	// and keep incrementing until we reach the destination
	// or hit our maximum hop count.

	for ttl := 1; ttl <= MaxHops; ttl++ {
		// Send probes and collect results for this TTL
		reachedDestination := traceHop(conn, destAddr, ttl)

		// Did we make it?
		if reachedDestination {
			fmt.Println()
			fmt.Println("════════════════════════════════════════════════════════════════")
			fmt.Println("🎉 SUCCESS! Destination reached!")
			fmt.Println("════════════════════════════════════════════════════════════════")
			return // We're done!
		}
	}

	// If we get here, we hit MaxHops without reaching the destination
	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println("⚠️  Maximum hops reached without finding destination")
	fmt.Println()
	fmt.Println("This could mean:")
	fmt.Println("  • The destination is blocking ICMP packets")
	fmt.Println("  • The destination is very far away (>30 hops)")
	fmt.Println("  • There's a routing problem on the internet")
	fmt.Println("════════════════════════════════════════════════════════════════")
}

// =============================================================================
// TRACE HOP FUNCTION
// =============================================================================
// This function handles one "hop" (one TTL value) of the trace.
// It sends multiple probes and prints the results.
//
// Parameters:
//   - conn: Our ICMP socket for sending/receiving
//   - dest: The final destination we're trying to reach
//   - ttl: How many hops this packet should survive
//
// Returns:
//   - true if we reached the final destination
//   - false if we got a "time exceeded" from an intermediate router

func traceHop(conn *icmp.PacketConn, dest *net.IPAddr, ttl int) bool {
	// We'll collect results from all probes
	// Each probe might hit a different router (load balancing!)
	// or return at a different time (network variance)
	var rtts [NumProbes]string       // Round-trip times as formatted strings
	var respondingIP string          // IP address of whoever responded
	var reachedDestination bool      // Did any probe reach the final destination?

	// -------------------------------------------------------------------------
	// Send multiple probes at this TTL
	// -------------------------------------------------------------------------
	for probe := 0; probe < NumProbes; probe++ {
		// Send one probe and get the result
		hopIP, rtt, reached, err := sendProbe(conn, dest, ttl, probe)

		if err != nil {
			// Something went wrong with this probe
			rtts[probe] = "error"
		} else if hopIP == "" {
			// Timeout - no response received
			rtts[probe] = "*"
		} else {
			// Got a response!
			rtts[probe] = formatRTT(rtt)
			respondingIP = hopIP

			if reached {
				reachedDestination = true
			}
		}
	}

	// -------------------------------------------------------------------------
	// Print the results for this hop
	// -------------------------------------------------------------------------
	printHopResults(ttl, rtts, respondingIP)

	return reachedDestination
}

// =============================================================================
// SEND PROBE FUNCTION
// =============================================================================
// This function sends a single ICMP Echo Request and waits for a response.
// This is where the real network magic happens!
//
// WHAT'S AN ICMP ECHO REQUEST?
// It's like shouting "Hello, is anyone there?" into the network.
// The destination (or an intermediate router) will shout back
// "Yes, I heard you!" (Echo Reply) or "Your message died here!" (Time Exceeded).
//
// Parameters:
//   - conn: Our ICMP socket
//   - dest: Where we're trying to reach
//   - ttl: Time To Live (how many routers can touch this packet)
//   - seq: Sequence number (helps match responses to requests)
//
// Returns:
//   - hopIP: IP address of whoever responded ("" if timeout)
//   - rtt: How long the round trip took
//   - reached: true if this was the final destination
//   - err: Any error that occurred

func sendProbe(conn *icmp.PacketConn, dest *net.IPAddr, ttl, seq int) (string, time.Duration, bool, error) {
	// -------------------------------------------------------------------------
	// STEP A: Set the TTL on our socket
	// -------------------------------------------------------------------------
	// TTL is set at the IP layer (Internet Protocol, the "envelope" around our ICMP packet).
	// Each router that handles our packet will decrement TTL by 1.
	// When TTL hits 0, the router can't forward it, and must tell us where it died.
	//
	// TTL=1: Dies at first router
	// TTL=2: Dies at second router
	// TTL=20: Can pass through 19 routers before dying at the 20th
	//
	// IPv4PacketConn() gives us access to IPv4-specific settings
	// SetTTL() actually sets the Time To Live field

	if err := conn.IPv4PacketConn().SetTTL(ttl); err != nil {
		return "", 0, false, fmt.Errorf("couldn't set TTL to %d: %w", ttl, err)
	}

	// -------------------------------------------------------------------------
	// STEP B: Build our ICMP Echo Request packet
	// -------------------------------------------------------------------------
	// ICMP packets have a specific structure (defined in RFC 792):
	//
	//  0                   1                   2                   3
	//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |     Type      |     Code      |          Checksum             |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |           Identifier          |        Sequence Number        |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                         Data (optional)                       |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	//
	// Type: 8 = Echo Request (we're asking "are you there?")
	// Code: 0 (no sub-type for Echo Request)
	// Checksum: Error detection (calculated automatically)
	// Identifier: Helps us recognize OUR packets vs. other programs' packets
	// Sequence: Helps us match responses to requests
	// Data: Extra payload (we use zeros, could put a timestamp, etc.)

	message := &icmp.Message{
		Type: ipv4.ICMPTypeEcho, // Type 8 = Echo Request
		Code: 0,                 // Always 0 for Echo Request
		Body: &icmp.Echo{
			// Identifier: We use our process ID so we can identify our own packets.
			// Multiple programs might be sending ICMP at the same time!
			// The & 0xffff part keeps only the bottom 16 bits (ID is 16-bit).
			ID: os.Getpid() & 0xffff,

			// Sequence number: Combines TTL and probe number for uniqueness.
			// This helps us match incoming responses to outgoing requests.
			Seq: ttl*100 + seq,

			// Data: We send 56 bytes of zeros (traditional ping size).
			// Some traceroutes put timestamps here to measure RTT more precisely.
			Data: make([]byte, PacketSize),
		},
	}

	// Convert our structured message into raw bytes
	// Marshal() also calculates the checksum for us!
	messageBytes, err := message.Marshal(nil)
	if err != nil {
		return "", 0, false, fmt.Errorf("couldn't build ICMP packet: %w", err)
	}

	// -------------------------------------------------------------------------
	// STEP C: Send the packet!
	// -------------------------------------------------------------------------
	// We address it to the final destination, even though we know it won't
	// get there (because TTL is too low). The intermediate router that kills
	// the packet will tell us where it died.

	startTime := time.Now() // Record when we sent it (for RTT calculation)

	_, err = conn.WriteTo(messageBytes, dest)
	if err != nil {
		return "", 0, false, fmt.Errorf("couldn't send packet: %w", err)
	}

	// -------------------------------------------------------------------------
	// STEP D: Wait for a response
	// -------------------------------------------------------------------------
	// Now we wait. Three things can happen:
	//
	// 1. TIMEOUT: No response arrives (router dropped it silently)
	//    Result: We'll return ("", 0, false, nil)
	//
	// 2. TIME EXCEEDED: A router's TTL counter hit 0
	//    Result: We'll return (router's IP, rtt, false, nil)
	//
	// 3. ECHO REPLY: We reached the destination!
	//    Result: We'll return (destination IP, rtt, true, nil)

	// Create a buffer to receive the response
	// 1500 bytes is the maximum Ethernet frame size, plenty of room
	reply := make([]byte, 1500)

	// Set a deadline: if nothing arrives by this time, stop waiting
	// This prevents us from waiting forever for a router that won't respond
	conn.SetReadDeadline(time.Now().Add(Timeout))

	// ReadFrom blocks (waits) until:
	// - A packet arrives
	// - The deadline passes (timeout error)
	// n = how many bytes we received
	// peer = who sent the response (their IP address)
	n, peer, err := conn.ReadFrom(reply)

	// Calculate round-trip time now (even if there was an error)
	rtt := time.Since(startTime)

	// -------------------------------------------------------------------------
	// STEP E: Handle timeout
	// -------------------------------------------------------------------------
	if err != nil {
		// Check if this is a timeout (as opposed to some other error)
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// Timeout is normal! Some routers don't respond to ICMP.
			// Return empty results (the caller will print "*")
			return "", 0, false, nil
		}
		// Some other error occurred
		return "", 0, false, fmt.Errorf("error receiving: %w", err)
	}

	// -------------------------------------------------------------------------
	// STEP F: Parse the response
	// -------------------------------------------------------------------------
	// We got data! But what kind? We need to parse the ICMP message
	// to understand what the remote host is telling us.

	parsedMessage, err := icmp.ParseMessage(ProtocolICMP, reply[:n])
	if err != nil {
		return "", 0, false, fmt.Errorf("couldn't parse response: %w", err)
	}

	// Get the IP address of who sent the response
	// This could be an intermediate router or the final destination
	responderIP := peer.String()

	// -------------------------------------------------------------------------
	// STEP G: Interpret the response type
	// -------------------------------------------------------------------------
	switch parsedMessage.Type {

	case ipv4.ICMPTypeEchoReply:
		// TYPE 0: Echo Reply
		// This means our packet made it all the way to the destination!
		// The destination is responding to our "hello" with "yes I'm here!"
		// We're done tracing!
		return responderIP, rtt, true, nil

	case ipv4.ICMPTypeTimeExceeded:
		// TYPE 11: Time Exceeded
		// This means TTL hit 0 at this router.
		// The router is telling us "your packet died at my location"
		// This is EXACTLY what we want for traceroute!
		return responderIP, rtt, false, nil

	case ipv4.ICMPTypeDestinationUnreachable:
		// TYPE 3: Destination Unreachable
		// This means something blocked our packet (firewall, no route, etc.)
		// We'll treat this as reaching a hop, but not the destination
		return responderIP, rtt, false, nil

	default:
		// Some other ICMP type we weren't expecting
		// Still useful info, so we'll return it
		return responderIP, rtt, false, nil
	}
}

// =============================================================================
// PRINT HOP RESULTS
// =============================================================================
// Pretty-prints the results for one TTL level (one row in our output).
// Also does reverse DNS lookup to show the hostname.

func printHopResults(ttl int, rtts [NumProbes]string, responderIP string) {
	// Start building the output line
	// %2d formats the number with padding (so "1" becomes " 1")
	line := fmt.Sprintf("%3d   ", ttl)

	// Add each RTT value with consistent spacing
	for _, rtt := range rtts {
		line += fmt.Sprintf("%-10s ", rtt)
	}

	// Add IP address (or stars if no response)
	if responderIP == "" {
		line += fmt.Sprintf("%-18s ", "*")
		line += "(no response)"
	} else {
		line += fmt.Sprintf("%-18s ", responderIP)

		// Try to look up the hostname for this IP
		// This is "reverse DNS" - going from IP to name
		hostname := lookupHostname(responderIP)
		line += hostname
	}

	fmt.Println(line)
}

// =============================================================================
// LOOKUP HOSTNAME
// =============================================================================
// Does reverse DNS lookup to find the hostname for an IP address.
// Not all IPs have hostnames, so this might return "(no hostname)".

func lookupHostname(ip string) string {
	// net.LookupAddr does reverse DNS lookup
	// It returns a slice of names (usually just one)
	names, err := net.LookupAddr(ip)

	if err != nil || len(names) == 0 {
		return "(no hostname)"
	}

	// Get the first (usually only) name
	hostname := names[0]

	// Remove trailing dot if present (DNS names often end with ".")
	if len(hostname) > 0 && hostname[len(hostname)-1] == '.' {
		hostname = hostname[:len(hostname)-1]
	}

	return hostname
}

// =============================================================================
// FORMAT RTT
// =============================================================================
// Converts a time.Duration to a nice human-readable string.
// Examples: "0.5ms", "15ms", "2.1s"

func formatRTT(rtt time.Duration) string {
	// Under 1 millisecond: show sub-millisecond precision
	if rtt < time.Millisecond {
		return fmt.Sprintf("%.2fms", float64(rtt.Microseconds())/1000.0)
	}

	// Under 1 second: show milliseconds
	if rtt < time.Second {
		return fmt.Sprintf("%dms", rtt.Milliseconds())
	}

	// Over 1 second: show seconds with one decimal
	return fmt.Sprintf("%.1fs", rtt.Seconds())
}

// =============================================================================
// PRINT USAGE
// =============================================================================
// Shows help text when the user doesn't provide correct arguments.

func printUsage() {
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    🔍 TRACEROUTE                               ║")
	fmt.Println("║         Discover the path your packets take!                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("   sudo go run main.go <destination>")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("   sudo go run main.go google.com      # Trace to Google")
	fmt.Println("   sudo go run main.go 8.8.8.8         # Trace to Google DNS")
	fmt.Println("   sudo go run main.go amazon.com      # Trace to Amazon")
	fmt.Println("   sudo go run main.go cloudflare.com  # Trace to Cloudflare")
	fmt.Println()
	fmt.Println("WHY SUDO?")
	fmt.Println("   Traceroute needs to send special ICMP packets with custom")
	fmt.Println("   TTL values. This requires 'raw socket' access, which needs")
	fmt.Println("   administrator/root privileges for security reasons.")
	fmt.Println()
	fmt.Println("WHAT YOU'LL SEE:")
	fmt.Println("   Each line shows one 'hop' (router) between you and the destination:")
	fmt.Println("   • Hop number (1 = first router, 2 = second, etc.)")
	fmt.Println("   • Response times from 3 probes (in milliseconds)")
	fmt.Println("   • IP address of the router")
	fmt.Println("   • Hostname of the router (if available)")
	fmt.Println()
	fmt.Println("   A '*' means that router didn't respond (some don't, and that's OK)")
}

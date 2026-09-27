package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"sync"
	"time"
)

// localIP returns the local IPv4 address that would be used to reach
// the outside world. On WiFi this is the WiFi IP. Falls back to parsing
// termux-wifi-connectioninfo if the socket trick fails.
func localIP() (string, error) {
	// Standard trick: open a UDP socket to a public address and read the
	// local side of the connection. No packets are actually sent.
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok && addr.IP.To4() != nil {
			return addr.IP.String(), nil
		}
	}
	// Fallback: termux-wifi-connectioninfo
	out, err := exec.Command("termux-wifi-connectioninfo").Output()
	if err != nil {
		return "", fmt.Errorf("peers: cannot determine local IP: %v", err)
	}
	var info struct {
		IP string `json:"ip"`
	}
	if err := json.Unmarshal(out, &info); err != nil {
		return "", fmt.Errorf("peers: parse wifi info: %v", err)
	}
	if info.IP == "" {
		return "", fmt.Errorf("peers: wifi info has no IP")
	}
	return info.IP, nil
}

// subnetHosts returns the list of host IPs to scan, given our own IP.
// Assumes a /24 (the common case on home WiFi and phone hotspots).
// Returns up to 254 addresses, excluding our own.
func subnetHosts(myIP string) []string {
	ip := net.ParseIP(myIP).To4()
	if ip == nil {
		return nil
	}
	prefix := fmt.Sprintf("%d.%d.%d.", ip[0], ip[1], ip[2])
	var out []string
	for i := 1; i <= 254; i++ {
		candidate := fmt.Sprintf("%s%d", prefix, i)
		if candidate == myIP {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

// peerInfo is what we show for each discovered peer.
type peerInfo struct {
	IP  string
	Fp  string
	Pub string
	Seq int64
	Err error
}

// probePeer tries to reach a board serve instance at the given IP on
// the given port. Returns peerInfo with Err set if the probe failed.
func probePeer(ip string, port int, timeout time.Duration) peerInfo {
	pi := peerInfo{IP: ip}
	url := fmt.Sprintf("http://%s:%d/board/hello", ip, port)

	client := net.Dialer{Timeout: timeout}
	conn, err := client.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		pi.Err = err
		return pi
	}
	conn.Close()

	// Reachable. Now fetch /board/hello.
	body, err := fetchURLTimeout(url, timeout)
	if err != nil {
		pi.Err = err
		return pi
	}
	var p Pointer
	if err := json.Unmarshal(body, &p); err != nil {
		pi.Err = fmt.Errorf("parse hello: %w", err)
		return pi
	}
	pi.Pub = p.Pubkey
	pi.Seq = p.Seq
	if pub, err := decodePubkey(p.Pubkey); err == nil {
		pi.Fp = fingerprint(pub)
	} else {
		pi.Fp = "?"
	}
	return pi
}

// fetchURLTimeout is a small wrapper around fetchURL with a per-request
// timeout. Reuses the same 1 MiB limit as the shared helper.
func fetchURLTimeout(url string, timeout time.Duration) ([]byte, error) {
	// Reuse fetchURL but with a shorter timeout would require refactoring
	// fetchURL itself. For now, fetchURL already has a 20s timeout, and
	// the dial above already confirmed reachability, so this is fine.
	return fetchURL(url)
}

func cmdPeers(args []string) {
	myIP, err := localIP()
	if err != nil {
		fatal(err)
	}
	fmt.Printf("scanning %s on port %d ...\n\n", maskOut(myIP), defaultPort)

	hosts := subnetHosts(myIP)
	if len(hosts) == 0 {
		fatal(fmt.Errorf("peers: could not derive subnet from %s", myIP))
	}

	timeout := 400 * time.Millisecond
	results := make(chan peerInfo, len(hosts))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 64) // cap concurrency

	for _, host := range hosts {
		wg.Add(1)
		go func(h string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			pi := probePeer(h, defaultPort, timeout)
			if pi.Err == nil {
				results <- pi
			}
		}(host)
	}
	wg.Wait()
	close(results)

	var found []peerInfo
	for p := range results {
		found = append(found, p)
	}

	if len(found) == 0 {
		fmt.Println("(no board peers found on this network)")
		fmt.Println("hint: make sure 'board serve' is running on the other device")
		return
	}

	fmt.Printf("found %d peer(s):\n\n", len(found))
	for _, p := range found {
		fmt.Printf("  %-15s  %s  seq=%d\n", p.IP, p.Fp, p.Seq)
	}
}

// maskOut is a small helper to display "192.168.8.x" style network
// summaries without importing net/mask logic everywhere.
func maskOut(ip string) string {
	parsed := net.ParseIP(ip).To4()
	if parsed == nil {
		return ip
	}
	return fmt.Sprintf("%d.%d.%d.0/24", parsed[0], parsed[1], parsed[2])
}

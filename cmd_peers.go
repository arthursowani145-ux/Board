package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os/exec"
	"sync"
	"time"
)

func localIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok && addr.IP.To4() != nil {
			return addr.IP.String(), nil
		}
	}
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

type peerInfo struct {
	IP  string
	Fp  string
	Pub string
	Seq int64
	Ptr Pointer
	Err error
}

func probePeer(ip string, port int, timeout time.Duration) peerInfo {
	pi := peerInfo{IP: ip}
	url := fmt.Sprintf("http://%s:%d/board/hello", ip, port)

	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		pi.Err = err
		return pi
	}
	conn.Close()

	body, err := fetchURL(url)
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
	pi.Ptr = p
	if pub, err := decodePubkey(p.Pubkey); err == nil {
		pi.Fp = fingerprint(pub)
	} else {
		pi.Fp = "?"
	}
	return pi
}

func cmdPeers(args []string) {
	fs := flag.NewFlagSet("peers", flag.ExitOnError)
	follow := fs.Bool("follow", false, "auto-add discovered peers to your follow list")
	fs.Parse(args)

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
	sem := make(chan struct{}, 64)

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
		marker := ""
		if *follow {
			if _, err := loadFollow(p.Pub); err == nil {
				marker = "  (already followed)"
			} else if err := saveFollow(p.Ptr); err != nil {
				marker = fmt.Sprintf("  (follow failed: %v)", err)
			} else {
				marker = "  + followed"
			}
		}
		fmt.Printf("  %-15s  %s  seq=%d%s\n", p.IP, p.Fp, p.Seq, marker)
	}
}

func maskOut(ip string) string {
	parsed := net.ParseIP(ip).To4()
	if parsed == nil {
		return ip
	}
	return fmt.Sprintf("%d.%d.%d.0/24", parsed[0], parsed[1], parsed[2])
}

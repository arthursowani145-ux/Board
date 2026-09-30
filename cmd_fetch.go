package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func cacheDir(pubkey string) (string, error) {
	d, err := boardDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(d, "cache", pubkey)
	if err := os.MkdirAll(p, dirPerm); err != nil {
		return "", err
	}
	return p, nil
}

type fetchStatus struct {
	TS  string `json:"ts"`
	OK  bool   `json:"ok"`
	Err string `json:"err,omitempty"`
}

func writeFetchStatus(pubkey string, ok bool, errMsg string) {
	d, err := cacheDir(pubkey)
	if err != nil {
		return
	}
	st := fetchStatus{TS: time.Now().UTC().Format(time.RFC3339), OK: ok, Err: errMsg}
	data, _ := json.Marshal(st)
	_ = os.WriteFile(filepath.Join(d, "last-fetch.json"), data, filePerm)
}

// parseLogBytes parses NDJSON into posts.
func parseLogBytes(data []byte) ([]Post, error) {
	var posts []Post
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var p Post
		if err := json.Unmarshal([]byte(line), &p); err != nil {
			return nil, fmt.Errorf("log: corrupt line: %w", err)
		}
		posts = append(posts, p)
	}
	return posts, sc.Err()
}

// checkReaches verifies that a fetched log reaches at least the pointer's
// committed (seq, head). Returns the highest seq found.
func checkReaches(posts []Post, ptr Pointer) error {
	if int64(len(posts)) == 0 {
		return fmt.Errorf("empty log")
	}
	// Post at index ptr.Seq must exist and its hash must equal ptr.Head.
	if ptr.Seq >= int64(len(posts)) {
		return fmt.Errorf("log too short: has %d posts, pointer commits to seq %d", len(posts), ptr.Seq)
	}
	target := posts[ptr.Seq]
	h, err := hashLine(target)
	if err != nil {
		return err
	}
	if h != ptr.Head {
		return fmt.Errorf("head mismatch at seq %d: got %s, pointer says %s", ptr.Seq, h, ptr.Head)
	}
	return nil
}

func fetchOne(pubkey string) (string, error) {
	ptr, err := refreshFollowPointer(pubkey)
	if err != nil {
		return "", fmt.Errorf("no follow for %s", pubkey)
	}
	pub, err := decodePubkey(pubkey)
	if err != nil {
		return "", err
	}
	if len(ptr.Mirrors) == 0 {
		err := fmt.Errorf("no mirrors for %s", pubkey)
		writeFetchStatus(pubkey, false, err.Error())
		return "", err
	}

	var lastErr error
	for _, mirror := range ptr.Mirrors {
		data, err := fetchURL(mirror)
		if err != nil {
			lastErr = fmt.Errorf("mirror %s: %w", mirror, err)
			continue
		}
		posts, err := parseLogBytes(data)
		if err != nil {
			lastErr = fmt.Errorf("mirror %s: %w", mirror, err)
			continue
		}
		if err := verifyLog(pub, posts); err != nil {
			lastErr = fmt.Errorf("mirror %s: verify: %w", mirror, err)
			continue
		}
		if err := checkReaches(posts, ptr); err != nil {
			lastErr = fmt.Errorf("mirror %s: %w", mirror, err)
			continue
		}
		cd, err := cacheDir(pubkey)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(cd, "posts.ndjson"), data, filePerm); err != nil {
			return "", err
		}
		writeFetchStatus(pubkey, true, "")
		return mirror, nil
	}
	msg := "all mirrors failed"
	if lastErr != nil {
		msg = lastErr.Error()
	}
	writeFetchStatus(pubkey, false, msg)
	return "", fmt.Errorf("%s", msg)
}

func cmdFetch(args []string) {
	var targets []string
	if len(args) > 0 {
		pk, err := resolvePubkeyInput(args[0])
		if err != nil {
			fatal(err)
		}
		targets = []string{pk}
	} else {
		follows, err := listFollows()
		if err != nil {
			fatal(err)
		}
		for _, f := range follows {
			targets = append(targets, f.Pubkey)
		}
	}
	if len(targets) == 0 {
		fmt.Println("(nothing to fetch)")
		return
	}
	okCount := 0
	for _, pk := range targets {
		pub, _ := decodePubkey(pk)
		fp := pk
		if pub != nil {
			fp = fingerprint(pub)
		}
		mirror, err := fetchOne(pk)
		if err != nil {
			fmt.Printf("%s  FAIL: %s\n", fp, err)
			continue
		}
		fmt.Printf("%s  OK via %s\n", fp, shortenMirror(mirror))
		okCount++
	}
	fmt.Printf("fetched %d/%d\n", okCount, len(targets))
}

// offline and to reject stale or hostile responses.
func refreshFollowPointer(pubkey string) (Pointer, error) {
	cached, err := loadFollow(pubkey)
	if err != nil {
		return Pointer{}, err
	}
	src := loadFollowSource(pubkey)
	if src == "" {
		return cached, nil
	}
	data, err := fetchURL(src)
	if err != nil {
		return cached, nil
	}
	fresh, err := parsePointer(strings.TrimSpace(string(data)))
	if err != nil {
		return cached, nil
	}
	if fresh.Pubkey != pubkey {
		return cached, nil
	}
	if fresh.Seq < cached.Seq {
		return cached, nil
	}
	_ = saveFollow(fresh)
	return fresh, nil
}

// shortenMirror turns a full mirror URL into a short label for fetch
// output. LAN URLs get "LAN (host:port)"; everything else gets "gist"
// if it's a GitHub gist, or the host otherwise.
func shortenMirror(url string) string {
	// http://192.168.0.206:8848/posts.ndjson -> LAN (192.168.0.206:8848)
	if strings.HasPrefix(url, "http://") {
		rest := strings.TrimPrefix(url, "http://")
		if idx := strings.Index(rest, "/"); idx > 0 {
			hostPort := rest[:idx]
			if strings.HasPrefix(hostPort, "192.168.") || strings.HasPrefix(hostPort, "10.") || strings.HasPrefix(hostPort, "172.") {
				return "LAN (" + hostPort + ")"
			}
			return hostPort
		}
	}
	// https://gist.githubusercontent.com/... -> gist
	if strings.HasPrefix(url, "https://gist.githubusercontent.com/") {
		return "gist"
	}
	// anything else -> just the host
	if strings.HasPrefix(url, "https://") {
		rest := strings.TrimPrefix(url, "https://")
		if idx := strings.Index(rest, "/"); idx > 0 {
			return rest[:idx]
		}
		return rest
	}
	return url
}

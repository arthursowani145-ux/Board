package main

import (
	"bufio"
	"crypto/ed25519"
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

func fetchOne(pubkey string) error {
	ptr, err := loadFollow(pubkey)
	if err != nil {
		return fmt.Errorf("no follow for %s", pubkey)
	}
	pub, err := decodePubkey(pubkey)
	if err != nil {
		return err
	}
	if len(ptr.Mirrors) == 0 {
		err := fmt.Errorf("no mirrors for %s", pubkey)
		writeFetchStatus(pubkey, false, err.Error())
		return err
	}

	// Try each mirror in order.
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
		// Good: cache it.
		cd, err := cacheDir(pubkey)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(cd, "posts.ndjson"), data, filePerm); err != nil {
			return err
		}
		writeFetchStatus(pubkey, true, "")
		return nil
	}
	msg := "all mirrors failed"
	if lastErr != nil {
		msg = lastErr.Error()
	}
	writeFetchStatus(pubkey, false, msg)
	return fmt.Errorf("%s", msg)
}

func cmdFetch(args []string) {
	var targets []string
	if len(args) > 0 {
		// resolve one pubkey (full or fp)
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
		if err := fetchOne(pk); err != nil {
			fmt.Printf("%s  FAIL: %s\n", fp, err)
			continue
		}
		fmt.Printf("%s  OK\n", fp)
		okCount++
	}
	fmt.Printf("fetched %d/%d\n", okCount, len(targets))
}

// silence unused import when ed25519 isn't referenced elsewhere in this file
var _ = ed25519.PublicKey(nil)

package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func fetchURL(url string) ([]byte, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

// resolvePointerInput takes one of:
//
//	board:v1:<b64>
//	{...}
//	https://...
//	http://...
//
// and returns a verified Pointer.
func resolvePointerInput(s string) (Pointer, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		body, err := fetchURL(s)
		if err != nil {
			return Pointer{}, fmt.Errorf("follow: fetch %s: %w", s, err)
		}
		return parsePointer(strings.TrimSpace(string(body)))
	}
	return parsePointer(s)
}

func cmdFollow(args []string) {
	fs := flag.NewFlagSet("follow", flag.ExitOnError)
	force := fs.Bool("force", false, "replace an existing follow without confirmation")
	fs.Parse(args)
	rest := fs.Args()
	if len(rest) != 2 {
		fatal(fmt.Errorf("usage: board follow <pubkey-or-fingerprint> <pointer>"))
	}
	inputPub, inputPtr := rest[0], rest[1]

	ptr, err := resolvePointerInput(inputPtr)
	if err != nil {
		fatal(err)
	}

	// If input is a full pubkey, it must match the pointer's pubkey.
	// If input is a fingerprint, it must match the pointer's pubkey's
	// fingerprint. Either way, no mismatch is allowed.
	if strings.Contains(inputPub, "-") && len(inputPub) < 30 {
		pub, err := decodePubkey(ptr.Pubkey)
		if err != nil {
			fatal(err)
		}
		if fingerprint(pub) != inputPub {
			fatal(fmt.Errorf("follow: pointer pubkey fingerprint is %s, not %s", fingerprint(pub), inputPub))
		}
	} else {
		if inputPub != ptr.Pubkey {
			fatal(fmt.Errorf("follow: pubkey mismatch: given %s, pointer says %s", inputPub, ptr.Pubkey))
		}
	}

	// Existing follow? Handle collision unless --force.
	if existing, err := loadFollow(ptr.Pubkey); err == nil {
		if !*force {
			if existing.Seq == ptr.Seq && existing.Head == ptr.Head {
				fmt.Println("already following, pointer unchanged")
				return
			}
			fatal(fmt.Errorf("follow: already following this pubkey (seq %d -> %d); use 'board fetch' to update, or --force to replace", existing.Seq, ptr.Seq))
		}
	}

	if err := saveFollow(ptr); err != nil {
		fatal(err)
	}
	// If the input was a URL, remember it so fetch can refresh the pointer.
	if strings.HasPrefix(inputPtr, "http://") || strings.HasPrefix(inputPtr, "https://") {
		if err := saveFollowSource(ptr.Pubkey, inputPtr); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save pointer source: %v\n", err)
		}
	}

	pub, _ := decodePubkey(ptr.Pubkey)
	fmt.Printf("following %s (seq %d)\n", fingerprint(pub), ptr.Seq)
}

func cmdFollows() {
	follows, err := listFollows()
	if err != nil {
		fatal(err)
	}
	if len(follows) == 0 {
		fmt.Println("(not following anyone)")
		return
	}
	for _, p := range follows {
		pub, err := decodePubkey(p.Pubkey)
		fp := p.Pubkey
		if err == nil {
			fp = fingerprint(pub)
		}
		fmt.Printf("%s  seq=%-6d mirrors=%d  ts=%s\n", fp, p.Seq, len(p.Mirrors), p.TS)
	}
}

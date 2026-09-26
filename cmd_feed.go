package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type feedEntry struct {
	Pubkey string
	Fp     string
	Seq    int64
	Title  string
	Type   string
	Link   string
	TS     string
	PostID string
}

func loadCachedPosts(pubkey string) ([]Post, error) {
	d, err := boardDir()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(d, "cache", pubkey, "posts.ndjson")
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseLogBytes(data)
}

func loadFetchStatus(pubkey string) fetchStatus {
	var st fetchStatus
	d, err := boardDir()
	if err != nil {
		return st
	}
	p := filepath.Join(d, "cache", pubkey, "last-fetch.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return st
	}
	_ = json.Unmarshal(data, &st)
	return st
}

func cmdFeed(args []string) {
	follows, err := listFollows()
	if err != nil {
		fatal(err)
	}
	if len(follows) == 0 {
		fmt.Println("(not following anyone; use 'board follow')")
		return
	}

	var entries []feedEntry
	var stale []string
	for _, f := range follows {
		pub, _ := decodePubkey(f.Pubkey)
		fp := f.Pubkey
		if pub != nil {
			fp = fingerprint(pub)
		}
		posts, err := loadCachedPosts(f.Pubkey)
		if err != nil || posts == nil {
			stale = append(stale, fmt.Sprintf("%s (never fetched)", fp))
			continue
		}
		// Report fetch failures as staleness.
		st := loadFetchStatus(f.Pubkey)
		if !st.OK && st.Err != "" {
			stale = append(stale, fmt.Sprintf("%s (%s)", fp, st.Err))
		}
		for _, p := range posts {
			entries = append(entries, feedEntry{
				Pubkey: f.Pubkey,
				Fp:     fp,
				Seq:    p.Seq,
				Title:  p.Title,
				Type:   p.Type,
				Link:   p.Link,
				TS:     p.TS,
				PostID: fp + ":" + fmt.Sprint(p.Seq),
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, entries[i].TS)
		tj, _ := time.Parse(time.RFC3339, entries[j].TS)
		return ti.After(tj)
	})

	for _, e := range entries {
		fmt.Printf("%s  [%s] %s\n    %s\n", e.TS, e.Type, e.Title, e.Link)
		fmt.Printf("    id: %s\n", e.PostID)
	}
	if len(stale) > 0 {
		fmt.Println()
		fmt.Println("stale follows:")
		for _, s := range stale {
			fmt.Println("  " + s)
		}
	}
}

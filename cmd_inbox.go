package main

import (
	"crypto/ed25519"
	"fmt"
	"sort"
	"time"
)

type inboxEntry struct {
	FromFp  string
	FromPub string
	Seq     int64
	TS      string
	Text    string
	PostID  string
}

func collectInbox(myPubB64 string, priv ed25519.PrivateKey, pub ed25519.PublicKey) ([]inboxEntry, []string) {
	follows, err := listFollows()
	if err != nil {
		return nil, []string{fmt.Sprintf("list follows: %v", err)}
	}
	var entries []inboxEntry
	var problems []string

	for _, f := range follows {
		posts, err := loadCachedPosts(f.Pubkey)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", shortPub(f.Pubkey), err))
			continue
		}
		fromPub, err := decodePubkey(f.Pubkey)
		if err != nil {
			continue
		}
		fromFp := fingerprint(fromPub)

		for _, p := range posts {
			if p.Type != "msg" {
				continue
			}
			if p.To != myPubB64 {
				continue
			}
			if p.Content == "" {
				continue
			}
			blob, err := b64dec(p.Content)
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s seq %d: bad b64", fromFp, p.Seq))
				continue
			}
			plaintext, err := DecryptFrom(priv, pub, blob)
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s seq %d: %v", fromFp, p.Seq, err))
				continue
			}
			entries = append(entries, inboxEntry{
				FromFp:  fromFp,
				FromPub: f.Pubkey,
				Seq:     p.Seq,
				TS:      p.TS,
				Text:    string(plaintext),
				PostID:  fromFp + ":" + fmt.Sprint(p.Seq),
			})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, entries[i].TS)
		tj, _ := time.Parse(time.RFC3339, entries[j].TS)
		return ti.After(tj)
	})
	return entries, problems
}

func shortPub(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12] + "…"
}

func cmdInbox(args []string) {
	priv, err := loadOrCreateIdentity()
	if err != nil {
		fatal(err)
	}
	pub := priv.Public().(ed25519.PublicKey)
	myPubB64 := pubkeyB64(pub)

	entries, problems := collectInbox(myPubB64, priv, pub)
	if len(entries) == 0 && len(problems) == 0 {
		fmt.Println("(no messages)")
		return
	}
	for _, e := range entries {
		fmt.Printf("%s  from %s  (seq %d)\n", e.TS, e.FromFp, e.Seq)
		fmt.Printf("    %s\n", e.Text)
		fmt.Printf("    id: %s\n\n", e.PostID)
	}
	if len(problems) > 0 {
		fmt.Println("problems:")
		for _, p := range problems {
			fmt.Println("  " + p)
		}
	}
}

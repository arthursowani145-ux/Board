package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// findFilePost scans a contact's cached log for the most recent
// type=file post with the given title.
func findFilePost(authorPub, name string) (Post, error) {
	posts, err := loadCachedPosts(authorPub)
	if err != nil {
		return Post{}, fmt.Errorf("get: load cached log: %w", err)
	}
	if len(posts) == 0 {
		return Post{}, fmt.Errorf("get: no cached posts for %s (run 'board fetch' first)", shortPub(authorPub))
	}
	var latest Post
	found := false
	for _, p := range posts {
		if p.Type != "file" {
			continue
		}
		if p.Title != name {
			continue
		}
		if !found || p.Seq > latest.Seq {
			latest = p
			found = true
		}
	}
	if !found {
		return Post{}, fmt.Errorf("get: no file named %q from %s", name, shortPub(authorPub))
	}
	return latest, nil
}

// downloadsDir returns ~/downloads/, creating it if needed.
func downloadsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(home, "downloads")
	if err := os.MkdirAll(d, 0755); err != nil {
		return "", err
	}
	return d, nil
}

func cmdGet(args []string) {
	if len(args) < 1 {
		fatal(fmt.Errorf("usage: board get <peer>/<name> [--into <dir>]"))
	}
	spec := args[0]
	parts := strings.SplitN(spec, "/", 2)
	if len(parts) != 2 {
		fatal(fmt.Errorf("get: expected '<peer>/<name>', got %q", spec))
	}
	contactInput, name := parts[0], parts[1]

	destDir := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--into" && i+1 < len(args) {
			destDir = args[i+1]
			i++
		}
	}
	if destDir == "" {
		d, err := downloadsDir()
		if err != nil {
			fatal(err)
		}
		destDir = d
	}

	authorPub, err := resolveContact(contactInput)
	if err != nil {
		fatal(err)
	}
	authorPubKey, err := decodePubkey(authorPub)
	if err != nil {
		fatal(err)
	}
	authorFp := fingerprint(authorPubKey)

	post, err := findFilePost(authorPub, name)
	if err != nil {
		fatal(err)
	}
	if post.Digest == "" {
		fatal(fmt.Errorf("get: post has no digest"))
	}

	url, err := resolveMirrorForPost(post, authorPub)
	if err != nil {
		fatal(err)
	}

	data, err := fetchURL(url)
	if err != nil {
		fatal(fmt.Errorf("get: fetch %s: %w", url, err))
	}

	gotDigest := fmt.Sprintf("%x", sha256.Sum256(data))
	if gotDigest != post.Digest {
		fatal(fmt.Errorf("get: digest mismatch\n  want %s\n  got  %s", post.Digest, gotDigest))
	}

	destPath := filepath.Join(destDir, name)
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		fatal(fmt.Errorf("get: write %s: %w", destPath, err))
	}

	fmt.Printf("got %s from %s (seq %d)\n", name, authorFp, post.Seq)
	fmt.Printf("  digest: %s\n", gotDigest)
	fmt.Printf("  saved:  %s\n", destPath)
	fmt.Printf("  bytes:  %d\n", len(data))
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// findLatestProjectPost scans a contact's cached log for the most recent
// post of type "project" with the given title (project name). Returns
// the post and its author's pubkey.
func findLatestProjectPost(authorPub string, projectName string) (Post, error) {
	posts, err := loadCachedPosts(authorPub)
	if err != nil {
		return Post{}, fmt.Errorf("pull: load cached log: %w", err)
	}
	if len(posts) == 0 {
		return Post{}, fmt.Errorf("pull: no cached posts for %s (run 'board fetch' first)", shortPub(authorPub))
	}
	var latest Post
	found := false
	for _, p := range posts {
		if p.Type != "project" {
			continue
		}
		if p.Title != projectName {
			continue
		}
		if !found || p.Seq > latest.Seq {
			latest = p
			found = true
		}
	}
	if !found {
		return Post{}, fmt.Errorf("pull: no project post named %q from %s", projectName, shortPub(authorPub))
	}
	return latest, nil
}

// resolveMirrorForPost returns a full URL to fetch the bundle from,
// given the post's Link and the author's pointer (which lists mirrors).
func resolveMirrorForPost(post Post, authorPub string) (string, error) {
	ptr, err := loadFollow(authorPub)
	if err != nil {
		return "", fmt.Errorf("pull: no pointer for %s: %w", shortPub(authorPub), err)
	}
	if len(ptr.Mirrors) == 0 {
		return "", fmt.Errorf("pull: %s has no mirrors configured", shortPub(authorPub))
	}
	// If the post's link is absolute, use it directly.
	if strings.HasPrefix(post.Link, "http://") || strings.HasPrefix(post.Link, "https://") {
		return post.Link, nil
	}
	// Otherwise, resolve it relative to each mirror's base URL.
	// A mirror is typically http://host:port/posts.ndjson
	// We want http://host:port/<link>
	var lastErr error
	for _, mirror := range ptr.Mirrors {
		base := mirror
		// Strip trailing path component if it looks like a filename.
		if idx := strings.LastIndex(base, "/"); idx > 8 {
			base = base[:idx]
		}
		url := base + post.Link
		// Try it now (cheap: HEAD would be better, but GET is fine for v1)
		if _, err := fetchURL(url); err == nil {
			return url, nil
		} else {
			lastErr = err
		}
	}
	return "", fmt.Errorf("pull: no mirror served the bundle: %v", lastErr)
}

func cmdPull(args []string) {
	if len(args) < 1 {
		fatal(fmt.Errorf("usage: board pull <contact>/<project> [--into <dir>]"))
	}
	spec := args[0]
	parts := strings.SplitN(spec, "/", 2)
	if len(parts) != 2 {
		fatal(fmt.Errorf("pull: expected '<contact>/<project>', got %q", spec))
	}
	contactInput, projectName := parts[0], parts[1]

	// Optional --into flag.
	destDir := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--into" && i+1 < len(args) {
			destDir = args[i+1]
			i++
		}
	}
	if destDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fatal(err)
		}
		destDir = filepath.Join(home, "code", projectName)
	}

	// Resolve the contact.
	authorPub, err := resolveContact(contactInput)
	if err != nil {
		fatal(err)
	}
	authorPubKey, err := decodePubkey(authorPub)
	if err != nil {
		fatal(err)
	}
	authorFp := fingerprint(authorPubKey)

	// Find the latest project post.
	post, err := findLatestProjectPost(authorPub, projectName)
	if err != nil {
		fatal(err)
	}
	if post.Digest == "" {
		fatal(fmt.Errorf("pull: project post has no digest"))
	}

	// Resolve the mirror URL for the bundle.
	bundleURL, err := resolveMirrorForPost(post, authorPub)
	if err != nil {
		fatal(err)
	}

	// Download the bundle.
	bundle, err := fetchURL(bundleURL)
	if err != nil {
		fatal(fmt.Errorf("pull: fetch %s: %w", bundleURL, err))
	}

	// Verify the digest matches what the author signed.
	gotDigest := bundleDigest(bundle)
	if gotDigest != post.Digest {
		fatal(fmt.Errorf("pull: bundle digest mismatch\n  want %s\n  got  %s", post.Digest, gotDigest))
	}

	// Extract into destDir.
	if err := extractBundle(bundle, destDir); err != nil {
		fatal(err)
	}

	fmt.Printf("pulled %s from %s (seq %d)\n", projectName, authorFp, post.Seq)
	fmt.Printf("  digest: %s\n", gotDigest)
	fmt.Printf("  into:   %s\n", destDir)
}

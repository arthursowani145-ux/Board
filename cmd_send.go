package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// sendFile reads a single file, computes its SHA-256, copies it into
// the public dir so board serve will expose it, and publishes a
// type=file post pointing at it.
func sendFile(path string) (Post, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Post{}, fmt.Errorf("send: read %s: %w", path, err)
	}
	base := filepath.Base(path)
	if base == "" || base == "." || base == "/" {
		return Post{}, fmt.Errorf("send: cannot derive filename from %s", path)
	}

	digest := fmt.Sprintf("%x", sha256.Sum256(data))

	pub, err := publicDir()
	if err != nil {
		return Post{}, err
	}
	destPath := filepath.Join(pub, base)
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return Post{}, fmt.Errorf("send: write %s: %w", destPath, err)
	}

	return publishContentPost(base, "file", "/"+base, digest, map[string]interface{}{
		"size_bytes": len(data),
		"path":       destPath,
	})
}

func cmdSend(args []string) {
	if len(args) < 2 {
		fatal(fmt.Errorf("usage: board send <peer> <path>"))
	}
	peerInput := args[0]
	path := args[1]

	// Resolve the peer to a full pubkey. We don't strictly need it for
	// the send itself (the content goes into our own log), but resolving
	// it confirms the peer exists in our follows, and it gives us a
	// clean error if it doesn't.
	if _, err := resolveContact(peerInput); err != nil {
		fatal(err)
	}

	// Stat the path.
	fi, err := os.Stat(path)
	if err != nil {
		fatal(fmt.Errorf("send: %w", err))
	}

	if fi.IsDir() {
		// Directory: must be a git repo. Shell out to cmdPush, which
		// handles the bundle/publish flow.
		if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
			fatal(fmt.Errorf("send: %s is a directory but not a git repository", path))
		}
		cmdPush([]string{path})
		return
	}

	if !fi.Mode().IsRegular() {
		fatal(fmt.Errorf("send: %s is not a regular file or directory", path))
	}

	// Regular file: send as type=file.
	_, err = sendFile(path)
	if err != nil {
		fatal(err)
	}
}

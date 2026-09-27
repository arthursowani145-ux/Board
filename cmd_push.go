package main

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// publicDir returns ~/board-public/, the canonical directory that a
// user serves over HTTP when they want to publish bundles. We create
// it if missing.
func publicDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(home, "board-public")
	if err := os.MkdirAll(d, 0755); err != nil {
		return "", err
	}
	return d, nil
}

// projectNameFromDir returns the project name for a directory: its base
// name, sanitized so it can appear in a URL and a filename.
func projectNameFromDir(dir string) string {
	base := filepath.Base(filepath.Clean(dir))
	// keep letters, digits, dash, underscore; replace everything else with -
	var b strings.Builder
	for _, r := range base {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}

func cmdPush(args []string) {
	if len(args) != 1 {
		fatal(fmt.Errorf("usage: board push <project-dir>"))
	}
	projectDir := args[0]
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		fatal(err)
	}
	if _, err := os.Stat(filepath.Join(absDir, ".git")); err != nil {
		fatal(fmt.Errorf("push: %s is not a git repository", absDir))
	}

	name := projectNameFromDir(absDir)
	if name == "" || name == "-" {
		fatal(fmt.Errorf("push: cannot derive project name from %s", absDir))
	}

	// Create the bundle.
	bundle, err := createBundle(absDir)
	if err != nil {
		fatal(err)
	}
	digest := bundleDigest(bundle)

	// Place it in the public dir.
	pub, err := publicDir()
	if err != nil {
		fatal(err)
	}
	bundlePath := filepath.Join(pub, name+".bundle")
	if err := os.WriteFile(bundlePath, bundle, 0644); err != nil {
		fatal(fmt.Errorf("push: write bundle: %w", err))
	}

	// Load identity and log.
	priv, err := loadOrCreateIdentity()
	if err != nil {
		fatal(err)
	}
	posts, err := readMyLog()
	if err != nil {
		fatal(err)
	}

	// Build the project post.
	p, err := newPost(posts, name, "project", "")
	if err != nil {
		fatal(err)
	}
	p.Digest = digest
	// The link field carries the URL of the bundle. For v1 we assume the
	// user serves ~/board-public/ and the bundle is at /<name>.bundle.
	// In a later version we could add --url flag to override this.
	p.Link = "/" + name + ".bundle"

	signed, err := appendPost(priv, p)
	if err != nil {
		fatal(err)
	}

	// Self-verify the whole log.
	pubKey := priv.Public().(ed25519.PublicKey)
	all, _ := readMyLog()
	if err := verifyLog(pubKey, all); err != nil {
		fatal(fmt.Errorf("post-append self-verify failed: %w", err))
	}

	// Emit a JSON summary so the user can see what happened and how to
	// make the bundle reachable.
	out := map[string]interface{}{
		"project":    name,
		"seq":        signed.Seq,
		"digest":     digest,
		"bundle":     bundlePath,
		"link":       signed.Link,
		"size_bytes": len(bundle),
	}
	enc, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(enc))
	fmt.Println()
	fmt.Println("To make the bundle reachable, serve ~/board-public/ over HTTP")
	fmt.Println("and add the URL as a mirror, e.g.:")
	fmt.Println("  cd ~/board-public && python3 -m http.server 8765 --bind 0.0.0.0")
	fmt.Println("  board mirror add http://<your-lan-ip>:8765/")
}

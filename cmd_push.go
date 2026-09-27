package main

import (
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

	bundle, err := createBundle(absDir)
	if err != nil {
		fatal(err)
	}
	digest := bundleDigest(bundle)

	pub, err := publicDir()
	if err != nil {
		fatal(err)
	}
	bundlePath := filepath.Join(pub, name+".bundle")
	if err := os.WriteFile(bundlePath, bundle, 0644); err != nil {
		fatal(fmt.Errorf("push: write bundle: %w", err))
	}

	_, err = publishContentPost(name, "project", "/"+name+".bundle", digest, map[string]interface{}{
		"bundle":     bundlePath,
		"size_bytes": len(bundle),
	})
	if err != nil {
		fatal(err)
	}
}

package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ghUser returns the currently authenticated GitHub username from gh.
func ghUser() (string, error) {
	cmd := exec.Command("gh", "api", "user", "--jq", ".login")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh api user: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	user := strings.TrimSpace(stdout.String())
	if user == "" {
		return "", fmt.Errorf("gh api user returned empty")
	}
	return user, nil
}

func cmdMirrorAddGist(args []string) {
	if len(args) != 1 {
		fatal(fmt.Errorf("usage: board mirror add-gist <gist-id>"))
	}
	gistID := args[0]

	user, err := ghUser()
	if err != nil {
		fatal(fmt.Errorf("mirror add-gist: need gh authenticated: %w", err))
	}

	mirrorURL := fmt.Sprintf("https://gist.githubusercontent.com/%s/%s/raw/posts.ndjson", user, gistID)
	pointerURL := fmt.Sprintf("https://gist.githubusercontent.com/%s/%s/raw/pointer.json", user, gistID)

	// Register uploader. Replace any existing "my-gist" with a notice.
	ups, err := loadUploaders()
	if err != nil {
		fatal(err)
	}
	replaced := false
	if _, ok := ups["my-gist"]; ok {
		replaced = true
	}
	ups["my-gist"] = Uploader{
		Type:   "gist",
		GistID: gistID,
	}
	if err := saveUploaders(ups); err != nil {
		fatal(err)
	}

	// Register mirror. Preserve existing mirrors, but drop any prior
	// gist mirror for the same host to avoid duplicates.
	mirrors, err := loadMirrors()
	if err != nil {
		fatal(err)
	}
	var cleaned []string
	for _, m := range mirrors {
		if strings.HasPrefix(m, "https://gist.githubusercontent.com/") {
			continue
		}
		cleaned = append(cleaned, m)
	}
	cleaned = append(cleaned, mirrorURL)
	if err := saveMirrors(cleaned); err != nil {
		fatal(err)
	}

	if replaced {
		fmt.Println("replaced existing 'my-gist' uploader")
	}
	fmt.Println("uploader: my-gist")
	fmt.Println("mirror:  ", mirrorURL)
	fmt.Println("pointer: ", pointerURL)
	fmt.Println()
	fmt.Println("Next: run 'board mirror push' to publish your log to this gist.")
}

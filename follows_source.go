package main

import (
	"os"
	"path/filepath"
	"strings"
)

func followSourcePath(pubkey string) (string, error) {
	d, err := followsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, pubkey+".source"), nil
}

// loadFollowSource returns the URL this follow's pointer is refreshed
// from, or "" if the follow has no source (was set up from a static
// pointer string).
func loadFollowSource(pubkey string) string {
	p, err := followSourcePath(pubkey)
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func saveFollowSource(pubkey, url string) error {
	p, err := followSourcePath(pubkey)
	if err != nil {
		return err
	}
	return os.WriteFile(p, []byte(url+"\n"), filePerm)
}

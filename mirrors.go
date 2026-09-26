package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func mirrorsPath() (string, error) {
	d, err := boardDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "mirrors.json"), nil
}

func loadMirrors() ([]string, error) {
	p, err := mirrorsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var mirrors []string
	if err := json.Unmarshal(data, &mirrors); err != nil {
		return nil, fmt.Errorf("mirrors: corrupt file: %w", err)
	}
	return mirrors, nil
}

func saveMirrors(mirrors []string) error {
	p, err := mirrorsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(mirrors, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, filePerm)
}

func addMirror(url string) error {
	if url == "" {
		return fmt.Errorf("mirror: empty url")
	}
	mirrors, err := loadMirrors()
	if err != nil {
		return err
	}
	for _, m := range mirrors {
		if m == url {
			return nil
		}
	}
	mirrors = append(mirrors, url)
	return saveMirrors(mirrors)
}

func removeMirror(url string) error {
	mirrors, err := loadMirrors()
	if err != nil {
		return err
	}
	out := mirrors[:0]
	found := false
	for _, m := range mirrors {
		if m == url {
			found = true
			continue
		}
		out = append(out, m)
	}
	if !found {
		return fmt.Errorf("mirror: not found: %s", url)
	}
	return saveMirrors(out)
}

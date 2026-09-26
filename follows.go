package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func followsDir() (string, error) {
	d, err := boardDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(d, "follows")
	if err := os.MkdirAll(p, dirPerm); err != nil {
		return "", err
	}
	return p, nil
}

func followPath(pubkey string) (string, error) {
	d, err := followsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, pubkey+".json"), nil
}

func loadFollow(pubkey string) (Pointer, error) {
	p, err := followPath(pubkey)
	if err != nil {
		return Pointer{}, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return Pointer{}, err
	}
	var ptr Pointer
	if err := json.Unmarshal(data, &ptr); err != nil {
		return Pointer{}, fmt.Errorf("follow: corrupt pointer file: %w", err)
	}
	return ptr, nil
}

func saveFollow(ptr Pointer) error {
	p, err := followPath(ptr.Pubkey)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(ptr, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, filePerm)
}

func listFollows() ([]Pointer, error) {
	d, err := followsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(d)
	if err != nil {
		return nil, err
	}
	var out []Pointer
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		pubkey := strings.TrimSuffix(e.Name(), ".json")
		ptr, err := loadFollow(pubkey)
		if err != nil {
			continue
		}
		out = append(out, ptr)
	}
	return out, nil
}

func resolvePubkeyInput(s string) (string, error) {
	// Accept either a full base64url pubkey or a fingerprint like
	// "jhft-IZR3-M3E". Return the full pubkey.
	if strings.Contains(s, "-") && len(s) < 30 {
		// looks like a fingerprint; find a follow whose fingerprint matches
		follows, err := listFollows()
		if err != nil {
			return "", err
		}
		for _, f := range follows {
			pub, err := decodePubkey(f.Pubkey)
			if err != nil {
				continue
			}
			if fingerprint(pub) == s {
				return f.Pubkey, nil
			}
		}
		return "", fmt.Errorf("follow: no followed pubkey with fingerprint %q", s)
	}
	// assume full pubkey; validate
	if _, err := decodePubkey(s); err != nil {
		return "", fmt.Errorf("follow: not a valid pubkey: %w", err)
	}
	return s, nil
}

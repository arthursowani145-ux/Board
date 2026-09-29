package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os/exec"
	"strings"
)

func cmdPointer(args []string) {
	fs := flag.NewFlagSet("pointer", flag.ExitOnError)
	form := fs.String("form", "b64", "output form: json | b64 | url")
	fs.Parse(args)

	priv, err := loadOrCreateIdentity()
	if err != nil {
		fatal(err)
	}
	posts, err := readMyLog()
	if err != nil {
		fatal(err)
	}
	mirrors, err := loadMirrors()
	if err != nil {
		fatal(err)
	}
	p, err := buildPointer(priv, posts, mirrors)
	if err != nil {
		fatal(err)
	}

	switch *form {
	case "json":
		out, _ := json.MarshalIndent(p, "", "  ")
		fmt.Println(string(out))

	case "b64":
		s, err := encodePointerCompact(p)
		if err != nil {
			fatal(err)
		}
		fmt.Println(s)

	case "url":
		// Requires a gist uploader to host the pointer.
		ups, err := loadUploaders()
		if err != nil {
			fatal(err)
		}
		var gist *Uploader
		for _, u := range ups {
			if u.Type == "gist" {
				copyOfU := u
				gist = &copyOfU
				break
			}
		}
		if gist == nil {
			fatal(fmt.Errorf("pointer: no gist uploader configured; run 'board mirror add-gist <id>' first"))
		}
		user, err := ghUser()
		if err != nil {
			fatal(fmt.Errorf("pointer: %w", err))
		}

		// Publish the log itself first, so the pointer's committed head
		// matches what's actually served at the mirror.
		if err := pushViaGist("my-gist", *gist); err != nil {
			fatal(fmt.Errorf("pointer: publish log: %w", err))
		}

		// Now publish the pointer.
		raw, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			fatal(err)
		}
		body, err := json.Marshal(map[string]interface{}{
			"files": map[string]interface{}{
				"pointer.json": map[string]string{
					"content": string(raw),
				},
			},
		})
		if err != nil {
			fatal(err)
		}
		cmd := exec.Command("gh", "api", "-X", "PATCH", "/gists/"+gist.GistID, "--input", "-")
		cmd.Stdin = bytes.NewReader(body)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			fatal(fmt.Errorf("pointer: gh api: %v: %s", err, strings.TrimSpace(stderr.String())))
		}

		url := fmt.Sprintf("https://gist.githubusercontent.com/%s/%s/raw/pointer.json", user, gist.GistID)
		fmt.Println(url)

	default:
		fatal(fmt.Errorf("unknown form: %s", *form))
	}
}

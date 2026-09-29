package main

import (
"bytes"
"crypto/sha256"
"fmt"
"os"
"os/exec"
"strings"
)

// pushViaGist uploads the log to a GitHub gist using the gh CLI.
// The gist is edited in place, so its stable "raw" URL keeps pointing
// at the latest content.
func pushViaGist(name string, u Uploader) error {
if u.GistID == "" {
return fmt.Errorf("uploader %q: gist_id missing", name)
}
lp, err := myLogPath()
if err != nil {
return err
}
logData, err := os.ReadFile(lp)
if err != nil {
return fmt.Errorf("uploader %q: read log: %w", name, err)
}

cmd := exec.Command("gh", "gist", "edit", u.GistID, "-f", "posts.ndjson", "-")
cmd.Stdin = bytes.NewReader(logData)
var stderr bytes.Buffer
cmd.Stderr = &stderr
if err := cmd.Run(); err != nil {
return fmt.Errorf("uploader %q: gh gist edit: %v: %s", name, err, strings.TrimSpace(stderr.String()))
}
return nil
}

func pushOneUploader(name string, u Uploader) error {
switch u.Type {
case "gist":
return pushViaGist(name, u)
case "":
return fmt.Errorf("uploader %q: no type set", name)
default:
return fmt.Errorf("uploader %q: unknown type %q", name, u.Type)
}
}

func cmdMirrorPush(args []string) {
uploaders, err := loadUploaders()
if err != nil {
fatal(err)
}
if len(uploaders) == 0 {
fmt.Println("(no uploaders configured)")
fmt.Println()
fmt.Println("To set one up, create ~/.board/uploaders.json with:")
fmt.Println(`  {"my-gist": {"type": "gist", "gist_id": "<id>"}}`)
return
}

var names []string
if len(args) == 1 {
if _, ok := uploaders[args[0]]; !ok {
fatal(fmt.Errorf("mirror push: no uploader named %q", args[0]))
}
names = []string{args[0]}
} else {
for n := range uploaders {
names = append(names, n)
}
}

// Compute the digest of the log so the user sees what went up.
lp, _ := myLogPath()
logData, _ := os.ReadFile(lp)
digest := fmt.Sprintf("%x", sha256.Sum256(logData))

for _, n := range names {
if err := pushOneUploader(n, uploaders[n]); err != nil {
fmt.Printf("%-15s  FAIL: %v\n", n, err)
continue
}
fmt.Printf("%-15s  OK\n", n)
}
fmt.Printf("pushed %d uploader(s), log digest %s\n", len(names), digest[:16])
}

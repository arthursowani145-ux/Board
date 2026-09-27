package main

import (
"bytes"
"fmt"
"os"
"os/exec"
"path/filepath"
"strings"
)

// createBundle runs `git bundle create - --all` inside projectDir and
// returns the bundle bytes. A bundle is a single-file, self-contained
// git repository: full history, all branches, clonable with
// `git clone <file>` as if it were a remote.
//
// We use stdout capture rather than a temp file so the caller gets the
// bytes directly and can hash, sign, or serve them without touching disk.
func createBundle(projectDir string) ([]byte, error) {
// Verify projectDir is a git repository.
gitDir := filepath.Join(projectDir, ".git")
if _, err := os.Stat(gitDir); err != nil {
return nil, fmt.Errorf("bundle: %s is not a git repository (no .git)", projectDir)
}

cmd := exec.Command("git", "bundle", "create", "-", "--all")
cmd.Dir = projectDir
var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout
cmd.Stderr = &stderr
if err := cmd.Run(); err != nil {
return nil, fmt.Errorf("bundle: git bundle failed: %v: %s", err, strings.TrimSpace(stderr.String()))
}
if stdout.Len() == 0 {
return nil, fmt.Errorf("bundle: git bundle produced no output")
}
return stdout.Bytes(), nil
}

// bundleName returns a stable filename for a project's bundle,
// derived from the project directory name. Used when placing the
// bundle into a served directory.
func bundleName(projectDir string) string {
base := filepath.Base(filepath.Clean(projectDir))
return base + ".bundle"
}

// extractBundle clones a bundle into destDir. destDir must not exist;
// we refuse to overwrite anything.
func extractBundle(bundle []byte, destDir string) error {
if _, err := os.Stat(destDir); err == nil {
return fmt.Errorf("bundle: destination %s already exists", destDir)
}
// Write bundle to a temp file, clone from it, remove the temp file.
tmp, err := os.CreateTemp("", "board-bundle-*.bundle")
if err != nil {
return fmt.Errorf("bundle: temp file: %w", err)
}
tmpPath := tmp.Name()
defer os.Remove(tmpPath)
if _, err := tmp.Write(bundle); err != nil {
tmp.Close()
return fmt.Errorf("bundle: write temp: %w", err)
}
if err := tmp.Close(); err != nil {
return fmt.Errorf("bundle: close temp: %w", err)
}

// git clone <bundlefile> <dest>
cmd := exec.Command("git", "clone", tmpPath, destDir)
var stderr bytes.Buffer
cmd.Stderr = &stderr
if err := cmd.Run(); err != nil {
return fmt.Errorf("bundle: git clone failed: %v: %s", err, strings.TrimSpace(stderr.String()))
}
return nil
}

// bundleDigest returns the hex sha256 of the bundle bytes. This is what
// goes into the post's Digest field, so the recipient can verify the
// bundle they downloaded is exactly the one that was signed.
func bundleDigest(bundle []byte) string {
h := sha256Sum(bundle)
return fmt.Sprintf("%x", h[:])
}

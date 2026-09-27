package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

func setupTestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitIn(t, dir, "init")
	gitIn(t, dir, "config", "user.email", "test@example.com")
	gitIn(t, dir, "config", "user.name", "Test")

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "README.md")
	gitIn(t, dir, "commit", "-m", "first commit")

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "main.go")
	gitIn(t, dir, "commit", "-m", "second commit")

	return dir
}

func TestBundleCreateAndExtract(t *testing.T) {
	src := setupTestProject(t)

	bundle, err := createBundle(src)
	if err != nil {
		t.Fatalf("createBundle: %v", err)
	}
	if len(bundle) == 0 {
		t.Fatal("createBundle returned empty bundle")
	}

	parent := t.TempDir()
	dest := filepath.Join(parent, "recovered")

	if err := extractBundle(bundle, dest); err != nil {
		t.Fatalf("extractBundle: %v", err)
	}

	for _, name := range []string{"README.md", "main.go"} {
		if _, err := os.Stat(filepath.Join(dest, name)); err != nil {
			t.Fatalf("extracted project missing %s: %v", name, err)
		}
	}

	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = dest
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-list in extracted: %v", err)
	}
	if got := string(out); got != "2\n" {
		t.Fatalf("extracted history has %q commits, want 2", got)
	}
}

func TestBundleDigestDeterministic(t *testing.T) {
	src := setupTestProject(t)
	b1, err := createBundle(src)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := createBundle(src)
	if err != nil {
		t.Fatal(err)
	}
	if bundleDigest(b1) != bundleDigest(b2) {
		t.Fatal("same repo produced different bundle digests")
	}
}

func TestExtractRefusesExistingDir(t *testing.T) {
	src := setupTestProject(t)
	bundle, err := createBundle(src)
	if err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := extractBundle(bundle, dest); err == nil {
		t.Fatal("extractBundle overwrote an existing directory")
	}
}

func TestCreateBundleNotARepo(t *testing.T) {
	dir := t.TempDir()
	if _, err := createBundle(dir); err == nil {
		t.Fatal("createBundle accepted a non-git directory")
	}
}

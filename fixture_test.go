package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func loadFixture(t *testing.T) ([]Post, ed25519.PublicKey) {
	t.Helper()
	raw, err := os.ReadFile("test/fixtures/known_good_posts.log")
	if err != nil {
		t.Fatalf("read fixture log: %v", err)
	}
	pubRaw, err := os.ReadFile("test/fixtures/known_good_pubkey.txt")
	if err != nil {
		t.Fatalf("read fixture pubkey: %v", err)
	}
	pubB64 := strings.TrimSpace(string(pubRaw))
	pub, err := base64.RawURLEncoding.DecodeString(pubB64)
	if err != nil {
		t.Fatalf("decode pubkey: %v", err)
	}
	var posts []Post
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var p Post
		if err := json.Unmarshal([]byte(line), &p); err != nil {
			t.Fatalf("parse fixture post: %v", err)
		}
		posts = append(posts, p)
	}
	if len(posts) == 0 {
		t.Fatal("fixture has zero posts")
	}
	return posts, ed25519.PublicKey(pub)
}

func TestKnownGoodLogStillVerifies(t *testing.T) {
	posts, pub := loadFixture(t)
	if err := verifyLog(pub, posts); err != nil {
		t.Fatalf("known-good log no longer verifies: %v", err)
	}
	t.Logf("known-good log verified: %d posts", len(posts))
}

func TestKnownGoodCanonicalBytesStable(t *testing.T) {
	posts, pub := loadFixture(t)
	for i, p := range posts {
		sig, err := base64.RawURLEncoding.DecodeString(p.Sig)
		if err != nil {
			t.Fatalf("post %d: bad sig b64: %v", i, err)
		}
		toVerify, err := signedBytes(p)
		if err != nil {
			t.Fatalf("post %d: canonicalize: %v", i, err)
		}
		if !ed25519.Verify(pub, toVerify, sig) {
			t.Fatalf("post %d: canonical bytes changed", i)
		}
	}
}

func TestKnownGoodCanonicalFormHasNoExtraKeys(t *testing.T) {
	posts, _ := loadFixture(t)
	for i, p := range posts {
		canon, err := signedBytes(p)
		if err != nil {
			t.Fatalf("post %d: %v", i, err)
		}
		s := string(canon)
		checks := []string{`"to"`, `"content"`, `"digest"`}
		for _, key := range checks {
			if strings.Contains(s, key) {
				t.Fatalf("post %d: canonical form contains %s key; omitempty broken:\n%s", i, key, s)
			}
		}
	}
}

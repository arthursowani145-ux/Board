package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// loadFixture reads the known-good log and pubkey captured before any
// changes to the Post struct. This is the immutability contract: any
// change to signing must leave these bytes verifiable.
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

// TestKnownGoodLogStillVerifies is the contract: the exact bytes we
// captured before touching the Post struct must always verify.
// If this test ever fails, a struct change broke existing signatures
// and MUST be reverted.
func TestKnownGoodLogStillVerifies(t *testing.T) {
	posts, pub := loadFixture(t)
	if err := verifyLog(pub, posts); err != nil {
		t.Fatalf("known-good log no longer verifies: %v\n"+
			"This means a change to the Post struct or the canonicalization "+
			"broke existing signatures. REVERT immediately.", err)
	}
	t.Logf("known-good log verified: %d posts", len(posts))
}

// TestKnownGoodCanonicalBytesStable is the byte-level contract: for
// each fixture post, the canonical bytes we compute today must equal
// the bytes that were signed. We can't recompute "the old bytes"
// without the old code, so instead we verify that re-signing the same
// post with the same key produces a signature the pubkey accepts —
// which is only true if the canonical form is unchanged.
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
			t.Fatalf("post %d: canonical bytes changed — signature no longer verifies", i)
		}
	}
}

// TestKnownGoodCanonicalFormHasNoExtraKeys asserts that the canonical
// form of a fixture post does not contain any of the fields we're
// about to add (to, content). If those keys start appearing for
// non-message posts, omitempty is not doing its job and old signatures
// will break.
func TestKnownGoodCanonicalFormHasNoExtraKeys(t *testing.T) {
	posts, _ := loadFixture(t)
	for i, p := range posts {
		canon, err := signedBytes(p)
		if err != nil {
			t.Fatalf("post %d: %v", i, err)
		}
		s := string(canon)
		if strings.Contains(s, `"to"`) {
			t.Fatalf("post %d: canonical form contains \"to\" key; omitempty broken:\n%s", i, s)
		}
		if strings.Contains(s, `"content"`) {
			t.Fatalf("post %d: canonical form contains \"content\" key; omitempty broken:\n%s", i, s)
		}
	}
}

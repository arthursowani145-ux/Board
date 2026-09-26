package main

import "testing"

func TestPointerRoundTrip(t *testing.T) {
	priv, err := loadOrCreateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	posts, err := readMyLog()
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Skip("no posts to build a pointer from")
	}
	p, err := buildPointer(priv, posts, []string{"https://example.com/log"})
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyPointer(p); err != nil {
		t.Fatalf("self-verify failed: %v", err)
	}
	s, err := encodePointerCompact(p)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := parsePointer(s)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if p2.Pubkey != p.Pubkey || p2.Seq != p.Seq || p2.Head != p.Head {
		t.Fatalf("round-trip mismatch: %+v vs %+v", p, p2)
	}
}

func TestPointerTamperDetected(t *testing.T) {
	priv, err := loadOrCreateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	posts, err := readMyLog()
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Skip("no posts")
	}
	p, err := buildPointer(priv, posts, nil)
	if err != nil {
		t.Fatal(err)
	}
	p.Seq = p.Seq + 1
	if err := verifyPointer(p); err == nil {
		t.Fatal("tampered pointer verified; expected signature failure")
	}
}

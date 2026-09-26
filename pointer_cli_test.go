package main

import (
	"os"
	"testing"
)

func TestPointerFromDiskRoundTrip(t *testing.T) {
	priv, err := loadOrCreateIdentity()
	if err != nil {
		t.Fatal(err)
	}
	posts, err := readMyLog()
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) == 0 {
		t.Skip("no posts on disk")
	}
	mirrors, err := loadMirrors()
	if err != nil {
		t.Fatal(err)
	}
	if len(mirrors) == 0 {
		t.Skip("no mirrors configured")
	}
	p, err := buildPointer(priv, posts, mirrors)
	if err != nil {
		t.Fatal(err)
	}
	s, err := encodePointerCompact(p)
	if err != nil {
		t.Fatal(err)
	}
	// simulate the follower side: parse the string we just produced
	p2, err := parsePointer(s)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p2.Pubkey != p.Pubkey || p2.Seq != p.Seq || p2.Head != p.Head || len(p2.Mirrors) != len(p.Mirrors) {
		t.Fatalf("mismatch after round-trip:\n  p1=%+v\n  p2=%+v", p, p2)
	}
}

func TestMain(m *testing.M) {
	// In case tests want a stable environment.
	os.Exit(m.Run())
}

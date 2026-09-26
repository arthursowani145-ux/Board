package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const pointerVersion = 1

type Pointer struct {
	V       int      `json:"v"`
	Pubkey  string   `json:"pubkey"`
	Seq     int64    `json:"seq"`
	Head    string   `json:"head"`
	Mirrors []string `json:"mirrors"`
	TS      string   `json:"ts"`
	Sig     string   `json:"sig"`
}

func pointerSignedBytes(p Pointer) ([]byte, error) {
	p.Sig = ""
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return CanonicalizeJSON(raw)
}

func buildPointer(priv ed25519.PrivateKey, posts []Post, mirrors []string) (Pointer, error) {
	if len(posts) == 0 {
		return Pointer{}, fmt.Errorf("pointer: log is empty; publish something first")
	}
	pub := priv.Public().(ed25519.PublicKey)
	last := posts[len(posts)-1]
	head, err := hashLine(last)
	if err != nil {
		return Pointer{}, err
	}
	if mirrors == nil {
		mirrors = []string{}
	}
	p := Pointer{
		V:       pointerVersion,
		Pubkey:  pubkeyB64(pub),
		Seq:     last.Seq,
		Head:    head,
		Mirrors: mirrors,
		TS:      time.Now().UTC().Format(time.RFC3339),
	}
	toSign, err := pointerSignedBytes(p)
	if err != nil {
		return Pointer{}, err
	}
	sig := ed25519.Sign(priv, toSign)
	p.Sig = base64.RawURLEncoding.EncodeToString(sig)
	return p, nil
}

func verifyPointer(p Pointer) error {
	if p.V != pointerVersion {
		return fmt.Errorf("pointer: unsupported version %d", p.V)
	}
	pub, err := decodePubkey(p.Pubkey)
	if err != nil {
		return fmt.Errorf("pointer: bad pubkey: %w", err)
	}
	if p.Seq < 0 {
		return fmt.Errorf("pointer: negative seq")
	}
	if len(p.Head) != 64 {
		return fmt.Errorf("pointer: head must be 64 hex chars")
	}
	sig, err := base64.RawURLEncoding.DecodeString(p.Sig)
	if err != nil {
		return fmt.Errorf("pointer: bad sig b64: %w", err)
	}
	toVerify, err := pointerSignedBytes(p)
	if err != nil {
		return err
	}
	if !ed25519.Verify(pub, toVerify, sig) {
		return fmt.Errorf("pointer: signature invalid")
	}
	return nil
}

// encodePointerCompact returns "board:v1:<base64url(json)>"
func encodePointerCompact(p Pointer) (string, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return "board:v1:" + base64.RawURLEncoding.EncodeToString(raw), nil
}

// parsePointer accepts any of the three forms and returns a verified Pointer.
func parsePointer(s string) (Pointer, error) {
	s = strings.TrimSpace(s)
	var raw []byte
	if strings.HasPrefix(s, "board:v1:") {
		b64 := strings.TrimPrefix(s, "board:v1:")
		dec, err := base64.RawURLEncoding.DecodeString(b64)
		if err != nil {
			return Pointer{}, fmt.Errorf("pointer: bad base64: %w", err)
		}
		raw = dec
	} else if strings.HasPrefix(s, "{") {
		raw = []byte(s)
	} else {
		return Pointer{}, fmt.Errorf("pointer: unrecognized format (expected board:v1:..., {...}, or a URL handled elsewhere)")
	}
	var p Pointer
	if err := json.Unmarshal(raw, &p); err != nil {
		return Pointer{}, fmt.Errorf("pointer: parse: %w", err)
	}
	if err := verifyPointer(p); err != nil {
		return Pointer{}, err
	}
	return p, nil
}

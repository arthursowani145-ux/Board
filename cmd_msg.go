package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
)

// resolveContact takes a fingerprint, full pubkey, or a substring of
// either and returns the full pubkey of a followed contact.
// This is the "phone book lookup" that messaging depends on.
func resolveContact(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("msg: empty contact")
	}
	follows, err := listFollows()
	if err != nil {
		return "", err
	}
	if len(follows) == 0 {
		return "", fmt.Errorf("msg: you follow no one; use 'board follow' first")
	}
	// Exact match on full pubkey
	for _, f := range follows {
		if f.Pubkey == input {
			return f.Pubkey, nil
		}
	}
	// Exact match on fingerprint
	for _, f := range follows {
		pub, err := decodePubkey(f.Pubkey)
		if err != nil {
			continue
		}
		if fingerprint(pub) == input {
			return f.Pubkey, nil
		}
	}
	// Substring match on either (case-insensitive-ish, but base64 is exact)
	var matches []string
	for _, f := range follows {
		pub, err := decodePubkey(f.Pubkey)
		if err != nil {
			continue
		}
		fp := fingerprint(pub)
		if strings.Contains(fp, input) || strings.Contains(f.Pubkey, input) {
			matches = append(matches, f.Pubkey)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("msg: %q matches %d contacts; be more specific", input, len(matches))
	}
	return "", fmt.Errorf("msg: no followed contact matches %q", input)
}

// recipientPubkey parses and validates a full pubkey string.
func recipientPubkey(s string) (ed25519.PublicKey, error) {
	return decodePubkey(s)
}

// b64 encodes a ciphertext blob for storage in Post.Content.
func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// b64dec is the inverse.
func b64dec(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

func cmdMsg(args []string) {
	if len(args) < 2 {
		fatal(fmt.Errorf("usage: board msg <contact> <text...>"))
	}
	contactInput := args[0]
	text := strings.Join(args[1:], " ")
	if strings.TrimSpace(text) == "" {
		fatal(fmt.Errorf("msg: empty message"))
	}

	// Look up the recipient in our phone book (follows).
	recipientB64, err := resolveContact(contactInput)
	if err != nil {
		fatal(err)
	}
	recipientPub, err := recipientPubkey(recipientB64)
	if err != nil {
		fatal(err)
	}

	// Encrypt the message to the recipient.
	ciphertext, err := EncryptTo(recipientPub, []byte(text))
	if err != nil {
		fatal(fmt.Errorf("msg: encrypt: %w", err))
	}

	// Load our identity and log.
	priv, err := loadOrCreateIdentity()
	if err != nil {
		fatal(err)
	}
	posts, err := readMyLog()
	if err != nil {
		fatal(err)
	}

	// Build the message post. Title is empty; content is the ciphertext.
	p, err := newPost(posts, "", "msg", "")
	if err != nil {
		fatal(err)
	}
	p.To = recipientB64
	p.Content = b64(ciphertext)

	signed, err := appendPost(priv, p)
	if err != nil {
		fatal(err)
	}

	// Self-verify the whole log after append, same as publish does.
	pub := priv.Public().(ed25519.PublicKey)
	all, _ := readMyLog()
	if err := verifyLog(pub, all); err != nil {
		fatal(fmt.Errorf("post-append self-verify failed: %w", err))
	}

	fmt.Printf("msg sent to %s (seq %d)\n", contactInput, signed.Seq)
}

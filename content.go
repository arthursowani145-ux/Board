package main

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
)

// publishContentPost is the shared "publish a post that carries a piece
// of verified content" path used by board push (projects) and board
// send (files). It:
//   - loads our identity
//   - reads the current log
//   - builds a new post with the given title, type, link, and digest
//   - appends and self-verifies
//   - prints a JSON summary
//
// The caller is responsible for having already placed the content at
// a URL that the "link" field points at (e.g. /<name>.bundle for a
// project, /<name> for a file). The content must already be under the
// public dir so board serve will expose it.
func publishContentPost(name, contentType, link, digest string, extra map[string]interface{}) (Post, error) {
	priv, err := loadOrCreateIdentity()
	if err != nil {
		return Post{}, err
	}
	posts, err := readMyLog()
	if err != nil {
		return Post{}, err
	}

	p, err := newPost(posts, name, contentType, link)
	if err != nil {
		return Post{}, err
	}
	p.Digest = digest

	signed, err := appendPost(priv, p)
	if err != nil {
		return Post{}, err
	}

	// Self-verify the whole log so a broken write is caught immediately.
	pubKey := priv.Public().(ed25519.PublicKey)
	all, _ := readMyLog()
	if err := verifyLog(pubKey, all); err != nil {
		return Post{}, fmt.Errorf("post-append self-verify failed: %w", err)
	}

	// Build the JSON summary. Start with the common fields, then merge
	// any content-specific ones.
	out := map[string]interface{}{
		"name":   name,
		"type":   contentType,
		"seq":    signed.Seq,
		"digest": digest,
		"link":   link,
	}
	for k, v := range extra {
		out[k] = v
	}
	enc, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(enc))

	return signed, nil
}

package main

import (
	"bufio"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Post struct {
	Seq   int64  `json:"seq"`
	Prev  string `json:"prev"`
	Title string `json:"title"`
	Type  string `json:"type"`
	Link  string `json:"link"`
	TS    string `json:"ts"`
	Sig   string `json:"sig"`
}

func myLogPath() (string, error) {
	d, err := boardDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "posts.log"), nil
}

func readMyLog() ([]Post, error) {
	p, err := myLogPath()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var posts []Post
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var p Post
		if err := json.Unmarshal([]byte(line), &p); err != nil {
			return nil, fmt.Errorf("log: corrupt line: %w", err)
		}
		posts = append(posts, p)
	}
	return posts, sc.Err()
}

func signedBytes(p Post) ([]byte, error) {
	p.Sig = ""
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return CanonicalizeJSON(raw)
}

func appendPost(priv ed25519.PrivateKey, p Post) (Post, error) {
	toSign, err := signedBytes(p)
	if err != nil {
		return Post{}, err
	}
	sig := ed25519.Sign(priv, toSign)
	p.Sig = base64.RawURLEncoding.EncodeToString(sig)

	line, err := json.Marshal(p)
	if err != nil {
		return Post{}, err
	}
	lp, err := myLogPath()
	if err != nil {
		return Post{}, err
	}
	f, err := os.OpenFile(lp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm)
	if err != nil {
		return Post{}, err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return Post{}, err
	}
	return p, nil
}

func hashLine(p Post) (string, error) {
	b, err := signedBytes(p)
	if err != nil {
		return "", err
	}
	h := sha256Sum(b)
	return fmt.Sprintf("%x", h[:]), nil
}

func newPost(posts []Post, title, typ, link string) (Post, error) {
	var seq int64 = 0
	var prev string = ""
	if len(posts) > 0 {
		last := posts[len(posts)-1]
		seq = last.Seq + 1
		h, err := hashLine(last)
		if err != nil {
			return Post{}, err
		}
		prev = h
	}
	return Post{
		Seq:   seq,
		Prev:  prev,
		Title: title,
		Type:  typ,
		Link:  link,
		TS:    time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func verifyLog(pub ed25519.PublicKey, posts []Post) error {
	var prevHash string
	for i, p := range posts {
		if p.Seq != int64(i) {
			return fmt.Errorf("verify: post %d: seq=%d (want %d)", i, p.Seq, i)
		}
		if p.Prev != prevHash {
			return fmt.Errorf("verify: post %d: prev=%q (want %q)", i, p.Prev, prevHash)
		}
		sig, err := base64.RawURLEncoding.DecodeString(p.Sig)
		if err != nil {
			return fmt.Errorf("verify: post %d: bad sig b64: %w", i, err)
		}
		toVerify, err := signedBytes(p)
		if err != nil {
			return fmt.Errorf("verify: post %d: %w", i, err)
		}
		if !ed25519.Verify(pub, toVerify, sig) {
			return fmt.Errorf("verify: post %d: signature invalid", i)
		}
		h, err := hashLine(p)
		if err != nil {
			return err
		}
		prevHash = h
	}
	return nil
}

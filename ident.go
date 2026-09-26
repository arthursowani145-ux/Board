package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

const (
	dirPerm  = 0700
	filePerm = 0600
)

func boardDir() (string, error) {
	if v := os.Getenv("BOARD_HOME"); v != "" {
		if err := os.MkdirAll(v, dirPerm); err != nil {
			return "", err
		}
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(home, ".board")
	if err := os.MkdirAll(d, dirPerm); err != nil {
		return "", err
	}
	return d, nil
}

func identityPath() (string, error) {
	d, err := boardDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "identity.pem"), nil
}

func loadOrCreateIdentity() (ed25519.PrivateKey, error) {
	p, err := identityPath()
	if err != nil {
		return nil, err
	}
	if data, err := os.ReadFile(p); err == nil {
		block, _ := pem.Decode(data)
		if block == nil || block.Type != "BOARD ED25519 PRIVATE KEY" {
			return nil, fmt.Errorf("identity: bad PEM")
		}
		if len(block.Bytes) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("identity: wrong key size")
		}
		return ed25519.PrivateKey(block.Bytes), nil
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	block := &pem.Block{Type: "BOARD ED25519 PRIVATE KEY", Bytes: priv}
	if err := os.WriteFile(p, pem.EncodeToMemory(block), filePerm); err != nil {
		return nil, err
	}
	return priv, nil
}

func pubkeyB64(pub ed25519.PublicKey) string {
	return base64.RawURLEncoding.EncodeToString(pub)
}

func decodePubkey(s string) (ed25519.PublicKey, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("ident: bad pubkey size")
	}
	return ed25519.PublicKey(b), nil
}

func fingerprint(pub ed25519.PublicKey) string {
	h := sha256Sum(pub)
	s := base64.RawURLEncoding.EncodeToString(h[:8])
	var out []byte
	for i := 0; i < len(s); i++ {
		if i > 0 && i%4 == 0 {
			out = append(out, '-')
		}
		out = append(out, s[i])
	}
	return string(out)
}

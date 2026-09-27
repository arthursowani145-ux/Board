package main

import (
"crypto/ed25519"
"crypto/rand"
"crypto/sha512"
"fmt"

"filippo.io/edwards25519"
"golang.org/x/crypto/nacl/box"
)

func Ed25519ToX25519(priv ed25519.PrivateKey, pub ed25519.PublicKey) (xPriv [32]byte, xPub [32]byte, err error) {
if len(priv) != ed25519.PrivateKeySize {
return xPriv, xPub, fmt.Errorf("crypto: bad ed25519 private key size %d", len(priv))
}
if len(pub) != ed25519.PublicKeySize {
return xPriv, xPub, fmt.Errorf("crypto: bad ed25519 public key size %d", len(pub))
}

seed := priv.Seed()
h := sha512.Sum512(seed)
copy(xPriv[:], h[:32])
xPriv[0] &= 248
xPriv[31] &= 127
xPriv[31] |= 64

edPoint, err := new(edwards25519.Point).SetBytes(pub)
if err != nil {
return xPriv, xPub, fmt.Errorf("crypto: decode ed25519 pubkey: %w", err)
}
copy(xPub[:], edPoint.BytesMontgomery())
return xPriv, xPub, nil
}

func EncryptTo(recipient ed25519.PublicKey, plaintext []byte) ([]byte, error) {
recipPoint, err := new(edwards25519.Point).SetBytes(recipient)
if err != nil {
return nil, fmt.Errorf("crypto: decode recipient pubkey: %w", err)
}
var recipX [32]byte
copy(recipX[:], recipPoint.BytesMontgomery())

ephPub, ephPriv, err := box.GenerateKey(rand.Reader)
if err != nil {
return nil, err
}
var nonce [24]byte
if _, err := rand.Read(nonce[:]); err != nil {
return nil, err
}
sealed := box.Seal(nil, plaintext, &nonce, &recipX, ephPriv)

out := make([]byte, 0, 32+24+len(sealed))
out = append(out, ephPub[:]...)
out = append(out, nonce[:]...)
out = append(out, sealed...)
return out, nil
}

func DecryptFrom(recipientPriv ed25519.PrivateKey, recipientPub ed25519.PublicKey, blob []byte) ([]byte, error) {
if len(blob) < 32+24+box.Overhead {
return nil, fmt.Errorf("crypto: blob too short (%d bytes)", len(blob))
}
xPriv, _, err := Ed25519ToX25519(recipientPriv, recipientPub)
if err != nil {
return nil, err
}
var ephPub [32]byte
copy(ephPub[:], blob[:32])
var nonce [24]byte
copy(nonce[:], blob[32:56])
plaintext, ok := box.Open(nil, blob[56:], &nonce, &ephPub, &xPriv)
if !ok {
return nil, fmt.Errorf("crypto: decryption failed (bad key or corrupted ciphertext)")
}
return plaintext, nil
}

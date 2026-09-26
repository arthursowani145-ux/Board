package main

import (
	"crypto/ed25519"
	"fmt"
)

func cmdID() {
	priv, err := loadOrCreateIdentity()
	if err != nil {
		fatal(err)
	}
	pub := priv.Public().(ed25519.PublicKey)
	fmt.Println(pubkeyB64(pub))
	fmt.Println("fingerprint:", fingerprint(pub))
}

package main

import (
	"crypto/ed25519"
	"fmt"
)

func cmdInit() {
	priv, err := loadOrCreateIdentity()
	if err != nil {
		fatal(err)
	}
	pub := priv.Public().(ed25519.PublicKey)
	fmt.Println("identity ready")
	fmt.Println("pubkey:     ", pubkeyB64(pub))
	fmt.Println("fingerprint:", fingerprint(pub))
	fmt.Println()
	fmt.Println("Read the fingerprint aloud to a friend to verify it is really you.")
	fmt.Println("Store it somewhere safe. If you lose ~/.board/identity.pem, you lose your identity.")
}

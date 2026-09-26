package main

import (
	"crypto/sha256"
	"fmt"
	"os"
)

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func sha256Sum(b []byte) [32]byte {
	return sha256.Sum256(b)
}

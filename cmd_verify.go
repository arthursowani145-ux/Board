package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func cmdVerify(args []string) {
	if len(args) != 2 {
		fatal(fmt.Errorf("usage: board verify <logfile> <pubkey-b64>"))
	}
	logfile, pubB64 := args[0], args[1]

	pub, err := decodePubkey(pubB64)
	if err != nil {
		fatal(err)
	}
	f, err := os.Open(logfile)
	if err != nil {
		fatal(err)
	}
	defer f.Close()

	var posts []Post
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var p Post
		if err := json.Unmarshal([]byte(line), &p); err != nil {
			fatal(fmt.Errorf("parse: %w", err))
		}
		posts = append(posts, p)
	}
	if err := sc.Err(); err != nil {
		fatal(err)
	}

	if err := verifyLog(pub, posts); err != nil {
		fatal(err)
	}
	fmt.Printf("OK: %d posts, chain intact, all signatures valid\n", len(posts))
	fmt.Println("fingerprint:", fingerprint(pub))
}

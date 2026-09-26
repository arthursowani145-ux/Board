package main

import (
	"crypto/ed25519"
	"encoding/json"
	"flag"
	"fmt"
)

func cmdPublish(args []string) {
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	title := fs.String("title", "", "post title")
	typ := fs.String("type", "", "post type (e.g. skill, video, note)")
	link := fs.String("link", "", "link (mega:, https:, etc.)")
	fs.Parse(args)

	if *title == "" || *typ == "" || *link == "" {
		fatal(fmt.Errorf("--title, --type, and --link are required"))
	}

	priv, err := loadOrCreateIdentity()
	if err != nil {
		fatal(err)
	}
	posts, err := readMyLog()
	if err != nil {
		fatal(err)
	}
	p, err := newPost(posts, *title, *typ, *link)
	if err != nil {
		fatal(err)
	}
	signed, err := appendPost(priv, p)
	if err != nil {
		fatal(err)
	}

	pub := priv.Public().(ed25519.PublicKey)
	all, _ := readMyLog()
	if err := verifyLog(pub, all); err != nil {
		fatal(fmt.Errorf("post-append self-verify failed: %w", err))
	}
	out, _ := json.Marshal(signed)
	fmt.Println(string(out))
}

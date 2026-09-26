package main

import (
	"encoding/json"
	"flag"
	"fmt"
)

func cmdPointer(args []string) {
	fs := flag.NewFlagSet("pointer", flag.ExitOnError)
	form := fs.String("form", "b64", "output form: json | b64 | url")
	fs.Parse(args)

	priv, err := loadOrCreateIdentity()
	if err != nil {
		fatal(err)
	}
	posts, err := readMyLog()
	if err != nil {
		fatal(err)
	}
	mirrors, err := loadMirrors()
	if err != nil {
		fatal(err)
	}
	p, err := buildPointer(priv, posts, mirrors)
	if err != nil {
		fatal(err)
	}

	switch *form {
	case "json":
		out, _ := json.MarshalIndent(p, "", "  ")
		fmt.Println(string(out))
	case "b64":
		s, err := encodePointerCompact(p)
		if err != nil {
			fatal(err)
		}
		fmt.Println(s)
	case "url":
		if len(p.Mirrors) == 0 {
			fatal(fmt.Errorf("pointer: no mirrors set; use 'board mirror add <url>' first"))
		}
		fmt.Println(p.Mirrors[0])
	default:
		fatal(fmt.Errorf("unknown form: %s", *form))
	}
}

package main

import (
	"fmt"
)

func cmdMirror(args []string) {
	if len(args) < 1 {
		fatal(fmt.Errorf("usage: board mirror add|list|remove [url]"))
	}
	switch args[0] {
	case "add":
		if len(args) != 2 {
			fatal(fmt.Errorf("usage: board mirror add <url>"))
		}
		if err := addMirror(args[1]); err != nil {
			fatal(err)
		}
		fmt.Println("mirror added:", args[1])
	case "list":
		mirrors, err := loadMirrors()
		if err != nil {
			fatal(err)
		}
		if len(mirrors) == 0 {
			fmt.Println("(no mirrors)")
			return
		}
		for _, m := range mirrors {
			fmt.Println(m)
		}
	case "remove":
		if len(args) != 2 {
			fatal(fmt.Errorf("usage: board mirror remove <url>"))
		}
		if err := removeMirror(args[1]); err != nil {
			fatal(err)
		}
		fmt.Println("mirror removed:", args[1])
	default:
		fatal(fmt.Errorf("unknown mirror subcommand: %s", args[0]))
	}
}

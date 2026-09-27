package main

import (
	"flag"
	"fmt"
)

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", defaultPort, "TCP port to listen on")
	fs.Parse(args)

	if err := runServe(*port); err != nil {
		fatal(fmt.Errorf("serve: %w", err))
	}
}

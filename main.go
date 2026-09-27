package main

import (
	"fmt"
	"os"
)

func usage() {
	fmt.Fprintf(os.Stderr, `board — signed append-only log for Termux

Usage:
  board init                    generate identity
  board id                      show pubkey + fingerprint
  board publish --title T --type K --link L
                                append a signed post
  board export                  print full signed log to stdout
  board verify <logfile> <pubkey-b64>
                                verify a signed log
  board mirror add <url>        add a storage location for your log
  board mirror list             list mirrors
  board mirror remove <url>     remove a mirror
  board pointer [--form F]      print your current pointer (F: json|b64|url)
  board msg <contact> <text>     send an encrypted message to a contact
  board inbox                   read your encrypted messages
  board push <dir>              publish a git project as a signed bundle
  board pull <contact>/<proj>   fetch a project bundle and clone it
  board serve [--port N]        listen for peers (default port 8848)
  board peers                   find other board instances on this network
  board follow <pubkey> <ptr>   follow someone (ptr: board:v1:..., {...}, or URL)
  board follows                 list who you follow
  board fetch [<pubkey>]        fetch and verify logs (all follows, or one)
  board feed                    read cache, print newest posts first

`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "init":
		cmdInit()
	case "id":
		cmdID()
	case "publish":
		cmdPublish(os.Args[2:])
	case "msg":
		cmdMsg(os.Args[2:])
	case "inbox":
		cmdInbox(os.Args[2:])
	case "push":
		cmdPush(os.Args[2:])
	case "pull":
		cmdPull(os.Args[2:])
	case "serve":
		cmdServe(os.Args[2:])
	case "peers":
		cmdPeers(os.Args[2:])
	case "export":
		cmdExport()
	case "verify":
		cmdVerify(os.Args[2:])
	case "mirror":
		cmdMirror(os.Args[2:])
	case "pointer":
		cmdPointer(os.Args[2:])
	case "follow":
		cmdFollow(os.Args[2:])
	case "follows":
		cmdFollows()
	case "fetch":
		cmdFetch(os.Args[2:])
	case "feed":
		cmdFeed(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

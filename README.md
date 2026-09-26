# board

A signed, append-only log for Termux. No server. No account. No company.

Each post is signed with an Ed25519 key that never leaves your device.
Each post commits to the hash of the one before it, forming a chain.
Anyone with your public key can verify the whole log offline.

## Why

Collaboration between people who already know each other shouldn't require
a platform in the middle. No signup, no rate limit, no terms of service,
no fear that the thing you depend on gets shut down or sold.

## Install

```sh
pkg install golang git
git clone https://github.com/arthursowani145-ux/Board.git
cd Board
go build -o board .
./board init

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
board follow <pubkey> <ptr>   follow someone
board follows                 list who you follow
board fetch [<pubkey>]        fetch and verify logs
board feed                    read cache, print newest first

How it works

Identity. An Ed25519 keypair, generated at board init. Your public
key is your address. Your fingerprint is what you read aloud to a friend
on a voice call to verify it's really you.

Posts. Signed, hash-chained. Each post commits to the hash of the
previous post's signed bytes. Delete a post in the middle, reorder them,
or edit any field, and verification fails. Proven by test/mid_deletion.sh.

Pointers. A small signed JSON blob that commits to your log's current
(seq, head) and lists the mirrors where your log lives. This is the
thing you share. It fits in an SMS or a QR code.

Mirrors. Your log can live anywhere that serves bytes over HTTP — a
gist, a MEGA share, an IPFS gateway, a friend's server. The integrity is
in the signature and the chain, not in the host. Any mirror that serves
a truncated log is caught, because the pointer commits to a head hash
the truncated log can't reach.

Fetch. Pull a log from a mirror, verify every signature, walk the
chain, check the head matches the pointer's commitment. If any step
fails, the log is rejected and the old cache is kept.

Feed. Read the verified cache. Newest-first. Offline. No network call.

What it protects against

· Forgery — a post can't be attributed to a key that didn't sign it
· Content tampering — any edit invalidates the signature
· Splicing within a log — insert, delete, reorder all break the chain
· Truncation, given a fresh pointer — a shortened log can't reach the head
· Third-party truncation — a hostile mirror can't show a shorter valid log

What it does not protect against

· The author showing different histories to different followers
  (equivocation — same problem Certificate Transparency needs gossip for)
· Stale prefixes with no freshness signal
· Compromised device, loss of identity, metadata, or a malicious peer
  inside a collaborative session

See THREAT_MODEL.md for the full statement.

Status

Prototype. Two instances on one phone pass end-to-end via
test/two_instance.sh. Two physical devices over a real network is the
next milestone.

Tests

```sh
go test ./...
./test/mid_deletion.sh
./test/two_instance.sh
```

License

MIT. See LICENSE.

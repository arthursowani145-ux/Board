# board — Threat Model

This document states what `board` protects against, what it does not,
and where the boundary is. It is deliberately short and deliberately
honest. If a claim isn't here, the software doesn't make it.

## What board is

An append-only, cryptographically signed log. Each post is signed by
its author's Ed25519 key. Each post commits to the hash of the previous
post, forming a chain. Anyone with the author's public key can verify
the entire log offline.

## What board defends against

1. **Forgery.** A post cannot be attributed to a key that didn't sign it.
   Ed25519 signatures, verified on every read.

2. **Tampering with content.** A published post cannot be edited after
   the fact. Any change to any field invalidates the signature.

3. **Splicing within a log.** A post cannot be inserted, deleted, or
   reordered *within* a log without breaking the `prev`-hash chain.
   Proven by test (see `test/mid_deletion.sh`).

4. **Truncation from the end, given a fresh pointer.** If a follower
   has seen a pointer at `(seq=N, head=H)` and later fetches a log that
   doesn't reach `(N, H)`, the mismatch is detectable. The pointer
   commits to both sequence number and head hash.

5. **Third-party truncation.** Someone who controls the storage where
   a log lives (a mirror, a MEGA share, an IPFS gateway) cannot show a
   follower a valid-looking *shorter* log and have it pass, *provided*
   the follower has a recent pointer.

## What board does NOT defend against

1. **The author showing different histories to different followers.**
   This is equivocation. It is the same problem Certificate Transparency
   needs gossip + monitors to solve. `board` does not solve it. A user
   who wants this property must compare notes with another follower
   out-of-band.

2. **Stale prefixes with no freshness signal.** If Alice hands Bob
   posts 0-2 of a 5-post log, and Bob has never seen a pointer past
   seq 2, Bob cannot tell. A signed prefix is a valid log. This is
   structural to append-only logs without consensus or gossip.

3. **Compromised device.** If the phone is rooted, has malware, or an
   attacker has physical access, keys and plaintext are exposed.
   Nothing at the software layer defends against this.

4. **Loss of identity.** `~/.board/identity.pem` is the identity. Lose
   it, lose the ability to sign as that identity. There is no recovery,
   no reset, no support to contact. This is intentional.

5. **Metadata.** That two IPs connected, when, and how much data moved
   is visible to a network observer. The relay design (v0.2) minimizes
   this but does not eliminate it.

6. **The peer inside a session.** In a collaborative session, the other
   person is trusted by definition. board is not a defense against a
   collaborator who is malicious.

## The decision, stated plainly

For a personal, pull-based feed between people who already know each
other, truncation-by-a-third-party is the threat worth defeating, and
the pointer mechanism defeats it. Equivocation-by-the-author is a real
threat but a rarer one, and defending against it requires infrastructure
that board does not have and does not want to require.

We choose to leave it unsolved, and to say so, rather than to imply a
guarantee the software does not provide.

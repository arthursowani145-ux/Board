# 📖 board — the manual

**You and your peer both have Termux on your phones. You want to send each other files, projects, and private messages — without going through GitHub, WhatsApp, Telegram, or any other company.**

This manual walks through every command, in order, with the actual output you'll see. It starts with two phones on the same WiFi, and ends with two phones on different continents. Everything in between is one command at a time.

---

## 🎯 What board is, in three lines

- A **signed log** that lives on your phone. Every post you make is signed with a key that never leaves the device.
- A way to **follow** your peer's log and fetch it from wherever they store it — no server of yours in the middle.
- Three things you can do with it: **send files, share projects, and exchange encrypted messages** — all verifiable, all without a company in the loop.

If you want the deep design, read [`THREAT_MODEL.md`](THREAT_MODEL.md). If you want to *use* it, keep reading.

---

## 🛠️ Install

You need Termux. If you're here, you have it. Open Termux and run:

```bash
pkg install golang git
git clone https://github.com/arthursowani145-ux/Board.git
cd Board
go build -o board .
The go build step takes a minute or two the first time. No output means it worked. You now have a board binary in the Board/ directory.

> 💡 **Tip:** If you want to run board from anywhere without ./board, move it to Termux's bin: mv board $PREFIX/bin/board. Then just board works.

Test that it built:

```bash
./board help
```

You should see the full command list. If you do, you're ready.

---

## 🔑 Your identity

Every board install generates its own identity the first time you run it. Your identity is an Ed25519 key pair — a private key that stays on your device, and a public key that you share.

Create your identity:

```bash
./board init
```

You'll see something like:

```
identity ready
pubkey:      RI11CL95J__kT6WbQi3YUrkt7NTcMxuvSFQ3GSXvvaY
fingerprint: jhft-IZR3-M3E

Read the fingerprint aloud to a friend to verify it's really you.
Store it somewhere safe. If you lose ~/.board/identity.pem, you lose your identity.
```

Two things to notice:

- The pubkey is your address. It's long, but it's what identifies you to other board users.
- The fingerprint is a short, human-readable version — jhft-IZR3-M3E above. This is what you'd read aloud to your peer on a phone call to confirm it's really you, and what they'd type when they want to send you something.

Save the fingerprint. Write it down. Send it to your peer over whatever channel you trust. They'll need it, and you'll need theirs.
> ⚠️ Your private key is at ~/.board/identity.pem. If you lose it, you lose your identity. There's no recovery, no reset, no account to contact. Back up the file if you care about the identity persisting. This is the same tradeoff as a physical key: nobody can reset it for you, and nobody can take it from you.

To see your identity again at any time:

```bash
./board id
```

That's it for setup. You have a key. Your peer has a key. Now you need to find each other.

---


## 📡 Finding each other on the same WiFi

The easiest way to use `board` is when you and your peer are on the same network — the same WiFi, or one phone's hotspot that the other has joined.

The flow is four commands total. Two on each phone.

### On both phones: start serving

`board serve` is a small HTTP server that exposes your log and any files you've published. It runs in the foreground, so open a Termux session for it on each phone and leave it alone.

**On your phone:**

```bash
./board serve
On your peer's phone:

```bash
./board serve
```

Each prints a banner like:

```
board serve
  fingerprint: jhft-IZR3-M3E
  pubkey:      RI11CL95J__kT6WbQi3YUrkt7NTcMxuvSFQ3GSXvvaY
  serving:     /data/data/com.termux/files/home/board-public
  listening:   :8848

Local mirror: http://192.168.8.109:8848/posts.ndjson
Discovery:    http://192.168.8.109:8848/board/hello
Ctrl+C to stop.
```

The Local mirror URL is important — that's where your peer will fetch your log from, and it's automatically registered in your mirror list so any pointer you generate carries it.

Leave both servers running. Open a second Termux session on each phone for the next steps.

On one phone: find the other

In a fresh Termux session, run:

```bash
./board peers
```

This scans your local network (the /24 subnet, port 8848) for other board instances. After a couple of seconds, you'll see:

```
scanning 192.168.8.0/24 on port 8848 ...

found 1 peer(s):

  192.168.8.102    CWPX-LJQN-0Hc  seq=0
```

That's your peer. The columns are:

- Their IP address on the local network
- Their fingerprint — the short identity string from board id
- The current state of their log — seq=0 means they haven't published anything yet

Verify the fingerprint matches what they told you. If it doesn't, something is wrong and you should not follow them. The fingerprint is your peer's cryptographic identity; the IP is just where they happen to be reachable from right now.

Follow them — one command

Now add them to your follow list and update their pointer in one shot:

```bash
./board peers --follow
```

Same output as before, but with a marker at the end of each line:

```
  192.168.8.102    CWPX-LJQN-0Hc  seq=0  + followed
```

+ followed means they're now in your contact list. From this point on, they're a contact — board msg, board send, and board push all accept their fingerprint (CWPX-LJQN-0Hc) or their full pubkey.

If you run board peers --follow again, you'll see:

```
  192.168.8.102    CWPX-LJQN-0Hc  seq=0  (already followed)
```

That's idempotency — the command is safe to run over and over.

Now do it the other way

On your peer's phone, in a second session, run the same thing:

```bash
./board peers --follow
```They should find you and add you. Now you're both in each other's contact lists, and you're ready to actually send things.

Check who you're following

Anytime you want to see your contact list:

```bash
./board follows
```

You'll see one line per contact, with their fingerprint, the last known sequence number of their log, and how many mirrors they have registered. That's the whole state of your follow list.

When you're done for the day

Just Ctrl+C on each phone's board serve. Your contact list, your log, your mirror list, and your identity all persist. Next time you want to use board, start serve again on both and you're back where you left off.

> 💡 **Tip:** If board peers finds nobody and you're sure both phones are on the same WiFi and both servers are running, the network itself might be isolating clients. Many public WiFi networks (hotels, cafés, airports) and some home routers block devices from talking to each other, even on the same network. If that happens, ping <peer-ip> will fail too. There's nothing board can do about a network that doesn't let its own devices talk — you'd need to use the over-the-internet mode instead.

---


## 📤 Sending things to your peer

Once you've followed each other (previous section), you can send three kinds of things:

- **Files** — any file. Scripts, images, PDFs, whatever.
- **Projects** — a git repository, complete with history.
- **Messages** — text, encrypted so only the recipient can read it.

All three go into *your* log. Your peer fetches your log (they're already following you) and picks up the content. **You don't need to coordinate with them in real time** — send whenever, and they'll see it when they next fetch.

> 💡 **Tip:** If you and your peer are on the same WiFi and both running `board serve`, everything below just works. If you're on different networks, you need to set up a mirror first — see the [over-the-internet section](#-over-the-internet) below.

### 📎 Sending a file

Send any file:

```bash
./board send <peer-fingerprint> <path-to-file>

For example, if your peer's fingerprint is CWPX-LJQN-0Hc and you want to send them a script:

```bash
./board send CWPX-LJQN-0Hc ~/downloads/my-script.sh
```

You'll see a summary:

```
{
  "digest": "9a37ff508e339f12441a943175438a4d165a01202800f089aaabb2c240705e6c",
  "link": "/my-script.sh",
  "name": "my-script.sh",
  "path": "/data/data/com.termux/files/home/board-public/my-script.sh",
  "seq": 7,
  "size_bytes": 99410,
  "type": "file"
}
```

**What just happened, step by step:**

1. board read the file and computed its SHA-256 hash — that's the digest above.
2. It copied the file to ~/board-public/, where board serve will expose it.
3. It published a signed post to your log at sequence number 7, containing the filename, a link, and the digest.
4. It verified the post immediately after publishing (that's the self-verify step running silently).

Note the seq number. Yours will be different — whatever your next post index is. That's your log growing.

📥 Receiving a file

On your peer's phone, they need to first fetch your log (which they're following) and then get the file.

Step 1 — fetch new posts from everyone they follow:

```bash
./board fetch
```

They'll see something like:

```
jhft-IZR3-M3E  OK
fetched 1/1
```

That means they pulled your updated log and verified it.

Step 2 — look at what arrived:

```bash
./board feed
```

The new file post appears at the top:

```
2026-09-27T09:05:08Z  [file] my-script.sh
    /my-script.sh
    id: jhft-IZR3-M3E:7
```

Step 3 — download the file:

```bash
./board get <your-fingerprint>/<filename>
```

Continuing the example:

```bash
./board get jhft-IZR3-M3E/my-script.sh
```

Expected output:

```
got my-script.sh from jhft-IZR3-M3E (seq 7)
  digest: 9a37ff508e339f12441a943175438a4d165a01202800f089aaabb2c240705e6c
  saved:  /data/data/com.termux/files/home/downloads/my-script.sh
  bytes:  99410
```
The digest matches what you signed. That's the important part — board get downloads the file from wherever it's hosted (your peer's server, in this case), computes its SHA-256, and refuses to write it if the hash doesn't match what was in the signed post. If anything altered the file in transit, the download fails loudly.

The file is saved to ~/downloads/<filename>. To save somewhere else, add --into <dir>:

```bash
./board get jhft-IZR3-M3E/my-script.sh --into ~/scripts/
```

📦 Sending a project

A project is a git repository. Sending one is nearly identical to sending a file, but it packs the entire git history — branches, commits, everything — into a single bundle.

Point board send at a directory instead of a file:

```bash
./board send CWPX-LJQN-0Hc ~/code/my-project
```

If ~/code/my-project/ is a git repository (has a .git/ directory), board send detects it, bundles it, and publishes a type=project post.

You'll see:

```
{
  "bundle": "/data/data/com.termux/files/home/board-public/my-project.bundle",
  "digest": "c5d9d3d0f1ca8f873c53a51e6fb6b6ae4069968ec10e0cba2449855a0bd66d9c",
  "link": "/my-project.bundle",
  "name": "my-project",
  "seq": 8,
  "size_bytes": 591,
  "type": "project"
}
```

Note: board send and board push are the same operation. board push ~/code/my-project does exactly the same thing. Use whichever feels natural.

📥 Receiving a project

Same as receiving a file, but the post type is project:

```bash
./board fetch
./board pull <your-fingerprint>/<project-name>
```

For example:

```bash
./board pull jhft-IZR3-M3E/my-project
```

Expected:

```
pulled my-project from jhft-IZR3-M3E (seq 8)
  digest: c5d9d3d0f1ca8f873c53a51e6fb6b6ae4069968ec10e0cba2449855a0bd66d9c
  into:   /data/data/com.termux/files/home/code/my-project
```

The project is cloned into ~/code/<name>/. It's a full git repository — you can cd into it, git log, git diff, everything. The commit history is identical to the sender's because git bundles are content-addressed.

If a directory with that name already exists, board pull refuses — it won't overwrite. Use --into <dir> to choose a different destination:

```bash
./board pull jhft-IZR3-M3E/my-project --into ~/code/my-project-from-you
```

✉️ Sending a message

Messages are text, encrypted to the recipient's identity. Only they can read them — anyone else following you sees a [msg] post with ciphertext, but nothing readable.

```bash
./board msg <peer-fingerprint> "your message text here"
```

For example:

```bash
./board msg CWPX-LJQN-0Hc "hey, checking that the WiFi test is working"
```

You'll see:

```
msg sent to CWPX-LJQN-0Hc (seq 9)
```

That's it — the message is in your log, encrypted to your peer's identity.

📬 Reading your inbox

On your peer's phone, messages arrive via the same fetch they'd run anyway:


```bash
./board fetch
./board inbox
```

Expected:

```
2026-09-27T09:20:11Z  from jhft-IZR3-M3E  (seq 9)
    hey, checking that the WiFi test is working
    id: jhft-IZR3-M3E:9
```

board inbox scans all cached posts from all your follows, filters for messages addressed to your pubkey, decrypts each with your private key, and shows them newest-first.

Anyone else following you does not see this. They see the post exists — [msg] in their feed — but the content is ciphertext. Only the intended recipient can decrypt it.

🔄 Sending back

Your peer can reply the same way. Their phone:

```bash
./board msg RI11CL95J__kT6WbQi3YUrkt7NTcMxuvSFQ3GSXvvaY "got it, works here too"
```

Then on your phone:

```bash
./board fetch
./board inbox
```

That's a full round-trip: encrypted message from A to B, reply encrypted from B to A, both fetched from live logs, both verified.

---


## 📰 Reading your feed

`board feed` is what you run when you want to see everything that's arrived. It reads from your local cache — no network call — so it's instant, and it works offline.

```bash
./board feed
You'll see every post from every peer you follow, newest first:

```
2026-09-27T09:20:11Z  [msg]
    id: jhft-IZR3-M3E:9
2026-09-27T09:05:08Z  [file] my-script.sh
    /my-script.sh
    id: jhft-IZR3-M3E:7
2026-09-27T08:32:41Z  [msg]
    id: jhft-IZR3-M3E:6
2026-09-27T04:25:11Z  [project] my-project
    /my-project.bundle
    id: jhft-IZR3-M3E:8
2026-09-26T01:26:22Z  [note] hello world
    mega:something
    id: jhft-IZR3-M3E:0
```

Each line has:

- The timestamp — when the post was created
- The type — file, project, msg, note, or anything else the author chose
- The title — the post's name (for messages, this is blank)
- The link — where the content lives (URL path, or an external URL)
- The post ID — <fingerprint>:<seq> — this is what you reference when you want to get a specific post

The post types

note — a plain signed post. Just text and a link. Useful for announcements, links to things you want to share, or anything that doesn't fit another type.

```bash
./board publish --title "check this out" --type note --link "https://example.com"
```

file — a file you've sent with board send. The link is a path served by your board serve. The digest is the file's SHA-256.

project — a git repository you've shared. The link is a .bundle file served by your board serve. Same digest verification.

msg — an encrypted message. The title is blank, the link is empty, the content is ciphertext encrypted to a specific recipient's pubkey.

Following someone's full history

If you follow someone for a while and want to see everything they've ever published, not just what's new:

```bash
./board export
```

That prints your own log. To see a peer's, use board feed and then get the specific post you're interested in:

```bash
./board get <their-fingerprint>/<filename>
```

for files, or

```bash
./board pull <their-fingerprint>/<project-name>
```

for projects.

The post ID

Every post has an ID like jhft-IZR3-M3E:7. The first part is the author's fingerprint, the second is the sequence number of the post in their log.

You can use this ID for a few things:

- Reference a specific post when asking someone about it
- Check that a post exists — board feed | grep <post-id>
- Verify who wrote it — if the fingerprint matches the person you expected, the post is from them

The sequence number is a commitment: jhft-IZR3-M3E:7 means "this is the 8th post from jhft-IZR3-M3E, and it can't be edited after the fact." If the author ever changed post 7, the signature on every post after it would break, because each post commits to the hash of the one before.

**What if something doesn't show up?**

If you're expecting a post and it isn't in board feed:

1. Run board fetch to pull new posts from everyone you follow.
2. Check the fetch output — if a peer fails, it'll say so and why.
3. Check board follows — that lists everyone you follow. If someone's missing, you didn't follow them yet.
4. Check that board serve is running on the other side — if their server is down, you can't fetch their log.

The most common cause is just "I haven't fetched recently." board feed reads from cache; it doesn't reach the network. You have to explicitly fetch to see new posts.

Fetching at a rhythm

There's no built-in scheduler. You fetch when you want new content. If you want it to happen automatically:

- Termux's termux-job-scheduler can run a command periodically. Set it to run board fetch every 15 minutes.
- A simple cron if you have cronie installed: */15 * * * * board fetch.
- Just run it manually. Most people don't need to fetch constantly — the store-and-forward model means messages sit in the log until you pick them up. There's no penalty for being offline.

> 💡 **Tip:** The name of the game is store-and-forward. Your peer can publish whenever they want, and you'll see it whenever you fetch next. Unlike a chat app, there's no "online" state to maintain. You both publish into your own logs, and both logs sit there waiting for whoever wants to read them.

---


## 🌍 Over the internet

Everything so far assumed you and your peer are on the same WiFi. That's the easy case, and it's the one most people will use.

But the design doesn't require it. **Two phones on different networks, anywhere in the world, can use `board` too.** The only thing that has to change is how the log gets from one phone to the other.

On the same WiFi, your phone serves your log and your peer fetches directly. Over the internet, neither phone can reach the other — carrier NAT, firewalls, no shared address. You need a place in the middle that both of you can reach. That's the **mirror**.

**A mirror is just a URL that serves your log over HTTPS.** A GitHub gist, a MEGA share, a pastebin, a VPS, an IPFS gateway, a friend's server — anything that returns the bytes of your log file. The mirror doesn't need to be trusted. It can't read your messages, can't alter your posts, and can't hide them without you noticing. It's a dumb file host.

Here's the flow.

### Both of you: set up a public mirror

Each of you needs to publish your own log somewhere the other can reach.

The easiest option is a **GitHub gist**. If you have a GitHub account:

1. On your phone, export your log: `./board export > posts.ndjson`
2. Go to `gist.github.com` (mobile browser is fine)
3. Create a new gist, name it `posts.ndjson`, paste the contents
4. Click **Create public gist**
5. Click the **Raw** button on the file
6. **Copy the URL from the address bar.** It looks like:
   `https://gist.githubusercontent.com/<user>/<id>/raw/posts.ndjson`

That raw URL is your mirror.

Register it:

```bash
./board mirror add 'https://gist.githubusercontent.com/<user>/<id>/raw/posts.ndjson'
```

Then generate a fresh pointer:

```bash
./board pointer --form b64
```

You'll get a string starting with board:v1:. Copy it and send it to your peer — over SMS, WhatsApp, Telegram, email, a phone call, whatever you already trust. The pointer is not secret. If someone intercepts it, they still can't forge it or read anything it points at that isn't already public. And you can always issue a new one if you're worried.

> 💡 **Tip:** The pointer is signed. If anyone alters it in transit — changes the pubkey, the seq, the mirror URL — verification fails immediately. That's what makes it safe to send over a channel you don't fully trust.

Your peer: follow you

Your peer receives the pointer string. On their phone:

```bash
./board follow -- <your-fingerprint> '<pointer string>'
```

For example:

```bash
./board follow -- jhft-IZR3-M3E 'board:v1:eyJ2IjoxLCJwdWJrZXkiOiJSSTEx...'
```

The -- before the pubkey is important. Your pubkey might start with a dash, and -- tells the shell and board to stop interpreting everything after it as flags. Get in the habit of including it.

If the pointer verifies, they'll see:

```
following jhft-IZR3-M3E (seq 7)
```

Then they fetch:

```bash
./board fetch
```

Expected:

```
jhft-IZR3-M3E  OK
fetched 1/1
```

fetch downloaded your log from the gist mirror, verified every signature, confirmed the chain reaches the head hash the pointer committed to, and cached it. Your peer now has an authenticated copy of your log.

And you follow them

Same thing in reverse. Your peer generates their pointer:

```bash
./board pointer --form b64
```

They send it to you. You run:

```bash
./board follow -- <their-fingerprint> '<their-pointer>'
./board fetch
```

Now you're both following each other, and everything in the previous section works — messages, files, projects — just with the mirror as the intermediate storage instead of direct WiFi.

Keeping the mirror up to date

Here's the one part that's manual: when you publish something new, your log changes, and the mirror needs to know about it.

Two steps after every board msg, board send, or board publish:

1. Re-export your log: ./board export > posts.ndjson
2. Update the gist with the new contents

Then regenerate your pointer and send it to your peer:

```bash
./board pointer --form b64
```

Your peer replaces their follow with the new pointer:

```bash
./board follow --force -- <your-fingerprint> '<new-pointer>'
```

That's the only friction. If both of you update your gists once a day, you effectively have daily mail between two phones, cryptographically verified, with no company holding it.

If you want to automate the update

Two options:

Option A — a VPS or public host. If you have access to a small server, sync your log there on a cron and skip the manual gist update. Serve posts.ndjson from a fixed URL and it never changes.

Option B — a script that does it in one command. If you have gh (GitHub CLI) installed in Termux, you can update the gist from the command line:

```bash
./board export | gh gist edit <gist-id> -f posts.ndjson -
```

Wrap that in a shell function and you've got a one-command publish.

The full flow, in four lines

Once both sides have a mirror and have followed each other, the whole thing is:

```
# on your phone, to send a message
./board msg <their-fingerprint> "hello"
./board export > posts.ndjson   # then update your gist with this

# on their phone, to read it
./board fetch
./board inbox
```

Four commands and a gist update. Same cryptographic guarantees as the same-WiFi case — verified signatures, verified chain, encrypted messages. Just with a slower and less private transport (the mirror operator can see that somebody fetched a log, but not what's in it, and not who wrote it, since the content is signed by you).

The honest limit

A mirror is a shared point of failure. If your gist is deleted, your peer can't fetch anything new. If GitHub is blocked in your peer's country, they can't reach it. That's why the mirror list accepts multiple URLs — you can host the same log in several places and a follower tries each in turn.

To add a second mirror:

```bash
./board mirror add 'https://another-host.com/posts.ndjson'
./board pointer --form b64
```

The new pointer will list both. Your peer re-follows and their fetch tries the first that responds.

No single mirror is required. Any mirror that serves the bytes works. The signature and chain make the mirror just a pipe, not a trusted party.

---


## 🔒 What board does and doesn't do

This is the honest version. If any claim isn't here, the tool doesn't make it. The full statement is in [`THREAT_MODEL.md`](THREAT_MODEL.md); this is the short form for users.

### What it protects against

**Forgery.** Every post is signed with an Ed25519 key that never leaves the device. A post can't be attributed to a key that didn't sign it.

**Tampering with content.** Any change to any field of any post — even one character — invalidates the signature. Editing a published post is impossible; publishing a different one is trivial, but you'll be able to tell.

**Splicing within a log.** A post can't be inserted, deleted, or reordered within a log without breaking the hash chain. Every post commits to the hash of the previous post, so any change to history is detectable by the next reader.

**Truncation, given a fresh pointer.** If you've seen a pointer committing to `(seq=N, head=H)` and later fetch a log that doesn't reach that state, the mismatch is caught. This means a mirror can't show you a shorter log and have it pass as valid.

**Third-party truncation.** A hostile mirror — one controlled by someone who wants to hide your peer's recent posts — can't succeed if the follower has a recent pointer. The pointer commits to the head, and a shorter log won't reach it.

### What it does not protect against

**The author showing different histories to different followers.** This is equivocation. It's the same problem Certificate Transparency needs gossip + monitors to solve. `board` does not solve it. If you want to defend against a peer deliberately showing you and someone else different versions of their log, you have to compare notes with that other person out-of-band.

**Stale prefixes with no freshness signal.** If your peer hands you their first three posts, and you've never seen a pointer past seq 2, you can't tell. A signed prefix is a valid log. There's no way to know more posts exist unless someone tells you.

**A compromised device.** If the phone is rooted, infected, or someone has physical access, keys and content are exposed. Nothing at the software layer defends against that.

**Loss of identity.** Your private key lives in `~/.board/identity.pem`. Lose it, lose the ability to sign as that identity. There is no recovery, no reset, no support to contact. This is intentional.

**Metadata.** That two devices connected, when, and how much data moved is visible to a network observer. The mirror you use can see that *somebody* fetched your log. It can't read the content or forge it, but it can see the pattern of use.

**Malicious collaborators inside a session.** If you're working with someone inside a shared workspace, they're trusted by definition. `board` is not a defense against a collaborator who is doing something harmful.

### The decision, stated plainly

For a personal, pull-based exchange between two people who already know each other, this tradeoff is reasonable. The threats that are solved — forgery, tampering, splicing, truncation — are the ones that matter most. The threats that aren't solved — equivocation, metadata, device compromise — are either rare in the target use case or unsolvable at this layer.

We state this boundary clearly rather than implying a guarantee the software doesn't provide.

---

## 📋 Command reference

The complete list. Every command, what it does, and the arguments.

### Identity

```bash
board init                    # generate identity (idempotent)
board id                      # show pubkey + fingerprint
```

Publishing

```bash
board publish --title T --type K --link L
                              # append a signed post
board export                  # print full signed log to stdout
board verify <logfile> <pubkey-b64>
                              # verify a signed log
```

Mirrors

```bash
board mirror add <url>        # register a mirror
board mirror list             # list registered mirrors
board mirror remove <url>     # remove a mirror
```

Pointers and following

```bash
board pointer [--form F]      # print current pointer (F: json|b64|url)
board follow <pubkey> <ptr>   # follow someone
board follows                 # list who you follow
board fetch [<pubkey>]        # fetch and verify logs (all follows, or one)
board feed                    # read cache, print newest posts first
```

Local network

```bash
board serve [--port N]        # serve your log + discovery (default 8848)
board peers [--follow]        # find other instances on the network
```

Messaging

```bash
board msg <contact> <text>    # send an encrypted message
board inbox                   # read your encrypted messages
```

Files and projects

```bash
board send <peer> <path>      # send a file or git project
board get <peer>/<name>       # download a file from a peer
board push <dir>              # publish a git project as a signed bundle
board pull <peer>/<proj>      # fetch a project bundle and clone it
```

Where things live

```
~/.board/
  identity.pem       # your private key (0600, back this up)
  posts.log          # your signed log
  mirrors.json       # your mirror list
  follows/           # pointers for people you follow
  cache/             # verified copies of their logs

~/board-public/      # files served by board serve
  posts.ndjson       # your log (served live from ~/.board/posts.log)
  <name>.bundle      # project bundles
  <name>             # files sent with board send

~/downloads/         # files received with board get
~/code/              # projects received with board pull
```

---

## 🚀 What's next

This is a working tool, not a finished product. Things that would make it better, in rough order of usefulness:

**Onboarding.** A board demo command that runs a full publish-follow-fetch-message cycle with two local identities in a temp directory. One command, one minute, "here's what the tool actually does."

**QR code pointers.** board pointer --form qr renders the pointer as a scannable QR code. Turns the "paste this 200-character string" step into "scan this."

**Automatic mirror updates.** Right now, in the over-the-internet mode, you have to manually re-upload your log after every publish. A board publish that also pushes to your configured mirror would fix that.

**Peers updating stale follows.** board peers --follow currently says "(already followed)" if a peer is in your list. It should detect a changed pointer (new seq, new mirror URL) and refresh the entry automatically. Small fix, big quality-of-life improvement.

**Group messaging.** Right now messages are strictly one-to-one. A shared group post that all followers can read, but with a designated group key, would be a different primitive.

None of these are required. The tool works today. But if you want to contribute, these are the ones that would have the biggest impact.

---

## 🔧 Contribute

The repo is small and readable. All Go. MIT licensed.

https://github.com/arthursowani145-ux/Board

If you find a bug, open an issue. If you find a security bug, please be specific — the threat model is the contract, and if the code breaks the contract, that's worth knowing.

If you want to test the tool on your own device, the fastest way is:

```bash
./test/msg_crypto_e2e.sh
```

That runs a three-identity messaging cycle in a temp directory and prints ALL CHECKS PASSED on success. It's the shortest possible proof that the tool works on your device.

Welcome aboard. 🖤

---


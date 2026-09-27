#!/data/data/com.termux/files/usr/bin/bash
# Simplest end-to-end messaging test, no HTTP, no fetch.
# A publishes a message to B. B decrypts it. C cannot.
# This exercises the whole msg path using the local log directly.
set -e
cd "$(dirname "$0")/.."
BOARD="$PWD/board"

A=~/msg-test/alice
B=~/msg-test/bob
C=~/msg-test/carol
rm -rf ~/msg-test
mkdir -p "$A" "$B" "$C"

echo "=== init three identities ==="
BOARD_HOME="$A" "$BOARD" init >/dev/null
BOARD_HOME="$B" "$BOARD" init >/dev/null
BOARD_HOME="$C" "$BOARD" init >/dev/null

A_PUB=$(BOARD_HOME="$A" "$BOARD" id | head -1)
B_PUB=$(BOARD_HOME="$B" "$BOARD" id | head -1)
C_PUB=$(BOARD_HOME="$C" "$BOARD" id | head -1)

echo "=== A adds B as a contact (writes B's pointer into A's follows) ==="
# A needs B in its follow list to send. Simulate by publishing B's
# pointer for A. B has no posts, so B publishes one first.
BOARD_HOME="$B" "$BOARD" publish --title "bob" --type note --link "mega:b0" >/dev/null
BOARD_HOME="$B" "$BOARD" mirror add "http://127.0.0.1:9/placeholder.ndjson"
B_PTR=$(BOARD_HOME="$B" "$BOARD" pointer --form b64)
BOARD_HOME="$A" "$BOARD" follow "$B_PUB" "$B_PTR"

echo "=== A sends an encrypted message to B ==="
BOARD_HOME="$A" "$BOARD" msg "$B_PUB" "hey bob, this is a secret for your eyes only"

echo "=== A exports its log to a file C and B can read ==="
BOARD_HOME="$A" "$BOARD" export > ~/msg-test/alice-log.ndjson

echo "=== Simulate fetch: copy A's log into B's and C's cache and add A as a follow ==="
BOARD_HOME="$A" "$BOARD" publish --title "dummy" --type note --link "mega:dummy" >/dev/null
A_PTR=$(BOARD_HOME="$A" "$BOARD" pointer --form b64)

for side in "$B" "$C"; do
  BOARD_HOME="$side" "$BOARD" follow "$A_PUB" "$A_PTR"
  mkdir -p "$side/cache/$A_PUB"
  cp ~/msg-test/alice-log.ndjson "$side/cache/$A_PUB/posts.ndjson"
done

echo "--- B's inbox ---"
BOARD_HOME="$B" "$BOARD" inbox

echo "--- C's inbox ---"
BOARD_HOME="$C" "$BOARD" inbox

echo
echo "=== RESULT ==="
B_INBOX=$(BOARD_HOME="$B" "$BOARD" inbox)
C_INBOX=$(BOARD_HOME="$C" "$BOARD" inbox)

B_OK=0
C_OK=0

if echo "$B_INBOX" | grep -q "secret for your eyes only"; then
  echo "PASS: Bob reads the message"
  B_OK=1
else
  echo "FAIL: Bob cannot read the message"
  echo "$B_INBOX"
fi

if echo "$C_INBOX" | grep -q "secret for your eyes only"; then
  echo "FAIL: Carol can read Bob's message — key isolation broken"
else
  echo "PASS: Carol cannot read the message"
  C_OK=1
fi

if [ "$B_OK" -eq 1 ] && [ "$C_OK" -eq 1 ]; then
  echo
  echo "ALL CHECKS PASSED"
  exit 0
fi
exit 1

#!/data/data/com.termux/files/usr/bin/bash
# End-to-end test: two isolated board instances on one phone.
# Instance A publishes and serves its log; instance B follows A's pointer,
# fetches from A's mirror, verifies, and shows A's posts in its feed.
set -e
cd "$(dirname "$0")/.."
BOARD="$PWD/board"

A=~/board-a
B=~/board-b
rm -rf "$A" "$B"
mkdir -p "$A/public" "$B"

echo "=== A: init + publish ==="
BOARD_HOME="$A" "$BOARD" init | head -2
BOARD_HOME="$A" "$BOARD" publish --title "A-post-1" --type note --link "mega:a1" >/dev/null
BOARD_HOME="$A" "$BOARD" publish --title "A-post-2" --type note --link "mega:a2" >/dev/null
BOARD_HOME="$A" "$BOARD" publish --title "A-post-3" --type note --link "mega:a3" >/dev/null
A_PUB=$(BOARD_HOME="$A" "$BOARD" id | head -1)
echo "A pubkey: $A_PUB"

echo "=== A: serve log on 127.0.0.1:8765 ==="
BOARD_HOME="$A" "$BOARD" export > "$A/public/posts.ndjson"
( cd "$A/public" && python3 -m http.server 8765 --bind 127.0.0.1 >/dev/null 2>&1 ) &
HTTP_PID=$!
sleep 1
trap 'kill $HTTP_PID 2>/dev/null || true' EXIT

BOARD_HOME="$A" "$BOARD" mirror add "http://127.0.0.1:8765/posts.ndjson"
A_PTR=$(BOARD_HOME="$A" "$BOARD" pointer --form b64)
echo "A pointer: ${A_PTR:0:40}..."

echo "=== B: init + follow A ==="
BOARD_HOME="$B" "$BOARD" init | head -2
BOARD_HOME="$B" "$BOARD" follow "$A_PUB" "$A_PTR"
BOARD_HOME="$B" "$BOARD" follows

echo "=== B: fetch A's log ==="
BOARD_HOME="$B" "$BOARD" fetch

echo "=== B: feed ==="
BOARD_HOME="$B" "$BOARD" feed

echo
echo "=== RESULT ==="
FEED=$(BOARD_HOME="$B" "$BOARD" feed)
if echo "$FEED" | grep -q "A-post-1" && echo "$FEED" | grep -q "A-post-3"; then
  echo "PASS: B sees A's posts"
  exit 0
else
  echo "FAIL: A's posts missing from B's feed"
  echo "$FEED"
  exit 1
fi

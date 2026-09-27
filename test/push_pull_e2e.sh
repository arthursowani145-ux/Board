#!/data/data/com.termux/files/usr/bin/bash
# End-to-end collaboration test: Alice has a git project. She pushes it
# as a signed bundle. Bob follows Alice, fetches her log, pulls the
# project, and gets a working git repo. Bob's copy has the same files
# and the same history, and the bundle is verified against the digest.
set -e
cd "$(dirname "$0")/.."
BOARD="$PWD/board"

A=~/pp-test/alice
B=~/pp-test/bob
rm -rf ~/pp-test
mkdir -p "$A" "$B"

echo "=== init two identities ==="
BOARD_HOME="$A" "$BOARD" init >/dev/null
BOARD_HOME="$B" "$BOARD" init >/dev/null
A_PUB=$(BOARD_HOME="$A" "$BOARD" id | head -1)
B_PUB=$(BOARD_HOME="$B" "$BOARD" id | head -1)
echo "A=$A_PUB"
echo "B=$B_PUB"

echo "=== create a test project that Alice owns ==="
PROJ=~/pp-test/myproject
mkdir -p "$PROJ"
cd "$PROJ"
git init -q
git config user.email "alice@example.com"
git config user.name "Alice"
echo "hello from alice" > README.md
git add README.md
git commit -q -m "first commit"
echo "package main" > main.go
git add main.go
git commit -q -m "second commit"
cd -
echo "project has $(git -C "$PROJ" rev-list --count HEAD) commits"

echo "=== Alice pushes the project ==="
BOARD_HOME="$A" "$BOARD" push "$PROJ"

echo "=== Alice serves ~/board-public/ and her log ==="
# Alice needs to serve BOTH her log (for the project post) and the bundle.
# Easiest: create a combined public dir.
COMBINED=~/pp-test/alice-public
mkdir -p "$COMBINED"
BOARD_HOME="$A" "$BOARD" export > "$COMBINED/posts.ndjson"
cp ~/board-public/myproject.bundle "$COMBINED/myproject.bundle" 2>/dev/null || \
  cp "$HOME/board-public/myproject.bundle" "$COMBINED/myproject.bundle"
ls -la "$COMBINED/"

( cd "$COMBINED" && python3 -m http.server 8901 --bind 127.0.0.1 >/dev/null 2>&1 ) &
HTTP_PID=$!
sleep 1
trap 'kill $HTTP_PID 2>/dev/null || true' EXIT

echo "=== Alice sets up her mirror and produces a pointer ==="
BOARD_HOME="$A" "$BOARD" mirror add "http://127.0.0.1:8901/posts.ndjson"
A_PTR=$(BOARD_HOME="$A" "$BOARD" pointer --form b64)

echo "=== Bob follows Alice and fetches her log ==="
BOARD_HOME="$B" "$BOARD" follow -- "$A_PUB" "$A_PTR"
BOARD_HOME="$B" "$BOARD" fetch
BOARD_HOME="$B" "$BOARD" feed

echo "=== Bob pulls the project ==="
BOARD_HOME="$B" "$BOARD" pull "$A_PUB/myproject"

echo "=== Verify Bob's copy ==="
BOB_PROJ=~/code/myproject
if [ ! -d "$BOB_PROJ" ]; then
  echo "FAIL: Bob's copy was not created at $BOB_PROJ"
  exit 1
fi

# Check the files.
if [ ! -f "$BOB_PROJ/README.md" ] || [ ! -f "$BOB_PROJ/main.go" ]; then
  echo "FAIL: Bob's copy missing files"
  ls -la "$BOB_PROJ"
  exit 1
fi

# Check the history.
BOB_COMMITS=$(git -C "$BOB_PROJ" rev-list --count HEAD)
echo "Bob's copy has $BOB_COMMITS commits"
if [ "$BOB_COMMITS" != "2" ]; then
  echo "FAIL: expected 2 commits, got $BOB_COMMITS"
  exit 1
fi

# Check content matches.
if ! grep -q "hello from alice" "$BOB_PROJ/README.md"; then
  echo "FAIL: Bob's README doesn't match Alice's"
  exit 1
fi

# Confirm git log matches Alice's.
A_LOG=$(git -C "$PROJ" log --oneline)
B_LOG=$(git -C "$BOB_PROJ" log --oneline)
if [ "$A_LOG" != "$B_LOG" ]; then
  echo "FAIL: git logs differ"
  echo "Alice:"; echo "$A_LOG"
  echo "Bob:";   echo "$B_LOG"
  exit 1
fi

echo
echo "=== RESULT ==="
echo "PASS: Bob pulled a verified copy of Alice's project"
echo "  same files, same history, same commits"
echo
echo "ALL CHECKS PASSED"

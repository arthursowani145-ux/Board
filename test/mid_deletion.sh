#!/data/data/com.termux/files/usr/bin/bash
# Proves the prev-hash chain catches a middle deletion that the seq
# check would otherwise mask.
set -e
cd "$(dirname "$0")/.."

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# fresh log
rm -f ~/.board/posts.log
for i in 0 1 2 3; do
  ./board publish --title "p$i" --type note --link "mega:$i" >/dev/null
done

./board export > "$TMP/full.ndjson"
PUB=$(./board id | head -1)

./board verify "$TMP/full.ndjson" "$PUB" >/dev/null
echo "full log: OK"

# drop post 2 and renumber the tail so seq check is satisfied
awk 'NR!=3' "$TMP/full.ndjson" | sed 's/"seq":3/"seq":2/' > "$TMP/mid.ndjson"

if ./board verify "$TMP/mid.ndjson" "$PUB" 2>"$TMP/err"; then
  echo "FAIL: mid-deletion passed verification"
  exit 1
fi

if grep -q "prev=" "$TMP/err"; then
  echo "mid-deletion: correctly caught by prev mismatch"
else
  echo "mid-deletion: caught, but not by prev check:"
  cat "$TMP/err"
  exit 1
fi

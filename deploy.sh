#!/bin/sh
set -e

REMOTE="root@objecthub.xyz"
SSH="ssh -p 2243 -i $HOME/.ssh/id_ed25519"
SCP="scp -P 2243 -i $HOME/.ssh/id_ed25519"

BIN="/usr/local/OjhubGo/ojhub-go"
TMP="/tmp/ojhub-go.new"

echo "==> Building..."
CGO_ENABLED=0 GOAMD64=v3 go127 build -pgo=auto -trimpath -ldflags="-s -w" -o ojhub-go .

echo "==> Uploading..."
$SCP ojhub-go "$REMOTE:$TMP"

echo "==> Deploying..."
# i thunk you have rc.d script with all exported environment variables
$SSH "$REMOTE" "mv '$TMP' '$BIN' &&service ojhub restart >/dev/null 2>&1"

curl -s https://objecthub.xyz/server/133/loginT.php | jq empty

echo "==> Done."

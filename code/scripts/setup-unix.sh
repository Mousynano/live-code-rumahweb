#!/usr/bin/env sh
set -eu

CODE_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
REPO_ROOT=$(CDPATH= cd -- "$CODE_ROOT/.." && pwd)

command -v go >/dev/null 2>&1 || { echo "Go is not installed. Install the current stable release from https://go.dev/dl/." >&2; exit 1; }
command -v node >/dev/null 2>&1 || { echo "Node.js is not installed." >&2; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "npm is not installed." >&2; exit 1; }

if [ ! -f "$REPO_ROOT/.env" ]; then
  cp "$REPO_ROOT/.env.example" "$REPO_ROOT/.env"
  echo "Created $REPO_ROOT/.env from .env.example"
fi

(cd "$CODE_ROOT/server" && go mod download && go test ./...)
(cd "$CODE_ROOT/client" && npm ci && npm test)

echo "Setup complete. Generate a bcrypt hash with:"
echo "  cd $CODE_ROOT/server && go run ./cmd/hash-password 'your-password'"
echo "Then update ADMIN_PASSWORD_HASH and WHOISFREAKS_API_KEY in $REPO_ROOT/.env"

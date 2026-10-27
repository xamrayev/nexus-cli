#!/usr/bin/env bash
# Nexus-CLI smoke test: builds the binary and checks commands that work
# without a DeepSeek account or network access.
# Usage: scripts/smoke.sh            (builds ./bin/nexus itself)
#        NEXUS_BIN=path scripts/smoke.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="${NEXUS_BIN:-$ROOT/bin/nexus}"
PORT="${SMOKE_PORT:-19655}"
WORK="$(mktemp -d)"
SERVER_PID=""

cleanup() {
  [ -n "$SERVER_PID" ] && kill "$SERVER_PID" 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT

pass() { printf '  \033[32m✓\033[0m %s\n' "$1"; }
fail() { printf '  \033[31m✗\033[0m %s\n' "$1"; exit 1; }

if [ -z "${NEXUS_BIN:-}" ]; then
  echo "▶ build"
  (cd "$ROOT" && go build -o "$BIN" .)
fi

# Run from an empty dir so ./accounts and auth files are never picked up.
cd "$WORK"
export HOME="$WORK"

echo "▶ CLI commands"
"$BIN" version | grep -q "Nexus-CLI versiyasi" && pass "nexus version" || fail "nexus version"
"$BIN" --help  | grep -q "Foydalanish"          && pass "nexus --help"  || fail "nexus --help"
"$BIN" models  | grep -q "deepseek"             && pass "nexus models"  || fail "nexus models"
"$BIN" launch  | grep -q "Mavjud platformalar"  && pass "nexus launch (menu)" || fail "nexus launch (menu)"

echo "▶ local API server"
"$BIN" serve --host 127.0.0.1 --port "$PORT" --key smoke-key --accounts-dir "$WORK/accounts" >"$WORK/serve.log" 2>&1 &
SERVER_PID=$!

for _ in $(seq 1 50); do
  curl -fsS "http://127.0.0.1:$PORT/health" >/dev/null 2>&1 && break
  sleep 0.2
done

curl -fsS "http://127.0.0.1:$PORT/health" | grep -q '"status":"ok"' \
  && pass "GET /health" || { cat "$WORK/serve.log"; fail "GET /health"; }

code=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$PORT/v1/models")
[ "$code" = "401" ] && pass "GET /v1/models without key → 401" || fail "GET /v1/models without key → $code"

curl -fsS -H "Authorization: Bearer smoke-key" "http://127.0.0.1:$PORT/v1/models" | grep -q '"object":"list"' \
  && pass "GET /v1/models with key" || fail "GET /v1/models with key"

curl -fsS "http://127.0.0.1:$PORT/api/version" | grep -q '"version"' \
  && pass "GET /api/version (Ollama)" || fail "GET /api/version (Ollama)"

echo "✅ smoke OK"

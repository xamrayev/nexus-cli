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
# check NAME PATTERN TEXT — capture first, then grep (avoids SIGPIPE under pipefail)
check() { if printf '%s' "$3" | grep -q -- "$2"; then pass "$1"; else printf '%s\n' "$3"; fail "$1"; fi; }

if [ -z "${NEXUS_BIN:-}" ]; then
  echo "▶ build"
  (cd "$ROOT" && go build -o "$BIN" ./cmd/nexus)
fi

# Run from an empty dir so ./accounts and auth files are never picked up.
cd "$WORK"
export HOME="$WORK"

echo "▶ CLI commands"
check "nexus version"       "Nexus-CLI versiyasi"  "$("$BIN" version 2>&1)"
check "nexus --help"        "Foydalanish"          "$("$BIN" --help 2>&1)"
check "nexus models"        "deepseek"             "$("$BIN" models 2>&1)"
check "nexus launch (menu)" "Mavjud platformalar"  "$("$BIN" launch 2>&1)"

echo "▶ local API server"
"$BIN" serve --host 127.0.0.1 --port "$PORT" --key smoke-key --accounts-dir "$WORK/accounts" >"$WORK/serve.log" 2>&1 &
SERVER_PID=$!

for _ in $(seq 1 50); do
  curl -fsS "http://127.0.0.1:$PORT/health" >/dev/null 2>&1 && break
  sleep 0.2
done

health="$(curl -sS "http://127.0.0.1:$PORT/health" 2>&1 || true)"
printf '%s' "$health" | grep -q '"status":"ok"' || cat "$WORK/serve.log"
check "GET /health" '"status":"ok"' "$health"

code=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$PORT/v1/models")
[ "$code" = "401" ] && pass "GET /v1/models without key → 401" || fail "GET /v1/models without key → $code"

check "GET /v1/models with key" '"object":"list"' \
  "$(curl -sS -H "Authorization: Bearer smoke-key" "http://127.0.0.1:$PORT/v1/models" 2>&1)"
check "GET /api/version (Ollama)" '"version"' "$(curl -sS "http://127.0.0.1:$PORT/api/version" 2>&1)"

echo "✅ smoke OK"

#!/usr/bin/env bash
# Integration test for v0.8.3 — Agent-Native CLI & Natural Language Interface
# Tests: --json flag, envault status, envault agent-check
set -uo pipefail

ENVAULT=$(realpath "${1:-./envault}")
PASS=0
FAIL=0
TEST_DIR=$(mktemp -d)
trap 'rm -rf "$TEST_DIR"' EXIT

pass() { echo "PASS: $1"; PASS=$((PASS+1)); }
fail() { echo "FAIL: $1"; FAIL=$((FAIL+1)); }

# Setup: fresh vault with one recipient (memstore — no real keychain needed)
cd "$TEST_DIR"
git init -q
git config user.email "test@example.com"
git config user.name "Test"

# Init vault (no key needed for status/agent-check)
"$ENVAULT" init 2>/dev/null && pass "envault init" || fail "envault init"

# --- --json flag ---
out=$("$ENVAULT" --json list 2>/dev/null)
if echo "$out" | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['ok']==True" 2>/dev/null; then
  pass "--json flag produces valid JSON"
else
  fail "--json flag produces valid JSON (got: $out)"
fi

# --- envault status (human mode) ---
out=$("$ENVAULT" status 2>/dev/null)
if echo "$out" | grep -q "initialized"; then
  pass "envault status shows initialized"
else
  fail "envault status shows initialized (got: $out)"
fi

# --- envault status (JSON mode) ---
out=$("$ENVAULT" status --json 2>/dev/null)
if echo "$out" | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['ok']==True; assert 'initialized' in d['data']" 2>/dev/null; then
  pass "envault status --json produces valid JSON with initialized field"
else
  fail "envault status --json produces valid JSON with initialized field (got: $out)"
fi

# initialized=true in status output
if echo "$out" | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['data']['initialized']==True" 2>/dev/null; then
  pass "envault status reports initialized=true"
else
  fail "envault status reports initialized=true (got: $out)"
fi

# --- envault agent-check (JSON mode, not ready) ---
out=$("$ENVAULT" agent-check --json 2>/dev/null || true)
if echo "$out" | python3 -c "import sys,json; d=json.load(sys.stdin); assert d['ok']==True; assert 'ready' in d['data']" 2>/dev/null; then
  pass "envault agent-check --json produces JSON with ready field"
else
  fail "envault agent-check --json produces JSON with ready field (got: $out)"
fi

# agent-check exits non-zero when not configured
if "$ENVAULT" agent-check 2>/dev/null; then
  fail "envault agent-check should exit non-zero when not configured"
else
  pass "envault agent-check exits non-zero when not configured"
fi

# --- envault agent-check (human mode, error output) ---
out=$("$ENVAULT" agent-check 2>/dev/null || true)
if echo "$out" | grep -q "Claude Code hook\|Privacy Shield\|Output masking"; then
  pass "envault agent-check human mode shows check labels"
else
  fail "envault agent-check human mode shows check labels (got: $out)"
fi

# --- uninitialized vault: status still works ---
UNINIT_DIR=$(mktemp -d)
cd "$UNINIT_DIR"
git init -q
out=$("$ENVAULT" status 2>/dev/null)
if echo "$out" | grep -q "✗\|initialized"; then
  pass "envault status works in uninitialized repo"
else
  fail "envault status works in uninitialized repo (got: $out)"
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]

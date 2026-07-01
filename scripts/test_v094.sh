#!/usr/bin/env bash
# Integration test for v0.9.4 — Key-Unlock Agent (ssh-agent-style, for
# Claude Code MCP UX). Tests: unlock/status/lock/stop lifecycle, that `envault
# run` and `envault mcp serve` (envault_run) both work with NO
# ENVAULT_PASSPHRASE once unlocked, that `envault hook postuse` masking also
# works via the agent, and that TTL expiry correctly falls back to requiring
# a passphrase again.
set -uo pipefail

ENVAULT=$(realpath "${1:-./envault}")
PASS=0
FAIL=0

# Isolate the agent socket to a short path (must fit in a Unix socket path,
# ~104 bytes on macOS) via ENVAULT_AGENT_SOCKET rather than overriding HOME:
# HOME also controls where the OS keychain looks for the login keychain, and
# overriding it breaks `key new`/`add` (macOS `security` fails to find a
# keychain to store into). This must also never touch a real, already-running
# agent on the dev machine.
TEST_HOME=$(mktemp -d /tmp/envault-agent-it-XXXXXX)
export ENVAULT_AGENT_SOCKET="$TEST_HOME/agent.sock"
TEST_DIR=$(mktemp -d)
KEY_ID="test-agent-$$@example.com"
trap '"$ENVAULT" agent stop >/dev/null 2>&1; "$ENVAULT" uninstall --keys >/dev/null 2>&1; rm -rf "$TEST_DIR" "$TEST_HOME"' EXIT

pass() { echo "PASS: $1"; PASS=$((PASS+1)); }
fail() { echo "FAIL: $1"; FAIL=$((FAIL+1)); }

cd "$TEST_DIR"
git init -q
git config user.email "test@example.com"
git config user.name "Test"
"$ENVAULT" init >/dev/null 2>&1 && pass "envault init" || fail "envault init"

# --- no agent running yet ---
out=$("$ENVAULT" agent status 2>&1)
if echo "$out" | grep -qi "no agent running"; then
  pass "agent status: reports no agent running before unlock"
else
  fail "agent status before unlock (got: $out)"
fi

# --- seed an identity and a secret (ENVAULT_PASSPHRASE only for setup) ---
export ENVAULT_PASSPHRASE="test-passphrase-123"
"$ENVAULT" key new --id "$KEY_ID" >/dev/null 2>&1 && pass "key new" || fail "key new"
echo "bar123" | "$ENVAULT" add FOO >/dev/null 2>&1 && pass "envault add" || fail "envault add"

# --- unlock the agent, then drop ENVAULT_PASSPHRASE for everything after ---
"$ENVAULT" agent unlock --ttl 1h >/dev/null 2>&1 && pass "agent unlock" || fail "agent unlock"
unset ENVAULT_PASSPHRASE

out=$("$ENVAULT" agent status 2>&1)
if echo "$out" | grep -q "$KEY_ID"; then
  pass "agent status: shows the unlocked identity"
else
  fail "agent status after unlock (got: $out)"
fi

# --- envault run works with NO ENVAULT_PASSPHRASE via the agent ---
out=$("$ENVAULT" run -- printenv FOO 2>&1)
if echo "$out" | grep -q "bar123"; then
  pass "envault run works passphrase-free via the agent"
else
  fail "envault run passphrase-free (got: $out)"
fi

# --- envault mcp serve (envault_run) also works with NO ENVAULT_PASSPHRASE ---
req='{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"envault_run","arguments":{"command":["printenv","FOO"]}}}'
out=$(printf '%s\n' "$req" | "$ENVAULT" mcp serve 2>/dev/null)
if echo "$out" | python3 -c "
import sys, json
r = json.loads(sys.stdin.read())
result = json.loads(r['result']['content'][0]['text'])
assert result['exit_code'] == 0
assert 'bar123' in result['stdout']
" 2>/dev/null; then
  pass "envault mcp serve (envault_run) works passphrase-free via the agent"
else
  fail "envault_run via MCP passphrase-free (got: $out)"
fi

# --- postuse masking also works via the agent, no ENVAULT_PASSPHRASE ---
postuse_input=$(python3 -c "
import json
print(json.dumps({'tool_name':'Bash','tool_input':{'command':'env'},'tool_response':'FOO=bar123'}))
")
out=$(printf '%s' "$postuse_input" | "$ENVAULT" hook postuse 2>&1)
if echo "$out" | grep -q "<ENVAULT:FOO>" && ! echo "$out" | grep -q "bar123"; then
  pass "postuse masks secrets via the agent, no ENVAULT_PASSPHRASE needed"
else
  fail "postuse masking passphrase-free (got: $out)"
fi

# --- lock clears the cache; run then requires a passphrase again ---
"$ENVAULT" agent lock >/dev/null 2>&1 && pass "agent lock" || fail "agent lock"
out=$("$ENVAULT" run -- printenv FOO 2>&1)
if [ $? -ne 0 ] || ! echo "$out" | grep -q "bar123"; then
  pass "envault run requires a passphrase again after agent lock"
else
  fail "envault run should fail after agent lock (got: $out)"
fi

# --- TTL expiry: unlock briefly, then confirm it's gone ---
export ENVAULT_PASSPHRASE="test-passphrase-123"
"$ENVAULT" agent unlock --ttl 1s >/dev/null 2>&1
unset ENVAULT_PASSPHRASE
sleep 2
out=$("$ENVAULT" run -- printenv FOO 2>&1)
if echo "$out" | grep -qi "passphrase"; then
  pass "key expires from the agent after its TTL elapses"
else
  fail "TTL expiry (got: $out)"
fi

# --- stop terminates the agent process entirely ---
"$ENVAULT" agent stop >/dev/null 2>&1 && pass "agent stop" || fail "agent stop"
out=$("$ENVAULT" agent status 2>&1)
if echo "$out" | grep -qi "no agent running"; then
  pass "agent status: no agent running after stop"
else
  fail "agent status after stop (got: $out)"
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]

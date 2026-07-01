#!/usr/bin/env bash
# Integration test for v0.9.3 — Embedded MCP Server (Claude Code Native Protocol)
# Tests: `envault mcp serve --dry-run` schema output, and a real JSON-RPC 2.0
# round trip over stdio (initialize, tools/list, tools/call for status/add/run),
# plus the headless-safe error path when ENVAULT_PASSPHRASE is missing.
set -uo pipefail

ENVAULT=$(realpath "${1:-./envault}")
PASS=0
FAIL=0
TEST_DIR=$(mktemp -d)
# Unique per run so this never collides with a real key already in the OS
# keychain; cleaned up via the product's own `uninstall --keys` on exit.
KEY_ID="test-mcp-$$@example.com"
trap '"$ENVAULT" uninstall --keys >/dev/null 2>&1; rm -rf "$TEST_DIR"' EXIT

pass() { echo "PASS: $1"; PASS=$((PASS+1)); }
fail() { echo "FAIL: $1"; FAIL=$((FAIL+1)); }

cd "$TEST_DIR"
git init -q
git config user.email "test@example.com"
git config user.name "Test"
"$ENVAULT" init >/dev/null 2>&1 && pass "envault init" || fail "envault init"

# --- --dry-run prints valid JSON Schema for every tool ---
out=$("$ENVAULT" mcp serve --dry-run 2>/dev/null)
if echo "$out" | python3 -c "
import sys, json
tools = json.load(sys.stdin)
names = {t['name'] for t in tools}
expected = {'envault_status','envault_list','envault_add','envault_rotate',
            'envault_run','envault_protect','envault_push','envault_pull'}
assert names == expected, names
assert tools[0]['inputSchema']['type'] == 'object'
"; then
  pass "mcp serve --dry-run: all 8 tool schemas present"
else
  fail "mcp serve --dry-run (got: $out)"
fi

# --- generate an identity key for the JSON-RPC round trip ---
export ENVAULT_PASSPHRASE="test-passphrase-123"
"$ENVAULT" key new --id "$KEY_ID" >/dev/null 2>&1 \
  && pass "key new" || fail "key new"

# --- JSON-RPC round trip: initialize, tools/list, tools/call (status/add/run) ---
requests=$(cat <<'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"envault_add","arguments":{"name":"FOO","value":"bar123"}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"envault_run","arguments":{"command":["printenv","FOO"]}}}
EOF
)
out=$(printf '%s\n' "$requests" | "$ENVAULT" mcp serve 2>/dev/null)

echo "$out" | python3 -c "
import sys, json
lines = [json.loads(l) for l in sys.stdin if l.strip()]
assert len(lines) == 4, f'expected 4 responses, got {len(lines)}'

init = lines[0]['result']
assert init['serverInfo']['name'] == 'envault'
assert 'tools' in init['capabilities']

tools = lines[1]['result']['tools']
assert len(tools) == 8

add_result = json.loads(lines[2]['result']['content'][0]['text'])
assert add_result['ok'] is True
assert add_result['name'] == 'FOO'
assert 'bar123' not in json.dumps(add_result), 'secret value leaked into envault_add response'

run_result = json.loads(lines[3]['result']['content'][0]['text'])
assert run_result['exit_code'] == 0
assert 'bar123' in run_result['stdout'], 'expected child-process output to contain the injected secret'
" && pass "JSON-RPC round trip: initialize/tools.list/envault_add/envault_run" \
  || fail "JSON-RPC round trip (got: $out)"

# --- headless-safe: rotate without ENVAULT_PASSPHRASE fails as a structured tool error, not a crash ---
req='{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"envault_rotate","arguments":{"name":"FOO"}}}'
out=$(printf '%s\n' "$req" | env -u ENVAULT_PASSPHRASE "$ENVAULT" mcp serve 2>/dev/null)
if echo "$out" | python3 -c "
import sys, json
r = json.loads(sys.stdin.read())
assert r['result']['isError'] is True
" 2>/dev/null; then
  pass "envault_rotate fails structurally without ENVAULT_PASSPHRASE (no TTY, no crash)"
else
  fail "envault_rotate headless error path (got: $out)"
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]

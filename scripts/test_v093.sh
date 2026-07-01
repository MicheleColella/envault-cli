#!/usr/bin/env bash
# Integration test for v0.9.3 — Embedded MCP Server (Claude Code Native Protocol)
# Tests: `envault mcp serve --dry-run` schema output, a real JSON-RPC 2.0 round
# trip over stdio (initialize, tools/list, envault_run), the headless-safe error
# path when ENVAULT_PASSPHRASE is missing, that `envault_add` is NOT exposed as
# an MCP tool, and that the preuse hook blocks `envault add`/`set` in Bash.
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

# --- --dry-run prints valid JSON Schema for every tool — envault_add absent ---
out=$("$ENVAULT" mcp serve --dry-run 2>/dev/null)
if echo "$out" | python3 -c "
import sys, json
tools = json.load(sys.stdin)
names = {t['name'] for t in tools}
expected = {'envault_status','envault_list','envault_rotate',
            'envault_run','envault_protect','envault_push','envault_pull'}
assert names == expected, names
assert 'envault_add' not in names, 'envault_add must not be exposed via MCP'
assert tools[0]['inputSchema']['type'] == 'object'
"; then
  pass "mcp serve --dry-run: 7 tool schemas present, envault_add absent"
else
  fail "mcp serve --dry-run (got: $out)"
fi

# --- generate an identity key and seal a secret via the CLI (terminal-only path) ---
export ENVAULT_PASSPHRASE="test-passphrase-123"
"$ENVAULT" key new --id "$KEY_ID" >/dev/null 2>&1 \
  && pass "key new" || fail "key new"
echo "bar123" | "$ENVAULT" add FOO >/dev/null 2>&1 \
  && pass "envault add (CLI, terminal-only)" || fail "envault add"

# --- JSON-RPC round trip: initialize, tools/list, envault_run ---
requests=$(cat <<'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"envault_run","arguments":{"command":["printenv","FOO"]}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"envault_add","arguments":{"name":"X","value":"y"}}}
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
assert len(tools) == 7

run_result = json.loads(lines[2]['result']['content'][0]['text'])
assert run_result['exit_code'] == 0
assert 'bar123' in run_result['stdout'], 'expected child-process output to contain the injected secret'

# envault_add must not exist as a callable MCP tool at all.
assert lines[3]['error']['code'] == -32602, f'expected unknown-tool protocol error, got {lines[3]}'
" && pass "JSON-RPC round trip: initialize/tools.list/envault_run, envault_add rejected" \
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

# --- preuse hook blocks envault add/set in Bash, including via a pipe ---
preuse_blocks() {
  local cmd="$1"
  local input
  input=$(python3 -c "import json,sys; print(json.dumps({'tool_name':'Bash','tool_input':{'command':sys.argv[1]}}))" "$cmd")
  printf '%s' "$input" | "$ENVAULT" hook preuse >/tmp/preuse_out_$$ 2>&1
  local code=$?
  local msg
  msg=$(cat /tmp/preuse_out_$$)
  rm -f /tmp/preuse_out_$$
  [ "$code" -ne 0 ] && echo "$msg" | grep -q "own terminal"
}

if preuse_blocks 'echo "sk-live-123" | envault add API_KEY'; then
  pass "preuse blocks 'echo secret | envault add' via Bash"
else
  fail "preuse should block 'echo secret | envault add' via Bash"
fi

if preuse_blocks 'envault set DB_URL <<< newvalue'; then
  pass "preuse blocks 'envault set <<<' via Bash"
else
  fail "preuse should block 'envault set <<<' via Bash"
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]

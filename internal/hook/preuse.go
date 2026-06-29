package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// ErrBlockToolCall is returned by RunHookPreuse when the Bash command must be
// blocked. The caller must exit non-zero so Claude Code denies the tool call —
// any text already written to the output writer is shown to Claude as the reason.
var ErrBlockToolCall = fmt.Errorf("tool call blocked by envault hook")

// PreuseInput is the subset of the Claude Code PreToolUse hook JSON we care about.
type PreuseInput struct {
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
}

// RunHookPreuse reads Claude Code's PreToolUse JSON from r.
// When a Bash command in an envault repo invokes `envault cat` or `envault export`
// without --force, it writes a human-readable block reason to w and returns
// ErrBlockToolCall. The caller must then exit non-zero so Claude Code denies the
// tool use and shows the message to Claude instead.
// For all other commands it returns nil (tool call is allowed unchanged).
func RunHookPreuse(r io.Reader, w io.Writer) error {
	var input PreuseInput
	if err := json.NewDecoder(r).Decode(&input); err != nil {
		return nil // non-fatal: allow the tool call unchanged
	}

	if input.ToolName != "Bash" {
		return nil
	}

	cmd, _ := input.ToolInput["command"].(string)
	if cmd == "" {
		return nil
	}

	// Only intercept when inside an envault repo.
	wd, err := os.Getwd()
	if err != nil || !IsEnvaultDir(wd) {
		return nil
	}

	if IsSensitiveEnvaultCmd(cmd) {
		_, _ = fmt.Fprintln(w,
			"envault: plaintext output blocked — secrets must not appear in the model context.\n"+
				"Use `envault run -- <cmd>` to inject secrets in-memory into a child process.\n"+
				"If you really need the plaintext value, pass --force to override.",
		)
		return ErrBlockToolCall
	}

	return nil
}

// IsSensitiveEnvaultCmd reports whether cmd invokes `envault cat` or
// `envault export` as the primary command (not as an argument to another tool)
// without the --force override flag.
func IsSensitiveEnvaultCmd(cmd string) bool {
	fields := strings.Fields(cmd)

	// Skip leading VAR=value environment assignments (e.g. CLAUDE_CODE=1 envault …)
	start := 0
	for start < len(fields) && strings.ContainsRune(fields[start], '=') {
		start++
	}

	if start >= len(fields) {
		return false
	}

	// Only match when envault is the first executable token.
	first := fields[start]
	if first != "envault" && !strings.HasSuffix(first, "/envault") {
		return false
	}

	if start+1 >= len(fields) {
		return false
	}
	sub := fields[start+1]
	if sub != "cat" && sub != "export" {
		return false
	}

	// Allow explicit --force override anywhere after the subcommand.
	for _, flag := range fields[start:] {
		if flag == "--force" {
			return false
		}
	}
	return true
}

// IsEnvaultDir returns true when .envault/ exists under root.
func IsEnvaultDir(root string) bool {
	_, err := os.Stat(root + "/.envault")
	return err == nil
}

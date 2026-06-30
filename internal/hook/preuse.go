package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MicheleColella/envault-cli/internal/audit"
	"github.com/MicheleColella/envault-cli/internal/protect"
)

// ErrBlockToolCall is returned by RunHookPreuse when the tool call must be
// blocked. The caller must exit non-zero so Claude Code denies the tool call —
// any text already written to the output writer is shown to Claude as the reason.
var ErrBlockToolCall = fmt.Errorf("tool call blocked by envault hook")

// PreuseInput is the subset of the Claude Code PreToolUse hook JSON we care about.
type PreuseInput struct {
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
}

// filePathTools are tools whose primary target is a file accessible via file_path.
// NotebookEdit uses notebook_path instead — handled separately.
var filePathTools = map[string]string{
	"Read":         "file_path",
	"Write":        "file_path",
	"Edit":         "file_path",
	"MultiEdit":    "file_path",
	"NotebookEdit": "notebook_path",
}

// RunHookPreuse reads Claude Code's PreToolUse JSON from r.
//
// Blocking rules (in priority order):
//  1. Read/Write/Edit/NotebookEdit tools whose file_path matches a protected pattern.
//  2. Bash commands that reference a protected path (best-effort heuristic; full
//     adversarial coverage is in v0.8.4).
//  3. Bash commands that invoke `envault cat` or `envault export` without --force.
//
// Each blocked call is appended to the audit log (.envault/ai-secure.log).
// Returns ErrBlockToolCall when a call is denied; nil otherwise.
func RunHookPreuse(r io.Reader, w io.Writer) error {
	var input PreuseInput
	if err := json.NewDecoder(r).Decode(&input); err != nil {
		return nil // non-fatal: allow unchanged
	}

	wd, err := os.Getwd()
	if err != nil || !IsEnvaultDir(wd) {
		return nil
	}

	patterns, _ := protect.LoadPatterns(wd) // ignore error: no patterns = no blocking

	// --- File tool protection ---
	if paramKey, isFileTool := filePathTools[input.ToolName]; isFileTool {
		filePath, _ := input.ToolInput[paramKey].(string)
		if filePath != "" && len(patterns) > 0 {
			if matched, ok := protect.MatchesAny(filePath, patterns); ok {
				_ = audit.AppendEntry(wd, input.ToolName, audit.ActionBlockedPath, filePath, matched)
				_, _ = fmt.Fprintf(w,
					"[ENVAULT PROTECTED: %s — file contents encrypted. Use `envault run` to access at runtime.]\n"+
						"Pattern: %s",
					filePath, matched,
				)
				return ErrBlockToolCall
			}
		}
		return nil
	}

	// --- Bash tool ---
	if input.ToolName != "Bash" {
		return nil
	}

	cmd, _ := input.ToolInput["command"].(string)
	if cmd == "" {
		return nil
	}

	// Protected path check in Bash command.
	if len(patterns) > 0 {
		if matched, tok, ok := protect.ContainsProtectedPath(cmd, patterns); ok {
			_ = audit.AppendEntry(wd, "Bash", audit.ActionBlockedCmd, snippetOf(cmd, 120), matched)
			_, _ = fmt.Fprintf(w,
				"[ENVAULT PROTECTED: %s — path matches protected pattern %q. Use `envault run` to access at runtime.]\n"+
					"Blocked command: %s",
				tok, matched, snippetOf(cmd, 200),
			)
			return ErrBlockToolCall
		}
	}

	// envault cat / export without --force.
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

func snippetOf(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

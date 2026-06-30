package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// claudeHookID is the stable marker embedded in the hook entry so we can find/remove it.
const claudeHookID = "envault"

// InstallClaudeHook writes a PreToolUse(Bash) hook entry into settings.json.
// When global is true, writes to ~/.claude/settings.json; otherwise writes to
// <repoRoot>/.claude/settings.json. Existing content is preserved; idempotent.
func InstallClaudeHook(repoRoot string, global bool) error {
	path := claudeSettingsPath(repoRoot, global)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create .claude directory: %w", err)
	}

	data, err := readSettings(path)
	if err != nil {
		return err
	}

	if isClaudeHookPresent(data) {
		return nil
	}

	addClaudeHook(data, hookBaseCmd())

	return writeSettings(path, data)
}

// UninstallClaudeHook removes the envault PreToolUse hook from settings.json.
// When global is true, targets ~/.claude/settings.json. Returns nil when the
// hook was not installed or the file does not exist.
func UninstallClaudeHook(repoRoot string, global bool) error {
	path := claudeSettingsPath(repoRoot, global)

	data, err := readSettings(path)
	if err != nil {
		return err
	}

	if !isClaudeHookPresent(data) {
		return nil
	}

	removeClaudeHook(data)
	return writeSettings(path, data)
}

// IsClaudeHookInstalled reports whether the envault hook is present in settings.json.
// When global is true, checks ~/.claude/settings.json.
func IsClaudeHookInstalled(repoRoot string, global bool) bool {
	data, err := readSettings(claudeSettingsPath(repoRoot, global))
	if err != nil {
		return false
	}
	return isClaudeHookPresent(data)
}

// hookBaseCmd returns the `<binary> hook` prefix used to build hook commands.
// Resolves the absolute path of the running envault binary so Claude Code can
// invoke it even when the binary is not on PATH (e.g. a local test copy).
// Falls back to the bare name "envault hook" when the path cannot be resolved.
func hookBaseCmd() string {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		return "envault hook"
	}
	real, err := filepath.EvalSymlinks(exe)
	if err == nil {
		exe = real
	}
	return exe + " hook"
}

// --- helpers ---

// claudeSettingsPath returns the target settings.json path.
// When global is true, returns ~/.claude/settings.json.
func claudeSettingsPath(repoRoot string, global bool) string {
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "~"
		}
		return filepath.Join(home, ".claude", "settings.json")
	}
	return filepath.Join(repoRoot, ".claude", "settings.json")
}

// readSettings reads and parses settings.json as a generic map.
// Returns an empty map when the file does not exist.
func readSettings(path string) (map[string]interface{}, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]interface{}{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read settings.json: %w", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("parse settings.json: %w", err)
	}
	return out, nil
}

func writeSettings(path string, data map[string]interface{}) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings.json: %w", err)
	}
	// Backup the previous file before overwriting.
	if prev, err := os.ReadFile(path); err == nil {
		_ = os.WriteFile(path+".bak", prev, 0o600) //nolint:gosec // G703: path is derived from our own claudeSettingsPath, not user input
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o600); err != nil {
		return fmt.Errorf("write settings.json: %w", err)
	}
	return nil
}

// isClaudeHookPresent checks whether the envault PreToolUse hook entry is already in data.
func isClaudeHookPresent(data map[string]interface{}) bool {
	for _, g := range preToolUseGroups(data) {
		if matchesEnvaultGroup(g) {
			return true
		}
	}
	return false
}

// addClaudeHook appends the envault PreToolUse and PostToolUse hook groups.
func addClaudeHook(data map[string]interface{}, cmd string) {
	hooks := hooksMap(data)

	pre := preToolUseGroups(data)
	pre = append(pre, envaultPreHookGroup(cmd))
	hooks["PreToolUse"] = pre

	post := postToolUseGroups(data)
	post = append(post, envaultPostHookGroup(cmd))
	hooks["PostToolUse"] = post

	data["hooks"] = hooks
}

// removeClaudeHook removes the envault PreToolUse and PostToolUse hook groups.
func removeClaudeHook(data map[string]interface{}) {
	hooks := hooksMap(data)

	filterGroups := func(groups []interface{}) []interface{} {
		out := make([]interface{}, 0, len(groups))
		for _, g := range groups {
			if !matchesEnvaultGroup(g) {
				out = append(out, g)
			}
		}
		return out
	}

	if pre := filterGroups(preToolUseGroups(data)); len(pre) == 0 {
		delete(hooks, "PreToolUse")
	} else {
		hooks["PreToolUse"] = pre
	}

	if post := filterGroups(postToolUseGroups(data)); len(post) == 0 {
		delete(hooks, "PostToolUse")
	} else {
		hooks["PostToolUse"] = post
	}

	if len(hooks) == 0 {
		delete(data, "hooks")
	} else {
		data["hooks"] = hooks
	}
}

// envaultPreHookGroup returns the PreToolUse hook group entry.
// matcher ".*" intercepts all tools so protected-path blocking covers Read/Write/Edit
// as well as Bash. The handler routes by tool_name internally.
func envaultPreHookGroup(cmd string) map[string]interface{} {
	return map[string]interface{}{
		"matcher": ".*",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":     "command",
				"command":  cmd + " preuse",
				"_envault": claudeHookID,
			},
		},
	}
}

// envaultPostHookGroup returns the PostToolUse hook group entry for placeholder injection.
func envaultPostHookGroup(cmd string) map[string]interface{} {
	return map[string]interface{}{
		"matcher": ".*",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":     "command",
				"command":  cmd + " postuse",
				"_envault": claudeHookID,
			},
		},
	}
}

// matchesEnvaultGroup returns true when g is the envault-managed hook group.
// Detection uses the stable _envault marker field as primary signal, with
// a command-suffix fallback for entries written before path-aware install.
func matchesEnvaultGroup(g interface{}) bool {
	m, ok := g.(map[string]interface{})
	if !ok {
		return false
	}
	hooks, _ := m["hooks"].([]interface{})
	for _, h := range hooks {
		hm, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		// Primary: stable marker field (present since v0.8.0).
		if hm["_envault"] == claudeHookID {
			return true
		}
		// Fallback: match by command suffix for legacy entries without the marker.
		cmd, _ := hm["command"].(string)
		if strings.HasSuffix(cmd, "/envault hook preuse") ||
			strings.HasSuffix(cmd, "/envault hook postuse") ||
			cmd == "envault hook preuse" ||
			cmd == "envault hook postuse" {
			return true
		}
	}
	return false
}

// hooksMap returns the "hooks" key from data as a map, creating it if missing.
func hooksMap(data map[string]interface{}) map[string]interface{} {
	if v, ok := data["hooks"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	m := map[string]interface{}{}
	data["hooks"] = m
	return m
}

// preToolUseGroups returns the PreToolUse array from hooks, or nil if absent.
func preToolUseGroups(data map[string]interface{}) []interface{} {
	return hookGroups(data, "PreToolUse")
}

// postToolUseGroups returns the PostToolUse array from hooks, or nil if absent.
func postToolUseGroups(data map[string]interface{}) []interface{} {
	return hookGroups(data, "PostToolUse")
}

func hookGroups(data map[string]interface{}, key string) []interface{} {
	hooks := hooksMap(data)
	v, ok := hooks[key]
	if !ok {
		return nil
	}
	if s, ok := v.([]interface{}); ok {
		return s
	}
	return nil
}

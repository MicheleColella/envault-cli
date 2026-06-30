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

	addClaudeHook(data, hookCommand())

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

// hookCommand returns the full shell command to register as the PreToolUse hook.
// It resolves the absolute path of the running envault binary so Claude Code can
// invoke it even when the binary is not on PATH (e.g. a local test copy).
// Falls back to the bare name "envault hook preuse" when the path cannot be resolved.
func hookCommand() string {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		return "envault hook preuse"
	}
	real, err := filepath.EvalSymlinks(exe)
	if err == nil {
		exe = real
	}
	return exe + " hook preuse"
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
		_ = os.WriteFile(path+".bak", prev, 0o600)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o600); err != nil {
		return fmt.Errorf("write settings.json: %w", err)
	}
	return nil
}

// isClaudeHookPresent checks whether the envault hook entry is already in data.
func isClaudeHookPresent(data map[string]interface{}) bool {
	groups := preToolUseGroups(data)
	for _, g := range groups {
		if matchesEnvaultGroup(g) {
			return true
		}
	}
	return false
}

// addClaudeHook appends the envault hook group to PreToolUse.
func addClaudeHook(data map[string]interface{}, cmd string) {
	hooks := hooksMap(data)
	groups := preToolUseGroups(data)
	groups = append(groups, envaultHookGroup(cmd))
	hooks["PreToolUse"] = groups
	data["hooks"] = hooks
}

// removeClaudeHook removes the envault hook group from PreToolUse.
func removeClaudeHook(data map[string]interface{}) {
	hooks := hooksMap(data)
	groups := preToolUseGroups(data)

	filtered := make([]interface{}, 0, len(groups))
	for _, g := range groups {
		if !matchesEnvaultGroup(g) {
			filtered = append(filtered, g)
		}
	}

	if len(filtered) == 0 {
		delete(hooks, "PreToolUse")
	} else {
		hooks["PreToolUse"] = filtered
	}

	if len(hooks) == 0 {
		delete(data, "hooks")
	} else {
		data["hooks"] = hooks
	}
}

// envaultHookGroup returns the settings.json hook-group entry for the given command.
func envaultHookGroup(cmd string) map[string]interface{} {
	return map[string]interface{}{
		"matcher": "Bash",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": cmd,
				// Stable ID used to locate/remove this entry regardless of the
				// binary path that was in effect when the hook was installed.
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
		if cmd == "envault hook preuse" || strings.HasSuffix(cmd, "/envault hook preuse") {
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
	hooks := hooksMap(data)
	v, ok := hooks["PreToolUse"]
	if !ok {
		return nil
	}
	if s, ok := v.([]interface{}); ok {
		return s
	}
	return nil
}

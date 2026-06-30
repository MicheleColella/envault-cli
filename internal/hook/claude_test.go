package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallClaudeHook_WritesSettingsFile(t *testing.T) {
	dir := t.TempDir()

	if err := InstallClaudeHook(dir, false); err != nil {
		t.Fatalf("InstallClaudeHook: %v", err)
	}

	path := filepath.Join(dir, ".claude", "settings.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("settings.json not created: %v", err)
	}

	var data map[string]interface{}
	b, _ := os.ReadFile(path)
	if err := json.Unmarshal(b, &data); err != nil {
		t.Fatalf("settings.json not valid JSON: %v", err)
	}
}

func TestInstallClaudeHook_Idempotent(t *testing.T) {
	dir := t.TempDir()

	if err := InstallClaudeHook(dir, false); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if err := InstallClaudeHook(dir, false); err != nil {
		t.Fatalf("second install: %v", err)
	}

	// Ensure there is exactly one envault hook group in PreToolUse.
	data, _ := readSettings(claudeSettingsPath(dir, false))
	groups := preToolUseGroups(data)
	count := 0
	for _, g := range groups {
		if matchesEnvaultGroup(g) {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 envault hook group after two installs, got %d", count)
	}
}

func TestIsClaudeHookInstalled_FalseBeforeInstall(t *testing.T) {
	dir := t.TempDir()
	if IsClaudeHookInstalled(dir, false) {
		t.Error("expected false before install")
	}
}

func TestIsClaudeHookInstalled_TrueAfterInstall(t *testing.T) {
	dir := t.TempDir()
	if err := InstallClaudeHook(dir, false); err != nil {
		t.Fatalf("InstallClaudeHook: %v", err)
	}
	if !IsClaudeHookInstalled(dir, false) {
		t.Error("expected true after install")
	}
}

func TestUninstallClaudeHook_RemovesEntry(t *testing.T) {
	dir := t.TempDir()

	if err := InstallClaudeHook(dir, false); err != nil {
		t.Fatalf("install: %v", err)
	}
	if err := UninstallClaudeHook(dir, false); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	if IsClaudeHookInstalled(dir, false) {
		t.Error("hook still reported as installed after uninstall")
	}
}

func TestUninstallClaudeHook_NoopWhenNotInstalled(t *testing.T) {
	dir := t.TempDir()
	if err := UninstallClaudeHook(dir, false); err != nil {
		t.Fatalf("uninstall on clean dir: %v", err)
	}
}

func TestInstallClaudeHook_PreservesExistingKeys(t *testing.T) {
	dir := t.TempDir()

	// Write a settings.json with some pre-existing content.
	existing := map[string]interface{}{
		"theme": "dark",
		"hooks": map[string]interface{}{
			"Stop": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{"type": "command", "command": "echo done"},
					},
				},
			},
		},
	}
	clDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(clDir, 0o700)
	b, _ := json.MarshalIndent(existing, "", "  ")
	_ = os.WriteFile(filepath.Join(clDir, "settings.json"), b, 0o600)

	if err := InstallClaudeHook(dir, false); err != nil {
		t.Fatalf("InstallClaudeHook: %v", err)
	}

	data, err := readSettings(claudeSettingsPath(dir, false))
	if err != nil {
		t.Fatalf("readSettings: %v", err)
	}

	// "theme" key must still be present.
	if data["theme"] != "dark" {
		t.Error("existing 'theme' key was overwritten")
	}

	// "Stop" hook must still be present.
	hooks := hooksMap(data)
	if _, ok := hooks["Stop"]; !ok {
		t.Error("existing 'Stop' hook was removed")
	}

	// Envault PreToolUse hook must also be present.
	if !IsClaudeHookInstalled(dir, false) {
		t.Error("envault hook not installed after merge")
	}
}

func TestUninstallClaudeHook_LeavesOtherHooksIntact(t *testing.T) {
	dir := t.TempDir()

	// Install envault + add another pre-tool-use group.
	if err := InstallClaudeHook(dir, false); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Inject a second PreToolUse group manually.
	path := claudeSettingsPath(dir, false)
	data, _ := readSettings(path)
	hooks := hooksMap(data)
	groups := preToolUseGroups(data)
	groups = append(groups, map[string]interface{}{
		"matcher": "Bash",
		"hooks": []interface{}{
			map[string]interface{}{"type": "command", "command": "other-hook"},
		},
	})
	hooks["PreToolUse"] = groups
	data["hooks"] = hooks
	_ = writeSettings(path, data)

	// Uninstall envault hook.
	if err := UninstallClaudeHook(dir, false); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	// Other hook must still be present.
	data, _ = readSettings(path)
	groups = preToolUseGroups(data)
	found := false
	for _, g := range groups {
		m, ok := g.(map[string]interface{})
		if !ok {
			continue
		}
		hooks2, _ := m["hooks"].([]interface{})
		for _, h := range hooks2 {
			hm, ok := h.(map[string]interface{})
			if ok && hm["command"] == "other-hook" {
				found = true
			}
		}
	}
	if !found {
		t.Error("other-hook was removed during envault uninstall")
	}
}

func TestInstallClaudeHook_CreatesBackup(t *testing.T) {
	dir := t.TempDir()

	// Write initial settings.json.
	clDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(clDir, 0o700)
	initial := []byte(`{"theme":"dark"}` + "\n")
	settingsPath := filepath.Join(clDir, "settings.json")
	_ = os.WriteFile(settingsPath, initial, 0o600)

	if err := InstallClaudeHook(dir, false); err != nil {
		t.Fatalf("InstallClaudeHook: %v", err)
	}

	bak, err := os.ReadFile(settingsPath + ".bak")
	if err != nil {
		t.Fatalf("backup not created: %v", err)
	}
	if string(bak) != string(initial) {
		t.Errorf("backup content mismatch: got %q, want %q", bak, initial)
	}
}

// TestSnapshotSettingsJSON verifies that repeated install/uninstall cycles
// never leave settings.json in an invalid or corrupted state.
func TestSnapshotSettingsJSON_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	for i := range 3 {
		if err := InstallClaudeHook(dir, false); err != nil {
			t.Fatalf("install cycle %d: %v", i, err)
		}
		b, err := os.ReadFile(claudeSettingsPath(dir, false))
		if err != nil {
			t.Fatalf("read settings.json cycle %d: %v", i, err)
		}
		var data map[string]interface{}
		if err := json.Unmarshal(b, &data); err != nil {
			t.Fatalf("settings.json invalid JSON after install cycle %d: %v\n%s", i, err, b)
		}

		if err := UninstallClaudeHook(dir, false); err != nil {
			t.Fatalf("uninstall cycle %d: %v", i, err)
		}
		b, err = os.ReadFile(claudeSettingsPath(dir, false))
		if err != nil {
			t.Fatalf("read settings.json after uninstall cycle %d: %v", i, err)
		}
		if err := json.Unmarshal(b, &data); err != nil {
			t.Fatalf("settings.json invalid JSON after uninstall cycle %d: %v\n%s", i, err, b)
		}
	}
}

// TestInstallClaudeHook_ToleratesMalformedJSON verifies that installing into a
// settings.json that contains malformed JSON does not silently produce a
// corrupted file — it should return an error, not write garbage.
func TestInstallClaudeHook_ToleratesMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	clDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(clDir, 0o700)
	_ = os.WriteFile(filepath.Join(clDir, "settings.json"), []byte("{broken json}"), 0o600)

	err := InstallClaudeHook(dir, false)
	if err != nil {
		// Error is the correct behaviour — don't corrupt.
		return
	}

	// If it succeeded, the file must be valid JSON.
	b, readErr := os.ReadFile(filepath.Join(clDir, "settings.json"))
	if readErr != nil {
		t.Fatalf("read settings.json: %v", readErr)
	}
	var data map[string]interface{}
	if jsonErr := json.Unmarshal(b, &data); jsonErr != nil {
		t.Errorf("settings.json corrupted after install into malformed file: %v\n%s", jsonErr, b)
	}
}

func TestInstallClaudeHook_GlobalUsesHomeDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Use a dummy repoRoot — global install must ignore it.
	dummy := t.TempDir()

	if err := InstallClaudeHook(dummy, true); err != nil {
		t.Fatalf("InstallClaudeHook global: %v", err)
	}

	globalPath := filepath.Join(home, ".claude", "settings.json")
	if _, err := os.Stat(globalPath); err != nil {
		t.Fatalf("global settings.json not created at %s: %v", globalPath, err)
	}

	if !IsClaudeHookInstalled(dummy, true) {
		t.Error("IsClaudeHookInstalled(global) returned false after global install")
	}

	// Local path must not have been touched.
	localPath := filepath.Join(dummy, ".claude", "settings.json")
	if _, err := os.Stat(localPath); err == nil {
		t.Error("local settings.json was written during global install")
	}
}

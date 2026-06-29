package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- RunHookPreuse ----------------------------------------------------------

func TestRunHookPreuse_AllowsNonSensitiveCommand(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".envault"), 0o700)

	origWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	input := map[string]interface{}{
		"tool_name":  "Bash",
		"tool_input": map[string]interface{}{"command": "npm install"},
	}
	b, _ := json.Marshal(input)
	var w bytes.Buffer

	if err := RunHookPreuse(bytes.NewReader(b), &w); err != nil {
		t.Fatalf("expected no error for non-sensitive command, got: %v", err)
	}
	if w.Len() != 0 {
		t.Errorf("expected no output for allowed command, got: %s", w.String())
	}
}

func TestRunHookPreuse_BlocksEnvaultCat(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".envault"), 0o700)

	origWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	for _, cmd := range []string{
		"envault cat DB_URL",
		"./envault cat API_KEY",
		"envault export",
		"/usr/local/bin/envault cat SECRET",
	} {
		input := map[string]interface{}{
			"tool_name":  "Bash",
			"tool_input": map[string]interface{}{"command": cmd},
		}
		b, _ := json.Marshal(input)
		var w bytes.Buffer

		err := RunHookPreuse(bytes.NewReader(b), &w)
		if err == nil {
			t.Errorf("cmd %q: expected ErrBlockToolCall, got nil", cmd)
			continue
		}
		if w.Len() == 0 {
			t.Errorf("cmd %q: expected block reason written to output, got nothing", cmd)
		}
		if !strings.Contains(w.String(), "envault run") {
			t.Errorf("cmd %q: block message should mention 'envault run', got: %s", cmd, w.String())
		}
	}
}

func TestRunHookPreuse_AllowsCatWithForce(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".envault"), 0o700)

	origWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	input := map[string]interface{}{
		"tool_name":  "Bash",
		"tool_input": map[string]interface{}{"command": "envault cat DB_URL --force"},
	}
	b, _ := json.Marshal(input)
	var w bytes.Buffer

	if err := RunHookPreuse(bytes.NewReader(b), &w); err != nil {
		t.Fatalf("expected no error for cat --force, got: %v", err)
	}
	if w.Len() != 0 {
		t.Errorf("expected no output for cat --force, got: %s", w.String())
	}
}

func TestRunHookPreuse_NoopOutsideEnvaultRepo(t *testing.T) {
	dir := t.TempDir() // no .envault/

	origWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	input := map[string]interface{}{
		"tool_name":  "Bash",
		"tool_input": map[string]interface{}{"command": "envault cat DB_URL"},
	}
	b, _ := json.Marshal(input)
	var w bytes.Buffer

	if err := RunHookPreuse(bytes.NewReader(b), &w); err != nil {
		t.Fatalf("expected no error outside envault repo, got: %v", err)
	}
	if w.Len() != 0 {
		t.Errorf("expected no output outside envault repo, got: %s", w.String())
	}
}

func TestRunHookPreuse_NoopForNonBashTool(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".envault"), 0o700)

	origWd, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	input := map[string]interface{}{
		"tool_name":  "Read",
		"tool_input": map[string]interface{}{"file_path": "/etc/hosts"},
	}
	b, _ := json.Marshal(input)
	var w bytes.Buffer

	if err := RunHookPreuse(bytes.NewReader(b), &w); err != nil {
		t.Fatalf("expected no error for non-Bash tool, got: %v", err)
	}
	if w.Len() != 0 {
		t.Errorf("expected no output for non-Bash tool, got: %s", w.String())
	}
}

func TestRunHookPreuse_InvalidJSONIsNoop(t *testing.T) {
	var w bytes.Buffer
	if err := RunHookPreuse(strings.NewReader("not json at all"), &w); err != nil {
		t.Fatalf("unexpected error on invalid JSON: %v", err)
	}
	if w.Len() != 0 {
		t.Errorf("expected no output for invalid JSON input, got: %s", w.String())
	}
}

// ---- IsSensitiveEnvaultCmd --------------------------------------------------

func TestIsSensitiveEnvaultCmd(t *testing.T) {
	sensitive := []string{
		"envault cat DB_URL",
		"./envault cat KEY",
		"/usr/local/bin/envault cat KEY",
		"envault export",
		"./envault export",
	}
	notSensitive := []string{
		"envault cat DB_URL --force",
		"envault list",
		"envault run -- npm start",
		"npm install",
		"echo envault cat",   // envault is not a command here
		"envault add DB_URL", // not cat/export
	}

	for _, cmd := range sensitive {
		if !IsSensitiveEnvaultCmd(cmd) {
			t.Errorf("expected %q to be sensitive, got false", cmd)
		}
	}
	for _, cmd := range notSensitive {
		if IsSensitiveEnvaultCmd(cmd) {
			t.Errorf("expected %q to NOT be sensitive, got true", cmd)
		}
	}
}

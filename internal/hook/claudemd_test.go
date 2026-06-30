package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectClaudeMD_CreatesFileWhenAbsent(t *testing.T) {
	dir := t.TempDir()

	if err := InjectClaudeMD(dir); err != nil {
		t.Fatalf("InjectClaudeMD: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}
	if !strings.Contains(string(b), claudeMDMarkerStart) {
		t.Error("CLAUDE.md missing start marker")
	}
	if !strings.Contains(string(b), claudeMDMarkerEnd) {
		t.Error("CLAUDE.md missing end marker")
	}
}

func TestInjectClaudeMD_AppendsToExistingFile(t *testing.T) {
	dir := t.TempDir()
	existing := "# My Project\n\nSome content.\n"
	_ = os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(existing), 0o644)

	if err := InjectClaudeMD(dir); err != nil {
		t.Fatalf("InjectClaudeMD: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	content := string(b)

	if !strings.HasPrefix(content, existing) {
		t.Error("existing content was not preserved at start of file")
	}
	if !strings.Contains(content, claudeMDMarkerStart) {
		t.Error("start marker not appended")
	}
	if !strings.Contains(content, "envault list") {
		t.Error("command list not present in injected section")
	}
}

func TestInjectClaudeMD_Idempotent(t *testing.T) {
	dir := t.TempDir()

	if err := InjectClaudeMD(dir); err != nil {
		t.Fatalf("first inject: %v", err)
	}
	if err := InjectClaudeMD(dir); err != nil {
		t.Fatalf("second inject: %v", err)
	}

	b, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	count := strings.Count(string(b), claudeMDMarkerStart)
	if count != 1 {
		t.Errorf("expected exactly 1 start marker after two injects, got %d", count)
	}
}

func TestInjectClaudeMD_ReplacesExistingSection(t *testing.T) {
	dir := t.TempDir()

	// Write an old section with outdated content.
	old := "# Project\n\n" + claudeMDMarkerStart + "\nold content\n" + claudeMDMarkerEnd + "\n"
	_ = os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(old), 0o644)

	if err := InjectClaudeMD(dir); err != nil {
		t.Fatalf("InjectClaudeMD: %v", err)
	}

	b, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	content := string(b)

	if strings.Contains(content, "old content") {
		t.Error("old section content still present after replace")
	}
	if strings.Count(content, claudeMDMarkerStart) != 1 {
		t.Error("unexpected number of start markers after replace")
	}
	if !strings.Contains(content, "envault list") {
		t.Error("new section content not present after replace")
	}
}

func TestIsClaudeMDInjected_FalseWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	if IsClaudeMDInjected(dir) {
		t.Error("expected false when no CLAUDE.md")
	}
}

func TestIsClaudeMDInjected_FalseWhenNoMarker(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Project\n"), 0o644)
	if IsClaudeMDInjected(dir) {
		t.Error("expected false when CLAUDE.md has no marker")
	}
}

func TestIsClaudeMDInjected_TrueAfterInject(t *testing.T) {
	dir := t.TempDir()
	if err := InjectClaudeMD(dir); err != nil {
		t.Fatalf("InjectClaudeMD: %v", err)
	}
	if !IsClaudeMDInjected(dir) {
		t.Error("expected true after inject")
	}
}

func TestRemoveClaudeMDSection_RemovesSection(t *testing.T) {
	dir := t.TempDir()

	if err := InjectClaudeMD(dir); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if err := RemoveClaudeMDSection(dir); err != nil {
		t.Fatalf("remove: %v", err)
	}

	if IsClaudeMDInjected(dir) {
		t.Error("section still present after remove")
	}
}

func TestRemoveClaudeMDSection_PreservesOtherContent(t *testing.T) {
	dir := t.TempDir()
	before := "# Project\n\nSome content.\n"
	_ = os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(before), 0o644)

	if err := InjectClaudeMD(dir); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if err := RemoveClaudeMDSection(dir); err != nil {
		t.Fatalf("remove: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read after remove: %v", err)
	}
	if !strings.Contains(string(b), "Some content.") {
		t.Error("existing content was lost after remove")
	}
}

func TestRemoveClaudeMDSection_NoopWhenNotInjected(t *testing.T) {
	dir := t.TempDir()
	if err := RemoveClaudeMDSection(dir); err != nil {
		t.Fatalf("remove on empty dir: %v", err)
	}
}

func TestRemoveClaudeMDSection_DeletesFileWhenOnlySection(t *testing.T) {
	dir := t.TempDir()

	// Inject into an empty file → CLAUDE.md contains only the Envault section.
	if err := InjectClaudeMD(dir); err != nil {
		t.Fatalf("inject: %v", err)
	}
	if err := RemoveClaudeMDSection(dir); err != nil {
		t.Fatalf("remove: %v", err)
	}

	// File should be removed entirely (it had no other content).
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err == nil {
		t.Error("CLAUDE.md should be removed when it only contained the Envault section")
	}
}

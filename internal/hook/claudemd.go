package hook

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	claudeMDMarkerStart = "<!-- envault:start -->"
	claudeMDMarkerEnd   = "<!-- envault:end -->"
)

// claudeMDContent is the Envault section injected into the project CLAUDE.md.
// Uses string concatenation to avoid raw-string backtick conflicts.
var claudeMDContent = claudeMDMarkerStart + "\n" +
	"## Envault\n\n" +
	"This vault is managed by Envault. Secrets are encrypted at rest — never plaintext on disk.\n\n" +
	"**Security rules for Claude Code:**\n" +
	"- NEVER run `envault cat` or `envault export` without `--force` — the PreToolUse hook blocks them to prevent plaintext in model context\n" +
	"- Prefer `envault run -- <cmd>` to inject secrets in memory without writing them anywhere\n" +
	"- Do not print or log the contents of `.envault/secrets.enc`\n\n" +
	"**Common commands:**\n" +
	"- `envault list` — show all sealed entries (names only, no decryption)\n" +
	"- `envault add <KEY>` — seal a new secret\n" +
	"- `envault run -- <cmd>` — run a command with secrets injected in memory\n" +
	"- `envault push` / `envault pull` — sync vault with team via Git\n" +
	"- `envault key list` — show vault recipients\n" +
	claudeMDMarkerEnd + "\n"

// InjectClaudeMD adds (or replaces) the Envault section in <repoRoot>/CLAUDE.md.
// The section is bounded by HTML comment markers for idempotent updates.
// Creates CLAUDE.md if it does not exist.
func InjectClaudeMD(repoRoot string) error {
	path := claudeMDPath(repoRoot)

	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	existing := string(b)

	var updated string
	if start := strings.Index(existing, claudeMDMarkerStart); start >= 0 {
		end := strings.Index(existing, claudeMDMarkerEnd)
		if end < 0 {
			end = start
		} else {
			end += len(claudeMDMarkerEnd)
			// Consume trailing newline after end marker if present.
			if end < len(existing) && existing[end] == '\n' {
				end++
			}
		}
		updated = existing[:start] + claudeMDContent + existing[end:]
	} else if existing == "" {
		updated = claudeMDContent
	} else {
		sep := "\n"
		if strings.HasSuffix(existing, "\n\n") {
			sep = ""
		} else if !strings.HasSuffix(existing, "\n") {
			sep = "\n\n"
		}
		updated = existing + sep + claudeMDContent
	}

	return os.WriteFile(path, []byte(updated), 0o644)
}

// RemoveClaudeMDSection removes the Envault section from <repoRoot>/CLAUDE.md.
// Returns nil when the file or section does not exist.
func RemoveClaudeMDSection(repoRoot string) error {
	path := claudeMDPath(repoRoot)
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	content := string(b)
	start := strings.Index(content, claudeMDMarkerStart)
	if start < 0 {
		return nil
	}
	end := strings.Index(content, claudeMDMarkerEnd)
	if end < 0 {
		end = start
	} else {
		end += len(claudeMDMarkerEnd)
		if end < len(content) && content[end] == '\n' {
			end++
		}
	}

	before := strings.TrimRight(content[:start], "\n")
	after := strings.TrimLeft(content[end:], "\n")

	var result string
	switch {
	case before == "" && after == "":
		result = ""
	case before == "":
		result = after
	case after == "":
		result = before + "\n"
	default:
		result = before + "\n\n" + after
	}

	if result == "" {
		return os.Remove(path)
	}
	return os.WriteFile(path, []byte(result), 0o644)
}

// IsClaudeMDInjected reports whether the Envault section is present in <repoRoot>/CLAUDE.md.
func IsClaudeMDInjected(repoRoot string) bool {
	b, err := os.ReadFile(claudeMDPath(repoRoot))
	if err != nil {
		return false
	}
	return strings.Contains(string(b), claudeMDMarkerStart)
}

func claudeMDPath(repoRoot string) string {
	return filepath.Join(repoRoot, "CLAUDE.md")
}

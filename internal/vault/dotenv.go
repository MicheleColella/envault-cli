package vault

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// EnvVar is a single key/value pair parsed from a dotenv file.
type EnvVar struct {
	Key   string
	Value string
}

// ParseDotenv parses dotenv content: one KEY=VALUE per line. Blank lines and
// comments (lines starting with '#') are skipped, an optional leading "export "
// is honored, and a value wrapped in matching single or double quotes is
// unquoted. Inline comments are intentionally not stripped so values may
// contain '#'.
func ParseDotenv(r io.Reader) ([]EnvVar, error) {
	var out []EnvVar
	sc := bufio.NewScanner(r)
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		key, value, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("dotenv line %d: missing '=' in %q", lineNum, line)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("dotenv line %d: empty key", lineNum)
		}
		out = append(out, EnvVar{Key: key, Value: unquoteValue(strings.TrimSpace(value))})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan dotenv: %w", err)
	}
	return out, nil
}

// unquoteValue strips a single pair of matching surrounding quotes, if present.
func unquoteValue(v string) string {
	if len(v) >= 2 {
		first, last := v[0], v[len(v)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}

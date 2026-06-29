// Package scan implements the secret detection engine used by the envault
// pre-commit hook and the `envault scan` CLI command.
// It provides pattern-based rules and a Shannon-entropy heuristic for finding
// plaintext credentials in unified diffs and tracked file trees.
package scan

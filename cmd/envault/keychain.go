package main

import (
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/MicheleColella/envault-cli/internal/keychain"
)

// passphraseEnv lets non-interactive callers (CI) supply the keychain passphrase
// without a TTY prompt. It is LESS SECURE than interactive entry: environment
// variables are visible to other same-user processes (e.g. via /proc or `ps -E`)
// and may be captured by shell history or process inspection. Use only in CI.
const passphraseEnv = "ENVAULT_PASSPHRASE" //nolint:gosec // G101 false positive: this is the env var NAME, not a credential value

// openKeychain returns the OS keychain wrapped in the passphrase-protection
// decorator. Every private key is encrypted under a passphrase-derived KEK
// before it reaches the OS secret store, so a silently extracted blob is
// useless ciphertext.
func openKeychain() (keychain.Store, error) {
	inner, err := keychain.New()
	if err != nil {
		return nil, err
	}
	return keychain.NewProtected(inner, askPassphrase), nil
}

// askPassphrase obtains the keychain passphrase, preferring the ENVAULT_PASSPHRASE
// environment variable (for CI) and falling back to a hidden interactive prompt.
func askPassphrase(prompt string) ([]byte, error) {
	if env := os.Getenv(passphraseEnv); env != "" {
		return []byte(env), nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, fmt.Errorf(
			"no passphrase available: set %s or run interactively to unlock the keychain",
			passphraseEnv,
		)
	}

	fmt.Fprintf(os.Stderr, "%s: ", prompt)
	pass, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("read passphrase: %w", err)
	}
	if len(pass) == 0 {
		return nil, fmt.Errorf("passphrase must not be empty")
	}
	return pass, nil
}

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	envcrypto "github.com/MicheleColella/envault-cli/internal/crypto"
	"github.com/MicheleColella/envault-cli/internal/keychain"
	"github.com/MicheleColella/envault-cli/internal/ui"
	"github.com/MicheleColella/envault-cli/internal/vault"
)

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run -- <command> [args...]",
		Short: "Inject decrypted secrets and execute a command",
		Long: "Decrypt all env secrets into memory and run the given command with\n" +
			"them injected as environment variables. Secrets are never written to\n" +
			"disk and are zeroed from memory when the command exits.\n\n" +
			"Use -- to separate envault flags from the child command:\n" +
			"  envault run -- npm start\n" +
			"  envault run -- go test ./...",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kc, err := keychain.New()
			if err != nil {
				return fmt.Errorf("open keychain: %w", err)
			}
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			err = runRun(wd, args, kc)
			var ece exitCodeError
			if errors.As(err, &ece) {
				os.Exit(ece.code)
			}
			return err
		},
	}
}

// exitCodeError carries the exit code of a child process so the parent can
// exit with the same code after all deferred cleanup has run.
type exitCodeError struct{ code int }

func (e exitCodeError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

// runRun decrypts all KindEnv secrets into memory, injects them into the
// child's environment, and runs args[0] with args[1:]. On child exit the
// plaintext is zeroed. The child's exit code is returned as exitCodeError so
// the caller can propagate it after cleanup.
func runRun(repoRoot string, args []string, kc keychain.Store) error {
	if !vault.IsInitialized(repoRoot) {
		return fmt.Errorf("vault not initialized — run `envault init` first")
	}

	store, err := vault.LoadStore(repoRoot)
	if err != nil {
		return err
	}

	var envEntries []vault.Entry
	for _, e := range store.Entries {
		if e.Kind == vault.KindEnv {
			envEntries = append(envEntries, e)
		}
	}

	priv, id, err := loadCurrentUserKey(repoRoot, kc)
	if err != nil {
		return err
	}
	defer clear(priv[:])

	ui.Info(fmt.Sprintf("decrypting %d secret(s) in memory as %s…", len(envEntries), id))

	plaintexts := make([][]byte, len(envEntries))
	extraEnv := make([]string, len(envEntries))

	for i, e := range envEntries {
		pt, err := envcrypto.Unseal(e.Envelope, priv)
		if err != nil {
			for j := 0; j < i; j++ {
				clear(plaintexts[j])
			}
			return fmt.Errorf("decrypt %s: %w", e.Name, err)
		}
		plaintexts[i] = pt
		extraEnv[i] = fmt.Sprintf("%s=%s", e.Name, string(pt))
	}
	defer func() {
		for _, pt := range plaintexts {
			clear(pt)
		}
	}()

	ui.OK(fmt.Sprintf("injected %d env var(s) — 0 bytes written to disk", len(envEntries)))

	// Merge parent env with vault secrets; vault entries override duplicates.
	childEnv := append(os.Environ(), extraEnv...)

	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // user-supplied command is intentional
	cmd.Env = childEnv
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %q: %w", args[0], err)
	}

	// Forward SIGINT and SIGTERM to the child so it can clean up.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range sigCh {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	waitErr := cmd.Wait()
	signal.Stop(sigCh)
	close(sigCh)

	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return exitCodeError{exitErr.ExitCode()}
		}
		return waitErr
	}
	return nil
}

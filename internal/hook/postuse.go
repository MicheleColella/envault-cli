package hook

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MicheleColella/envault-cli/internal/audit"
	envcrypto "github.com/MicheleColella/envault-cli/internal/crypto"
	"github.com/MicheleColella/envault-cli/internal/keychain"
	"github.com/MicheleColella/envault-cli/internal/vault"
)

// PostuseInput is the Claude Code PostToolUse hook JSON.
type PostuseInput struct {
	ToolName     string          `json:"tool_name"`
	ToolInput    json.RawMessage `json:"tool_input"`
	ToolResponse json.RawMessage `json:"tool_response"`
}

// RunHookPostuse reads Claude Code's PostToolUse JSON from r.
//
// If ENVAULT_PASSPHRASE is set, decrypts all KindEnv vault secrets and scans
// the tool response for their plaintext values (and base64-encoded variants),
// replacing each match with a structured placeholder
// `<ENVAULT:NAME>` (or `<ENVAULT:NAME|base64>`) before writing to w and exiting
// with code 2 so Claude Code uses the masked output instead of the original.
//
// When the passphrase is unavailable or the vault is absent the function returns
// nil (pass-through), logging the skip to the audit log.
func RunHookPostuse(r io.Reader, w io.Writer) error {
	var input PostuseInput
	if err := json.NewDecoder(r).Decode(&input); err != nil {
		return nil
	}

	wd, err := os.Getwd()
	if err != nil || !IsEnvaultDir(wd) {
		return nil
	}

	passphrase := os.Getenv("ENVAULT_PASSPHRASE")
	if passphrase == "" {
		return nil // cannot decrypt without passphrase in hook context
	}

	secrets, err := loadSecretsForMasking(wd, []byte(passphrase))
	if err != nil || len(secrets) == 0 {
		return nil
	}

	// Stringify the tool response for scanning.
	responseStr := string(input.ToolResponse)
	masked, names := maskSecrets(responseStr, secrets)

	if len(names) == 0 {
		return nil // nothing to mask — pass through
	}

	for _, name := range names {
		_ = audit.AppendEntry(wd, input.ToolName, audit.ActionMasked, name, "")
	}

	// Write masked response to w. Exit 2 causes Claude Code to use our output
	// instead of the original tool response, preventing plaintext from entering
	// the model context.
	_, _ = fmt.Fprint(w, masked)
	return ErrBlockToolCall // caller must exit 2
}

// secretValue pairs an env-var name with its decrypted plaintext.
type secretValue struct {
	Name      string
	Plaintext []byte
}

// loadSecretsForMasking opens the keychain with passphrase, decrypts all
// KindEnv entries, and returns the plaintext values. Caller must not store
// the returned slices beyond the call lifetime.
func loadSecretsForMasking(repoRoot string, passphrase []byte) ([]secretValue, error) {
	kc, err := keychain.New()
	if err != nil {
		return nil, err
	}

	askFunc := func(_ string) ([]byte, error) { return passphrase, nil }
	protected := keychain.NewProtected(kc, askFunc)

	store, err := vault.LoadStore(repoRoot)
	if err != nil {
		return nil, err
	}

	// Find which recipient key we own.
	recipients, err := vault.ListRecipients(repoRoot)
	if err != nil {
		return nil, err
	}

	var priv envcrypto.PrivateKey
	var found bool
	for _, rec := range recipients {
		raw, err := protected.Unseal(rec.ID)
		if err != nil {
			continue
		}
		if len(raw) == 32 {
			copy(priv[:], raw)
			clear(raw)
			found = true
			break
		}
		clear(raw)
	}
	if !found {
		return nil, nil
	}
	defer clear(priv[:])

	var out []secretValue
	for _, e := range store.Entries {
		if e.Kind != vault.KindEnv {
			continue
		}
		pt, err := envcrypto.Unseal(e.Envelope, priv)
		if err != nil {
			continue
		}
		out = append(out, secretValue{Name: e.Name, Plaintext: pt})
	}
	return out, nil
}

// maskSecrets scans text for each secret's plaintext (and its base64 variant)
// and replaces matches with structured placeholders.
// Returns the masked text and the names of secrets that were replaced.
func maskSecrets(text string, secrets []secretValue) (string, []string) {
	var replaced []string
	result := text
	for _, s := range secrets {
		if len(s.Plaintext) == 0 {
			continue
		}
		plain := string(s.Plaintext)
		b64 := base64.StdEncoding.EncodeToString(s.Plaintext)

		placeholderB64 := fmt.Sprintf("<ENVAULT:%s|base64>", s.Name)
		placeholderPlain := fmt.Sprintf("<ENVAULT:%s>", s.Name)

		newText := strings.ReplaceAll(result, b64, placeholderB64)
		if newText != result {
			replaced = append(replaced, s.Name)
			result = newText
		}
		newText = strings.ReplaceAll(result, plain, placeholderPlain)
		if newText != result {
			if len(replaced) == 0 || replaced[len(replaced)-1] != s.Name {
				replaced = append(replaced, s.Name)
			}
			result = newText
		}
	}
	return result, replaced
}

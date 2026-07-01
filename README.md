# Envault

> Git-backed, zero-trust secrets manager for developer teams.

Envault encrypts your API keys and tokens directly inside your existing Git repo.
No central vault, no third-party trust, no `.env` files committed in plaintext.
Private keys never leave your machine. Secrets are injected into processes in memory
at runtime — never written to disk.

---

## Why Envault

Most teams leak secrets without realising it:

- `.env` files committed to repos (anyone with clone access has the keys)
- Secrets passed over Slack, email, or copy-paste
- A shared vault that requires trusting a third-party server

Envault is different: it uses your team's existing Git remote as the transport.
Each secret is encrypted end-to-end — only team members who have been granted
access can decrypt. Remove a member from the vault and they lose future access.
No new infrastructure required.

---

## Install

**One-line install** (macOS / Linux, amd64 / arm64) — downloads the latest signed
release, verifies its checksum, and drops `envault` on your `PATH`:

```sh
curl -fsSL https://raw.githubusercontent.com/MicheleColella/envault-cli/main/scripts/install.sh | sh
```

Override the target with `ENVAULT_VERSION=v0.9.0` or `ENVAULT_INSTALL_DIR=~/.local/bin`.

**With Go** (any platform Go supports):

```sh
go install github.com/MicheleColella/envault-cli/cmd/envault@latest
```

**From source:**

```sh
make build && sudo make install   # builds ./envault and installs to /usr/local/bin
```

Releases are cross-compiled in CI and published to [GitHub Releases](https://github.com/MicheleColella/envault-cli/releases)
with a `checksums.txt` signed via keyless [cosign](https://github.com/sigstore/cosign).
Windows binaries are not yet published (pending a Windows keychain backend).

---

## Claude Code plugin

Envault ships as a [Claude Code](https://claude.com/claude-code) plugin: the AI
Privacy Shield hooks, an embedded MCP server, the `/envault:*` slash commands,
and a skill that teaches Claude the vault workflow, all enabled per-project
(never globally by default).

The MCP server (`envault mcp serve`) exposes typed, JSON-Schema-validated
tools (`envault_status`, `envault_list`, `envault_rotate`, `envault_run`,
`envault_protect`, `envault_push`, `envault_pull`) so Claude calls Envault
directly instead of constructing bash commands — there's no shell string to
parse, so no shell-injection surface, and tool responses carry only metadata
(name, algorithm, recipient count, timestamps), never a secret value. It runs
headless, per Claude Code session, with no persistent daemon. Sealing a
**new** secret (`add`/`set`) is deliberately not exposed to Claude at all,
via MCP or bash — that requires a human to type it in their own terminal, or
the plaintext would have to pass through the model's context first.

Since the MCP server is headless, operations needing your private key
(`rotate`, `run`, `protect encrypt`, `push`/`pull`) normally need
`ENVAULT_PASSPHRASE` set. Run `envault agent unlock` once from your own
terminal instead: it prompts for the passphrase interactively, then a small
background agent (`~/.envault/agent.sock`, ssh-agent-style) keeps the
decrypted key cached in memory for a bounded time (default 8h) — no
passphrase needed for the rest of that window, even in a Claude Code session
opened later from a different terminal. This is opt-in and widens the
key's exposure window in exchange for convenience; `envault status`/`doctor`
always show whether it's active.

```text
/plugin marketplace add MicheleColella/envault-cli
/plugin install envault@envault
```

`envault@envault` is `<plugin>@<marketplace>` — both are named `envault` in
[`marketplace.json`](.claude-plugin/marketplace.json).

Enabling the plugin is reversible (`/plugin uninstall envault`) and scoped via
`.claude/settings.json` (`enabledPlugins`). The plugin is **additive** — the CLI
installs above are still the way to use Envault from a plain terminal.

**Packaging decision:** the plugin's hooks call `envault` on your `PATH`; it does
**not** bundle platform-specific binaries. Install the binary once via any method
above, then enable the plugin. Run `envault doctor` if the hooks report the binary
is missing.

---

## Quick start

```sh
# 1. Initialise a vault in your repo
envault init

# 2. Generate your identity key (sealed in your OS keychain, encrypted at rest
#    under a passphrase you choose — the key never leaves your machine)
envault key new --id you@example.com

# 3. Add a secret
echo "sk-abc123" | envault add OPENAI_KEY

# 4. Push the encrypted vault to your remote
envault push

# 5. A teammate pulls and runs their app with secrets injected in memory
envault pull
envault run -- npm start
```

> Reading a key (e.g. `run`, `cat`, `export`) asks for the passphrase that protects
> it. For non-interactive/CI use, supply it via the `ENVAULT_PASSPHRASE` environment
> variable (less secure — visible to same-user processes).

---

## Commands

| Command | Description | Status |
|---|---|---|
| `envault init` | Initialise a vault in the current repo | ✅ |
| `envault agent unlock/lock/stop/status` | Unlock your key into a background agent for passphrase-free use (Claude Code MCP, `postuse` masking) | ✅ |
| `envault key new` | Generate an identity key (sealed in OS keychain) | ✅ |
| `envault key list` | List vault recipients | ✅ |
| `envault key export` | Export your public key to share with teammates | ✅ |
| `envault key import` | Add a teammate's public key as a recipient | ✅ |
| `envault key delete` | Remove a recipient from the vault | ✅ |
| `envault import <file.env>` | Bulk-import from an existing `.env` file | ✅ |
| `envault data store <file>` | Store an arbitrary file (JSON, PEM, binary…) | ✅ |
| `envault add <KEY>` | Add or update a single secret | ✅ |
| `envault set <KEY>` | Re-seal an existing secret with a new value | ✅ |
| `envault rm <KEY>` | Remove a secret from the vault | ✅ |
| `envault list` | List all secrets (names only — no plaintext) | ✅ |
| `envault cat <KEY>` | Decrypt and print a single secret | ✅ |
| `envault export` | Decrypt all env secrets as `export KEY=value` | ✅ |
| `envault rotate <KEY>` | Re-seal a secret with a fresh key for current recipients (true revocation) | ✅ |
| `envault push` | Stage, commit, and push the encrypted vault | ✅ |
| `envault pull` | Fetch and merge the vault; report changes | ✅ |
| `envault run [--only/--except] -- <cmd>` | Inject secrets in memory and run a command (0 bytes to disk) | ✅ |
| `envault exec` | Open `$SHELL` with all env secrets injected | ✅ |
| `envault scan [--staged/--all]` | Scan for secrets (pattern rules + entropy heuristic) | ✅ |
| `envault hook install --git` | Install a pre-commit hook that blocks secret leaks (`--uninstall` to remove) | ✅ |
| `envault protect add <path>` | Mark a path/glob off-limits to AI agents (blocked by the Envault plugin) | ✅ |
| `envault audit log show/verify` | Show or verify the tamper-evident AI access log | ✅ |
| `envault mcp serve [--project <path>] [--dry-run]` | Start the Envault MCP server for Claude Code (JSON-RPC 2.0 over stdio); `--dry-run` prints the tool schemas | ✅ |
| `envault status` | Structured health check of the vault, hooks, and shield | ✅ |
| `envault agent-check` | Verify the AI-agent environment is ready (exit 1 if not) | ✅ |
| `envault doctor` | Diagnose install state, hooks, keychain, and Git remote (no secrets exposed) | ✅ |
| `envault uninstall [--keys]` | Remove the vault and Git hook (`--keys` also clears keychain); `install.sh --uninstall` removes the binary. Claude Code: `/plugin uninstall envault` | ✅ |

> Add `--agent-safe` (alias `--json`) to any command for structured JSON output;
> in this mode `cat`/`export` refuse to print plaintext unless you pass `--force`.

---

## Security model

Envault is designed so that you do not have to trust anyone except your Git remote:

- **End-to-end encryption** — secrets are encrypted on your machine before they are committed. Each secret uses a random data key (AES-256-GCM) wrapped to every recipient's X25519 public key. Only recipients with a matching private key can decrypt.
- **Private keys never leave your machine** — they are sealed in the OS keychain (macOS Keychain via `security`, Linux kernel keyring via `keyctl`) and are never sent anywhere.
- **Private keys are encrypted at rest** — the keychain blob is itself encrypted under a passphrase-derived key (Argon2id → AES-256-GCM), so even a process that reads your keychain gets useless ciphertext without your passphrase.
- **Zero-trust remote** — the Git remote only ever stores ciphertext. Even if the remote is compromised, no secrets are exposed.
- **No disk writes** — secrets are decrypted in memory and injected directly into the child process. Nothing is written to a temp file.
- **Per-recipient access control** — adding or removing a teammate from the vault controls who can decrypt. `rotate` re-seals a secret with a fresh data key for the current recipients, truly revoking a removed member.
- **Leak prevention** — an optional Git pre-commit hook (`envault hook install --git`) scans the staged diff for `.env` files, private keys, and known API tokens, blocking the commit before a secret ships.
- **AI Privacy Shield** — the [Envault Claude Code plugin](#claude-code-plugin) blocks AI agents from reading protected paths or running `envault cat`/`export`, masks any vault secret that appears in tool output, and records every access in a tamper-evident audit log.
- **Integrity guaranteed** — ciphertext is authenticated; any tampering is detected and rejected before decryption.
- **Key-unlock agent is opt-in and clearly bounded** — `envault agent unlock` trades the "decrypt on demand, clear immediately" norm for a decrypted key cached in memory for a bounded TTL (default 8h), so headless callers (Claude Code) can skip the passphrase. Nothing changes unless you explicitly run it; `envault status`/`doctor` always show whether it's active.

---

## Status

Active development — the full core workflow is implemented end-to-end: init a vault,
manage keys, add/import secrets, push/pull over Git, `envault run -- <cmd>` to inject
secrets in memory, AI-agent integration (Claude Code Privacy Shield), and one-line
install from signed cross-platform releases.

| Milestone | Status |
|---|---|
| v0.1–0.2 — Scaffold, CI, crypto core | ✅ shipped |
| v0.3 — Vault init, key management | ✅ shipped |
| v0.4 — Secret import, add/set/rm, list, cat/export | ✅ shipped |
| v0.5 — Git push / pull, re-wrap & rotation, conflict merge | ✅ shipped |
| v0.6 — Runtime injection (`envault run`, `exec`) | ✅ shipped |
| v0.7 — Git pre-commit hook & secret detection | ✅ shipped |
| v0.8 — Claude Code & AI agent integration (Privacy Shield) | ✅ shipped |
| v0.9.0 — Installer & cross-platform signed releases | ✅ shipped |
| v0.9.1 — Clean uninstall & doctor | ✅ shipped |
| v0.9.2 — Claude Code plugin & marketplace distribution | ✅ shipped |
| v0.9.3 — Embedded MCP server (Claude Code native protocol) | ✅ shipped |
| v0.9.4 — Key-unlock agent (ssh-agent-style, passphrase-free Claude Code UX) | ✅ shipped |
| v0.10.0 — Integration testing (Gitea) | 🔜 next |
| v1.0.0 — Stable release | planned |

---

## Requirements

- Go 1.25+
- macOS or Linux
- Git (any version with remote support)

---

## Build

```sh
make build          # static binary → ./envault  (CGO_ENABLED=0)
go test ./...       # run the test suite
```

The binary embeds its version from the latest git tag:

```sh
./envault --version
```

---

## Contributing

The project is in active development. The roadmap is managed internally.
Feel free to open an issue to discuss ideas, report bugs, or ask questions.

---

## License

MIT

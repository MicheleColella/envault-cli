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

## Quick start

```sh
# 1. Build
make build

# 2. Initialise a vault in your repo
envault init

# 3. Generate your identity key (private key stays in your OS keychain)
envault key new --id you@example.com

# 4. Add a secret
echo "sk-abc123" | envault add OPENAI_KEY

# 5. Push the encrypted vault to your remote
envault push

# 6. A teammate pulls and runs their app with secrets injected (coming in v0.6.0)
envault run -- npm start
```

---

## Commands

| Command | Description | Status |
|---|---|---|
| `envault init` | Initialise a vault in the current repo | ✅ |
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
| `envault push` | Stage, commit, and push the encrypted vault | ✅ |
| `envault pull` | Fetch and merge the vault; report changes | ✅ |
| `envault run -- <cmd>` | Inject secrets in memory and run a command | 🚧 v0.6.0 |
| `envault hook` | Manage Git and Claude Code hooks | 🚧 v0.7.0 |

---

## Security model

Envault is designed so that you do not have to trust anyone except your Git remote:

- **End-to-end encryption** — secrets are encrypted on your machine before they are committed. Only recipients with a matching private key can decrypt.
- **Private keys never leave your machine** — they are sealed in the OS keychain (macOS Keychain or Linux Secret Service) and are never sent anywhere.
- **Zero-trust remote** — the Git remote only ever stores ciphertext. Even if the remote is compromised, no secrets are exposed.
- **No disk writes** — secrets are decrypted in memory and injected directly into the child process. Nothing is written to a temp file.
- **Per-recipient access control** — adding or removing a teammate from the vault controls who can decrypt. Each secret is independently encrypted for the current recipient set.
- **Integrity guaranteed** — ciphertext is authenticated; any tampering is detected and rejected before decryption.

---

## Status

Active development — core vault, key management, and Git sync are fully implemented.
Runtime injection and hook integration are coming next.

| Milestone | Status |
|---|---|
| v0.1–0.2 — Scaffold, CI, crypto core | ✅ shipped |
| v0.3 — Vault init, key management | ✅ shipped |
| v0.4 — Secret import, add/set/rm, list, cat/export | ✅ shipped |
| v0.5.0 — Git push / pull | ✅ shipped |
| v0.5.1 — Recipient re-wrap & rotation | 🔜 next |
| v0.6.0 — Runtime injection (`envault run`) | planned |
| v0.7.0 — Git pre-commit hook (leak prevention) | planned |
| v0.8.0 — Claude Code & AI agent integration | planned |
| v1.0.0 — Stable release | planned |

---

## Requirements

- Go 1.21+
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

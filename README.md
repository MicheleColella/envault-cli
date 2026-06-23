# Envault

> Git-backed, zero-trust secrets CLI for teams.

Envault encrypts your API keys and tokens directly inside your existing Git repo. Every secret stays ciphertext in the remote; private keys never leave the machine. Secrets are injected into a child process in memory at runtime — never written to disk.

---

## How it works

Envault uses **envelope encryption** (modelled on the [age](https://age-encryption.org) file format):

1. A fresh random **data encryption key (DEK)** is generated for each secret.
2. The secret payload is sealed with the DEK using **AES-256-GCM** (or ChaCha20-Poly1305).
3. The DEK is wrapped for each team member via **ephemeral X25519 ECDH** + **HKDF-SHA256**, producing a per-recipient encrypted block.
4. The resulting envelope (JSON) is committed to `.envault/` inside the repo.

Private keys never touch the network. The Git remote only ever stores ciphertext.

```
┌─────────────────────────────────────────────────┐
│                    Envelope                      │
│  version · suite · nonce · ciphertext(payload)  │
│  ┌──────────────────────────────────┐            │
│  │ Recipient 0                      │            │
│  │  ephemeral_public · nonce        │            │
│  │  wrapped_key = AES-GCM(DEK)      │            │
│  └──────────────────────────────────┘            │
│  ┌──────────────────────────────────┐            │
│  │ Recipient N  …                   │            │
│  └──────────────────────────────────┘            │
└─────────────────────────────────────────────────┘
```

## Status

Early development — core crypto is implemented and tested; vault, Git sync, and runtime injection are still stub commands.

| Milestone | Status |
|-----------|--------|
| v0.1.0 — Project scaffold & CLI skeleton | ✅ shipped |
| v0.1.1 — CI & build tooling | ✅ shipped |
| v0.2.0 — Crypto core (AES-256-GCM + X25519 envelope) | ✅ shipped |
| v0.2.1 — Crypto test vectors & tamper tests | planned |
| v0.3.0 — Vault init | planned |
| v0.4.0 — Secret import & local encryption | planned |
| v0.5.0 — Git sync (push / pull) | planned |
| v0.6.0 — Runtime injection | planned |
| v1.0.0 — Stable release | planned |

## Requirements

- Go 1.21+
- macOS or Linux
- Git

## Build

```sh
make build          # produces ./envault (static, CGO_ENABLED=0)
make test           # go test ./...
make lint           # golangci-lint run ./...
```

The binary embeds its version from `git describe`:

```sh
./envault --version
```

## Usage

```
envault <command> [flags]

Commands:
  init      Initialise a new vault in the current repo
  key       Manage your X25519 keypair
  import    Import existing environment variables into the vault
  add       Add or update a single secret
  list      List secrets visible to the current key
  push      Push encrypted vault to the remote
  pull      Pull and merge vault from the remote
  run       Inject secrets and run a command
  hook      Manage Git and Claude Code hooks
```

> Most commands are stubs — they will be implemented progressively through the roadmap above.

## Security design

| Property | Mechanism |
|---|---|
| Confidentiality | AES-256-GCM or ChaCha20-Poly1305 per secret |
| Key wrapping | Ephemeral X25519 ECDH + HKDF-SHA256 per recipient |
| Integrity | AEAD authentication tag; envelope metadata bound as additional data |
| Forward secrecy of wraps | Fresh ephemeral keypair per recipient per seal |
| Memory safety | Sensitive key material (`defer clear()`) zeroed after use |
| No secret on disk | Runtime injection via child process environment; never persisted |
| Zero-trust remote | Git remote holds only ciphertext; private keys are local-only |

## Project layout

```
cmd/envault/        Cobra CLI entry point and command files
internal/
  crypto/           Envelope encryption: Seal / Unseal, AEAD, X25519 keys
  vault/            Vault layout and secret records (planned)
  git/              Push / pull via go-git (planned)
  keychain/         OS keychain integration — macOS Keychain / Linux Secret Service (planned)
  hook/             Git pre-commit and Claude Code hook management (planned)
  ui/               Shared terminal output (colored glyphs, NO_COLOR-aware)
```

## Contributing

This project is in active early development. The roadmap is managed internally; feel free to open an issue to discuss ideas or report bugs.

## License

MIT

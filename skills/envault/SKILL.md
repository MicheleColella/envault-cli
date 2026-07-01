---
name: envault
description: Use when a repo has a .envault/ vault — Envault is a Git-backed, zero-trust secrets manager. Covers reading vault state, injecting secrets into commands, syncing with the team, and the AI Privacy Shield rules that keep plaintext out of model context. Triggers on "envault", ".envault", "secrets", "API key", ".env file", "inject secrets", "vault recipients".
---

# Envault

This repo's secrets are managed by **Envault**. They are encrypted at rest in
`.envault/secrets.enc` — never plaintext on disk. The Git remote only stores
ciphertext; private keys never leave the machine.

## Security rules (the AI Privacy Shield)

- **NEVER** run `envault cat` or `envault export` without `--force` — the PreToolUse
  hook blocks them to keep plaintext out of model context. Don't suggest `--force`
  to work around it.
- Prefer `envault run -- <cmd>` to inject secrets into a child process **in memory**,
  never writing them anywhere.
- Do not print, cat, or log the contents of `.envault/secrets.enc`.
- Paths registered via `envault protect add` are blocked from Read/Write/Edit/Bash.

## Common commands

| Goal | Command |
|---|---|
| List sealed entries (names only) | `envault list` |
| Seal a new secret | `envault add <KEY>` |
| Re-seal an existing secret | `envault set <KEY>` |
| Run a command with secrets injected in memory | `envault run -- <cmd>` |
| Open a shell with secrets injected | `envault exec` |
| Sync vault with the team via Git | `envault push` / `envault pull` |
| Rotate a secret (true revocation) | `envault rotate <KEY>` |
| Show recipients | `envault key list` |
| Vault health | `envault status` |
| Manage protected paths | `envault protect add\|list\|remove <path>` |
| Audit log | `envault audit log show` / `verify` |
| Diagnose install | `envault doctor` |

## MCP tools (preferred over bash when available)

If the `envault` MCP server is connected (tools named `envault_status`,
`envault_add`, `envault_list`, `envault_rotate`, `envault_run`,
`envault_protect`, `envault_push`, `envault_pull`), prefer calling those tools
directly instead of the equivalent bash command — parameters go through
JSON-Schema validation instead of a shell string, and tool responses only ever
contain metadata (name, algorithm, recipient count, timestamps), never a
secret value. `cat`/`export`/`data`/`import`/`key *` have no MCP equivalent —
those still go through bash, where the Privacy Shield hooks apply.

## Notes

- `envault status` / `list` / `audit` / `doctor` are safe, read-only, and never
  expose secret values — use them freely to understand vault state.
- If a command needs a passphrase non-interactively, it reads `ENVAULT_PASSPHRASE`.
- The binary must be on PATH (installed via the Envault installer, Homebrew, or
  `go install`). Run `envault doctor` if commands are not found.

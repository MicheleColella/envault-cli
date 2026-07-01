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
- **NEVER seal a brand-new secret yourself** — not via `envault add`/`envault set`
  in Bash (e.g. `echo value | envault add KEY`), and there is no MCP tool for it
  either. Doing this requires the plaintext to pass through your own context first,
  which is exactly what the Privacy Shield exists to prevent — the PreToolUse hook
  blocks it in Bash for the same reason. If the user wants to add or update a
  secret, tell them to run `envault add <KEY>` / `envault set <KEY>` themselves in
  their own terminal — do not offer to do it "for convenience" and do not suggest
  `--force` to work around the block.
- Prefer `envault run -- <cmd>` to inject secrets into a child process **in memory**,
  never writing them anywhere.
- Do not print, cat, or log the contents of `.envault/secrets.enc`.
- Paths registered via `envault protect add` are blocked from Read/Write/Edit/Bash.

## Common commands

| Goal | Command | Who runs it |
|---|---|---|
| List sealed entries (names only) | `envault list` | either |
| Seal a new secret | `envault add <KEY>` | **user only**, in their own terminal |
| Re-seal an existing secret with a new value | `envault set <KEY>` | **user only**, in their own terminal |
| Run a command with secrets injected in memory | `envault run -- <cmd>` | either |
| Open a shell with secrets injected | `envault exec` | user |
| Sync vault with the team via Git | `envault push` / `envault pull` | either |
| Rotate a secret (true revocation, no new value needed) | `envault rotate <KEY>` | either |
| Show recipients | `envault key list` | either |
| Vault health | `envault status` | either |
| Manage protected paths | `envault protect add\|list\|remove <path>` | either |
| Audit log | `envault audit log show` / `verify` | either |
| Diagnose install | `envault doctor` | either |
| Unlock the key for passphrase-free use | `envault agent unlock` | **user only**, in their own terminal |
| Check what's unlocked / lock again | `envault agent status` / `lock` | either |

## MCP tools (preferred over bash when available)

If the `envault` MCP server is connected (tools named `envault_status`,
`envault_list`, `envault_rotate`, `envault_run`, `envault_protect`,
`envault_push`, `envault_pull`), prefer calling those tools directly instead of
the equivalent bash command — parameters go through JSON-Schema validation
instead of a shell string, and tool responses only ever contain metadata
(name, algorithm, recipient count, timestamps), never a secret value.
`cat`/`export`/`data`/`import`/`key *`/**`add`**/**`set`** have no MCP
equivalent, by design — those still go through bash (where the Privacy Shield
hooks apply), or must be run by the user themselves for `add`/`set`.

## Passphrase-free operation (the key-unlock agent)

If the user has run `envault agent unlock` from their own terminal, commands
needing the private key (`envault_rotate`, `envault_run`,
`envault_protect` encrypt, `push`/`pull`, and `postuse` masking) work with no
passphrase prompt and no `ENVAULT_PASSPHRASE` — check `envault status` /
`envault doctor` for "Agent unlocked keys" / "Key-unlock agent" to see if
this is active. If a private-key operation fails with a passphrase error and
the user seems to expect it to "just work", suggest `envault agent unlock`
(run by them, in a real terminal — it prompts interactively) rather than
suggesting they set `ENVAULT_PASSPHRASE` in the environment, which is the
less secure, more manual fallback.

## Notes

- `envault status` / `list` / `audit` / `doctor` are safe, read-only, and never
  expose secret values — use them freely to understand vault state.
- If a command needs a passphrase non-interactively and no agent is unlocked,
  it reads `ENVAULT_PASSPHRASE`.
- The binary must be on PATH (installed via the Envault installer, Homebrew, or
  `go install`). Run `envault doctor` if commands are not found.

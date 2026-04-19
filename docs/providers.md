# Providers

TokenTop does not ask you to create or paste API keys for Codex or Claude. If you are already signed in to the [Codex CLI](https://github.com/openai/codex) or [Claude Code CLI](https://claude.com/claude-code), TokenTop picks those credentials up automatically — there is nothing else to configure. OpenRouter is the only provider that needs a key in your shell, because it does not ship a CLI that stores one.

| Provider   | Credential source                                              | Shows |
|------------|----------------------------------------------------------------|-------|
| Codex      | `~/.codex/auth.json`                                           | ChatGPT/Codex rate limits |
| OpenRouter | `OPENROUTER_API_KEY` env var                                   | Credit + activity |
| Claude     | macOS keychain (`Claude Code-credentials`) or `~/.claude/.credentials.json` | Session + weekly limits |

A provider is skipped (with a warning) if its credential is missing. TokenTop exits with an error only when *no* provider has valid credentials.

## Codex

**Setup:** install and sign in to the [Codex CLI](https://github.com/openai/codex) — TokenTop uses the same credentials with no extra steps.

```sh
codex login
```

This writes `~/.codex/auth.json` containing an OAuth access token and refresh token. TokenTop reads that file and queries ChatGPT's rate-limit API for your plan's primary and (optionally) code-review windows.

**Shown:** plan type, primary window usage, code-review window usage (when `codex_ui.code_review: true`), credits balance.

## OpenRouter

**Credential:** `OPENROUTER_API_KEY` environment variable.

```sh
export OPENROUTER_API_KEY="sk-or-..."
```

Put it in your shell rc (`~/.zshrc`, `~/.bashrc`) to persist.

**Key types:**

- **Standard key** — only the key's own usage is visible.
- **Management key** — unlocks the full view: daily spend history, top models by spend, and (with `openrouter_ui.api_keys: true`) the list of all keys on the account. Create a management key at <https://openrouter.ai/settings/keys>.
- **Free-tier key** — detected and labeled; limited data available.

The key type is detected automatically on first fetch and logged at info level.

## Claude (Anthropic)

**Setup:** install and sign in to [Claude Code](https://claude.com/claude-code) — TokenTop reuses its OAuth credentials with no extra configuration.

Lookup order:

1. **macOS only** — the macOS Keychain entry `Claude Code-credentials` (read via `security find-generic-password`). This is where recent versions of Claude Code store credentials on macOS.
2. `~/.claude/.credentials.json` — the file-based fallback used on Linux and older macOS installs.

TokenTop needs read access to the keychain entry; on first access macOS will prompt you to allow it. Click **Always Allow** to avoid repeated prompts.

**Shown:** subscription type, rate-limit tier, current session window, weekly window.

**How it works:** Anthropic does not expose a usage endpoint, so each refresh sends a minimal probe request to `POST /v1/messages` (`max_tokens: 1`, body `"hi"`) and reads the rate-limit response headers. The probe consumes a negligible amount of quota but is not zero — factor that in if you shorten the refresh interval.

## Adding a new provider

Each provider lives in `pkg/<name>/` with two files:

- `auth.go` — exposes `LoadAuth() (*Auth, error)` that resolves credentials from env/file/keychain.
- `usage.go` — exposes `FetchUsage(*Auth) (*Usage, error)` that returns a struct the TUI can render.

Then wire it up in `cmd/tokentop/main.go` (auth load + CLI flag) and add a view under `internal/tui/`. See `pkg/codex/` for the simplest complete example.

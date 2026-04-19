# Troubleshooting

## `Error: no providers available`

Every enabled provider failed to load credentials. See the specific `Warning:` line printed above the error for the reason — usually a missing file or env var. Cross-check with [providers.md](providers.md).

## Codex: `reading ~/.codex/auth.json: no such file or directory`

You have not signed in with the Codex CLI. Run:

```sh
codex login
```

If the file exists but fails to parse, it may have been rotated to a new format; updating the Codex CLI and signing in again usually fixes it.

## OpenRouter: `OPENROUTER_API_KEY is not set`

Export it in your shell:

```sh
export OPENROUTER_API_KEY="sk-or-..."
```

Add the line to your shell rc file to persist across sessions. A management key unlocks daily spend, top models, and the API keys panel — a standard key shows only your own key's usage.

## Claude: `no Claude credentials found: keychain: ...; file: ...`

On macOS, TokenTop first tries the Keychain entry `Claude Code-credentials`, then falls back to `~/.claude/.credentials.json`. If both fail:

- Confirm Claude Code is installed and you've signed in at least once.
- On macOS, run once to prime the keychain prompt and click **Always Allow**:

  ```sh
  security find-generic-password -s "Claude Code-credentials" -w
  ```

- On Linux, confirm `~/.claude/.credentials.json` exists and contains a `claudeAiOauth.accessToken` field.

## macOS Keychain prompts keep appearing

When you deny or dismiss the prompt, macOS re-asks every run. Choose **Always Allow** once and the prompt stops. If you already chose **Deny** by mistake, the simplest fix is:

1. Open **Keychain Access**, find **Claude Code-credentials** under **login → Passwords**, and delete it.
2. Sign in to Claude Code again (`claude` and follow the OAuth flow) to recreate the entry.
3. Launch `tokentop` and choose **Always Allow** on the fresh prompt.

Editing the entry's Access Control list directly is unreliable for unsigned binaries — the delete-and-resign flow is more predictable.

## The bars or percentages look wrong

Resize the terminal and press `r`. Very narrow terminals are clamped to a minimum bar width but may still look cramped — enable compact mode:

```yaml
codex_ui:
  compact: true
claude_ui:
  compact: true
```

## Logs

By default, logs are written to:

```
$XDG_STATE_HOME/tokentop/tokentop.log
# or, if XDG_STATE_HOME is unset:
~/.local/state/tokentop/tokentop.log
```

Turn up verbosity for a single run:

```yaml
log:
  level: debug
```

Or disable file logging entirely:

```yaml
log:
  level: off
```

## Nothing updates

Check the footer — if the countdown is running but the timestamp never updates, a fetch is failing silently beyond its retry budget. Set `log.level: debug` and inspect the log file for `*_refresh failed` entries.

## Still stuck

Open an issue with your OS, TokenTop version (`tokentop` prints it in the header), and the relevant log excerpt at <https://github.com/lwlee2608/tokentop/issues>.

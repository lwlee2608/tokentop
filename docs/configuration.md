# Configuration

TokenTop reads its config from `~/.config/tokentop/config.yaml`. The file is created with defaults on first run. If you add new options in a release, missing keys are backfilled automatically on startup — your existing values are preserved.

## File location

```
~/.config/tokentop/config.yaml
```

Any value may be overridden by an environment variable using dotted keys with `_` separators (e.g. `LOG_LEVEL=debug`, `PROVIDERS_CODEX_ENABLED=false`).

## Default config

```yaml
log:
  level: info
  path: ""

providers:
  codex:
    enabled: true
  openrouter:
    enabled: true
  anthropic:
    enabled: true

codex_ui:
  compact: false
  code_review: false
  pace_tick: true

claude_ui:
  compact: false
  pace_tick: true

openrouter_ui:
  summary: true
  daily_spend: true
  top_models: true
  api_keys: false
  metric: spend  # spend | requests | tokens
```

## Fields

### `log`

| Key     | Type   | Default | Description |
|---------|--------|---------|-------------|
| `level` | string | `info`  | One of `debug`, `info`, `warn`, `error`, or `off` to disable logging entirely. |
| `path`  | string | `""`    | Log file path. When empty, logs go to `$XDG_STATE_HOME/tokentop/tokentop.log` (or `~/.local/state/tokentop/tokentop.log`). |

### `providers`

Each provider has a single `enabled` boolean. Disabling a provider skips its auth loading and API calls entirely.

| Key                    | Default | Description |
|------------------------|---------|-------------|
| `providers.codex`      | `true`  | OpenAI Codex (ChatGPT subscription). |
| `providers.openrouter` | `true`  | OpenRouter. |
| `providers.anthropic`  | `true`  | Anthropic (Claude Code subscription). |

A provider with `enabled: true` but missing credentials is logged as a warning and skipped — TokenTop still starts as long as at least one provider has working credentials.

CLI flags can override the providers block for a single run — see [Keybindings & CLI flags](keybindings.md#cli-flags).

### `codex_ui`

| Key           | Default | Description |
|---------------|---------|-------------|
| `compact`     | `false` | Render Codex section in a single-line-per-bar compact form. |
| `code_review` | `false` | Show the separate code-review rate limit window. |
| `pace_tick`   | `true`  | Split usage bars at the time-elapsed mark to show over/under-pace burn. |

### `claude_ui`

| Key         | Default | Description |
|-------------|---------|-------------|
| `compact`   | `false` | Render Claude section in compact form. |
| `pace_tick` | `true`  | Split usage bars at the time-elapsed mark to show over/under-pace burn. |

### `openrouter_ui`

These settings only apply when the configured `OPENROUTER_API_KEY` is a **management key**. With a standard key, TokenTop renders the credit-limit bar and a fixed daily/weekly/monthly usage line regardless of these settings.

| Key           | Default | Description |
|---------------|---------|-------------|
| `summary`     | `true`  | Show credits remaining, total spend, and request/token totals. |
| `daily_spend` | `true`  | Show the last 30 days of spend as a chart. |
| `top_models`  | `true`  | Show the top models by spend. |
| `api_keys`    | `false` | List every key on the account with per-key spend. |
| `metric`      | `spend` | Initial metric for daily spend and top models charts. One of `spend`, `requests`, `tokens`. Cycle at runtime with `m`. |

## Examples

### Quiet logging, only OpenRouter, only summary

```yaml
log:
  level: warn

providers:
  codex:
    enabled: false
  openrouter:
    enabled: true
  anthropic:
    enabled: false

openrouter_ui:
  summary: true
  daily_spend: false
  top_models: false
  api_keys: false
```

### Compact everything

```yaml
codex_ui:
  compact: true
claude_ui:
  compact: true
```

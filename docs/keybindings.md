# Keybindings & CLI flags

TokenTop is read-only and has a tiny keymap.

| Key      | Action                                                 |
|----------|--------------------------------------------------------|
| `r`      | Refresh immediately                                    |
| `m`      | Cycle OpenRouter chart metric (spend → requests → tokens) |
| `q`      | Quit                                                   |
| `Ctrl+C` | Quit                                                   |

The default metric is configurable via `openrouter_ui.metric` in `~/.config/tokentop/config.yaml` (`spend`, `requests`, or `tokens`).

## CLI flags

Flags override the `providers` section of the config for the current run only.

| Flag           | Effect |
|----------------|--------|
| `--codex`      | Run with only Codex |
| `--claude`     | Run with only Claude |
| `--openrouter` | Run with only OpenRouter |
| `--all`        | Force all three providers on |
| `--version`    | Print version and exit |

Example:

```sh
tokentop --openrouter
```

## Refresh behavior

- **Automatic:** every 5 minutes.
- **Manual:** `r` triggers all enabled providers to fetch in parallel and resets the 5-minute timer.
- **Retries:** a failed fetch is retried up to 3 times with a 5-second delay before being surfaced as an error in the panel.

The footer always shows time-until-next-refresh and the timestamp of the last successful fetch.

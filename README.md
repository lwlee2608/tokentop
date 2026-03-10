# tokenburn

[![CI](https://github.com/lwlee2608/tokenburn/actions/workflows/ci.yml/badge.svg)](https://github.com/lwlee2608/tokenburn/actions/workflows/ci.yml)

A terminal dashboard for monitoring your [OpenAI Codex](https://chatgpt.com/codex) usage limits in real time.

![tokenburn screenshot](screenshot.jpg)

## Install

```sh
go install github.com/lwlee2608/tokenburn/cmd/tokenburn@latest
```

Or build from source:

```sh
git clone https://github.com/lwlee2608/tokenburn.git
cd tokenburn
make build
./bin/tokenburn
```

## Prerequisites

You need an active Codex session with auth credentials at `~/.codex/auth.json` (created automatically when you use [Codex CLI](https://github.com/openai/codex)).

## License

MIT

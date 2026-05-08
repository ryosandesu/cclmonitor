# cclmonitor

> Policy-based hook for Claude Code — monitor, notify, and block tool calls in real time.

[![Go](https://img.shields.io/badge/go-1.21+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS-lightgrey)](https://www.apple.com/macos/)

`cclmonitor` is a `PreToolUse` hook binary for [Claude Code](https://claude.ai/code). It intercepts every tool call — `Bash`, `Edit`, `Write`, `Read` — and evaluates it against your YAML policy. Dangerous commands get blocked before they run; everything else is logged with macOS notifications.

```
Claude Code  →  PreToolUse hook  →  cclmonitor
                                        │
                              ┌─────────┴──────────┐
                           deny?                allow/unknown?
                              │                      │
                    notify + exit 2            notify + exit 0
                    (tool blocked)             (tool proceeds)
```

---

## Features

- **Block by policy** — regex or glob rules per tool type; `deny` wins over `allow`
- **macOS notifications** — instant alert via `osascript` when a rule fires
- **JSONL audit log** — date-rotated log files under `~/.claude/`
- **Dedup suppression** — SQLite-backed deduplication to silence repeated events
- **Project overrides** — per-repo `.claude/cclmonitor.yaml` merges with global config
- **Dry-run mode** — `cclmonitor test` evaluates a value without blocking anything
- **Live log viewer** — `cclmonitor-tail` streams color-coded events to your terminal
- **Zero CGO** — pure Go SQLite (`modernc.org/sqlite`), no extra toolchain needed

---

## Installation

### Build from source

**Prerequisites:** Go 1.21+, macOS, Claude Code

```sh
git clone https://github.com/ryosandesu/cclmonitor.git
cd cclmonitor
make install
```

`make install` builds the binaries, copies them to `~/bin/`, and **auto-registers the hook** in `~/.claude/settings.json` (a backup is saved as `settings.json.bak`).

To use `cclmonitor-tail` and `cclmonitor test` from the terminal, add `~/bin/` to your PATH:

```sh
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

### Verify

```sh
cclmonitor --version
```

---

## Quick Start

### 1. Create your policy file

```sh
cp examples/cclmonitor.yaml ~/.claude/cclmonitor.yaml
```

### 2. Edit the rules

```yaml
# ~/.claude/cclmonitor.yaml
mode: prod

notify:
  channels: [osascript, logfile]
  dedup_window_sec: 60

rules:
  Bash:
    allow:
      - regex: '^(ls|pwd|cat|grep|git\s+(status|log|diff))\b'
    deny:
      - regex: '\brm\s+-rf\s+/'
      - regex: '\bcurl\b.*\|\s*(ba)?sh\b'

  Edit:
    allow:
      - glob: '<cwd>/**/*.{ts,go,py,md}'
    deny:
      - glob: '**/.env*'
```

### 3. Test your rules

```sh
cclmonitor test "rm -rf /"
# tool:    Bash
# value:   rm -rf /
# verdict: deny

cclmonitor test --tool Edit "/etc/passwd"
# tool:    Edit
# value:   /etc/passwd
# verdict: unknown
```

---

## Configuration Reference

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `mode` | `prod` \| `dev` | `prod` | `dev` notifies on `allow` hits too |
| `notify.channels` | list | `[]` | `osascript`, `logfile`, or both |
| `notify.logdir` | path | `~/.claude/` | Directory for JSONL log files |
| `notify.dbdir` | path | `~/.claude/` | Directory for the dedup SQLite DB |
| `notify.dedup_window_sec` | int | `0` | Suppress duplicate events for N seconds (`0` = disabled) |
| `notify.retain_days` | int | `30` | Delete log files older than N days |

### Rule matching

Each rule has exactly one of:

| Field | Matches against | Example |
|-------|----------------|---------|
| `regex` | Full value string | `'^ls\b'` |
| `glob` | File path | `'<cwd>/**/*.go'` |

**Evaluation order:** `deny` rules are checked first. The first match wins.

**`<cwd>` token:** Expands to the working directory reported by Claude Code. Use it in glob rules to scope a policy to the current project.

| Tool | Value evaluated |
|------|----------------|
| `Bash` | Command string |
| `Edit` | Target file path |
| `Write` | Target file path |
| `Read` | Target file path |

### Project-level overrides

Place a `.claude/cclmonitor.yaml` in any project root. It merges with your global config — project rules take precedence per tool section.

```
~/projects/my-app/
  .claude/
    cclmonitor.yaml    ← project overrides
~/.claude/
  cclmonitor.yaml      ← global policy
```

---

## Commands

### `cclmonitor test`

Dry-run a value against your current policy. No notifications, no blocking.

```sh
cclmonitor test [--tool TOOL] [--cwd DIR] <value>

# Examples
cclmonitor test "git push --force"
cclmonitor test --tool Edit "~/.ssh/id_rsa"
cclmonitor test --tool Bash --cwd ~/projects/myapp "npm install"
```

### `cclmonitor-tail`

Stream today's audit log to your terminal with color-coded verdicts.

```sh
cclmonitor-tail
```

```
14:32:01 [allow  ] Bash: ls -la
14:32:05 [deny   ] Bash: rm -rf /tmp
14:32:10 [unknown] Write: /tmp/output.txt
```

<span style="color:green">■</span> green = allow &nbsp; <span style="color:red">■</span> red = deny &nbsp; <span style="color:#b5a000">■</span> yellow = unknown

---

## Audit Log

When `logfile` is enabled, events are appended to date-rotated files:

```
~/.claude/
  cclmonitor.2024-01-15.log
  cclmonitor.2024-01-16.log
```

Each line is a JSON object:

```json
{"time":"2024-01-15T14:32:05Z","session_id":"abc123","tool_name":"Bash","value":"rm -rf /tmp","verdict":"deny"}
```

---

## How the hook works

`make install` writes the following entry into `~/.claude/settings.json`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [{ "type": "command", "command": "/Users/<you>/bin/cclmonitor" }]
      }
    ]
  }
}
```

The full path is written automatically so Claude Code can find the binary regardless of your shell's `PATH`.

Claude Code invokes `cclmonitor` synchronously before every tool call, passing a JSON payload on stdin. Exit code `2` blocks the tool; exit code `0` allows it.

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | allow or unknown — tool proceeds |
| `2` | deny — Claude Code blocks the tool call |

---

## Uninstall

```sh
make uninstall
```

Removes binaries from `~/bin/` and restores `~/.claude/settings.json` from backup.

---

## Development

```sh
# Run all tests
make test

# TDD cycle
go test -v ./internal/match/
go test -v ./cmd/cclmonitor/
```

This project follows a **Red → Green → Refactor** TDD cycle. Tests live alongside each package (`*_test.go`).

---

## License

[MIT](LICENSE)

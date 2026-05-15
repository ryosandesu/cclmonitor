# cclmonitor

> Policy-based hook for Claude Code — audit and block tool calls in real time.

[![Go](https://img.shields.io/badge/go-1.21+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

`cclmonitor` is a hook binary for [Claude Code](https://claude.ai/code). It intercepts every tool call — `Bash`, `Edit`, `Write`, `Read` — and evaluates it against your YAML policy. Dangerous commands get blocked before they run; everything else is recorded in a JSONL audit log.

```
Claude Code  →  PreToolUse hook (cclmonitor)
                    deny?          → log "denied" + exit 2  (tool blocked)
                    allow/unknown? → log "pending" + exit 0

             →  tool executes (or user cancels → PostToolUse never fires)

             →  PostToolUse hook (cclmonitor post)
                    interrupted? → log "interrupted"
                    allow?       → log "executed"
                    unknown?     → log "unknown"
```

---

## Features

- **Block by policy** — regex or glob rules per tool type; `deny` wins over `allow`
- **Five-verdict audit log** — `pending` / `executed` / `denied` / `unknown` / `interrupted`, date-rotated JSONL files
- **Accurate execution record** — PostToolUse hook confirms the tool actually ran
- **Project overrides** — per-repo `.claude/cclmonitor.yaml` merges with global config
- **Dry-run mode** — `cclmonitor test` evaluates a value without blocking anything
- **TUI dashboard** — `cclmonitor-ui` shows Compliance & Coverage scores, per-tool breakdown, 30-day heatmap, and live event feed

---

## Installation

**Prerequisites:** Go 1.26+, Claude Code

### macOS / Linux

```sh
git clone https://github.com/ryosandesu/cclmonitor.git
cd cclmonitor
make install
```

`make install` builds the binaries, copies them to `~/bin/`, and **auto-registers both hooks** (PreToolUse and PostToolUse) in `~/.claude/settings.json` (a backup is saved as `settings.json.bak`).

To use `cclmonitor test` from the terminal, add `~/bin/` to your PATH:

```sh
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

### Windows

`make` is not available by default on Windows, so build manually with PowerShell:

```powershell
git clone https://github.com/ryosandesu/cclmonitor.git
cd cclmonitor

# Build all binaries
go build -o bin\cclmonitor.exe .\cmd\cclmonitor
go build -o bin\cclmonitor-install.exe .\cmd\cclmonitor-install
go build -o bin\cclmonitor-ui.exe .\cmd\cclmonitor-ui

# Install to %USERPROFILE%\bin and register hooks
New-Item -ItemType Directory -Force -Path "$HOME\bin" | Out-Null
Move-Item -Force bin\*.exe "$HOME\bin\"
& "$HOME\bin\cclmonitor-install.exe"
```

This registers the hooks in `%USERPROFILE%\.claude\settings.json` (a backup is saved as `settings.json.bak`).

To use `cclmonitor test` from any terminal, add `%USERPROFILE%\bin` to your PATH:

```powershell
[Environment]::SetEnvironmentVariable("Path", "$env:Path;$HOME\bin", "User")
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
eventlog:
  retain_days: 30

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
| `eventlog.logdir` | path | `~/.claude/` | Directory for JSONL log files |
| `eventlog.retain_days` | int | `30` | Delete log files older than N days |
| `eventlog.grace_sec` | int | `60` | Seconds to wait before treating an unmatched `pending` as `cancelled` in `cclmonitor-ui` |

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

Dry-run a value against your current policy. No logging, no blocking.

```sh
cclmonitor test [--tool TOOL] [--cwd DIR] <value>

# Examples
cclmonitor test "git push --force"
cclmonitor test --tool Edit "~/.ssh/id_rsa"
cclmonitor test --tool Bash --cwd ~/projects/myapp "npm install"
```

### `cclmonitor suggest`

Analyze recent logs and propose new rules for `cclmonitor.yaml`. Each proposal is shown one at a time with `[y/N/q]` — `y` writes the rule, `n` skips, `q` exits.

```sh
cclmonitor suggest [--days 30] [--min-count 5] [--target global|project] [--dry-run]

# Typical flow: from inside your project root
cd ~/projects/myapp
cclmonitor suggest --target project
```

**Sources, in order:**

1. cclmonitor event log (`~/.claude/cclmonitor.YYYY-MM-DD.log` or your `eventlog.logdir`)
2. Claude Code transcripts (`~/.claude/projects/<encoded-cwd>/*.jsonl`) — used when cclmonitor logs are empty
3. **Built-in defaults** — applied when neither source has enough events (default threshold: 10). Adds secrets / shell-safety / git-safety deny rules with a single `[y/N]` prompt

**Safety:**

- Never suggests removing or relaxing existing `deny` rules
- Skips suggestions already present in the target yaml
- Backs up the target before the first write (`<path>.bak-YYYY-MM-DD-HHMMSS`)
- Writes atomically (temp file + rename)

**Tip:** run `suggest` from your project root so file-path suggestions get expressed as `<cwd>/...` globs.

### `cclmonitor-ui`

Full-screen TUI dashboard. Shows harness compliance scores, per-tool breakdown, 30-day timeline, and a live event feed.

```sh
cclmonitor-ui [--logdir ~/.claude/] [--snapshot] [--grace 60s]
```

| Key | Action |
|-----|--------|
| `1`–`4` | Switch tabs (Overview / Tools / Timeline / Events) |
| `t` / `7` / `m` / `a` | Period: today / 7d / 30d / all |
| `j` / `k` | Scroll events |
| `r` | Refresh |
| `s` | Pause / resume live updates |
| `q` | Quit |

#### Scores (Overview tab)

The Overview tab shows two scores derived from the verdict of each tool call.

**Verdicts** — every event lands in exactly one bucket:

| Verdict | Logged by | Meaning |
|---------|-----------|---------|
| `executed` | PostToolUse | allow rule matched; tool ran to completion |
| `denied` | PreToolUse | deny rule matched; tool was blocked |
| `cancelled` | — | `pending` with no matching PostToolUse; user cancelled at the prompt |
| `unknown` | PostToolUse | no rule matched; tool ran to completion |
| `interrupted` | PostToolUse | tool started but stopped mid-execution |

---

**Compliance Score** — *how well Claude operates within your allow rules*

```
executed ÷ (executed + denied + cancelled)
```

Denominator = events where your policy applied **and** the outcome is final.  
`unknown` is excluded (no rule was written for those calls, so they are outside the policy scope).  
`pending` is excluded (outcome not yet final).

| Score | Meaning |
|-------|---------|
| High | Claude is mostly attempting allowed operations |
| Low | Claude is frequently attempting operations your rules block |

---

**Coverage Score** — *how completely your rules cover real tool usage*

```
(executed + denied) ÷ (executed + denied + unknown)
```

Denominator = all calls confirmed as completed by PostToolUse.  
`cancelled` is excluded (tool never ran).  
A high `unknown` count means your rules have gaps — operations are running without any policy applied.

| Score | Meaning |
|-------|---------|
| High | Your rules cover almost all tool calls |
| Low | Many tool calls match no rule (consider adding more allow/deny entries) |

---

**Reading both scores together**

| Compliance | Coverage | Interpretation |
|------------|----------|----------------|
| High | High | Rules are thorough and Claude operates within policy ✅ |
| High | Low | Claude is well-behaved but rules have gaps (many `unknown`) |
| Low | High | Rules are thorough but Claude frequently attempts blocked operations |
| Low | Low | Rules are incomplete and Claude is frequently out of policy ⚠️ |

## Audit Log

Events are appended to date-rotated JSONL files:

```
~/.claude/
  cclmonitor.2024-01-15.log
  cclmonitor.2024-01-16.log
```

Each line is a JSON object:

```json
{"time":"2024-01-15T14:32:05Z","session_id":"abc123","tool_name":"Bash","value":"rm -rf /tmp","verdict":"denied"}
```

### Verdicts

| Verdict | When | Hook |
|---------|------|------|
| `pending` | allow/unknown — tool about to execute (or user will cancel) | PreToolUse |
| `executed` | allow rule matched, tool ran to completion | PostToolUse |
| `denied` | deny rule matched, tool blocked | PreToolUse |
| `unknown` | no rule matched, tool ran to completion | PostToolUse |
| `interrupted` | tool started but was stopped mid-execution (e.g. Ctrl+C) | PostToolUse |

A `pending` entry with no matching `tool_use_id` in PostToolUse means the user cancelled at the confirmation prompt.

---

## How the hook works

`make install` writes the following into `~/.claude/settings.json`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [{ "type": "command", "command": "/Users/<you>/bin/cclmonitor" }]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "",
        "hooks": [{ "type": "command", "command": "/Users/<you>/bin/cclmonitor post" }]
      }
    ]
  }
}
```

- **PreToolUse** evaluates the tool call before execution. Exit code `2` blocks the tool and logs `"denied"`. When a call is denied, cclmonitor also writes `{"reason": "..."}` to stdout — Claude Code displays this as the block reason and instructs the model not to attempt workarounds.
- **PostToolUse** fires after the tool actually ran. Logs `"executed"` or `"unknown"`. If the user cancels before execution, PostToolUse does not fire — leaving no log entry.

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

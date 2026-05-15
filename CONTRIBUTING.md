# Contributing to cclmonitor

Thank you for your interest in contributing!

## Architecture

### Execution flow

```
Claude Code invokes a tool
  → PreToolUse hook (cclmonitor)
      → deny match   : log "denied"  + exit 2  (tool blocked)
      → allow/unknown: log "pending" + exit 0
  → tool executes (or user cancels → PostToolUse never fires)
  → PostToolUse hook (cclmonitor post)
      → interrupted  : log "interrupted"
      → allow match  : log "executed"
      → unknown      : log "unknown"
```

### Package responsibilities

| Package | Role |
|---------|------|
| `cmd/cclmonitor/` | PreToolUse / PostToolUse entry points. Reads stdin → calls match engine → logs / blocks |
| `cmd/cclmonitor-install/` | Writes hook entries into `~/.claude/settings.json` (backs up before modifying) |
| `cmd/cclmonitor-ui/` | Full-screen TUI dashboard (Bubble Tea). Shows scores, per-tool breakdown, 30-day timeline, live event feed |
| `internal/hookio/` | Parses Claude Code stdin JSON. Defines per-tool `tool_input` structs |
| `internal/config/` | Loads and merges global + project-level YAML. Expands `<cwd>` tokens |
| `internal/match/` | Evaluates rules using `regexp` and `doublestar` (glob) |
| `internal/eventlog/` | Appends JSONL entries to log files; reads ranges and incremental diffs |
| `internal/metrics/` | Pairs Pre/PostToolUse events, computes Compliance / Coverage scores |
| `internal/suggest/` | Extracts patterns from logs and proposes new rules |

### Key design notes

- **`<cwd>` token** — expands to the working directory from the hook payload inside glob patterns.
- **Project-level overrides** — `.claude/cclmonitor.yaml` in a project root merges with global config; project rules take precedence per tool section.
- **`tool_use_id`** — present on every log entry; used to correlate PreToolUse and PostToolUse events.
- **`pending` verdict** — logged by PreToolUse for both `allow` and `unknown` outcomes, because the tool may still be cancelled by the user before it runs.

## Development workflow

This project follows a **Red → Green → Refactor** TDD cycle.

1. **Red** — write the test first (`*_test.go`). Confirm it fails.
2. **Green** — write the minimum implementation to make it pass.
3. **Refactor** — clean up without breaking tests.

Tests live alongside each package:

```
internal/match/
  match.go
  match_test.go
```

## Build & test

```sh
# Build all binaries to bin/
make build

# Run all tests
make test

# Run tests for a specific package (with verbose output)
go test -v ./internal/match/
go test -v ./cmd/cclmonitor/

# Install binaries to ~/bin/ and register hooks in ~/.claude/settings.json
make install

# Uninstall
make uninstall
```

## Commit message conventions

| Prefix | When to use | Version bump |
|--------|-------------|--------------|
| `fix:` | Bug fix | PATCH |
| `feat:` | New feature | MINOR |
| `feat!:` | Breaking change | MAJOR |
| `refactor:` | Code cleanup, no behavior change | — |
| `docs:` | Documentation only | — |
| `test:` | Tests only | — |
| `chore:` | Makefile, dependencies, release | — |

All commit messages, PR titles, and CHANGELOG entries must be written in **English**.

## Submitting a pull request

1. Fork the repository and create a feature branch.
2. Follow the TDD cycle above — PRs without tests for new behavior will not be merged.
3. Run `make test` and confirm all tests pass before opening a PR.
4. Keep changes focused. One concern per PR.

# Changelog

All notable changes to this project will be documented in this file.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
This project adheres to [Semantic Versioning](https://semver.org/).

## [0.9.1](https://github.com/ryosandesu/cclmonitor/compare/v0.9.0...v0.9.1) (2026-05-18)


### Bug Fixes

* compile regex at load time and add untracked verdict ([8276ca2](https://github.com/ryosandesu/cclmonitor/commit/8276ca2c1ebf3bd4319384c8dc520788e3e30b63))
* detect invalid regex at load time and add untracked verdict ([230d848](https://github.com/ryosandesu/cclmonitor/commit/230d8480b965f102bff6ad14164386aca0184808))

## [v0.9.1] - 2026-05-18

### Fixed
- Regex patterns in YAML config are now compiled at load time; invalid patterns are reported as an error instead of silently passing all tool calls through
- Tools not supported by cclmonitor (e.g. WebSearch) are now logged as `untracked` instead of `unknown` in PostToolUse

### Changed
- `untracked` verdict is excluded from Rule Coverage calculation in the UI

## [v0.9.0] - 2026-05-16

### Added
- GitHub Actions CI workflow (`.github/workflows/ci.yml`) — runs unit tests and integration tests on every push and pull request to `main`

## [v0.8.1] - 2026-05-16

### Added
- Integration test suite in `tests/` — builds and invokes the compiled binary as a subprocess; run with `make test-integration`

### Changed
- README, CONTRIBUTING, and SECURITY updated for public repository

## [v0.8.0] - 2026-05-15

### Removed
- `cclmonitor-tail` binary — live log viewer superseded by the Events tab in `cclmonitor-ui`

### Changed
- `cclmonitor suggest` flag defaults extracted as named constants in `suggest.go`
- `eventlog.TruncateDay` exported and shared with `metrics` to remove duplicate implementation

## [v0.7.0] - 2026-05-12

### Added
- `cclmonitor suggest` subcommand — scans cclmonitor event logs (or Claude Code transcripts as fallback) and proposes rule additions for `cclmonitor.yaml` via an interactive `[y/N/q]` prompt. Frequent `unknown` patterns become allow candidates; repeated `denied` patterns become deny candidates.
- Built-in baseline defaults (secrets / shell-safety / git-safety deny rules) applied via the suggest defaults mode when log data is insufficient.
- Structured YAML insertion: rules are added via `yaml.v3` Node API (not text append) with atomic write and per-session timestamped backup (`*.bak-YYYY-MM-DD-HHMMSS`).
- `--dry-run`, `--days`, `--min-count`, `--insufficient-threshold`, `--target {global|project}` flags for the new subcommand.
- PreToolUse deny now writes `{"reason": "..."}` to stdout in addition to exiting with code 2. Claude Code displays this as the block reason and instructs the model not to attempt workarounds.

### Changed
- `cclmonitor-ui` Overview: Recent Violations list is now sorted newest-first and timestamp includes date (`MM/DD HH:MM`)
- `cclmonitor-ui` value truncation changed to tail-based for file-path tools (Edit/Write/Read) so the filename is always visible; Bash commands continue to truncate from the right

## [v0.6.0] - 2026-05-11

### Added
- `eventlog.grace_sec` config key — sets the seconds `cclmonitor-ui` waits before treating an unmatched `pending` as `cancelled` (default: 60). Eliminates the need to pass `--grace` on every launch.

### Fixed
- Compliance percentage format unified to `%.1f%%` across all views (Overview Per-Tool section, Tools tab, and Timeline were inconsistent)

## [v0.5.1] - 2026-05-11

### Fixed
- `cclmonitor-ui` now reads `eventlog.logdir` from `~/.claude/cclmonitor.yaml` instead of always defaulting to `~/.claude/`, matching the behaviour of all other cclmonitor commands
- Events tab timestamp changed from `HH:MM:SS` to `YYYY-MM-DD HH:MM:SS` so dates are visible when viewing multi-day history

### Changed
- README: added Compliance / Coverage score explanation with verdict table and interpretation matrix

## [v0.5.0] - 2026-05-11

### Added
- Windows support: `cclmonitor-install` now appends `.exe` to hook commands on Windows so `settings.json` references the correct binary
- README installation section split into macOS/Linux and Windows subsections, with PowerShell instructions for Windows users

### Changed
- `cmd/cclmonitor`: project config path is now built with `filepath.Join` instead of string concatenation (transparent on POSIX, correct on Windows)
- README prerequisite updated to Go 1.26+ to match `go.mod`

## [v0.4.0] - 2026-05-11

### Added
- `cclmonitor-ui` — TUI dashboard with Overview, Tools, Timeline, and Events tabs
- Compliance Score (`executed / (executed + denied + cancelled)`) — measures how well Claude follows allow rules
- Coverage Score (`(executed + denied) / (executed + denied + unknown)`) — measures how complete the rule definitions are
- Period filter: Today / 7d / 30d / All (`t` / `7` / `m` / `a` keys)
- 30-day compliance trend chart (Timeline tab, always shown regardless of period filter)
- Live event feed with 500ms incremental polling

## [v0.3.1] - 2026-05-11

### Added
- `THIRD_PARTY_LICENSES` file listing license texts for bundled dependencies

## [v0.3.0] - 2026-05-11

### Added
- `pending` verdict: PreToolUse logs allow/unknown calls before execution; unmatched `pending` (no PostToolUse) indicates user cancelled at confirmation prompt
- `interrupted` verdict: PostToolUse logs `"interrupted"` when a tool is stopped mid-execution
- `tool_use_id` field in all log entries — enables correlation of Pre/PostToolUse pairs

### Fixed
- `CleanOldLogs` deleted files one day late in non-UTC timezones

### Changed
- `cclmonitor-tail` verdict colors corrected to match actual verdict strings (`executed`, `denied`, etc.)
- `cclmonitor-tail` adds blue for `pending`, cyan for `interrupted`

## [v0.2.0] - 2026-05-10

### Added
- PostToolUse hook (`cclmonitor post`) records commands that actually executed
- Three-verdict audit log: `executed` / `denied` / `unknown`
  - `executed` — allow rule matched, confirmed ran via PostToolUse
  - `denied` — deny rule matched, blocked by PreToolUse
  - `unknown` — no rule matched, confirmed ran via PostToolUse
- `cclmonitor-install` now registers both PreToolUse and PostToolUse hooks in `settings.json`

### Changed
- YAML key `notify:` renamed to `eventlog:` (config schema breaking change)
- Renamed internal package `notify` → `eventlog`
- PreToolUse no longer logs allow/unknown verdicts; PostToolUse handles these

### Removed
- macOS notifications via `osascript`
- SQLite-backed deduplication (`dedup_window_sec`, `dbdir` config keys)
- `mode: dev/prod` configuration option
- `notify.channels` configuration key

## [v0.1.0] - 2026-05-08

### Added
- `PreToolUse` hook binary that intercepts Claude Code tool calls
- Policy-based allow/deny rules via YAML config (`regex` and `glob` matching)
- Per-tool rule sections: `Bash`, `Edit`, `Write`, `Read`
- `<cwd>` token expansion in glob patterns for project-scoped rules
- Global config (`~/.claude/cclmonitor.yaml`) + project-level override (`.claude/cclmonitor.yaml`)
- JSONL audit log with daily rotation under `~/.claude/`
- `cclmonitor test` dry-run command to evaluate rules without blocking
- `cclmonitor-tail` live log viewer with color-coded verdicts
- `cclmonitor-install` auto-registers the hook into `~/.claude/settings.json`
- `make install` / `make uninstall` for one-command setup

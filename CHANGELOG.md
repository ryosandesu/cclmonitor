# Changelog

All notable changes to this project will be documented in this file.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
This project adheres to [Semantic Versioning](https://semver.org/).

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

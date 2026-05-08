# Changelog

All notable changes to this project will be documented in this file.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
This project adheres to [Semantic Versioning](https://semver.org/).

## [v0.1.0] - 2026-05-08

### Added
- `PreToolUse` hook binary that intercepts Claude Code tool calls
- Policy-based allow/deny rules via YAML config (`regex` and `glob` matching)
- Per-tool rule sections: `Bash`, `Edit`, `Write`, `Read`
- `<cwd>` token expansion in glob patterns for project-scoped rules
- Global config (`~/.claude/cclmonitor.yaml`) + project-level override (`.claude/cclmonitor.yaml`)
- `mode: dev` to notify on `allow` hits for policy coverage verification
- macOS notifications via `osascript`
- JSONL audit log with daily rotation under `~/.claude/`
- SQLite-backed deduplication (`dedup_window_sec`) to suppress repeated events
- `cclmonitor test` dry-run command to evaluate rules without blocking
- `cclmonitor-tail` live log viewer with color-coded verdicts
- `cclmonitor-install` auto-registers the hook into `~/.claude/settings.json`
- `make install` / `make uninstall` for one-command setup
- Zero CGO — pure Go SQLite via `modernc.org/sqlite`

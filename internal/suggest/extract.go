// Package suggest provides analysis of cclmonitor event logs to propose
// rule additions for cclmonitor.yaml. Pattern extraction lives here;
// counting/filtering lives in aggregate.go.
package suggest

import (
	"path/filepath"
	"strings"
)

// ExtractBashKey returns a canonical key for a Bash command value.
// It strips leading environment variable assignments (FOO=bar) and
// takes up to the first two whitespace-separated tokens.
// Returns ("", false) for empty/whitespace-only inputs or env-var-only inputs.
func ExtractBashKey(value string) (string, bool) {
	fields := strings.Fields(value)
	for len(fields) > 0 && isEnvAssignment(fields[0]) {
		fields = fields[1:]
	}
	if len(fields) == 0 {
		return "", false
	}
	if len(fields) == 1 {
		return fields[0], true
	}
	return fields[0] + " " + fields[1], true
}

// BashKeyToRegex converts a key (one or two tokens) into a regex pattern
// suitable for cclmonitor.yaml. The section selects the anchoring style:
//   - "allow": (^|[\s;&|]) — also matches chained commands like `cd x && pnpm install`
//   - "deny":  \b — catches the pattern anywhere in the command
func BashKeyToRegex(key, section string) string {
	tokens := strings.SplitN(key, " ", 2)
	var prefix string
	switch section {
	case "allow":
		prefix = `(^|[\s;&|])`
	default:
		prefix = `\b`
	}
	if len(tokens) == 1 {
		return prefix + tokens[0] + `\b`
	}
	return prefix + tokens[0] + `\s+` + tokens[1] + `\b`
}

// ExtractFileGlob converts an absolute file path to a glob pattern,
// generalizing by top-level directory under cwd and file extension.
//
//	cwd=/proj  path=/proj/src/components/Button.tsx → <cwd>/src/**/*.tsx
//	cwd=/proj  path=/proj/file.go                    → <cwd>/**/*.go
//	cwd=/proj  path=/proj/Makefile                   → <cwd>/**/Makefile
//	cwd=/proj  path=/etc/hosts                       → /etc/hosts (outside cwd, returned as-is)
func ExtractFileGlob(cwd, path string) (string, bool) {
	if path == "" {
		return "", false
	}

	// Outside cwd or no cwd info → return path as-is.
	if cwd == "" || !isUnderCwd(cwd, path) {
		return path, true
	}

	rel, err := filepath.Rel(cwd, path)
	if err != nil || rel == "." || rel == "" {
		return "", false
	}
	rel = filepath.ToSlash(rel)

	parts := strings.SplitN(rel, "/", 2)
	base := filepath.Base(rel)
	ext := filepath.Ext(base)

	// File directly in cwd root (no subdirectory).
	if len(parts) == 1 {
		if ext == "" {
			return "<cwd>/**/" + base, true
		}
		return "<cwd>/**/*" + ext, true
	}

	topDir := parts[0]
	if ext == "" {
		return "<cwd>/" + topDir + "/**/" + base, true
	}
	return "<cwd>/" + topDir + "/**/*" + ext, true
}

func isEnvAssignment(token string) bool {
	if token == "" {
		return false
	}
	eq := strings.IndexByte(token, '=')
	if eq <= 0 {
		return false
	}
	// Must be a valid identifier before '=': [A-Za-z_][A-Za-z0-9_]*
	name := token[:eq]
	for i, r := range name {
		if r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func isUnderCwd(cwd, path string) bool {
	cwd = filepath.Clean(cwd)
	path = filepath.Clean(path)
	if path == cwd {
		return true
	}
	prefix := cwd
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return strings.HasPrefix(path, prefix)
}

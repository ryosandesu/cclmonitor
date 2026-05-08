package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunDryRun_DenyVerdict(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeCfg(t, dir, `
rules:
  Bash:
    deny:
      - regex: '\brm\s+-rf'
`)
	var out bytes.Buffer
	code := runDryRun(&out, "Bash", "rm -rf /", dir, cfgPath)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	got := out.String()
	if !strings.Contains(got, "deny") {
		t.Errorf("output should contain 'deny', got: %s", got)
	}
	if !strings.Contains(got, "Bash") {
		t.Errorf("output should contain 'Bash', got: %s", got)
	}
	if !strings.Contains(got, "rm -rf /") {
		t.Errorf("output should contain value, got: %s", got)
	}
}

func TestRunDryRun_AllowVerdict(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeCfg(t, dir, `
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	var out bytes.Buffer
	code := runDryRun(&out, "Bash", "ls -la", dir, cfgPath)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "allow") {
		t.Errorf("output should contain 'allow', got: %s", out.String())
	}
}

func TestRunDryRun_UnknownVerdict(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeCfg(t, dir, `
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	var out bytes.Buffer
	runDryRun(&out, "Bash", "git status", dir, cfgPath)
	if !strings.Contains(out.String(), "unknown") {
		t.Errorf("output should contain 'unknown', got: %s", out.String())
	}
}

func TestRunDryRun_NoConfigReturns0(t *testing.T) {
	var out bytes.Buffer
	code := runDryRun(&out, "Bash", "anything", "/tmp", "/nonexistent/cfg.yaml")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

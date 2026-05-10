package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCfg(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "cclmonitor.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func bashPayload(cmd, cwd, session string) string {
	input, _ := json.Marshal(map[string]string{"command": cmd})
	p := map[string]any{
		"tool_name":  "Bash",
		"tool_input": json.RawMessage(input),
		"cwd":        cwd,
		"session_id": session,
	}
	b, _ := json.Marshal(p)
	return string(b)
}

func readTodayLog(t *testing.T, dir string) []byte {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read log dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "cclmonitor.") && strings.HasSuffix(e.Name(), ".log") {
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			return data
		}
	}
	t.Fatal("no cclmonitor log file found in dir")
	return nil
}

// --- PreToolUse (run) ---

func TestRun_DenyReturns2(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeCfg(t, dir, `
rules:
  Bash:
    deny:
      - regex: '\brm\s+-rf'
`)
	code := run(strings.NewReader(bashPayload("rm -rf /", dir, "s1")), cfgPath)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestRun_DenyWritesDeniedLog(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	cfgPath := writeCfg(t, dir, `
eventlog:
  logdir: `+logDir+`
rules:
  Bash:
    deny:
      - regex: '\brm\s+-rf'
`)
	run(strings.NewReader(bashPayload("rm -rf /", dir, "s1")), cfgPath)

	data := readTodayLog(t, logDir)
	if !strings.Contains(string(data), "denied") {
		t.Errorf("log should contain 'denied', got: %s", data)
	}
}

func TestRun_AllowReturns0WithNoLog(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	cfgPath := writeCfg(t, dir, `
eventlog:
  logdir: `+logDir+`
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	code := run(strings.NewReader(bashPayload("ls -la", dir, "s1")), cfgPath)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if _, err := os.ReadDir(logDir); err == nil {
		entries, _ := os.ReadDir(logDir)
		if len(entries) > 0 {
			t.Error("PreToolUse should not write log for allow verdict")
		}
	}
}

func TestRun_UnknownReturns0WithNoLog(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	cfgPath := writeCfg(t, dir, `
eventlog:
  logdir: `+logDir+`
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	code := run(strings.NewReader(bashPayload("git status", dir, "s1")), cfgPath)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if _, err := os.ReadDir(logDir); err == nil {
		entries, _ := os.ReadDir(logDir)
		if len(entries) > 0 {
			t.Error("PreToolUse should not write log for unknown verdict")
		}
	}
}

func TestRun_NoConfigReturns0(t *testing.T) {
	code := run(strings.NewReader(bashPayload("anything", "/tmp", "s1")), "/nonexistent/cclmonitor.yaml")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestRun_MalformedInputReturns0(t *testing.T) {
	code := run(strings.NewReader("not json"), "/nonexistent/cfg.yaml")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

// --- PostToolUse (runPost) ---

func TestRunPost_AllowWritesExecutedLog(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	cfgPath := writeCfg(t, dir, `
eventlog:
  logdir: `+logDir+`
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	runPost(strings.NewReader(bashPayload("ls -la", dir, "s1")), cfgPath)

	data := readTodayLog(t, logDir)
	if !strings.Contains(string(data), "executed") {
		t.Errorf("log should contain 'executed', got: %s", data)
	}
}

func TestRunPost_UnknownWritesUnknownLog(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	cfgPath := writeCfg(t, dir, `
eventlog:
  logdir: `+logDir+`
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	runPost(strings.NewReader(bashPayload("git status", dir, "s1")), cfgPath)

	data := readTodayLog(t, logDir)
	if !strings.Contains(string(data), "unknown") {
		t.Errorf("log should contain 'unknown', got: %s", data)
	}
}

func TestRunPost_DenyIsIgnored(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	cfgPath := writeCfg(t, dir, `
eventlog:
  logdir: `+logDir+`
rules:
  Bash:
    deny:
      - regex: '\brm\s+-rf'
`)
	code := runPost(strings.NewReader(bashPayload("rm -rf /", dir, "s1")), cfgPath)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if _, err := os.ReadDir(logDir); err == nil {
		entries, _ := os.ReadDir(logDir)
		if len(entries) > 0 {
			t.Error("PostToolUse should not write log for deny verdict")
		}
	}
}

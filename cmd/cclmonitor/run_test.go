package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCfg(t *testing.T, dir, content string) string {
	t.Helper()
	// inject logdir when missing so tests never write to the real ~/.claude/
	if !strings.Contains(content, "logdir:") {
		content = "eventlog:\n  logdir: " + dir + "\n" + content
	}
	path := filepath.Join(dir, "cclmonitor.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func webSearchPayload(query, cwd, session string) string {
	input, _ := json.Marshal(map[string]string{"query": query})
	p := map[string]any{
		"tool_name":   "WebSearch",
		"tool_input":  json.RawMessage(input),
		"cwd":         cwd,
		"session_id":  session,
		"tool_use_id": "toolu_test01",
	}
	b, _ := json.Marshal(p)
	return string(b)
}

func bashPayload(cmd, cwd, session string) string {
	input, _ := json.Marshal(map[string]string{"command": cmd})
	p := map[string]any{
		"tool_name":   "Bash",
		"tool_input":  json.RawMessage(input),
		"cwd":         cwd,
		"session_id":  session,
		"tool_use_id": "toolu_test01",
	}
	b, _ := json.Marshal(p)
	return string(b)
}

func bashPayloadInterrupted(cmd, cwd, session string) string {
	input, _ := json.Marshal(map[string]string{"command": cmd})
	p := map[string]any{
		"tool_name":   "Bash",
		"tool_input":  json.RawMessage(input),
		"cwd":         cwd,
		"session_id":  session,
		"tool_use_id": "toolu_test01",
		"tool_response": map[string]any{
			"interrupted": true,
		},
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
	code := run(strings.NewReader(bashPayload("rm -rf /", dir, "s1")), io.Discard, cfgPath)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestRun_DenyWritesReasonJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeCfg(t, dir, `
rules:
  Bash:
    deny:
      - regex: '\brm\s+-rf'
`)
	var buf strings.Builder
	run(strings.NewReader(bashPayload("rm -rf /", dir, "s1")), &buf, cfgPath)

	var out map[string]string
	if err := json.Unmarshal([]byte(buf.String()), &out); err != nil {
		t.Fatalf("stdout is not valid JSON: %v, got: %s", err, buf.String())
	}
	reason, ok := out["reason"]
	if !ok {
		t.Fatal("stdout JSON missing 'reason' field")
	}
	if !strings.Contains(reason, "Bash") {
		t.Errorf("reason should contain tool name, got: %s", reason)
	}
	if !strings.Contains(reason, "rm -rf /") {
		t.Errorf("reason should contain blocked value, got: %s", reason)
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
	run(strings.NewReader(bashPayload("rm -rf /", dir, "s1")), io.Discard, cfgPath)

	data := readTodayLog(t, logDir)
	if !strings.Contains(string(data), "denied") {
		t.Errorf("log should contain 'denied', got: %s", data)
	}
}

func TestRun_AllowWritesPendingLog(t *testing.T) {
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
	code := run(strings.NewReader(bashPayload("ls -la", dir, "s1")), io.Discard, cfgPath)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	data := readTodayLog(t, logDir)
	if !strings.Contains(string(data), "pending") {
		t.Errorf("log should contain 'pending', got: %s", data)
	}
}

func TestRun_UnknownWritesPendingLog(t *testing.T) {
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
	code := run(strings.NewReader(bashPayload("git status", dir, "s1")), io.Discard, cfgPath)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	data := readTodayLog(t, logDir)
	if !strings.Contains(string(data), "pending") {
		t.Errorf("log should contain 'pending', got: %s", data)
	}
}

func TestRun_NoConfigReturns0(t *testing.T) {
	// redirect HOME so resolveDir("") writes to a temp dir instead of ~/.claude/
	t.Setenv("HOME", t.TempDir())
	code := run(strings.NewReader(bashPayload("anything", "/tmp", "s1")), io.Discard, "/nonexistent/cclmonitor.yaml")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestRun_MalformedInputReturns0(t *testing.T) {
	code := run(strings.NewReader("not json"), io.Discard, "/nonexistent/cfg.yaml")
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

func TestRunPost_LogsToolUseID(t *testing.T) {
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
	if !strings.Contains(string(data), "toolu_test01") {
		t.Errorf("log should contain tool_use_id, got: %s", data)
	}
}

func TestRunPost_InterruptedWritesInterruptedLog(t *testing.T) {
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
	runPost(strings.NewReader(bashPayloadInterrupted("ls -la", dir, "s1")), cfgPath)

	data := readTodayLog(t, logDir)
	if !strings.Contains(string(data), "interrupted") {
		t.Errorf("log should contain 'interrupted', got: %s", data)
	}
}

func TestRun_UntrackedToolWritesPendingLog(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	cfgPath := writeCfg(t, dir, "eventlog:\n  logdir: "+logDir+"\n")
	code := run(strings.NewReader(webSearchPayload("golang", dir, "s1")), io.Discard, cfgPath)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	data := readTodayLog(t, logDir)
	if !strings.Contains(string(data), "pending") {
		t.Errorf("log should contain 'pending', got: %s", data)
	}
}

func TestRunPost_UntrackedToolWritesUntrackedLog(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	cfgPath := writeCfg(t, dir, "eventlog:\n  logdir: "+logDir+"\n")
	runPost(strings.NewReader(webSearchPayload("golang", dir, "s1")), cfgPath)
	data := readTodayLog(t, logDir)
	if !strings.Contains(string(data), "untracked") {
		t.Errorf("log should contain 'untracked', got: %s", data)
	}
}

func TestRun_DefaultVerdictDenyUnknownReturns2(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeCfg(t, dir, `
default_verdict: deny
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	code := run(strings.NewReader(bashPayload("git status", dir, "s1")), io.Discard, cfgPath)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestRun_DefaultVerdictDenyUnknownWritesUnknownLog(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	cfgPath := writeCfg(t, dir, `
default_verdict: deny
eventlog:
  logdir: `+logDir+`
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	run(strings.NewReader(bashPayload("git status", dir, "s1")), io.Discard, cfgPath)

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

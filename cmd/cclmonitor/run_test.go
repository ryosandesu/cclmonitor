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

func TestRun_AllowReturns0(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeCfg(t, dir, `
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	code := run(strings.NewReader(bashPayload("ls -la", dir, "s1")), cfgPath)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestRun_UnknownReturns0(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeCfg(t, dir, `
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	code := run(strings.NewReader(bashPayload("git status", dir, "s1")), cfgPath)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestRun_NoConfigReturns0(t *testing.T) {
	code := run(strings.NewReader(bashPayload("anything", "/tmp", "s1")), "/nonexistent/cclmonitor.yaml")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
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

func TestRun_DevModeWritesLogOnAllow(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	cfgPath := writeCfg(t, dir, `
mode: dev
notify:
  channels: [logfile]
  logdir: `+logDir+`
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	run(strings.NewReader(bashPayload("ls -la", dir, "s1")), cfgPath)

	data := readTodayLog(t, logDir)
	if !strings.Contains(string(data), "allow") {
		t.Errorf("log should contain 'allow', got: %s", data)
	}
}

func TestRun_UnknownWritesLog(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	cfgPath := writeCfg(t, dir, `
notify:
  channels: [logfile]
  logdir: `+logDir+`
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	run(strings.NewReader(bashPayload("git status", dir, "s1")), cfgPath)

	data := readTodayLog(t, logDir)
	if !strings.Contains(string(data), "unknown") {
		t.Errorf("log should contain 'unknown', got: %s", data)
	}
}

func TestRun_MalformedInputReturns0(t *testing.T) {
	code := run(strings.NewReader("not json"), "/nonexistent/cfg.yaml")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestRun_DedupSuppressesSecondNotification(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	dbDir := filepath.Join(dir, "db")
	cfgPath := writeCfg(t, dir, `
notify:
  channels: [logfile]
  logdir: `+logDir+`
  dbdir: `+dbDir+`
  dedup_window_sec: 60
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	// 1回目：ログに書かれる
	run(strings.NewReader(bashPayload("git status", dir, "s1")), cfgPath)
	data1 := readTodayLog(t, logDir)
	lines1 := strings.Count(string(data1), "\n")

	// 2回目：dedup により抑制されるのでログ行数が増えない
	run(strings.NewReader(bashPayload("git status", dir, "s1")), cfgPath)
	data2 := readTodayLog(t, logDir)
	lines2 := strings.Count(string(data2), "\n")

	if lines2 > lines1 {
		t.Errorf("second identical event should be deduplicated, lines: %d -> %d", lines1, lines2)
	}
}

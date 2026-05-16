//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var cclmonitorBin string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "cclmonitor-inttest-*")
	if err != nil {
		log.Fatalf("TempDir: %v", err)
	}
	defer os.RemoveAll(tmp)

	cclmonitorBin = filepath.Join(tmp, "cclmonitor")
	repoRoot, _ := filepath.Abs("..")
	build := exec.Command("go", "build", "-o", cclmonitorBin, "./cmd/cclmonitor")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		log.Fatalf("build failed: %v\n%s", err, out)
	}

	os.Exit(m.Run())
}

func bashStdin(cmd, cwd string) []byte {
	input, _ := json.Marshal(map[string]string{"command": cmd})
	p := map[string]any{
		"tool_name":   "Bash",
		"tool_input":  json.RawMessage(input),
		"cwd":         cwd,
		"session_id":  "integration-test",
		"tool_use_id": "toolu_inttest01",
	}
	b, _ := json.Marshal(p)
	return b
}

// writeProjectCfg creates <dir>/.claude/cclmonitor.yaml with the given content.
// The binary merges this project config on top of the global config, so tests
// can inject rules without touching the user's real ~/.claude/cclmonitor.yaml.
func writeProjectCfg(t *testing.T, dir, content string) {
	t.Helper()
	cfgDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "cclmonitor.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if e, ok := err.(*exec.ExitError); ok {
		return e.ExitCode()
	}
	return -1
}

// --- PreToolUse ---

func TestBinary_DenyExitsWithCode2(t *testing.T) {
	dir := t.TempDir()
	writeProjectCfg(t, dir, `
rules:
  Bash:
    deny:
      - regex: '\brm\s+-rf'
`)
	cmd := exec.Command(cclmonitorBin)
	cmd.Stdin = bytes.NewReader(bashStdin("rm -rf /", dir))
	var stdout strings.Builder
	cmd.Stdout = &stdout

	code := exitCode(cmd.Run())
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stdout.String(), "reason") {
		t.Errorf("stdout should contain reason JSON, got: %s", stdout.String())
	}
}

func TestBinary_AllowExitsWithCode0(t *testing.T) {
	dir := t.TempDir()
	writeProjectCfg(t, dir, `
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	cmd := exec.Command(cclmonitorBin)
	cmd.Stdin = bytes.NewReader(bashStdin("ls -la", dir))
	if code := exitCode(cmd.Run()); code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestBinary_UnknownExitsWithCode0(t *testing.T) {
	dir := t.TempDir()
	writeProjectCfg(t, dir, `
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	cmd := exec.Command(cclmonitorBin)
	cmd.Stdin = bytes.NewReader(bashStdin("git status", dir))
	if code := exitCode(cmd.Run()); code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

// --- PostToolUse (post subcommand) ---

func TestBinary_PostAllowExitsWithCode0(t *testing.T) {
	dir := t.TempDir()
	writeProjectCfg(t, dir, `
rules:
  Bash:
    allow:
      - regex: '^ls\b'
`)
	cmd := exec.Command(cclmonitorBin, "post")
	cmd.Stdin = bytes.NewReader(bashStdin("ls -la", dir))
	if code := exitCode(cmd.Run()); code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestBinary_PostDenyExitsWithCode0(t *testing.T) {
	dir := t.TempDir()
	writeProjectCfg(t, dir, `
rules:
  Bash:
    deny:
      - regex: '\brm\s+-rf'
`)
	// PostToolUse never blocks; deny verdict is silently ignored
	cmd := exec.Command(cclmonitorBin, "post")
	cmd.Stdin = bytes.NewReader(bashStdin("rm -rf /", dir))
	if code := exitCode(cmd.Run()); code != 0 {
		t.Errorf("exit code = %d, want 0 (post never blocks)", code)
	}
}

// --- version flag ---

func TestBinary_VersionFlag(t *testing.T) {
	cmd := exec.Command(cclmonitorBin, "--version")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("--version failed: %v", err)
	}
	if !strings.Contains(string(out), "cclmonitor") {
		t.Errorf("--version output should contain 'cclmonitor', got: %s", out)
	}
}

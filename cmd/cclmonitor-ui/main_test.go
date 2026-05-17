package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveGrace_FlagTakesPriority(t *testing.T) {
	home := t.TempDir()
	cfgPath := filepath.Join(home, ".claude", "cclmonitor.yaml")
	got := resolveGrace(30*time.Second, cfgPath)
	if got != 30*time.Second {
		t.Errorf("resolveGrace with flag = %v, want 30s", got)
	}
}

func TestResolveGrace_UsesCfgGraceSec(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "cclmonitor.yaml")
	os.WriteFile(cfgPath, []byte("eventlog:\n  grace_sec: 120\n"), 0600)
	got := resolveGrace(0, cfgPath)
	if got != 120*time.Second {
		t.Errorf("resolveGrace from config = %v, want 120s", got)
	}
}

func TestResolveGrace_FallsBackToDefault(t *testing.T) {
	home := t.TempDir()
	cfgPath := filepath.Join(home, ".claude", "nonexistent.yaml")
	got := resolveGrace(0, cfgPath)
	if got != 60*time.Second {
		t.Errorf("resolveGrace default = %v, want 60s", got)
	}
}

func TestResolveLogDir_FlagTakesPriority(t *testing.T) {
	home := t.TempDir()
	cfgPath := filepath.Join(home, ".claude", "cclmonitor.yaml")
	got := resolveLogDir("/explicit/path", cfgPath, home)
	if got != "/explicit/path" {
		t.Errorf("resolveLogDir with flag = %q, want /explicit/path", got)
	}
}

func TestResolveLogDir_UsesCfgLogDir(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "cclmonitor.yaml")
	os.WriteFile(cfgPath, []byte("eventlog:\n  logdir: ~/.claude/logs\n"), 0600)

	got := resolveLogDir("", cfgPath, home)
	want := filepath.Join(home, ".claude", "logs")
	if got != want {
		t.Errorf("resolveLogDir from config = %q, want %q", got, want)
	}
}

func TestResolveLogDir_FallsBackToDefault(t *testing.T) {
	home := t.TempDir()
	cfgPath := filepath.Join(home, ".claude", "cclmonitor.yaml") // does not exist
	got := resolveLogDir("", cfgPath, home)
	want := filepath.Join(home, ".claude")
	if got != want {
		t.Errorf("resolveLogDir default = %q, want %q", got, want)
	}
}

func TestResolveLogDir_ExpandsTildeInFlag(t *testing.T) {
	home := t.TempDir()
	got := resolveLogDir("~/mylogs", "", home)
	want := filepath.Join(home, "mylogs")
	if got != want {
		t.Errorf("resolveLogDir tilde in flag = %q, want %q", got, want)
	}
}

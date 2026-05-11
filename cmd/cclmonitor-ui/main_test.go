package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
	cfgPath := filepath.Join(home, ".claude", "cclmonitor.yaml") // 存在しない
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

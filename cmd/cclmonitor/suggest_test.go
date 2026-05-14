package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPromptYNQ(t *testing.T) {
	cases := []struct {
		input string
		want  rune
	}{
		{"y\n", 'y'},
		{"Y\n", 'y'},
		{"yes\n", 'y'},
		{"n\n", 'n'},
		{"\n", 'n'},
		{"NO\n", 'n'},
		{"q\n", 'q'},
		{"quit\n", 'q'},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			var out bytes.Buffer
			got, err := promptYNQ(strings.NewReader(c.input), &out, "Add?")
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("input=%q got=%q want=%q", c.input, got, c.want)
			}
		})
	}
}

func TestRunSuggest_GeneratesAndAppliesFromLogs(t *testing.T) {
	dir := t.TempDir()
	logdir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logdir, 0755); err != nil {
		t.Fatal(err)
	}

	// Seed 6 unknown Bash events for "pnpm install" on 2026-05-10
	day := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	logFile := filepath.Join(logdir, "cclmonitor.2026-05-10.log")
	var lines strings.Builder
	for i := 0; i < 6; i++ {
		lines.WriteString(`{"time":"` + day.Format(time.RFC3339Nano) + `","tool_name":"Bash","value":"pnpm install","verdict":"unknown"}` + "\n")
	}
	if err := os.WriteFile(logFile, []byte(lines.String()), 0600); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, "cclmonitor.yaml")
	opts := suggestOpts{
		Days:               30,
		MinCount:           5,
		Target:             target,
		InsufficientThresh: 1,
		LogDir:             logdir,
		CWD:                dir,
		HomeDir:            dir,
		Now:                time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC),
	}

	var out bytes.Buffer
	code := runSuggest(strings.NewReader("y\n"), &out, opts)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	// Verify the rule was added to the target yaml.
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `pnpm\s+install`) {
		t.Errorf("rule not inserted: %s", content)
	}

	// Verify backup was created.
	matches, _ := filepath.Glob(target + ".bak-*")
	// Note: target didn't exist before run, so no backup is expected
	if len(matches) != 0 {
		t.Errorf("expected no backup for new file, got %v", matches)
	}
}

func TestRunSuggest_DefaultsModeOnInsufficientLogs(t *testing.T) {
	dir := t.TempDir()
	logdir := filepath.Join(dir, "logs")
	os.MkdirAll(logdir, 0755)

	// No log files = no events at all.
	target := filepath.Join(dir, "cclmonitor.yaml")
	opts := suggestOpts{
		Days:               30,
		MinCount:           5,
		Target:             target,
		InsufficientThresh: 10,
		LogDir:             logdir,
		HomeDir:            dir, // no transcripts under HomeDir/.claude/projects either
		CWD:                dir,
		Now:                time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC),
	}

	var out bytes.Buffer
	code := runSuggest(strings.NewReader("y\n"), &out, opts)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	if !strings.Contains(out.String(), "baseline security defaults") {
		t.Errorf("expected defaults prompt in output: %s", out.String())
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	// Should contain some baseline deny rule
	if !strings.Contains(string(content), `sudo`) {
		t.Errorf("baseline not applied: %s", content)
	}
}

func TestRunSuggest_DryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	logdir := filepath.Join(dir, "logs")
	os.MkdirAll(logdir, 0755)

	day := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	logFile := filepath.Join(logdir, "cclmonitor.2026-05-10.log")
	var lines strings.Builder
	for i := 0; i < 6; i++ {
		lines.WriteString(`{"time":"` + day.Format(time.RFC3339Nano) + `","tool_name":"Bash","value":"pnpm install","verdict":"unknown"}` + "\n")
	}
	os.WriteFile(logFile, []byte(lines.String()), 0600)

	target := filepath.Join(dir, "cclmonitor.yaml")
	opts := suggestOpts{
		Days:               30,
		MinCount:           5,
		Target:             target,
		InsufficientThresh: 1,
		LogDir:             logdir,
		CWD:                dir,
		HomeDir:            dir,
		Now:                time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC),
		DryRun:             true,
	}

	var out bytes.Buffer
	code := runSuggest(strings.NewReader("y\n"), &out, opts)
	if code != 0 {
		t.Fatalf("exit code %d, want 0", code)
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("expected target file not to exist in dry-run, err=%v", err)
	}
}

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `
mode: prod
notify:
  channels: [osascript, logfile]
  logdir: ~/.claude/
  dedup_window_sec: 60
  retain_days: 30
rules:
  Bash:
    allow:
      - regex: '^(ls|pwd)\b'
    deny:
      - regex: '\brm\s+-rf\s+/'
  Edit:
    deny:
      - glob: '**/.env*'
`
	path := filepath.Join(dir, "cclmonitor.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile error: %v", err)
	}
	if cfg.Mode != "prod" {
		t.Errorf("Mode = %q, want %q", cfg.Mode, "prod")
	}
	if len(cfg.Notify.Channels) != 2 {
		t.Errorf("Channels len = %d, want 2", len(cfg.Notify.Channels))
	}
	if cfg.Notify.DedupWindowSec != 60 {
		t.Errorf("DedupWindowSec = %d, want 60", cfg.Notify.DedupWindowSec)
	}
	if cfg.Notify.RetainDays != 30 {
		t.Errorf("RetainDays = %d, want 30", cfg.Notify.RetainDays)
	}
	bashRules, ok := cfg.Rules["Bash"]
	if !ok {
		t.Fatal("no Bash rules")
	}
	if len(bashRules.Allow) != 1 {
		t.Errorf("Bash allow len = %d, want 1", len(bashRules.Allow))
	}
	if bashRules.Allow[0].Regex != `^(ls|pwd)\b` {
		t.Errorf("Bash allow regex = %q", bashRules.Allow[0].Regex)
	}
}

func TestLoadFile_NotFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/path/cclmonitor.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestExpandCwd(t *testing.T) {
	cfg := &Config{
		Rules: map[string]ToolRules{
			"Edit": {
				Allow: []Rule{{Glob: "<cwd>/**/*.go"}},
				Deny:  []Rule{{Glob: "<cwd>/.env"}},
			},
		},
	}
	cwd := "/home/user/project"
	result := ExpandCwd(cfg, cwd)

	editRules := result.Rules["Edit"]
	if editRules.Allow[0].Glob != "/home/user/project/**/*.go" {
		t.Errorf("Allow glob = %q", editRules.Allow[0].Glob)
	}
	if editRules.Deny[0].Glob != "/home/user/project/.env" {
		t.Errorf("Deny glob = %q", editRules.Deny[0].Glob)
	}
}

func TestExpandCwd_PreservesNonCwdPatterns(t *testing.T) {
	cfg := &Config{
		Rules: map[string]ToolRules{
			"Bash": {
				Allow: []Rule{{Regex: `^ls\b`}},
			},
		},
	}
	result := ExpandCwd(cfg, "/some/path")
	if result.Rules["Bash"].Allow[0].Regex != `^ls\b` {
		t.Error("regex should not be modified by ExpandCwd")
	}
}

func TestMerge(t *testing.T) {
	global := &Config{
		Mode: "prod",
		Notify: NotifyConfig{
			Channels:       []string{"osascript"},
			DedupWindowSec: 60,
		},
		Rules: map[string]ToolRules{
			"Bash": {
				Allow: []Rule{{Regex: `^ls\b`}},
			},
		},
	}
	project := &Config{
		Rules: map[string]ToolRules{
			"Edit": {
				Deny: []Rule{{Glob: "**/.env*"}},
			},
		},
	}

	merged := Merge(global, project)

	if merged.Mode != "prod" {
		t.Errorf("Mode = %q, want %q", merged.Mode, "prod")
	}
	if _, ok := merged.Rules["Bash"]; !ok {
		t.Error("global Bash rules should be present")
	}
	if _, ok := merged.Rules["Edit"]; !ok {
		t.Error("project Edit rules should be present")
	}
}

func TestMerge_ProjectAddsToGlobal(t *testing.T) {
	global := &Config{
		Mode: "prod",
		Rules: map[string]ToolRules{
			"Bash": {
				Allow: []Rule{{Regex: `^ls\b`}},
			},
		},
	}
	project := &Config{
		Mode: "dev",
		Rules: map[string]ToolRules{
			"Bash": {
				Allow: []Rule{{Regex: `^(go|make)\b`}},
			},
		},
	}

	merged := Merge(global, project)

	if merged.Mode != "dev" {
		t.Errorf("Mode = %q, want project's %q", merged.Mode, "dev")
	}
	if len(merged.Rules["Bash"].Allow) != 2 {
		t.Errorf("allow len = %d, want 2 (global + project)", len(merged.Rules["Bash"].Allow))
	}
}

func TestMerge_NilProject(t *testing.T) {
	global := &Config{Mode: "prod"}
	merged := Merge(global, nil)
	if merged.Mode != "prod" {
		t.Errorf("Mode = %q", merged.Mode)
	}
}

// グローバルと プロジェクトの deny は両方が適用される
func TestMerge_DenyIsAdditive(t *testing.T) {
	global := &Config{
		Rules: map[string]ToolRules{
			"Bash": {
				Allow: []Rule{{Regex: `^ls\b`}},
				Deny:  []Rule{{Regex: `\brm\s+-rf\b`}},
			},
		},
	}
	project := &Config{
		Rules: map[string]ToolRules{
			"Bash": {
				Deny: []Rule{{Regex: `\bgit\s+push\s+--force\b`}},
			},
		},
	}

	merged := Merge(global, project)

	if len(merged.Rules["Bash"].Deny) != 2 {
		t.Errorf("deny len = %d, want 2 (global + project)", len(merged.Rules["Bash"].Deny))
	}
	if len(merged.Rules["Bash"].Allow) != 1 {
		t.Errorf("allow len = %d, want 1 (global only)", len(merged.Rules["Bash"].Allow))
	}
}

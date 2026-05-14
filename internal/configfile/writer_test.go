package configfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInsert_AppendsToExistingAllow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cclmonitor.yaml")
	if err := os.WriteFile(path, []byte(`rules:
  Bash:
    allow:
      - regex: '^ls\b'
`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Insert(path, "Bash", "allow", "regex", `(^|[\s;&|])pnpm\s+install\b`); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := string(got)
	if !strings.Contains(out, `^ls\b`) {
		t.Errorf("existing rule lost: %s", out)
	}
	if !strings.Contains(out, `(^|[\s;&|])pnpm\s+install\b`) {
		t.Errorf("new rule not inserted: %s", out)
	}
}

func TestInsert_CreatesMissingSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cclmonitor.yaml")
	if err := os.WriteFile(path, []byte(`rules:
  Bash:
    allow:
      - regex: '^ls\b'
`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Insert(path, "Edit", "deny", "glob", "**/.env"); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(path)
	out := string(got)
	if !strings.Contains(out, "Edit:") {
		t.Errorf("Edit section missing: %s", out)
	}
	if !strings.Contains(out, "**/.env") {
		t.Errorf("glob not inserted: %s", out)
	}
}

func TestInsert_CreatesFileIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cclmonitor.yaml")

	if err := Insert(path, "Bash", "deny", "regex", `\bsudo\b`); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := string(got)
	if !strings.Contains(out, "rules:") {
		t.Errorf("rules root missing: %s", out)
	}
	if !strings.Contains(out, `\bsudo\b`) {
		t.Errorf("rule not present: %s", out)
	}
}

func TestInsert_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deeper", "cclmonitor.yaml")

	if err := Insert(path, "Bash", "deny", "regex", `\bsudo\b`); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestBackup_CreatesTimestampedCopy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cclmonitor.yaml")
	original := []byte("rules:\n  Bash:\n    allow: []\n")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}

	bakPath, err := Backup(path)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(bakPath, path+".bak-") {
		t.Errorf("bak path prefix wrong: %s", bakPath)
	}
	got, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Errorf("backup content mismatch: got %q want %q", got, original)
	}
}

func TestBackup_MissingSourceReturnsEmptyNoError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-such.yaml")

	bakPath, err := Backup(path)
	if err != nil {
		t.Errorf("expected no error for missing source, got %v", err)
	}
	if bakPath != "" {
		t.Errorf("expected empty path for missing source, got %q", bakPath)
	}
}

func TestInsert_GlobKindForGlobRule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cclmonitor.yaml")

	if err := Insert(path, "Read", "deny", "glob", "**/.env"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	out := string(got)
	if !strings.Contains(out, "glob:") {
		t.Errorf("glob key missing: %s", out)
	}
	if strings.Contains(out, "regex:") {
		t.Errorf("regex key should not appear for glob rule: %s", out)
	}
}

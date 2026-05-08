package notify

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendLog_WritesJSONL(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()
	event := Event{
		Time:      now,
		SessionID: "sess-1",
		ToolName:  "Bash",
		Value:     "rm -rf /",
		Verdict:   "deny",
	}

	if err := AppendLog(dir, event); err != nil {
		t.Fatalf("AppendLog error: %v", err)
	}

	path := filepath.Join(dir, "cclmonitor."+now.Format("2006-01-02")+".log")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var got Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, data)
	}
	if got.SessionID != "sess-1" {
		t.Errorf("SessionID = %q", got.SessionID)
	}
	if got.Verdict != "deny" {
		t.Errorf("Verdict = %q", got.Verdict)
	}
}

func TestAppendLog_AppendsMultipleLines(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()

	for _, verdict := range []string{"deny", "allow", "unknown"} {
		e := Event{
			Time:      now,
			SessionID: "sess",
			ToolName:  "Bash",
			Value:     "cmd",
			Verdict:   verdict,
		}
		if err := AppendLog(dir, e); err != nil {
			t.Fatal(err)
		}
	}

	path := filepath.Join(dir, "cclmonitor."+now.Format("2006-01-02")+".log")
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var lines []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatalf("invalid JSON line: %v", err)
		}
		lines = append(lines, e)
	}
	if len(lines) != 3 {
		t.Errorf("line count = %d, want 3", len(lines))
	}
	if lines[0].Verdict != "deny" || lines[1].Verdict != "allow" || lines[2].Verdict != "unknown" {
		t.Error("verdicts don't match expected order")
	}
}

func TestAppendLog_CreatesParentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir")
	now := time.Now().UTC()
	event := Event{Time: now, Verdict: "allow"}

	if err := AppendLog(dir, event); err != nil {
		t.Fatalf("AppendLog error: %v", err)
	}

	path := filepath.Join(dir, "cclmonitor."+now.Format("2006-01-02")+".log")
	if _, err := os.Stat(path); err != nil {
		t.Error("log file should exist")
	}
}

func TestCleanOldLogs_DeletesOldFiles(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	files := map[string]bool{
		"cclmonitor." + now.Format("2006-01-02") + ".log":                     true,  // 今日: 保持
		"cclmonitor." + now.AddDate(0, 0, -29).Format("2006-01-02") + ".log": true,  // 29日前: 保持
		"cclmonitor." + now.AddDate(0, 0, -30).Format("2006-01-02") + ".log": false, // 30日前: 削除
		"cclmonitor." + now.AddDate(0, 0, -31).Format("2006-01-02") + ".log": false, // 31日前: 削除
		"other.log": true, // 無関係なファイル: 触らない
	}
	for name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	CleanOldLogs(dir, 30)

	for name, shouldExist := range files {
		_, err := os.Stat(filepath.Join(dir, name))
		exists := err == nil
		if exists != shouldExist {
			if shouldExist {
				t.Errorf("%s should still exist", name)
			} else {
				t.Errorf("%s should have been deleted", name)
			}
		}
	}
}

func TestCleanOldLogs_DefaultsTo30Days(t *testing.T) {
	dir := t.TempDir()
	old := "cclmonitor." + time.Now().AddDate(0, 0, -31).Format("2006-01-02") + ".log"
	if err := os.WriteFile(filepath.Join(dir, old), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	CleanOldLogs(dir, 0) // 0 はデフォルト30日として扱う

	if _, err := os.Stat(filepath.Join(dir, old)); err == nil {
		t.Error("31-day-old file should be deleted with default 30-day retention")
	}
}

func TestOsascriptMessage(t *testing.T) {
	event := Event{
		ToolName: "Bash",
		Value:    "rm -rf /",
		Verdict:  "deny",
	}
	msg := osascriptMessage(event)
	if msg == "" {
		t.Error("message should not be empty")
	}
}

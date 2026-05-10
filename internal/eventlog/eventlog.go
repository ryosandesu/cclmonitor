package eventlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryosandesu/cclmonitor/internal/config"
)

type Event struct {
	Time      time.Time `json:"time"`
	SessionID string    `json:"session_id"`
	ToolUseID string    `json:"tool_use_id"`
	ToolName  string    `json:"tool_name"`
	Value     string    `json:"value"`
	Verdict   string    `json:"verdict"`
}

func Write(cfg config.EventLogConfig, event Event) error {
	dir := resolveDir(cfg.LogDir)
	if err := AppendLog(dir, event); err != nil {
		return err
	}
	go CleanOldLogs(dir, cfg.RetainDays)
	return nil
}

// AppendLog writes event as a JSON line to a date-based file under dir.
func AppendLog(dir string, event Event) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := logFilePath(dir, event.Time)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	line, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

// CleanOldLogs deletes cclmonitor.YYYY-MM-DD.log files older than retainDays.
// If retainDays <= 0, defaults to 30.
func CleanOldLogs(dir string, retainDays int) {
	if retainDays <= 0 {
		retainDays = 30
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cy, cm, cd := time.Now().AddDate(0, 0, -retainDays).Date()
	cutoff := time.Date(cy, cm, cd, 0, 0, 0, 0, time.Local)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "cclmonitor.") || !strings.HasSuffix(name, ".log") {
			continue
		}
		dateStr := name[len("cclmonitor.") : len(name)-len(".log")]
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		fy, fm, fd := t.Date()
		fileDate := time.Date(fy, fm, fd, 0, 0, 0, 0, time.Local)
		if !fileDate.After(cutoff) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}

func logFilePath(dir string, t time.Time) string {
	return filepath.Join(dir, "cclmonitor."+t.Format("2006-01-02")+".log")
}

func resolveDir(dir string) string {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "."
		}
		return filepath.Join(home, ".claude")
	}
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return dir
		}
		return filepath.Join(home, dir[2:])
	}
	return dir
}

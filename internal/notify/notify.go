package notify

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryosandesu/cclmonitor/internal/config"
)

type Event struct {
	Time      time.Time `json:"time"`
	SessionID string    `json:"session_id"`
	ToolName  string    `json:"tool_name"`
	Value     string    `json:"value"`
	Verdict   string    `json:"verdict"`
}

// Notify dispatches an event to each configured channel.
func Notify(cfg config.NotifyConfig, event Event) error {
	for _, ch := range cfg.Channels {
		switch ch {
		case "logfile":
			dir := resolveDir(cfg.LogDir)
			if err := AppendLog(dir, event); err != nil {
				return err
			}
			go CleanOldLogs(dir, cfg.RetainDays)
		case "osascript":
			if err := sendOsascript(event); err != nil {
				return err
			}
		}
	}
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
	cutoff := time.Now().AddDate(0, 0, -retainDays)
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
		if t.Before(cutoff) {
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

func osascriptMessage(event Event) string {
	return fmt.Sprintf("[%s] %s: %s", event.Verdict, event.ToolName, event.Value)
}

func sendOsascript(event Event) error {
	msg := osascriptMessage(event)
	script := fmt.Sprintf(
		`display notification %q with title "cclmonitor"`,
		msg,
	)
	return exec.Command("osascript", "-e", script).Start()
}

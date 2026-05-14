// Package claudelog parses Claude Code transcripts at
// ~/.claude/projects/<encoded-cwd>/*.jsonl and exposes tool_use events
// as eventlog.Event values. Used as a fallback data source by
// `cclmonitor suggest` when cclmonitor's own logs are unavailable.
package claudelog

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryosandesu/cclmonitor/internal/eventlog"
)

// transcriptLine is a partial schema of a single JSONL entry. Many fields
// (usage stats, parent IDs, etc.) are ignored — we only need timestamp and
// content blocks of type "tool_use".
type transcriptLine struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Message   struct {
		Content []toolUseBlock `json:"content"`
	} `json:"message"`
}

type toolUseBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ReadProjectTranscripts scans every *.jsonl file under dir and returns
// eventlog.Event values for each tool_use block within [from, to). Tools
// other than Bash/Edit/Write/Read are skipped. All events get verdict="unknown"
// because transcripts do not carry hook decisions.
//
// A non-existent dir is treated as empty (returns nil, nil).
func ReadProjectTranscripts(dir string, from, to time.Time) ([]eventlog.Event, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var events []eventlog.Event
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".jsonl") {
			continue
		}
		path := filepath.Join(dir, ent.Name())
		fileEvents, err := readFile(path, from, to)
		if err != nil {
			continue
		}
		events = append(events, fileEvents...)
	}
	return events, nil
}

// EncodeCwd converts an absolute path to the directory-name format used by
// Claude Code's transcript storage:
//
//	/Users/foo/Desktop/proj  →  -Users-foo-Desktop-proj
func EncodeCwd(cwd string) string {
	return strings.ReplaceAll(cwd, "/", "-")
}

func readFile(path string, from, to time.Time) ([]eventlog.Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []eventlog.Event
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for scanner.Scan() {
		var line transcriptLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type != "assistant" {
			continue
		}
		ts, err := time.Parse(time.RFC3339, line.Timestamp)
		if err != nil {
			continue
		}
		if ts.Before(from) || !ts.Before(to) {
			continue
		}
		for _, block := range line.Message.Content {
			if block.Type != "tool_use" {
				continue
			}
			value, ok := extractValue(block.Name, block.Input)
			if !ok {
				continue
			}
			events = append(events, eventlog.Event{
				Time:      ts,
				ToolUseID: block.ID,
				ToolName:  block.Name,
				Value:     value,
				Verdict:   "unknown",
			})
		}
	}
	return events, nil
}

func extractValue(tool string, input json.RawMessage) (string, bool) {
	switch tool {
	case "Bash":
		var in struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(input, &in); err != nil || in.Command == "" {
			return "", false
		}
		return in.Command, true
	case "Edit", "Write", "Read":
		var in struct {
			FilePath string `json:"file_path"`
		}
		if err := json.Unmarshal(input, &in); err != nil || in.FilePath == "" {
			return "", false
		}
		return in.FilePath, true
	default:
		return "", false
	}
}

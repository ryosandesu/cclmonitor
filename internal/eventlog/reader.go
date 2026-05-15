package eventlog

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// ReadRange returns Events from log files in dir where from <= event.Time < to.
// Missing files and corrupt JSON lines are silently skipped.
func ReadRange(dir string, from, to time.Time) ([]Event, error) {
	var events []Event
	for d := TruncateDay(from); d.Before(to); d = d.AddDate(0, 0, 1) {
		path := filepath.Join(dir, "cclmonitor."+d.Format("2006-01-02")+".log")
		f, err := os.Open(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var e Event
			if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
				continue
			}
			if (e.Time.Equal(from) || e.Time.After(from)) && e.Time.Before(to) {
				events = append(events, e)
			}
		}
		f.Close()
	}
	return events, nil
}

// Reader manages incremental reading of a single log file.
type Reader struct {
	path   string
	file   *os.File
	offset int64
}

// NewReader opens path and prepares for incremental Poll calls.
func NewReader(path string) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &Reader{path: path, file: f}, nil
}

// Poll returns events appended since the last call. Returns empty slice when nothing new.
func (r *Reader) Poll() ([]Event, error) {
	if _, err := r.file.Seek(r.offset, 0); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(r.file)
	var events []Event
	for scanner.Scan() {
		r.offset += int64(len(scanner.Bytes())) + 1 // +1 for newline
		var e Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		events = append(events, e)
	}
	return events, scanner.Err()
}

// Close releases the file handle.
func (r *Reader) Close() error {
	return r.file.Close()
}

func TruncateDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

package hookio_test

import (
	"strings"
	"testing"

	"github.com/ryosandesu/cclmonitor/internal/hookio"
)

func TestParse_Bash(t *testing.T) {
	input := `{"tool_name":"Bash","tool_input":{"command":"ls -la"},"cwd":"/home/user","session_id":"abc123"}`
	p, err := hookio.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ToolName != "Bash" {
		t.Errorf("want ToolName=Bash, got %q", p.ToolName)
	}
	if p.Cwd != "/home/user" {
		t.Errorf("want Cwd=/home/user, got %q", p.Cwd)
	}
	if p.SessionID != "abc123" {
		t.Errorf("want SessionID=abc123, got %q", p.SessionID)
	}
}

func TestMatchValue_Bash(t *testing.T) {
	input := `{"tool_name":"Bash","tool_input":{"command":"ls -la"},"cwd":"/home/user","session_id":"abc123"}`
	p, _ := hookio.Parse(strings.NewReader(input))
	v, err := hookio.MatchValue(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "ls -la" {
		t.Errorf("want 'ls -la', got %q", v)
	}
}

func TestMatchValue_Edit(t *testing.T) {
	input := `{"tool_name":"Edit","tool_input":{"file_path":"/tmp/foo.go","old_string":"a","new_string":"b"},"cwd":"/home/user","session_id":"abc123"}`
	p, _ := hookio.Parse(strings.NewReader(input))
	v, err := hookio.MatchValue(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "/tmp/foo.go" {
		t.Errorf("want '/tmp/foo.go', got %q", v)
	}
}

func TestMatchValue_Write(t *testing.T) {
	input := `{"tool_name":"Write","tool_input":{"file_path":"/tmp/bar.go","content":"hello"},"cwd":"/home/user","session_id":"abc123"}`
	p, _ := hookio.Parse(strings.NewReader(input))
	v, err := hookio.MatchValue(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "/tmp/bar.go" {
		t.Errorf("want '/tmp/bar.go', got %q", v)
	}
}

func TestMatchValue_Read(t *testing.T) {
	input := `{"tool_name":"Read","tool_input":{"file_path":"/tmp/baz.go"},"cwd":"/home/user","session_id":"abc123"}`
	p, _ := hookio.Parse(strings.NewReader(input))
	v, err := hookio.MatchValue(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "/tmp/baz.go" {
		t.Errorf("want '/tmp/baz.go', got %q", v)
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	_, err := hookio.Parse(strings.NewReader("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

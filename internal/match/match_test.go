package match

import (
	"testing"

	"github.com/ryosandesu/cclmonitor/internal/config"
)

func TestEvaluate_DenyByRegex(t *testing.T) {
	rules := config.ToolRules{
		Deny: []config.Rule{{Regex: `\brm\s+-rf\s+/`}},
	}
	verdict, err := Evaluate(rules, "rm -rf /")
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Deny {
		t.Errorf("verdict = %v, want Deny", verdict)
	}
}

func TestEvaluate_AllowByRegex(t *testing.T) {
	rules := config.ToolRules{
		Allow: []config.Rule{{Regex: `^(ls|pwd)\b`}},
	}
	verdict, err := Evaluate(rules, "ls -la")
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Allow {
		t.Errorf("verdict = %v, want Allow", verdict)
	}
}

func TestEvaluate_DenyByGlob(t *testing.T) {
	rules := config.ToolRules{
		Deny: []config.Rule{{Glob: "**/.env*"}},
	}
	verdict, err := Evaluate(rules, "/home/user/project/.env.local")
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Deny {
		t.Errorf("verdict = %v, want Deny", verdict)
	}
}

func TestEvaluate_AllowByGlob(t *testing.T) {
	rules := config.ToolRules{
		Allow: []config.Rule{{Glob: "/home/user/project/**/*.go"}},
	}
	verdict, err := Evaluate(rules, "/home/user/project/internal/foo/bar.go")
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Allow {
		t.Errorf("verdict = %v, want Allow", verdict)
	}
}

func TestEvaluate_Unknown(t *testing.T) {
	rules := config.ToolRules{
		Allow: []config.Rule{{Regex: `^ls\b`}},
	}
	verdict, err := Evaluate(rules, "git status")
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Unknown {
		t.Errorf("verdict = %v, want Unknown", verdict)
	}
}

func TestEvaluate_DenyBeforeAllow(t *testing.T) {
	// deny takes precedence when both deny and allow rules match
	rules := config.ToolRules{
		Allow: []config.Rule{{Regex: `^rm\b`}},
		Deny:  []config.Rule{{Regex: `\brm\s+-rf\s+/`}},
	}
	verdict, err := Evaluate(rules, "rm -rf /")
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Deny {
		t.Errorf("verdict = %v, want Deny", verdict)
	}
}

func TestEvaluate_NoRules(t *testing.T) {
	rules := config.ToolRules{}
	verdict, err := Evaluate(rules, "anything")
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Unknown {
		t.Errorf("verdict = %v, want Unknown", verdict)
	}
}

func TestEvaluate_InvalidRegex(t *testing.T) {
	rules := config.ToolRules{
		Deny: []config.Rule{{Regex: `[invalid`}},
	}
	_, err := Evaluate(rules, "something")
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

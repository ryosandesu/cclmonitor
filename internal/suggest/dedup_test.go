package suggest

import (
	"testing"

	"github.com/ryosandesu/cclmonitor/internal/config"
)

func TestIsDuplicate_RegexMatch(t *testing.T) {
	cfg := &config.Config{
		Rules: map[string]config.ToolRules{
			"Bash": {
				Allow: []config.Rule{{Regex: `(^|[\s;&|])pnpm\s+install\b`}},
			},
		},
	}
	s := Suggestion{Tool: "Bash", Section: "allow", Kind: "regex", Pattern: `(^|[\s;&|])pnpm\s+install\b`}
	if !IsDuplicate(cfg, s) {
		t.Error("expected duplicate, got false")
	}
}

func TestIsDuplicate_GlobMatch(t *testing.T) {
	cfg := &config.Config{
		Rules: map[string]config.ToolRules{
			"Edit": {
				Deny: []config.Rule{{Glob: "**/.env"}},
			},
		},
	}
	s := Suggestion{Tool: "Edit", Section: "deny", Kind: "glob", Pattern: "**/.env"}
	if !IsDuplicate(cfg, s) {
		t.Error("expected duplicate, got false")
	}
}

func TestIsDuplicate_NoMatch(t *testing.T) {
	cfg := &config.Config{
		Rules: map[string]config.ToolRules{
			"Bash": {
				Allow: []config.Rule{{Regex: `^ls\b`}},
			},
		},
	}
	s := Suggestion{Tool: "Bash", Section: "allow", Kind: "regex", Pattern: `(^|[\s;&|])pnpm\s+install\b`}
	if IsDuplicate(cfg, s) {
		t.Error("expected not duplicate, got true")
	}
}

func TestIsDuplicate_DifferentSection(t *testing.T) {
	cfg := &config.Config{
		Rules: map[string]config.ToolRules{
			"Bash": {
				Deny: []config.Rule{{Regex: `(^|[\s;&|])pnpm\s+install\b`}},
			},
		},
	}
	// Same regex but in deny; suggestion is for allow → not duplicate
	s := Suggestion{Tool: "Bash", Section: "allow", Kind: "regex", Pattern: `(^|[\s;&|])pnpm\s+install\b`}
	if IsDuplicate(cfg, s) {
		t.Error("expected not duplicate (different section), got true")
	}
}

func TestIsDuplicate_DifferentTool(t *testing.T) {
	cfg := &config.Config{
		Rules: map[string]config.ToolRules{
			"Edit": {
				Deny: []config.Rule{{Glob: "**/.env"}},
			},
		},
	}
	s := Suggestion{Tool: "Write", Section: "deny", Kind: "glob", Pattern: "**/.env"}
	if IsDuplicate(cfg, s) {
		t.Error("expected not duplicate (different tool), got true")
	}
}

func TestIsDuplicate_NilConfig(t *testing.T) {
	s := Suggestion{Tool: "Bash", Section: "allow", Kind: "regex", Pattern: `^ls\b`}
	if IsDuplicate(nil, s) {
		t.Error("nil config should never duplicate")
	}
}

func TestIsDuplicate_AlreadyDenied(t *testing.T) {
	// Pattern matches an existing deny — suggestion was generated from a denied event,
	// so this is the "already denied" case (informational, skip in UI).
	cfg := &config.Config{
		Rules: map[string]config.ToolRules{
			"Bash": {
				Deny: []config.Rule{{Regex: `\bnpm\s+install\b`}},
			},
		},
	}
	s := Suggestion{Tool: "Bash", Section: "deny", Kind: "regex", Pattern: `\bnpm\s+install\b`}
	if !IsDuplicate(cfg, s) {
		t.Error("expected duplicate, got false")
	}
}

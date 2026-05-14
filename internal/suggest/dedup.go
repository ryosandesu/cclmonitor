package suggest

import "github.com/ryosandesu/cclmonitor/internal/config"

// IsDuplicate reports whether the suggestion is already present in cfg
// as a string-identical rule under the same tool and section.
// Returns false for nil cfg.
func IsDuplicate(cfg *config.Config, s Suggestion) bool {
	if cfg == nil {
		return false
	}
	tr, ok := cfg.Rules[s.Tool]
	if !ok {
		return false
	}
	var rules []config.Rule
	switch s.Section {
	case "allow":
		rules = tr.Allow
	case "deny":
		rules = tr.Deny
	default:
		return false
	}
	for _, r := range rules {
		switch s.Kind {
		case "regex":
			if r.Regex == s.Pattern {
				return true
			}
		case "glob":
			if r.Glob == s.Pattern {
				return true
			}
		}
	}
	return false
}

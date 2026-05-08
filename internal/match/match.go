package match

import (
	"regexp"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/ryosandesu/cclmonitor/internal/config"
)

type Verdict int

const (
	Unknown Verdict = iota
	Allow
	Deny
)

// Evaluate checks value against deny rules first, then allow rules.
func Evaluate(rules config.ToolRules, value string) (Verdict, error) {
	matched, err := matchesAny(rules.Deny, value)
	if err != nil {
		return Unknown, err
	}
	if matched {
		return Deny, nil
	}

	matched, err = matchesAny(rules.Allow, value)
	if err != nil {
		return Unknown, err
	}
	if matched {
		return Allow, nil
	}

	return Unknown, nil
}

func matchesAny(rules []config.Rule, value string) (bool, error) {
	for _, r := range rules {
		if r.Regex != "" {
			re, err := regexp.Compile(r.Regex)
			if err != nil {
				return false, err
			}
			if re.MatchString(value) {
				return true, nil
			}
		}
		if r.Glob != "" {
			matched, err := doublestar.Match(r.Glob, value)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
	}
	return false, nil
}

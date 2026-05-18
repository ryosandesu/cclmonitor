package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	EventLog EventLogConfig       `yaml:"eventlog"`
	Rules    map[string]ToolRules `yaml:"rules"`
}

type EventLogConfig struct {
	LogDir     string `yaml:"logdir"`
	RetainDays int    `yaml:"retain_days"`
	GraceSec   int    `yaml:"grace_sec"`
}

type ToolRules struct {
	Allow []Rule `yaml:"allow"`
	Deny  []Rule `yaml:"deny"`
}

type Rule struct {
	Regex      string         `yaml:"regex"`
	Glob       string         `yaml:"glob"`
	CompiledRe *regexp.Regexp `yaml:"-"`
}

func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	for tool, rules := range cfg.Rules {
		allow, err := compileRules(rules.Allow)
		if err != nil {
			return nil, fmt.Errorf("tool %s allow: %w", tool, err)
		}
		deny, err := compileRules(rules.Deny)
		if err != nil {
			return nil, fmt.Errorf("tool %s deny: %w", tool, err)
		}
		cfg.Rules[tool] = ToolRules{Allow: allow, Deny: deny}
	}
	return &cfg, nil
}

func compileRules(rules []Rule) ([]Rule, error) {
	out := make([]Rule, len(rules))
	for i, r := range rules {
		out[i] = r
		if r.Regex != "" {
			re, err := regexp.Compile(r.Regex)
			if err != nil {
				return nil, fmt.Errorf("invalid regex %q: %w", r.Regex, err)
			}
			out[i].CompiledRe = re
		}
	}
	return out, nil
}

// ExpandCwd returns a new Config with <cwd> tokens in glob patterns replaced by cwd.
func ExpandCwd(cfg *Config, cwd string) *Config {
	out := &Config{
		EventLog: cfg.EventLog,
		Rules:    make(map[string]ToolRules, len(cfg.Rules)),
	}
	for tool, rules := range cfg.Rules {
		out.Rules[tool] = ToolRules{
			Allow: expandRules(rules.Allow, cwd),
			Deny:  expandRules(rules.Deny, cwd),
		}
	}
	return out
}

func expandRules(rules []Rule, cwd string) []Rule {
	result := make([]Rule, len(rules))
	for i, r := range rules {
		result[i] = Rule{
			Regex:      r.Regex,
			Glob:       strings.ReplaceAll(r.Glob, "<cwd>", cwd),
			CompiledRe: r.CompiledRe,
		}
	}
	return result
}

// Merge returns a new Config combining global and project settings.
// allow/deny are merged independently per tool:
//   - project explicitly sets a section (even empty []): project wins
//   - project omits a section (nil): global is inherited
func Merge(global, project *Config) *Config {
	if project == nil {
		return global
	}
	out := &Config{
		EventLog: global.EventLog,
		Rules:    make(map[string]ToolRules),
	}
	for tool, rules := range global.Rules {
		out.Rules[tool] = rules
	}
	if (project.EventLog != EventLogConfig{}) {
		out.EventLog = project.EventLog
	}
	for tool, projectRules := range project.Rules {
		base := out.Rules[tool]
		out.Rules[tool] = ToolRules{
			Allow: append(base.Allow, projectRules.Allow...),
			Deny:  append(base.Deny, projectRules.Deny...),
		}
	}
	return out
}

package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Mode   string              `yaml:"mode"`
	Notify NotifyConfig        `yaml:"notify"`
	Rules  map[string]ToolRules `yaml:"rules"`
}

type NotifyConfig struct {
	Channels       []string `yaml:"channels"`
	LogDir         string   `yaml:"logdir"`
	DBDir          string   `yaml:"dbdir"`
	DedupWindowSec int      `yaml:"dedup_window_sec"`
	RetainDays     int      `yaml:"retain_days"`
}

type ToolRules struct {
	Allow []Rule `yaml:"allow"`
	Deny  []Rule `yaml:"deny"`
}

type Rule struct {
	Regex string `yaml:"regex"`
	Glob  string `yaml:"glob"`
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
	return &cfg, nil
}

// ExpandCwd returns a new Config with <cwd> tokens in glob patterns replaced by cwd.
func ExpandCwd(cfg *Config, cwd string) *Config {
	out := &Config{
		Mode:   cfg.Mode,
		Notify: cfg.Notify,
		Rules:  make(map[string]ToolRules, len(cfg.Rules)),
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
			Regex: r.Regex,
			Glob:  strings.ReplaceAll(r.Glob, "<cwd>", cwd),
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
		Mode:   global.Mode,
		Notify: global.Notify,
		Rules:  make(map[string]ToolRules),
	}
	for tool, rules := range global.Rules {
		out.Rules[tool] = rules
	}
	if project.Mode != "" {
		out.Mode = project.Mode
	}
	if len(project.Notify.Channels) > 0 {
		out.Notify = project.Notify
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

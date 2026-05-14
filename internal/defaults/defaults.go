// Package defaults provides the built-in baseline security rules applied
// when a project has insufficient log data for cclmonitor suggest to
// generate meaningful suggestions.
//
// Defaults are deny-only and target patterns with near-zero false-positive
// rate across project types (secrets, system destruction, dangerous git).
package defaults

import "github.com/ryosandesu/cclmonitor/internal/config"

// Builtin returns the baseline Config. Rules are deny-only.
func Builtin() *config.Config {
	fileSecrets := []config.Rule{
		{Glob: "**/.env"},
		{Glob: "**/.env.local"},
		{Glob: "**/.env.test"},
		{Glob: "**/.env.production"},
		{Glob: "**/.env.development"},
		{Glob: "**/*.pem"},
		{Glob: "**/*.key"},
		{Glob: "**/id_rsa*"},
		{Glob: "**/*credentials*"},
		{Glob: "**/.ssh/**"},
	}
	fileSecretsRead := append([]config.Rule{}, fileSecrets...)
	fileSecretsRead = append(fileSecretsRead, config.Rule{Glob: "**/.gnupg/**"})

	return &config.Config{
		Rules: map[string]config.ToolRules{
			"Bash": {
				Deny: []config.Rule{
					{Regex: `\bsudo\b`},
					{Regex: `\brm\s+-rf?\s+(/|~|\$HOME)(\s|$)`},
					{Regex: `\bcurl\b.*\|\s*(ba)?sh\b`},
					{Regex: `\bgit\s+push\s+(--force\b|-f\b)`},
					{Regex: `\bgit\s+reset\s+--hard\b`},
					{Regex: `\bgit\s+clean\s+-[a-z]*f`},
				},
			},
			"Edit":  {Deny: fileSecrets},
			"Write": {Deny: fileSecrets},
			"Read":  {Deny: fileSecretsRead},
		},
	}
}

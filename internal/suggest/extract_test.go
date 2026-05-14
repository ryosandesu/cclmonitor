package suggest

import "testing"

func TestExtractBashKey(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
		ok    bool
	}{
		{"two tokens with flags", "pnpm install --frozen-lockfile", "pnpm install", true},
		{"single env var prefix", "FOO=bar pnpm install", "pnpm install", true},
		{"multiple env var prefixes", "FOO=bar BAZ=qux pnpm install", "pnpm install", true},
		{"two tokens simple", "docker ps", "docker ps", true},
		{"single token", "pnpm", "pnpm", true},
		{"empty string", "", "", false},
		{"whitespace only", "   ", "", false},
		{"env var only", "FOO=bar", "", false},
		{"node with script", "node script.js", "node script.js", true},
		{"command with leading whitespace", "  pnpm install", "pnpm install", true},
		{"npm install with arg", "npm install foo", "npm install", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractBashKey(tt.value)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("got = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBashKeyToRegex(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		section string
		want    string
	}{
		{"two-token allow", "pnpm install", "allow", `(^|[\s;&|])pnpm\s+install\b`},
		{"single-token allow", "pnpm", "allow", `(^|[\s;&|])pnpm\b`},
		{"two-token deny", "npm install", "deny", `\bnpm\s+install\b`},
		{"single-token deny", "sudo", "deny", `\bsudo\b`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BashKeyToRegex(tt.key, tt.section)
			if got != tt.want {
				t.Errorf("got = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractFileGlob(t *testing.T) {
	tests := []struct {
		name string
		cwd  string
		path string
		want string
		ok   bool
	}{
		{
			name: "tsx in src subdir",
			cwd:  "/Users/foo/proj",
			path: "/Users/foo/proj/src/components/Button.tsx",
			want: "<cwd>/src/**/*.tsx",
			ok:   true,
		},
		{
			name: "md in docs subdir",
			cwd:  "/Users/foo/proj",
			path: "/Users/foo/proj/docs/api/index.md",
			want: "<cwd>/docs/**/*.md",
			ok:   true,
		},
		{
			name: "yaml deeper",
			cwd:  "/Users/foo/proj",
			path: "/Users/foo/proj/k8s/local/01-postgres.yaml",
			want: "<cwd>/k8s/**/*.yaml",
			ok:   true,
		},
		{
			name: "file in cwd root with extension",
			cwd:  "/Users/foo/proj",
			path: "/Users/foo/proj/file.go",
			want: "<cwd>/**/*.go",
			ok:   true,
		},
		{
			name: "extensionless in subdir",
			cwd:  "/Users/foo/proj",
			path: "/Users/foo/proj/scripts/Makefile",
			want: "<cwd>/scripts/**/Makefile",
			ok:   true,
		},
		{
			name: "extensionless in root",
			cwd:  "/Users/foo/proj",
			path: "/Users/foo/proj/Makefile",
			want: "<cwd>/**/Makefile",
			ok:   true,
		},
		{
			name: "outside cwd",
			cwd:  "/Users/foo/proj",
			path: "/etc/hosts",
			want: "/etc/hosts",
			ok:   true,
		},
		{
			name: "empty path",
			cwd:  "/Users/foo/proj",
			path: "",
			want: "",
			ok:   false,
		},
		{
			name: "no cwd provided",
			cwd:  "",
			path: "/some/abs/path/x.go",
			want: "/some/abs/path/x.go",
			ok:   true,
		},
		{
			name: "cwd is exact path",
			cwd:  "/Users/foo/proj",
			path: "/Users/foo/proj",
			want: "",
			ok:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractFileGlob(tt.cwd, tt.path)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("got = %q, want %q", got, tt.want)
			}
		})
	}
}

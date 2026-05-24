package match

import "strings"

// SplitBashCommands splits a bash command string on shell metacharacters
// (;, |, &&, ||, &, newline, $(...), `...`) and returns the individual tokens.
// Metacharacters inside single or double quotes are not treated as delimiters.
// Each token is trimmed of leading/trailing whitespace; empty tokens are omitted.
func SplitBashCommands(cmd string) []string {
	var tokens []string
	var buf strings.Builder
	inSingle := false
	inDouble := false
	i := 0

	flush := func() {
		t := strings.TrimSpace(buf.String())
		if t != "" {
			tokens = append(tokens, t)
		}
		buf.Reset()
	}

	for i < len(cmd) {
		ch := cmd[i]

		// backslash escape: skip next character (only outside single quotes)
		if ch == '\\' && !inSingle {
			buf.WriteByte(ch)
			i++
			if i < len(cmd) {
				buf.WriteByte(cmd[i])
				i++
			}
			continue
		}

		// toggle single-quote mode
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			buf.WriteByte(ch)
			i++
			continue
		}

		// toggle double-quote mode
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			buf.WriteByte(ch)
			i++
			continue
		}

		// inside single quotes: no splitting at all
		if inSingle {
			buf.WriteByte(ch)
			i++
			continue
		}

		// $( ... ) — command substitution: split even inside double quotes
		// (bash expands $() inside double quotes, so it must be checked)
		if ch == '$' && i+1 < len(cmd) && cmd[i+1] == '(' {
			flush()
			i += 2 // skip '$('
			inner, advance := extractBalancedParen(cmd[i:])
			for _, t := range SplitBashCommands(inner) {
				tokens = append(tokens, t)
			}
			i += advance
			continue
		}

		// inside double quotes: no other splitting
		if inDouble {
			buf.WriteByte(ch)
			i++
			continue
		}

		// backtick substitution: split, recurse on inner content
		if ch == '`' {
			flush()
			i++ // skip opening backtick
			end := strings.IndexByte(cmd[i:], '`')
			if end < 0 {
				// unterminated backtick: treat rest as one token
				for _, t := range SplitBashCommands(cmd[i:]) {
					tokens = append(tokens, t)
				}
				i = len(cmd)
				continue
			}
			for _, t := range SplitBashCommands(cmd[i : i+end]) {
				tokens = append(tokens, t)
			}
			i += end + 1 // skip closing backtick
			continue
		}

		// || — logical OR
		if ch == '|' && i+1 < len(cmd) && cmd[i+1] == '|' {
			flush()
			i += 2
			continue
		}

		// && — logical AND
		if ch == '&' && i+1 < len(cmd) && cmd[i+1] == '&' {
			flush()
			i += 2
			continue
		}

		// | — pipe
		if ch == '|' {
			flush()
			i++
			continue
		}

		// ; — command separator
		if ch == ';' {
			flush()
			i++
			continue
		}

		// & — background execution
		if ch == '&' {
			flush()
			i++
			continue
		}

		// newline — command separator
		if ch == '\n' {
			flush()
			i++
			continue
		}

		buf.WriteByte(ch)
		i++
	}

	flush()
	return tokens
}

// extractBalancedParen reads characters until a matching ')' is found,
// tracking nested '(' and ')'. Returns the inner content and the number
// of bytes consumed (not including the closing ')').
func extractBalancedParen(s string) (inner string, advance int) {
	depth := 1
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[:i], i + 1
			}
		}
	}
	// no closing paren found: return entire string
	return s, len(s)
}

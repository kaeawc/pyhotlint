// Package suppress parses pyhotlint suppression pragmas from a Python
// source file and reports whether a finding for a given (rule, line)
// should be dropped.
//
// Pragma syntax:
//
//	# pyhotlint: ignore                 # all rules on this line
//	# pyhotlint: ignore[rule-id]        # one rule on this line
//	# pyhotlint: ignore[r1, r2]         # several rules on this line
//	# pyhotlint: ignore-file            # all rules in this file
//	# pyhotlint: ignore-file[rule-id]   # one rule across the file
//
// Whitespace around `:` and inside `[...]` is tolerated. The pragma may
// appear anywhere on a line — typically end-of-line for line scope, on
// its own line near the top for file scope, but the parser does not
// enforce position.
package suppress

import (
	"bufio"
	"bytes"
	"strings"
)

// allRules is the sentinel meaning "every rule" inside a suppression set.
const allRules = "*"

// Set holds the suppression state parsed from a single source file.
type Set struct {
	fileIgnored map[string]struct{}
	lineIgnored map[int]map[string]struct{}
}

// Parse extracts every suppression pragma from src.
func Parse(src []byte) *Set {
	s := &Set{
		fileIgnored: map[string]struct{}{},
		lineIgnored: map[int]map[string]struct{}{},
	}
	scanner := bufio.NewScanner(bytes.NewReader(src))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		s.parseLine(line, scanner.Text())
	}
	return s
}

// IsSuppressed reports whether a finding for ruleID at lineNumber
// (1-based) should be dropped.
func (s *Set) IsSuppressed(ruleID string, lineNumber int) bool {
	if s == nil {
		return false
	}
	if _, all := s.fileIgnored[allRules]; all {
		return true
	}
	if _, ok := s.fileIgnored[ruleID]; ok {
		return true
	}
	if entry, ok := s.lineIgnored[lineNumber]; ok {
		if _, all := entry[allRules]; all {
			return true
		}
		if _, ok := entry[ruleID]; ok {
			return true
		}
	}
	return false
}

func (s *Set) parseLine(line int, text string) {
	body, ok := pragmaBody(text)
	if !ok {
		return
	}
	switch {
	case strings.HasPrefix(body, "ignore-file"):
		s.recordFile(parseRuleList(body[len("ignore-file"):]))
	case strings.HasPrefix(body, "ignore"):
		s.recordLine(line, parseRuleList(body[len("ignore"):]))
	}
}

// pragmaBody returns the substring after `# pyhotlint:` (whitespace
// tolerated), or false if the line has no pyhotlint pragma.
func pragmaBody(text string) (string, bool) {
	idx := strings.Index(text, "#")
	if idx < 0 {
		return "", false
	}
	rest := strings.TrimSpace(text[idx+1:])
	if !strings.HasPrefix(rest, "pyhotlint") {
		return "", false
	}
	rest = strings.TrimSpace(rest[len("pyhotlint"):])
	if !strings.HasPrefix(rest, ":") {
		return "", false
	}
	return strings.TrimSpace(rest[1:]), true
}

// parseRuleList extracts the rule IDs inside `[r1, r2]`. Returns nil
// when no bracket follows (callers treat nil as "all rules").
func parseRuleList(s string) []string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "[") {
		return nil
	}
	end := strings.Index(s, "]")
	if end < 0 {
		return nil
	}
	var out []string
	for _, r := range strings.Split(s[1:end], ",") {
		if r = strings.TrimSpace(r); r != "" {
			out = append(out, r)
		}
	}
	return out
}

func (s *Set) recordFile(rules []string) {
	if len(rules) == 0 {
		s.fileIgnored[allRules] = struct{}{}
		return
	}
	for _, r := range rules {
		s.fileIgnored[r] = struct{}{}
	}
}

func (s *Set) recordLine(line int, rules []string) {
	entry, ok := s.lineIgnored[line]
	if !ok {
		entry = map[string]struct{}{}
		s.lineIgnored[line] = entry
	}
	if len(rules) == 0 {
		entry[allRules] = struct{}{}
		return
	}
	for _, r := range rules {
		entry[r] = struct{}{}
	}
}

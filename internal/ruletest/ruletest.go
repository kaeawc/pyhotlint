// Package ruletest is shared scaffolding for rule unit tests: fixture
// path resolution, parsing, single-rule dispatch, and the
// positive/negative bucket walker.
package ruletest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kaeawc/pyhotlint/internal/rules/v2"
	"github.com/kaeawc/pyhotlint/internal/scanner"
)

// FixtureRoot resolves tests/fixtures/<ruleID>/<bucket>/ relative to the
// test's working directory by walking up until it finds a tests/
// directory.
func FixtureRoot(t *testing.T, ruleID, bucket string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "tests", "fixtures", ruleID, bucket)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate tests/fixtures/%s/%s from %s", ruleID, bucket, wd)
	return ""
}

// RunRule parses path and runs only the rule with the given ID.
func RunRule(t *testing.T, ruleID, path string) []v2.Finding {
	t.Helper()
	pf, err := scanner.ParseFile(path)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	defer pf.Close()
	var rules []*v2.Rule
	for _, r := range v2.All() {
		if r.ID == ruleID {
			rules = append(rules, r)
		}
	}
	if len(rules) == 0 {
		t.Fatalf("rule %q not registered", ruleID)
	}
	return v2.Run(rules, pf.Path, pf.Source, pf.Tree.RootNode())
}

// WalkPositives runs the rule against every file in the positive bucket
// and asserts at least one finding per file.
func WalkPositives(t *testing.T, ruleID string) {
	t.Helper()
	walkBucket(t, ruleID, FixtureRoot(t, ruleID, "positive"), true)
}

// WalkNegatives runs the rule against every file in the negative bucket
// and asserts no findings.
func WalkNegatives(t *testing.T, ruleID string) {
	t.Helper()
	walkBucket(t, ruleID, FixtureRoot(t, ruleID, "negative"), false)
}

func walkBucket(t *testing.T, ruleID, root string, expectFindings bool) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no fixtures in %s", root)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			findings := RunRule(t, ruleID, filepath.Join(root, name))
			if expectFindings && len(findings) == 0 {
				t.Fatalf("expected findings in %s, got 0", name)
			}
			if !expectFindings && len(findings) != 0 {
				for _, f := range findings {
					t.Logf("unexpected finding: %s:%d:%d %s", f.File, f.Line, f.Col, f.Message)
				}
				t.Fatalf("expected no findings in %s, got %d", name, len(findings))
			}
		})
	}
}

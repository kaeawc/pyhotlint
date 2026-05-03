package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
)

// fakeRules returns a synthetic registry for Apply / WarnUnknownRules
// tests so we don't depend on the real rule set.
func fakeRules() []*v2.Rule {
	return []*v2.Rule{
		{ID: "rule-a", Severity: v2.SeverityWarning, Category: "cat"},
		{ID: "rule-b", Severity: v2.SeverityError, Category: "cat"},
		{ID: "rule-c", Severity: v2.SeverityInfo, Category: "cat"},
	}
}

func writeConfig(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

func TestLoad_Empty(t *testing.T) {
	p := writeConfig(t, "pyhotlint.yml", "")
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Rules) != 0 {
		t.Fatalf("expected no rules, got %v", cfg.Rules)
	}
}

func TestLoad_DisableAndOverride(t *testing.T) {
	p := writeConfig(t, "pyhotlint.yml", `rules:
  rule-a:
    enabled: false
  rule-b:
    severity: info
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Rules["rule-a"].Enabled == nil || *cfg.Rules["rule-a"].Enabled {
		t.Fatalf("rule-a enabled should be false, got %#v", cfg.Rules["rule-a"].Enabled)
	}
	if cfg.Rules["rule-b"].Severity != "info" {
		t.Fatalf("rule-b severity should be info, got %q", cfg.Rules["rule-b"].Severity)
	}
}

func TestLoad_InvalidSeverity(t *testing.T) {
	p := writeConfig(t, "pyhotlint.yml", `rules:
  rule-a:
    severity: critical
`)
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for invalid severity")
	}
}

func TestLoad_NotFound(t *testing.T) {
	if _, err := Load("/nonexistent/path/pyhotlint.yml"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestApply_PassThroughWhenNoConfig(t *testing.T) {
	rules := fakeRules()
	got := (*Config)(nil).Apply(rules)
	if len(got) != len(rules) {
		t.Fatalf("nil Apply should pass through all %d rules, got %d", len(rules), len(got))
	}
}

func TestApply_DisablesRule(t *testing.T) {
	disabled := false
	cfg := &Config{Rules: map[string]RuleConfig{"rule-b": {Enabled: &disabled}}}
	got := cfg.Apply(fakeRules())
	if len(got) != 2 {
		t.Fatalf("expected 2 rules after disabling one, got %d", len(got))
	}
	for _, r := range got {
		if r.ID == "rule-b" {
			t.Fatal("rule-b should have been removed")
		}
	}
}

func TestApply_OverridesSeverity(t *testing.T) {
	cfg := &Config{Rules: map[string]RuleConfig{"rule-a": {Severity: "error"}}}
	got := cfg.Apply(fakeRules())
	if len(got) != 3 {
		t.Fatalf("expected all 3 rules to survive, got %d", len(got))
	}
	for _, r := range got {
		if r.ID != "rule-a" {
			continue
		}
		if r.Severity != v2.SeverityError {
			t.Fatalf("rule-a severity should be error, got %q", r.Severity)
		}
	}
	// The original rule slice must NOT be mutated (we shallow-clone).
	for _, r := range fakeRules() {
		if r.ID == "rule-a" && r.Severity != v2.SeverityWarning {
			t.Fatal("Apply mutated the input rule slice")
		}
	}
}

func TestFind_DiscoversInDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pyhotlint.yml"), []byte(""), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, path, err := Find(dir)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config")
	}
	if filepath.Base(path) != "pyhotlint.yml" {
		t.Fatalf("expected pyhotlint.yml, got %s", path)
	}
}

func TestFind_DiscoversInParent(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".pyhotlint.yml"), []byte(""), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	child := filepath.Join(root, "src", "deep")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfg, path, err := Find(child)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config from parent dir")
	}
	if !strings.HasSuffix(path, ".pyhotlint.yml") {
		t.Fatalf("expected .pyhotlint.yml, got %s", path)
	}
}

func TestFind_NoneFound(t *testing.T) {
	dir := t.TempDir()
	cfg, path, err := Find(dir)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if cfg != nil || path != "" {
		t.Fatalf("expected nil/empty, got %v %q", cfg, path)
	}
}

func TestWarnUnknownRules(t *testing.T) {
	cfg := &Config{Rules: map[string]RuleConfig{
		"rule-a":    {},
		"misspeled": {},
	}}
	var buf bytes.Buffer
	cfg.WarnUnknownRules(&buf, fakeRules())
	got := buf.String()
	if !strings.Contains(got, "misspeled") {
		t.Fatalf("expected warning about misspeled, got %q", got)
	}
	if strings.Contains(got, "rule-a") {
		t.Fatalf("known rule-a should not be warned, got %q", got)
	}
}

package output

import (
	"bytes"
	"strings"
	"testing"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
)

func sampleListRules() []*v2.Rule {
	return []*v2.Rule{
		{ID: "z-rule", Severity: v2.SeverityWarning, Category: "alpha", Description: "z thing"},
		{ID: "a-rule", Severity: v2.SeverityError, Category: "alpha", Description: "a thing"},
		{ID: "m-rule", Severity: v2.SeverityInfo, Category: "beta", Description: "m thing"},
	}
}

func TestWriteRuleList_HeaderAndOrder(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteRuleList(&buf, sampleListRules()); err != nil {
		t.Fatalf("WriteRuleList: %v", err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected header + separator + rows, got %d lines: %q", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "ID") || !strings.Contains(lines[0], "SEVERITY") {
		t.Errorf("header missing expected columns: %q", lines[0])
	}
	// Body lines (skip header + separator)
	body := lines[2:]
	wantOrder := []string{"a-rule", "z-rule", "m-rule"} // alpha-a, alpha-z, beta-m
	if len(body) != len(wantOrder) {
		t.Fatalf("expected %d rule rows, got %d", len(wantOrder), len(body))
	}
	for i, want := range wantOrder {
		if !strings.HasPrefix(body[i], want) {
			t.Errorf("row %d: got %q, want prefix %q", i, body[i], want)
		}
	}
}

func TestWriteRuleList_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteRuleList(&buf, nil); err != nil {
		t.Fatalf("WriteRuleList: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("empty rule list should produce header + separator only, got %d lines: %q", len(lines), buf.String())
	}
	if !strings.HasPrefix(lines[0], "ID") {
		t.Errorf("expected header, got %q", lines[0])
	}
}

func TestWriteRuleList_DoesNotMutateInput(t *testing.T) {
	rules := sampleListRules()
	originalFirst := rules[0].ID
	var buf bytes.Buffer
	if err := WriteRuleList(&buf, rules); err != nil {
		t.Fatalf("WriteRuleList: %v", err)
	}
	if rules[0].ID != originalFirst {
		t.Fatalf("WriteRuleList mutated caller's slice order: first ID is now %q", rules[0].ID)
	}
}

func TestWriteRuleList_DescriptionUnclipped(t *testing.T) {
	long := strings.Repeat("description ", 20)
	rules := []*v2.Rule{
		{ID: "a", Severity: v2.SeverityWarning, Category: "x", Description: long},
	}
	var buf bytes.Buffer
	if err := WriteRuleList(&buf, rules); err != nil {
		t.Fatalf("WriteRuleList: %v", err)
	}
	if !strings.Contains(buf.String(), long) {
		t.Fatal("long description must not be clipped")
	}
}

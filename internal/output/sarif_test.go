package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
)

func sampleRules() []*v2.Rule {
	return []*v2.Rule{
		{ID: "rule-a", Severity: v2.SeverityError, Description: "rule a desc"},
		{ID: "rule-b", Severity: v2.SeverityWarning, Description: "rule b desc"},
		{ID: "rule-c", Severity: v2.SeverityInfo, Description: "rule c desc"},
	}
}

func decodeSARIF(t *testing.T, raw []byte) sarifLog {
	t.Helper()
	var log sarifLog
	if err := json.Unmarshal(raw, &log); err != nil {
		t.Fatalf("decode: %v\nraw: %s", err, raw)
	}
	return log
}

func TestWriteSARIF_EmptyEnvelope(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, nil, sampleRules(), "1.2.3"); err != nil {
		t.Fatalf("WriteSARIF: %v", err)
	}
	got := decodeSARIF(t, buf.Bytes())
	if got.Version != "2.1.0" {
		t.Fatalf("version: %q", got.Version)
	}
	if got.Schema != sarifSchema {
		t.Fatalf("schema: %q", got.Schema)
	}
	if len(got.Runs) != 1 {
		t.Fatalf("runs: %d", len(got.Runs))
	}
	if got.Runs[0].Tool.Driver.Name != driverName {
		t.Fatalf("driver name: %q", got.Runs[0].Tool.Driver.Name)
	}
	if got.Runs[0].Tool.Driver.Version != "1.2.3" {
		t.Fatalf("driver version: %q", got.Runs[0].Tool.Driver.Version)
	}
	if len(got.Runs[0].Tool.Driver.Rules) != 3 {
		t.Fatalf("rules: %d", len(got.Runs[0].Tool.Driver.Rules))
	}
	if len(got.Runs[0].Results) != 0 {
		t.Fatalf("expected zero results, got %d", len(got.Runs[0].Results))
	}
}

func TestWriteSARIF_SeverityMapping(t *testing.T) {
	findings := []v2.Finding{
		{Rule: "rule-a", Severity: v2.SeverityError, File: "x.py", Line: 1, Col: 1, EndLine: 1, EndCol: 2, Message: "error"},
		{Rule: "rule-b", Severity: v2.SeverityWarning, File: "x.py", Line: 2, Col: 1, EndLine: 2, EndCol: 2, Message: "warning"},
		{Rule: "rule-c", Severity: v2.SeverityInfo, File: "x.py", Line: 3, Col: 1, EndLine: 3, EndCol: 2, Message: "info"},
	}
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, findings, sampleRules(), ""); err != nil {
		t.Fatalf("WriteSARIF: %v", err)
	}
	got := decodeSARIF(t, buf.Bytes())
	wantLevels := []string{"error", "warning", "note"}
	for i, want := range wantLevels {
		if got.Runs[0].Results[i].Level != want {
			t.Errorf("result %d level: got %q, want %q", i, got.Runs[0].Results[i].Level, want)
		}
	}
}

func TestWriteSARIF_RuleIndexAlignment(t *testing.T) {
	findings := []v2.Finding{
		{Rule: "rule-c", Severity: v2.SeverityInfo, File: "x.py", Line: 1},
		{Rule: "rule-a", Severity: v2.SeverityError, File: "x.py", Line: 2},
	}
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, findings, sampleRules(), ""); err != nil {
		t.Fatalf("WriteSARIF: %v", err)
	}
	got := decodeSARIF(t, buf.Bytes())
	rules := got.Runs[0].Tool.Driver.Rules
	for _, r := range got.Runs[0].Results {
		if rules[r.RuleIndex].ID != r.RuleID {
			t.Errorf("ruleIndex %d -> %q does not match ruleId %q", r.RuleIndex, rules[r.RuleIndex].ID, r.RuleID)
		}
	}
}

func TestWriteSARIF_PathsUseForwardSlashes(t *testing.T) {
	findings := []v2.Finding{
		{Rule: "rule-a", Severity: v2.SeverityError, File: "src\\pkg\\mod.py", Line: 1, Col: 1, EndLine: 1, EndCol: 2, Message: "x"},
	}
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, findings, sampleRules(), ""); err != nil {
		t.Fatalf("WriteSARIF: %v", err)
	}
	got := decodeSARIF(t, buf.Bytes())
	uri := got.Runs[0].Results[0].Locations[0].PhysicalLocation.ArtifactLocation.URI
	if strings.Contains(uri, "\\") {
		t.Fatalf("URI must use forward slashes, got %q", uri)
	}
}

func TestWriteSARIF_DefaultDriverVersionIsDev(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, nil, sampleRules(), ""); err != nil {
		t.Fatalf("WriteSARIF: %v", err)
	}
	got := decodeSARIF(t, buf.Bytes())
	if got.Runs[0].Tool.Driver.Version != "dev" {
		t.Fatalf("expected default version 'dev', got %q", got.Runs[0].Tool.Driver.Version)
	}
}

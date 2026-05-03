package output

import (
	"encoding/json"
	"io"
	"strings"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
)

// SARIF v2.1.0. Spec:
// https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html

const (
	sarifSchema  = "https://json.schemastore.org/sarif-2.1.0.json"
	sarifVersion = "2.1.0"
	driverName   = "pyhotlint"
	driverURI    = "https://github.com/kaeawc/pyhotlint"
)

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID                   string      `json:"id"`
	Name                 string      `json:"name"`
	ShortDescription     sarifText   `json:"shortDescription"`
	FullDescription      sarifText   `json:"fullDescription"`
	DefaultConfiguration sarifConfig `json:"defaultConfiguration"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifConfig struct {
	Level string `json:"level"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	RuleIndex int             `json:"ruleIndex"`
	Level     string          `json:"level"`
	Message   sarifText       `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
	EndLine     int `json:"endLine"`
	EndColumn   int `json:"endColumn"`
}

// WriteSARIF emits findings as a SARIF v2.1.0 log. The rules slice
// becomes runs[0].tool.driver.rules; each finding's ruleIndex points
// into that slice. driverVersion is stamped into tool.driver.version
// — pass main.version; "dev" is used when empty.
func WriteSARIF(w io.Writer, findings []v2.Finding, rules []*v2.Rule, driverVersion string) error {
	if driverVersion == "" {
		driverVersion = "dev"
	}
	driver := sarifDriver{
		Name:           driverName,
		Version:        driverVersion,
		InformationURI: driverURI,
		Rules:          make([]sarifRule, 0, len(rules)),
	}
	idx := make(map[string]int, len(rules))
	for i, r := range rules {
		idx[r.ID] = i
		driver.Rules = append(driver.Rules, sarifRule{
			ID:                   r.ID,
			Name:                 r.ID,
			ShortDescription:     sarifText{Text: r.Description},
			FullDescription:      sarifText{Text: r.Description},
			DefaultConfiguration: sarifConfig{Level: severityToSARIFLevel(r.Severity)},
		})
	}

	results := make([]sarifResult, 0, len(findings))
	for _, f := range findings {
		results = append(results, sarifResult{
			RuleID:    f.Rule,
			RuleIndex: idx[f.Rule], // 0 when unknown — SARIF requires a non-negative int
			Level:     severityToSARIFLevel(f.Severity),
			Message:   sarifText{Text: f.Message},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: strings.ReplaceAll(f.File, "\\", "/")},
					Region: sarifRegion{
						StartLine:   f.Line,
						StartColumn: f.Col,
						EndLine:     f.EndLine,
						EndColumn:   f.EndCol,
					},
				},
			}},
		})
	}

	log := sarifLog{
		Schema:  sarifSchema,
		Version: sarifVersion,
		Runs: []sarifRun{{
			Tool:    sarifTool{Driver: driver},
			Results: results,
		}},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(log)
}

func severityToSARIFLevel(s v2.Severity) string {
	switch s {
	case v2.SeverityError:
		return "error"
	case v2.SeverityInfo:
		return "note"
	default:
		return "warning"
	}
}

// Package config loads pyhotlint.yml / .pyhotlint.yml from a project
// root and applies its overrides to the rule set: per-rule enable
// toggles and severity changes. Unknown rule IDs in the config are
// surfaced as warnings to the caller, never silent.
package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
)

// configFilenames is the search list for auto-discovery, in priority
// order. The first match wins.
var configFilenames = []string{
	"pyhotlint.yml",
	".pyhotlint.yml",
	"pyhotlint.yaml",
	".pyhotlint.yaml",
}

// Config is the parsed shape of a pyhotlint config file.
//
//	rules:
//	  sync-io-in-async-fn:
//	    enabled: false           # disable
//	  pickle-load-from-untrusted-path:
//	    severity: warning        # downgrade from error
type Config struct {
	Rules map[string]RuleConfig `yaml:"rules"`
}

// RuleConfig is the per-rule override block. A nil Enabled means
// "leave the default"; an empty Severity means "leave the default".
type RuleConfig struct {
	Enabled  *bool  `yaml:"enabled,omitempty"`
	Severity string `yaml:"severity,omitempty"`
}

// Find walks up from startDir looking for the first config filename in
// the search list. Returns (nil, "", nil) when no config exists; that
// is not an error.
func Find(startDir string) (*Config, string, error) {
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return nil, "", err
	}
	for {
		for _, name := range configFilenames {
			p := filepath.Join(abs, name)
			info, err := os.Stat(p)
			if err == nil && !info.IsDir() {
				cfg, err := Load(p)
				return cfg, p, err
			}
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return nil, "", nil
		}
		abs = parent
	}
}

// Load parses a single config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("validate %s: %w", path, err)
	}
	return &c, nil
}

func (c *Config) validate() error {
	for id, rc := range c.Rules {
		if rc.Severity == "" {
			continue
		}
		switch v2.Severity(rc.Severity) {
		case v2.SeverityError, v2.SeverityWarning, v2.SeverityInfo:
		default:
			return fmt.Errorf("rule %q: unknown severity %q (want error|warning|info)", id, rc.Severity)
		}
	}
	return nil
}

// Apply returns a new rule slice with the config's overrides folded in.
// Rules with `enabled: false` are dropped; rules with a `severity`
// override are shallow-cloned with the new severity. Rules not
// mentioned in the config pass through unchanged.
func (c *Config) Apply(rules []*v2.Rule) []*v2.Rule {
	if c == nil || len(c.Rules) == 0 {
		return rules
	}
	out := make([]*v2.Rule, 0, len(rules))
	for _, r := range rules {
		rc, ok := c.Rules[r.ID]
		if !ok {
			out = append(out, r)
			continue
		}
		if rc.Enabled != nil && !*rc.Enabled {
			continue
		}
		if rc.Severity == "" {
			out = append(out, r)
			continue
		}
		clone := *r
		clone.Severity = v2.Severity(rc.Severity)
		out = append(out, &clone)
	}
	return out
}

// WarnUnknownRules writes one warning to w for every rule ID in c that
// is not present in registered. Used to surface typos in config files.
func (c *Config) WarnUnknownRules(w io.Writer, registered []*v2.Rule) {
	if c == nil {
		return
	}
	known := make(map[string]struct{}, len(registered))
	for _, r := range registered {
		known[r.ID] = struct{}{}
	}
	for id := range c.Rules {
		if _, ok := known[id]; !ok {
			fmt.Fprintf(w, "pyhotlint: warning: unknown rule %q in config\n", id)
		}
	}
}

// Package project reads pyproject.toml and uv.lock to surface the
// facts version-drift rules need: the project's requested Python
// version, and a flattened map of resolved dependency versions.
//
// uv.lock is preferred when present because it carries the resolved
// versions; pyproject.toml dependencies fall back to PEP 508 specs
// (e.g. "torch>=2.0") which are not exact pins.
//
// Poetry and bare requirements.txt are not yet supported — the README
// MVP scope lists uv as the priority and other resolvers as follow-up.
package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Source identifies which file the dependency map came from.
type Source string

const (
	SourceNone      Source = "none"
	SourcePyProject Source = "pyproject"
	SourceUvLock    Source = "uv.lock"
)

// Project describes the relevant facts about a Python project root.
type Project struct {
	// Root is the absolute path of the directory containing pyproject.toml.
	Root string
	// PythonVersion is the requires-python spec (e.g. ">=3.10"); empty
	// when neither pyproject.toml nor uv.lock declares one.
	PythonVersion string
	// Dependencies maps PEP-503 normalized package name to version
	// spec (from pyproject) or resolved version (from uv.lock).
	Dependencies map[string]string
	// Source records which file the version data ultimately came from.
	Source Source
}

// Load walks up from start looking for a pyproject.toml. When found,
// returns a Project populated from it (and uv.lock if it sits beside).
// When no pyproject.toml is found anywhere up the tree, returns
// (nil, nil) — pyhotlint can run without project context.
func Load(start string) (*Project, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	dir := abs
	for {
		pp := filepath.Join(dir, "pyproject.toml")
		info, statErr := os.Stat(pp)
		if statErr == nil && !info.IsDir() {
			return loadFrom(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, nil
		}
		dir = parent
	}
}

func loadFrom(dir string) (*Project, error) {
	p := &Project{
		Root:         dir,
		Dependencies: map[string]string{},
		Source:       SourceNone,
	}
	if err := readPyProject(filepath.Join(dir, "pyproject.toml"), p); err != nil {
		return nil, fmt.Errorf("pyproject.toml: %w", err)
	}
	uvPath := filepath.Join(dir, "uv.lock")
	if info, err := os.Stat(uvPath); err == nil && !info.IsDir() {
		if err := readUvLock(uvPath, p); err != nil {
			return nil, fmt.Errorf("uv.lock: %w", err)
		}
		p.Source = SourceUvLock
	}
	return p, nil
}

// VersionOf returns the recorded version (or spec) for a package, or ""
// when the package is not in the dependency map. The query is
// PEP-503 normalized before lookup.
func (p *Project) VersionOf(pkg string) string {
	if p == nil {
		return ""
	}
	return p.Dependencies[normalizeName(pkg)]
}

type pyprojectFile struct {
	Project struct {
		RequiresPython string   `toml:"requires-python"`
		Dependencies   []string `toml:"dependencies"`
	} `toml:"project"`
}

func readPyProject(path string, p *Project) error {
	var doc pyprojectFile
	if _, err := toml.DecodeFile(path, &doc); err != nil {
		return err
	}
	p.PythonVersion = doc.Project.RequiresPython
	if len(doc.Project.Dependencies) == 0 {
		return nil
	}
	for _, d := range doc.Project.Dependencies {
		name, ver := parseDepSpec(d)
		if name == "" {
			continue
		}
		if _, exists := p.Dependencies[name]; !exists {
			p.Dependencies[name] = ver
		}
	}
	if p.Source == SourceNone {
		p.Source = SourcePyProject
	}
	return nil
}

type uvLockFile struct {
	RequiresPython string  `toml:"requires-python"`
	Package        []uvPkg `toml:"package"`
}

type uvPkg struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

func readUvLock(path string, p *Project) error {
	var doc uvLockFile
	if _, err := toml.DecodeFile(path, &doc); err != nil {
		return err
	}
	if doc.RequiresPython != "" && p.PythonVersion == "" {
		p.PythonVersion = doc.RequiresPython
	}
	for _, pkg := range doc.Package {
		name := normalizeName(pkg.Name)
		if name == "" {
			continue
		}
		// uv.lock is the resolved-version source of truth.
		p.Dependencies[name] = pkg.Version
	}
	return nil
}

// parseDepSpec extracts (name, version-spec) from a PEP 508 dep string
// such as "torch>=2.0", "transformers[serialization]==4.30; python_version >= '3.10'".
// Best-effort: environment markers and extras are stripped, leaving the
// bare name and version constraint.
func parseDepSpec(s string) (name, version string) {
	if idx := strings.Index(s, ";"); idx >= 0 {
		s = s[:idx]
	}
	if openIdx := strings.Index(s, "["); openIdx >= 0 {
		if closeIdx := strings.Index(s, "]"); closeIdx > openIdx {
			s = s[:openIdx] + s[closeIdx+1:]
		}
	}
	s = strings.TrimSpace(s)
	end := 0
	for end < len(s) && isNameByte(s[end]) {
		end++
	}
	if end == 0 {
		return "", ""
	}
	return normalizeName(s[:end]), strings.TrimSpace(s[end:])
}

func isNameByte(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z':
		return true
	case c >= 'A' && c <= 'Z':
		return true
	case c >= '0' && c <= '9':
		return true
	case c == '_' || c == '-' || c == '.':
		return true
	}
	return false
}

// normalizeName implements PEP 503: lowercase, runs of [_.-] collapsed
// to a single '-', leading and trailing '-' trimmed.
func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	b.Grow(len(s))
	prevDash := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' || c == '.' || c == '-' {
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
			continue
		}
		b.WriteByte(c)
		prevDash = false
	}
	return strings.TrimSuffix(b.String(), "-")
}

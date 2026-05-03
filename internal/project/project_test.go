package project

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return root
}

func TestLoad_NoPyProject(t *testing.T) {
	root := t.TempDir()
	p, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p != nil {
		t.Fatalf("expected nil project, got %#v", p)
	}
}

func TestLoad_PyProjectOnly(t *testing.T) {
	root := writeFiles(t, map[string]string{
		"pyproject.toml": `
[project]
name = "demo"
requires-python = ">=3.10"
dependencies = [
    "torch>=2.0",
    "transformers==4.30.0",
    "Pillow",
    "python-dateutil[serialization]>=2.8; python_version >= '3.10'",
]
`,
	})
	p, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p == nil {
		t.Fatal("expected project")
	}
	if p.PythonVersion != ">=3.10" {
		t.Fatalf("PythonVersion: %q", p.PythonVersion)
	}
	if got := p.VersionOf("torch"); got != ">=2.0" {
		t.Fatalf("torch version: %q", got)
	}
	if got := p.VersionOf("transformers"); got != "==4.30.0" {
		t.Fatalf("transformers version: %q", got)
	}
	if got := p.VersionOf("Pillow"); got != "" {
		t.Fatalf("Pillow (no spec): %q", got)
	}
	// Normalization: pyproject says "python-dateutil"; query with underscores.
	if got := p.VersionOf("python_dateutil"); got != ">=2.8" {
		t.Fatalf("python_dateutil normalized lookup: %q", got)
	}
	if p.Source != SourcePyProject {
		t.Fatalf("source: %q", p.Source)
	}
}

func TestLoad_UvLockOverridesPyProject(t *testing.T) {
	root := writeFiles(t, map[string]string{
		"pyproject.toml": `
[project]
name = "demo"
requires-python = ">=3.10"
dependencies = ["torch>=2.0"]
`,
		"uv.lock": `
version = 1
requires-python = ">=3.11"

[[package]]
name = "torch"
version = "2.1.2"

[[package]]
name = "transformers"
version = "4.36.0"
`,
	})
	p, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p.Source != SourceUvLock {
		t.Fatalf("source: %q, want uv.lock", p.Source)
	}
	if got := p.VersionOf("torch"); got != "2.1.2" {
		t.Fatalf("torch (resolved): %q", got)
	}
	// uv.lock supplies a package not in pyproject, still surfaces.
	if got := p.VersionOf("transformers"); got != "4.36.0" {
		t.Fatalf("transformers: %q", got)
	}
	// pyproject's requires-python wins because it was set first; uv's
	// requires-python is only used when pyproject did not specify.
	if p.PythonVersion != ">=3.10" {
		t.Fatalf("PythonVersion: %q", p.PythonVersion)
	}
}

func TestLoad_FromSubdirectory(t *testing.T) {
	root := writeFiles(t, map[string]string{
		"pyproject.toml":      `[project]` + "\n" + `name="demo"` + "\n",
		"src/pkg/__init__.py": "",
	})
	deep := filepath.Join(root, "src", "pkg")
	p, err := Load(deep)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p == nil {
		t.Fatal("expected project from parent")
	}
}

func TestLoad_InvalidToml(t *testing.T) {
	root := writeFiles(t, map[string]string{
		"pyproject.toml": "this is = not [valid",
	})
	if _, err := Load(root); err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}

func TestParseDepSpec(t *testing.T) {
	cases := []struct {
		in              string
		wantName, wantV string
	}{
		{"torch>=2.0", "torch", ">=2.0"},
		{"transformers==4.30.0", "transformers", "==4.30.0"},
		{"Pillow", "pillow", ""},
		{"python_dateutil>=2.8", "python-dateutil", ">=2.8"},
		{"requests[serialization]==2.31.0", "requests", "==2.31.0"},
		{"torch>=2.0; python_version >= '3.10'", "torch", ">=2.0"},
		{"  numpy  ~=  1.26  ", "numpy", "~=  1.26"},
		{"", "", ""},
	}
	for _, c := range cases {
		gotName, gotV := parseDepSpec(c.in)
		if gotName != c.wantName || gotV != c.wantV {
			t.Errorf("parseDepSpec(%q) = (%q, %q); want (%q, %q)", c.in, gotName, gotV, c.wantName, c.wantV)
		}
	}
}

func TestNormalizeName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Pillow", "pillow"},
		{"python_dateutil", "python-dateutil"},
		{"python.dateutil", "python-dateutil"},
		{"PYTHON__DATE--UTIL", "python-date-util"},
		{"trailing-", "trailing"},
	}
	for _, c := range cases {
		if got := normalizeName(c.in); got != c.want {
			t.Errorf("normalizeName(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestVersionOf_NilProject(t *testing.T) {
	var p *Project
	if got := p.VersionOf("torch"); got != "" {
		t.Fatalf("nil project should return empty, got %q", got)
	}
}

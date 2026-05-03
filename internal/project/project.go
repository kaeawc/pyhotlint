// Package project will read pyproject.toml + lockfiles (uv first, then
// poetry/pdm/requirements) to feed version-drift rules and the oracle.
// MVP stub: returns an empty Project.
package project

// Project describes the relevant facts about a Python project root.
type Project struct {
	Root          string
	PythonVersion string
	Dependencies  map[string]string // name -> resolved version
}

// Load is a stub for the MVP: returns an empty Project regardless of root.
// Real implementation will detect uv.lock / poetry.lock / requirements.txt.
func Load(root string) (*Project, error) {
	return &Project{Root: root, Dependencies: map[string]string{}}, nil
}

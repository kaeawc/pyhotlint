package walker

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// scaffold builds a temporary tree from a path -> contents map and
// returns the temp root. Empty contents create files; paths ending in /
// create directories.
func scaffold(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, contents := range files {
		full := filepath.Join(root, rel)
		if strings.HasSuffix(rel, "/") {
			if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatalf("mkdirall %s: %v", full, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdirall %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	return root
}

// rel converts absolute paths in got back to slash-separated paths
// relative to root, for assertion-friendly comparison.
func rel(t *testing.T, root string, got []string) []string {
	t.Helper()
	out := make([]string, len(got))
	for i, p := range got {
		r, err := filepath.Rel(root, p)
		if err != nil {
			t.Fatalf("rel: %v", err)
		}
		out[i] = filepath.ToSlash(r)
	}
	return out
}

func TestFindFiles_SingleFile(t *testing.T) {
	root := scaffold(t, map[string]string{
		"a.py": "x = 1\n",
	})
	got, err := FindFiles([]string{filepath.Join(root, "a.py")})
	if err != nil {
		t.Fatalf("FindFiles: %v", err)
	}
	want := []string{"a.py"}
	if !reflect.DeepEqual(rel(t, root, got), want) {
		t.Fatalf("got %v, want %v", rel(t, root, got), want)
	}
}

func TestFindFiles_NonPyExplicitFileIncluded(t *testing.T) {
	// Explicit file paths bypass the extension filter — the user asked
	// for that file by name.
	root := scaffold(t, map[string]string{
		"weird.txt": "x = 1\n",
	})
	got, err := FindFiles([]string{filepath.Join(root, "weird.txt")})
	if err != nil {
		t.Fatalf("FindFiles: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected explicit file to be included, got %v", got)
	}
}

func TestFindFiles_DirectoryRecursion(t *testing.T) {
	root := scaffold(t, map[string]string{
		"a.py":          "x = 1\n",
		"pkg/b.py":      "y = 2\n",
		"pkg/sub/c.pyi": "z: int\n",
		"pkg/sub/d.txt": "ignore\n",
		"pkg/sub/e.py":  "q = 4\n",
		"README.md":     "# readme\n",
	})
	got, err := FindFiles([]string{root})
	if err != nil {
		t.Fatalf("FindFiles: %v", err)
	}
	want := []string{
		"a.py",
		"pkg/b.py",
		"pkg/sub/c.pyi",
		"pkg/sub/e.py",
	}
	if !reflect.DeepEqual(rel(t, root, got), want) {
		t.Fatalf("got %v, want %v", rel(t, root, got), want)
	}
}

func TestFindFiles_SkipsKnownDirs(t *testing.T) {
	root := scaffold(t, map[string]string{
		"src/a.py":                "x = 1\n",
		".venv/lib/installed.py":  "should_skip = 1\n",
		"node_modules/pkg/foo.py": "should_skip = 1\n",
		"__pycache__/cached.py":   "should_skip = 1\n",
		".git/config":             "[core]\n",
		"dist/wheel.py":           "should_skip = 1\n",
	})
	got, err := FindFiles([]string{root})
	if err != nil {
		t.Fatalf("FindFiles: %v", err)
	}
	if !reflect.DeepEqual(rel(t, root, got), []string{"src/a.py"}) {
		t.Fatalf("got %v, want only src/a.py", rel(t, root, got))
	}
}

func TestFindFiles_GlobExpansion(t *testing.T) {
	root := scaffold(t, map[string]string{
		"a.py":  "",
		"b.py":  "",
		"c.txt": "",
	})
	got, err := FindFiles([]string{filepath.Join(root, "*.py")})
	if err != nil {
		t.Fatalf("FindFiles: %v", err)
	}
	want := []string{"a.py", "b.py"}
	if !reflect.DeepEqual(rel(t, root, got), want) {
		t.Fatalf("got %v, want %v", rel(t, root, got), want)
	}
}

func TestFindFiles_GlobNoMatch(t *testing.T) {
	root := scaffold(t, map[string]string{"a.py": ""})
	// A glob with no matches returns no files and no error — matches
	// ruff/mypy behavior so user shell-expanded globs do not blow up
	// when a directory is empty.
	got, err := FindFiles([]string{filepath.Join(root, "missing-*.py")})
	if err != nil {
		t.Fatalf("FindFiles: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no matches, got %v", got)
	}
}

func TestFindFiles_DeduplicatedAndSorted(t *testing.T) {
	root := scaffold(t, map[string]string{
		"a.py":     "",
		"b.py":     "",
		"sub/c.py": "",
	})
	got, err := FindFiles([]string{
		root,
		filepath.Join(root, "a.py"), // duplicate
		filepath.Join(root, "*.py"), // duplicates a.py and b.py
	})
	if err != nil {
		t.Fatalf("FindFiles: %v", err)
	}
	want := []string{"a.py", "b.py", "sub/c.py"}
	if !reflect.DeepEqual(rel(t, root, got), want) {
		t.Fatalf("got %v, want %v", rel(t, root, got), want)
	}
}

func TestFindFiles_MissingPathErrors(t *testing.T) {
	_, err := FindFiles([]string{"/nonexistent-path-pyhotlint-test"})
	if err == nil {
		t.Fatal("expected error for missing path")
	}
}

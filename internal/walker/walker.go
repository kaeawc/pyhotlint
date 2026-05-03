// Package walker expands the CLI's positional path arguments into a
// flat, deterministic list of Python source files. It accepts files,
// directories, and shell-style globs; directories are walked
// recursively with a small skiplist of well-known venv / cache / build
// directories that should never be linted.
package walker

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// skipDirs is the set of directory base names we never descend into.
// Conservative on purpose — we do not skip all hidden directories,
// only those known to hold non-source content.
var skipDirs = map[string]struct{}{
	".git":             {},
	".hg":              {},
	".svn":             {},
	".tox":             {},
	".venv":            {},
	".pytest_cache":    {},
	".mypy_cache":      {},
	".ruff_cache":      {},
	".pyhotlint_cache": {},
	"__pycache__":      {},
	"node_modules":     {},
	"venv":             {},
	"env":              {},
	"dist":             {},
	"build":            {},
}

// pyExtensions is the set of file extensions treated as Python source.
var pyExtensions = map[string]struct{}{
	".py":  {},
	".pyi": {},
}

// FindFiles expands paths into Python source files.
//
//   - A path that is a regular file is included verbatim regardless of
//     extension (the user asked for it explicitly).
//   - A path that is a directory is walked recursively; only files
//     matching pyExtensions are returned, and skipDirs subtrees are
//     pruned.
//   - A path that contains a shell-style glob meta-character is
//     expanded with filepath.Glob; each match is then re-expanded by
//     this function.
//   - A non-existent path that does not parse as a glob returns an
//     error; missing globs return no matches without erroring (matches
//     ruff/mypy behavior).
//
// The returned list is deduplicated and sorted lexicographically.
func FindFiles(paths []string) ([]string, error) {
	seen := map[string]struct{}{}
	var out []string

	for _, p := range paths {
		matches, err := expandPath(p)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			if _, dup := seen[m]; dup {
				continue
			}
			seen[m] = struct{}{}
			out = append(out, m)
		}
	}

	sort.Strings(out)
	return out, nil
}

func expandPath(p string) ([]string, error) {
	if hasGlobMeta(p) {
		matches, err := filepath.Glob(p)
		if err != nil {
			return nil, fmt.Errorf("glob %q: %w", p, err)
		}
		var out []string
		for _, m := range matches {
			children, err := expandPath(m)
			if err != nil {
				return nil, err
			}
			out = append(out, children...)
		}
		return out, nil
	}

	info, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", p, err)
	}
	if !info.IsDir() {
		return []string{p}, nil
	}
	return walkDir(p)
}

func walkDir(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			if _, skip := skipDirs[d.Name()]; skip {
				return fs.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if _, ok := pyExtensions[ext]; ok {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// hasGlobMeta reports whether p contains a shell glob meta-character
// understood by filepath.Match (`*`, `?`, `[`).
func hasGlobMeta(p string) bool {
	return strings.ContainsAny(p, "*?[")
}

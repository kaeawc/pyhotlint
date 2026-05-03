package oracle

import (
	"os/exec"
	"path/filepath"
	"runtime"
)

// DiscoverPython resolves the Python interpreter to attach to. Order:
//
//  1. <root>/.venv/bin/python (POSIX) or .venv/Scripts/python.exe (Windows).
//  2. python3 on PATH.
//  3. python on PATH.
//
// Returns "" when none are usable; callers should fall back to Stub.
func DiscoverPython(root string) string {
	if p := venvPython(root); p != "" {
		return p
	}
	for _, name := range []string{"python3", "python"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

func venvPython(root string) string {
	if root == "" {
		return ""
	}
	var rel string
	if runtime.GOOS == "windows" {
		rel = filepath.Join(".venv", "Scripts", "python.exe")
	} else {
		rel = filepath.Join(".venv", "bin", "python")
	}
	candidate := filepath.Join(root, rel)
	if _, err := exec.LookPath(candidate); err == nil {
		return candidate
	}
	return ""
}

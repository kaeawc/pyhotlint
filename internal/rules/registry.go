// Package rules is the import-side bundle: importing it triggers each
// taxonomy package's init() to register its rules into v2.
package rules

import (
	// Side-effect imports — each package's init() registers its rules.
	_ "github.com/kaeawc/pyhotlint/internal/rules/async"
	_ "github.com/kaeawc/pyhotlint/internal/rules/server"
	_ "github.com/kaeawc/pyhotlint/internal/rules/tensor"
	_ "github.com/kaeawc/pyhotlint/internal/rules/versioning"
)

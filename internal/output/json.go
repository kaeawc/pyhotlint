// Package output renders findings to user-visible formats. MVP: JSON only.
package output

import (
	"encoding/json"
	"io"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
)

// WriteJSON emits findings as a single JSON array.
func WriteJSON(w io.Writer, findings []v2.Finding) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(findings)
}

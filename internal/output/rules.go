package output

import (
	"fmt"
	"io"
	"sort"
	"strings"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
)

// WriteRuleList prints the registered rules in a stable, tabular form
// for human consumption. Sorted first by category then by ID so two
// runs of the same binary always produce identical output (useful in
// docs / CI snapshots).
func WriteRuleList(w io.Writer, rules []*v2.Rule) error {
	sorted := make([]*v2.Rule, len(rules))
	copy(sorted, rules)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Category != sorted[j].Category {
			return sorted[i].Category < sorted[j].Category
		}
		return sorted[i].ID < sorted[j].ID
	})

	idW := len("ID")
	sevW := len("SEVERITY")
	catW := len("CATEGORY")
	for _, r := range sorted {
		if n := len(r.ID); n > idW {
			idW = n
		}
		if n := len(string(r.Severity)); n > sevW {
			sevW = n
		}
		if n := len(r.Category); n > catW {
			catW = n
		}
	}

	if _, err := fmt.Fprintf(w, "%-*s  %-*s  %-*s  %s\n",
		idW, "ID", sevW, "SEVERITY", catW, "CATEGORY", "DESCRIPTION"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, strings.Repeat("-", idW+sevW+catW+15+30)); err != nil {
		return err
	}
	for _, r := range sorted {
		if _, err := fmt.Fprintf(w, "%-*s  %-*s  %-*s  %s\n",
			idW, r.ID, sevW, string(r.Severity), catW, r.Category, r.Description); err != nil {
			return err
		}
	}
	return nil
}

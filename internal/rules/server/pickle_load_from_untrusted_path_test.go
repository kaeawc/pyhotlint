package server

import (
	"testing"

	"github.com/kaeawc/pyhotlint/internal/ruletest"
)

func TestPickleLoadPositive(t *testing.T) {
	ruletest.WalkPositives(t, "pickle-load-from-untrusted-path")
}

func TestPickleLoadNegative(t *testing.T) {
	ruletest.WalkNegatives(t, "pickle-load-from-untrusted-path")
}

package server

import (
	"testing"

	"github.com/kaeawc/pyhotlint/internal/ruletest"
)

func TestMetricNotLabeledPositive(t *testing.T) {
	ruletest.WalkPositives(t, "metric-not-labeled")
}

func TestMetricNotLabeledNegative(t *testing.T) {
	ruletest.WalkNegatives(t, "metric-not-labeled")
}

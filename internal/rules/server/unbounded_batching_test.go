package server

import (
	"testing"

	"github.com/kaeawc/pyhotlint/internal/ruletest"
)

func TestUnboundedBatchingPositive(t *testing.T) {
	ruletest.WalkPositives(t, "unbounded-batching")
}

func TestUnboundedBatchingNegative(t *testing.T) {
	ruletest.WalkNegatives(t, "unbounded-batching")
}

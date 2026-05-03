package tensor

import (
	"testing"

	"github.com/kaeawc/pyhotlint/internal/ruletest"
)

func TestUntrackedGradInEvalPositive(t *testing.T) {
	ruletest.WalkPositives(t, "untracked-grad-in-eval")
}

func TestUntrackedGradInEvalNegative(t *testing.T) {
	ruletest.WalkNegatives(t, "untracked-grad-in-eval")
}

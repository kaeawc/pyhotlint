package async

import (
	"testing"

	"github.com/kaeawc/pyhotlint/internal/ruletest"
)

func TestLockHeldAcrossAwaitPositive(t *testing.T) {
	ruletest.WalkPositives(t, "lock-held-across-await")
}

func TestLockHeldAcrossAwaitNegative(t *testing.T) {
	ruletest.WalkNegatives(t, "lock-held-across-await")
}

package async

import (
	"testing"

	"github.com/kaeawc/pyhotlint/internal/ruletest"
)

func TestCPUBoundLoopInEventLoopPositive(t *testing.T) {
	ruletest.WalkPositives(t, "cpu-bound-loop-in-event-loop")
}

func TestCPUBoundLoopInEventLoopNegative(t *testing.T) {
	ruletest.WalkNegatives(t, "cpu-bound-loop-in-event-loop")
}

package tensor

import (
	"testing"

	"github.com/kaeawc/pyhotlint/internal/ruletest"
)

func TestHostDeviceCopyInLoopPositive(t *testing.T) {
	ruletest.WalkPositives(t, "host-device-copy-in-loop")
}

func TestHostDeviceCopyInLoopNegative(t *testing.T) {
	ruletest.WalkNegatives(t, "host-device-copy-in-loop")
}

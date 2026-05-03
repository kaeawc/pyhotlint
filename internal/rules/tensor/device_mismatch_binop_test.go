package tensor

import (
	"testing"

	"github.com/kaeawc/pyhotlint/internal/ruletest"
)

func TestDeviceMismatchBinopPositive(t *testing.T) {
	ruletest.WalkPositives(t, "device-mismatch-binop")
}

func TestDeviceMismatchBinopNegative(t *testing.T) {
	ruletest.WalkNegatives(t, "device-mismatch-binop")
}

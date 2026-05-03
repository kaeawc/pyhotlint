package versioning

import (
	"testing"

	"github.com/kaeawc/pyhotlint/internal/ruletest"
)

func TestTokenizerModelIDMismatchPositive(t *testing.T) {
	ruletest.WalkPositives(t, "tokenizer-model-id-mismatch")
}

func TestTokenizerModelIDMismatchNegative(t *testing.T) {
	ruletest.WalkNegatives(t, "tokenizer-model-id-mismatch")
}

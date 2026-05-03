package async

import (
	"testing"

	"github.com/kaeawc/pyhotlint/internal/ruletest"
)

func TestSyncIOInAsyncFnPositive(t *testing.T) {
	ruletest.WalkPositives(t, "sync-io-in-async-fn")
}

func TestSyncIOInAsyncFnNegative(t *testing.T) {
	ruletest.WalkNegatives(t, "sync-io-in-async-fn")
}

package v2

import "testing"

func TestCached_BuildRunsOnce(t *testing.T) {
	ctx := NewContext("f.py", []byte(""), nil, nil, nil, new([]Finding))
	calls := 0
	build := func() any {
		calls++
		return calls
	}
	if got := ctx.Cached("k", build).(int); got != 1 {
		t.Fatalf("first Cached: got %d, want 1", got)
	}
	if got := ctx.Cached("k", build).(int); got != 1 {
		t.Fatalf("second Cached: got %d, want 1 (build must not re-run)", got)
	}
	if calls != 1 {
		t.Fatalf("build ran %d times, want 1", calls)
	}
}

func TestCached_DifferentKeysDoNotCollide(t *testing.T) {
	ctx := NewContext("f.py", []byte(""), nil, nil, nil, new([]Finding))
	a := ctx.Cached("a", func() any { return "alpha" }).(string)
	b := ctx.Cached("b", func() any { return "beta" }).(string)
	if a != "alpha" || b != "beta" {
		t.Fatalf("got %q / %q; want alpha / beta", a, b)
	}
}

func TestNewContext_OracleNilFallsBackToStub(t *testing.T) {
	ctx := NewContext("f.py", []byte(""), nil, nil, nil, new([]Finding))
	if ctx.Oracle == nil {
		t.Fatal("nil oracle should fall back to Stub, got nil")
	}
	if got := ctx.Oracle.DeviceOf("x"); got.Known {
		t.Fatalf("Stub.DeviceOf should be Unknown, got %v", got)
	}
}

func TestNewContext_RootStored(t *testing.T) {
	// Pass a non-nil sentinel pointer; we only care that Context retains it.
	ctx := NewContext("f.py", []byte(""), nil, nil, nil, new([]Finding))
	if ctx.Root != nil {
		t.Fatal("expected nil Root when passed nil")
	}
}

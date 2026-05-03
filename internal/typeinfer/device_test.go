package typeinfer

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/kaeawc/pyhotlint/internal/scanner"
)

func parse(t *testing.T, src string) (*scanner.ParsedFile, func()) {
	t.Helper()
	pf, err := scanner.ParseSource("test.py", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return pf, func() { pf.Close() }
}

func TestDevice_String(t *testing.T) {
	cases := map[Device]string{
		DeviceCPU:     "cpu",
		DeviceCUDA:    "cuda",
		DeviceMPS:     "mps",
		DeviceUnknown: "unknown",
	}
	for d, want := range cases {
		if got := d.String(); got != want {
			t.Errorf("Device(%d).String() = %q, want %q", d, got, want)
		}
	}
}

func TestTracker_DirectCalls(t *testing.T) {
	src := `a = x.cpu()
b = x.cuda()
c = x.to("cuda:0")
d = x.to("cpu")
e = x.to("mps")
f = x.to(some_var)
`
	pf, cleanup := parse(t, src)
	defer cleanup()
	tk := NewDeviceTracker(pf.Tree.RootNode(), pf.Source)

	cases := map[string]Device{
		"a": DeviceCPU,
		"b": DeviceCUDA,
		"c": DeviceCUDA,
		"d": DeviceCPU,
		"e": DeviceMPS,
	}
	for name, want := range cases {
		if got := tk.bindings[name]; got != want {
			t.Errorf("binding %q: got %s, want %s", name, got, want)
		}
	}
	if _, ok := tk.bindings["f"]; ok {
		t.Errorf("f should be unbound (variable arg to .to)")
	}
}

func TestTracker_Reassignment(t *testing.T) {
	src := `x = base.cpu()
x = base.cuda()
`
	pf, cleanup := parse(t, src)
	defer cleanup()
	tk := NewDeviceTracker(pf.Tree.RootNode(), pf.Source)
	if got := tk.bindings["x"]; got != DeviceCUDA {
		t.Fatalf("after reassignment expected cuda, got %s", got)
	}
}

func TestTracker_NestedDefDoesNotLeak(t *testing.T) {
	src := `a = base.cpu()

def inner():
    a = base.cuda()
    return a
`
	pf, cleanup := parse(t, src)
	defer cleanup()
	tk := NewDeviceTracker(pf.Tree.RootNode(), pf.Source)
	if got := tk.bindings["a"]; got != DeviceCPU {
		t.Fatalf("module-level a should remain cpu (nested def must not leak), got %s", got)
	}
}

func TestTracker_BinopAgreed(t *testing.T) {
	src := `a = base.cpu()
b = base.cpu()
c = a + b
`
	pf, cleanup := parse(t, src)
	defer cleanup()
	tk := NewDeviceTracker(pf.Tree.RootNode(), pf.Source)
	if got := tk.bindings["c"]; got != DeviceCPU {
		t.Fatalf("a + b (both cpu) should be cpu, got %s", got)
	}
}

func TestTracker_BinopDisagreed(t *testing.T) {
	src := `a = base.cpu()
b = base.cuda()
c = a + b
`
	pf, cleanup := parse(t, src)
	defer cleanup()
	tk := NewDeviceTracker(pf.Tree.RootNode(), pf.Source)
	if got, ok := tk.bindings["c"]; ok {
		t.Fatalf("a + b (cpu vs cuda) should not bind c, got %s", got)
	}
}

func TestTracker_Parenthesized(t *testing.T) {
	src := `a = (base.cpu())
`
	pf, cleanup := parse(t, src)
	defer cleanup()
	tk := NewDeviceTracker(pf.Tree.RootNode(), pf.Source)
	if got := tk.bindings["a"]; got != DeviceCPU {
		t.Fatalf("parenthesized .cpu() should still resolve, got %s", got)
	}
}

func TestTracker_NilExpr(t *testing.T) {
	tk := NewDeviceTracker(nil, []byte(""))
	if got := tk.DeviceOf(nil); got != DeviceUnknown {
		t.Errorf("DeviceOf(nil) = %s, want unknown", got)
	}
}

func TestTracker_DeviceOfBinopExpression(t *testing.T) {
	// Verify DeviceOf works on a binop AST node directly (not just via
	// the bindings table), which is the path the rule consumer uses.
	src := `c = a.cpu() + b.cuda()
`
	pf, cleanup := parse(t, src)
	defer cleanup()
	tk := NewDeviceTracker(pf.Tree.RootNode(), pf.Source)

	binop := findFirstNode(pf.Tree.RootNode(), "binary_operator")
	if binop == nil {
		t.Fatal("could not locate binary_operator")
	}
	if got := tk.DeviceOf(binop); got != DeviceUnknown {
		t.Fatalf("a.cpu() + b.cuda() should be Unknown (devices disagree), got %s", got)
	}
}

// findFirstNode walks root in preorder and returns the first node
// whose type matches.
func findFirstNode(root *sitter.Node, t string) *sitter.Node {
	if root == nil {
		return nil
	}
	if root.Type() == t {
		return root
	}
	for i := 0; i < int(root.ChildCount()); i++ {
		c := root.Child(i)
		if c == nil {
			continue
		}
		if got := findFirstNode(c, t); got != nil {
			return got
		}
	}
	return nil
}

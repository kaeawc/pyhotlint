package versioning

import (
	"testing"

	"github.com/kaeawc/pyhotlint/internal/project"
	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
	"github.com/kaeawc/pyhotlint/internal/scanner"
)

const ruleID = "transformers-pinned-but-config-newer"

func runWithProject(t *testing.T, src string, proj *project.Project) []v2.Finding {
	t.Helper()
	pf, err := scanner.ParseSource("test.py", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defer pf.Close()
	var rules []*v2.Rule
	for _, r := range v2.All() {
		if r.ID == ruleID {
			rules = append(rules, r)
		}
	}
	if len(rules) == 0 {
		t.Fatal("rule not registered")
	}
	return v2.Run(rules, proj, nil, pf.Path, pf.Source, pf.Tree.RootNode())
}

func newProject(transformersVersion string) *project.Project {
	return &project.Project{
		Root:         ".",
		Dependencies: map[string]string{"transformers": transformersVersion},
		Source:       project.SourceUvLock,
	}
}

func TestPositive_OldExactPin(t *testing.T) {
	src := `from transformers import AutoModel
m = AutoModel.from_pretrained("bert-base-uncased", attn_implementation="flash_attention_2")
`
	got := runWithProject(t, src, newProject("4.20.0"))
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %#v", len(got), got)
	}
}

func TestPositive_OldPyProjectSpec(t *testing.T) {
	src := `from transformers import AutoModelForCausalLM
m = AutoModelForCausalLM.from_pretrained("meta-llama/Llama-3", quantization_config={"load_in_4bit": True})
`
	got := runWithProject(t, src, newProject(">=4.20,<4.30"))
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %#v", len(got), got)
	}
}

func TestNegative_NewEnoughPin(t *testing.T) {
	src := `from transformers import AutoModel
m = AutoModel.from_pretrained("bert-base-uncased", attn_implementation="sdpa")
`
	got := runWithProject(t, src, newProject("4.40.0"))
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d: %#v", len(got), got)
	}
}

func TestNegative_NoProject(t *testing.T) {
	src := `from transformers import AutoModel
m = AutoModel.from_pretrained("bert-base-uncased", attn_implementation="sdpa")
`
	got := runWithProject(t, src, nil)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (no project), got %d", len(got))
	}
}

func TestNegative_TransformersNotPinned(t *testing.T) {
	src := `from transformers import AutoModel
m = AutoModel.from_pretrained("bert", attn_implementation="sdpa")
`
	proj := &project.Project{Root: ".", Dependencies: map[string]string{}}
	got := runWithProject(t, src, proj)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (no pin), got %d", len(got))
	}
}

func TestNegative_KwargNotInTable(t *testing.T) {
	src := `from transformers import AutoModel
m = AutoModel.from_pretrained("bert", trust_remote_code=False, low_cpu_mem_usage=True)
`
	got := runWithProject(t, src, newProject("4.10.0"))
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (kwargs not tracked), got %d", len(got))
	}
}

func TestNegative_NotFromPretrained(t *testing.T) {
	src := `def configure(model, attn_implementation="sdpa"):
    return model
configure(None, attn_implementation="flash_attention_2")
`
	got := runWithProject(t, src, newProject("4.20.0"))
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (regular call, not from_pretrained), got %d", len(got))
	}
}

func TestParseVersionLowerBound(t *testing.T) {
	cases := []struct {
		in     string
		want   [3]int
		wantOk bool
	}{
		{"4.30.0", [3]int{4, 30, 0}, true},
		{">=4.30", [3]int{4, 30, 0}, true},
		{"~=4.30,<5", [3]int{4, 30, 0}, true},
		{"==4.30.0", [3]int{4, 30, 0}, true},
		{"  >=  4.30.5  ", [3]int{4, 30, 5}, true},
		{"4", [3]int{4, 0, 0}, true},
		{"", [3]int{}, false},
		{"latest", [3]int{}, false},
	}
	for _, c := range cases {
		got, ok := parseVersionLowerBound(c.in)
		if ok != c.wantOk || got != c.want {
			t.Errorf("parseVersionLowerBound(%q) = (%v, %v); want (%v, %v)", c.in, got, ok, c.want, c.wantOk)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	if compareVersions([3]int{4, 30, 0}, [3]int{4, 30, 0}) != 0 {
		t.Fatal("equal compare")
	}
	if compareVersions([3]int{4, 30, 0}, [3]int{4, 30, 1}) >= 0 {
		t.Fatal("less compare on patch")
	}
	if compareVersions([3]int{4, 30, 0}, [3]int{4, 29, 99}) <= 0 {
		t.Fatal("greater compare on minor")
	}
	if compareVersions([3]int{5, 0, 0}, [3]int{4, 99, 99}) <= 0 {
		t.Fatal("greater compare on major")
	}
}

package versioning

import (
	"fmt"
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
)

// configFieldRequirements lists transformers kwargs alongside the
// minimum (major, minor, patch) version that introduced them. The
// table is intentionally small for MVP — production rule libraries
// generate this from HF release notes; we hand-curate three of the
// most-common load-time kwargs that show up in real inference code.
var configFieldRequirements = map[string][3]int{
	"attn_implementation": {4, 36, 0}, // FlashAttention 2 selector
	"quantization_config": {4, 30, 0}, // bitsandbytes / GPTQ
	"device_map":          {4, 20, 0}, // Accelerate-backed sharding
}

func init() {
	v2.Register(&v2.Rule{
		ID:          "transformers-pinned-but-config-newer",
		Category:    "versioning",
		Severity:    v2.SeverityError,
		Description: "A from_pretrained kwarg requires a transformers version newer than the project pins.",
		NodeTypes:   []string{"call"},
		Needs:       v2.NeedsProject,
		Confidence:  0.85,
		Check:       checkTransformersConfigNewer,
	})
}

func checkTransformersConfigNewer(ctx *v2.Context, call *sitter.Node) {
	if ctx.Project == nil {
		return
	}
	pinned := ctx.Project.VersionOf("transformers")
	if pinned == "" {
		return
	}
	pinnedTuple, ok := parseVersionLowerBound(pinned)
	if !ok {
		return
	}
	if !isFromPretrainedCall(call, ctx.Source) {
		return
	}
	args := call.ChildByFieldName("arguments")
	if args == nil {
		return
	}
	for i := 0; i < int(args.NamedChildCount()); i++ {
		c := args.NamedChild(i)
		if c == nil || c.Type() != "keyword_argument" {
			continue
		}
		nameNode := c.ChildByFieldName("name")
		if nameNode == nil {
			continue
		}
		kw := string(ctx.Source[nameNode.StartByte():nameNode.EndByte()])
		req, has := configFieldRequirements[kw]
		if !has {
			continue
		}
		if compareVersions(pinnedTuple, req) >= 0 {
			continue
		}
		ctx.Emit(c, fmt.Sprintf(
			"kwarg %q requires transformers >= %d.%d.%d but project pins %s",
			kw, req[0], req[1], req[2], pinned))
	}
}

// isFromPretrainedCall reports whether call is a `<X>.from_pretrained(...)` invocation.
func isFromPretrainedCall(call *sitter.Node, src []byte) bool {
	fnExpr := call.ChildByFieldName("function")
	if fnExpr == nil || fnExpr.Type() != "attribute" {
		return false
	}
	leaf := fnExpr.ChildByFieldName("attribute")
	if leaf == nil {
		return false
	}
	return string(src[leaf.StartByte():leaf.EndByte()]) == "from_pretrained"
}

// parseVersionLowerBound extracts the leading (major, minor, patch)
// from a PEP 440 / PEP 508 version spec. Accepts forms like:
//
//	"4.30.0"           -> (4, 30, 0)
//	">=4.30"           -> (4, 30, 0)
//	"~=4.30,<5"        -> (4, 30, 0)
//	"==4.30.0"         -> (4, 30, 0)
//
// Best-effort: anything we cannot parse returns (_, false), and the
// rule simply skips so it does not false-positive on exotic specs.
func parseVersionLowerBound(spec string) ([3]int, bool) {
	s := strings.TrimSpace(spec)
	for _, prefix := range []string{">=", "<=", "==", "~=", "!=", ">", "<"} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimSpace(s[len(prefix):])
			break
		}
	}
	// Take up to first non-version character (comma, space, letter, etc.).
	end := 0
	for end < len(s) && (s[end] == '.' || (s[end] >= '0' && s[end] <= '9')) {
		end++
	}
	s = s[:end]
	if s == "" {
		return [3]int{}, false
	}
	parts := strings.Split(s, ".")
	var out [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}

func compareVersions(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// Package server holds rules in the "server hygiene" taxonomy,
// including security checks specific to model-serving stacks.
package server

import (
	sitter "github.com/smacker/go-tree-sitter"

	"github.com/kaeawc/pyhotlint/internal/rules/v2"
)

// pickleLoadCalls maps the dotted call name to the message used in the
// finding. Receiver/owner proof — only the explicit `pickle.` form
// matches; a user-defined `pickle` parameter shadowing the module is
// out of scope for MVP.
var pickleLoadCalls = map[string]string{
	"pickle.load":  "pickle.load deserializes arbitrary Python objects and can execute attacker-controlled code; treat all inputs as untrusted",
	"pickle.loads": "pickle.loads deserializes arbitrary Python objects and can execute attacker-controlled code; treat all inputs as untrusted",
}

func init() {
	v2.Register(&v2.Rule{
		ID:          "pickle-load-from-untrusted-path",
		Category:    "security",
		Severity:    v2.SeverityError,
		Description: "pickle.load / pickle.loads can execute arbitrary code on deserialization. Avoid on data sourced from disk paths, network input, or model registries that are not cryptographically verified.",
		NodeTypes:   []string{"call"},
		Confidence:  0.9,
		Check:       checkPickleLoad,
	})
}

func checkPickleLoad(ctx *v2.Context, call *sitter.Node) {
	fnExpr := call.ChildByFieldName("function")
	if fnExpr == nil || fnExpr.Type() != "attribute" {
		return
	}
	text := string(ctx.Source[fnExpr.StartByte():fnExpr.EndByte()])
	if msg, ok := pickleLoadCalls[text]; ok {
		ctx.Emit(call, msg)
	}
}

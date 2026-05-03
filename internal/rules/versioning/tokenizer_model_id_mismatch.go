// Package versioning holds rules in the "versioning / drift" taxonomy.
package versioning

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	v2 "github.com/kaeawc/pyhotlint/internal/rules/v2"
)

func init() {
	v2.Register(&v2.Rule{
		ID:          "tokenizer-model-id-mismatch",
		Category:    "versioning",
		Severity:    v2.SeverityError,
		Description: "Tokenizer and model in the same scope are loaded from different pretrained IDs; tokenization will not match the model's vocabulary.",
		NodeTypes:   []string{"module"},
		Confidence:  0.9,
		Check:       checkTokenizerModelIDMismatch,
	})
}

type fromPretrainedCall struct {
	kind  string // "tokenizer" or "model"
	id    string
	node  *sitter.Node
	scope *sitter.Node // nearest function_definition, or the module
}

func checkTokenizerModelIDMismatch(ctx *v2.Context, module *sitter.Node) {
	calls := collectFromPretrainedCalls(module, module, ctx.Source)
	byScope := map[*sitter.Node][]fromPretrainedCall{}
	for _, c := range calls {
		byScope[c.scope] = append(byScope[c.scope], c)
	}
	for _, group := range byScope {
		emitMismatches(ctx, group)
	}
}

// collectFromPretrainedCalls walks the subtree rooted at n, attributing
// each from_pretrained call to its nearest enclosing function (or to
// the module when at top level).
func collectFromPretrainedCalls(n, scope *sitter.Node, src []byte) []fromPretrainedCall {
	var out []fromPretrainedCall
	if n.Type() == "function_definition" {
		scope = n
	}
	if n.Type() == "call" {
		if entry, ok := classifyFromPretrained(n, src); ok {
			entry.scope = scope
			out = append(out, entry)
		}
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		c := n.Child(i)
		if c == nil {
			continue
		}
		out = append(out, collectFromPretrainedCalls(c, scope, src)...)
	}
	return out
}

func emitMismatches(ctx *v2.Context, group []fromPretrainedCall) {
	modelID, ok := firstModelID(group)
	if !ok {
		return
	}
	for _, c := range group {
		if c.kind == "tokenizer" && c.id != modelID {
			ctx.Emit(c.node, fmt.Sprintf("tokenizer ID %q does not match model ID %q in the same scope", c.id, modelID))
		}
	}
}

func firstModelID(group []fromPretrainedCall) (string, bool) {
	for _, c := range group {
		if c.kind == "model" {
			return c.id, true
		}
	}
	return "", false
}

func classifyFromPretrained(call *sitter.Node, src []byte) (fromPretrainedCall, bool) {
	fnExpr := call.ChildByFieldName("function")
	if fnExpr == nil || fnExpr.Type() != "attribute" {
		return fromPretrainedCall{}, false
	}
	attrLeaf := fnExpr.ChildByFieldName("attribute")
	if attrLeaf == nil {
		return fromPretrainedCall{}, false
	}
	if string(src[attrLeaf.StartByte():attrLeaf.EndByte()]) != "from_pretrained" {
		return fromPretrainedCall{}, false
	}
	receiver := fnExpr.ChildByFieldName("object")
	if receiver == nil {
		return fromPretrainedCall{}, false
	}
	receiverText := strings.TrimSpace(string(src[receiver.StartByte():receiver.EndByte()]))
	last := receiverText
	if idx := strings.LastIndex(receiverText, "."); idx >= 0 {
		last = receiverText[idx+1:]
	}
	kind := classifyClassName(last)
	if kind == "" {
		return fromPretrainedCall{}, false
	}
	args := call.ChildByFieldName("arguments")
	if args == nil {
		return fromPretrainedCall{}, false
	}
	first := args.NamedChild(0)
	if first == nil || first.Type() != "string" {
		return fromPretrainedCall{}, false
	}
	id := stringLiteralValue(first, src)
	if id == "" {
		return fromPretrainedCall{}, false
	}
	return fromPretrainedCall{kind: kind, id: id, node: call}, true
}

func classifyClassName(name string) string {
	switch {
	case strings.Contains(name, "Tokenizer"):
		return "tokenizer"
	case strings.Contains(name, "Model"),
		strings.Contains(name, "ForCausalLM"),
		strings.Contains(name, "ForSeq2SeqLM"),
		strings.Contains(name, "ForMaskedLM"),
		strings.Contains(name, "ForQuestionAnswering"),
		strings.Contains(name, "ForSequenceClassification"),
		strings.Contains(name, "ForTokenClassification"):
		return "model"
	}
	return ""
}

func stringLiteralValue(node *sitter.Node, src []byte) string {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		c := node.NamedChild(i)
		if c != nil && c.Type() == "string_content" {
			return string(src[c.StartByte():c.EndByte()])
		}
	}
	raw := string(src[node.StartByte():node.EndByte()])
	if len(raw) >= 2 && (raw[0] == '"' || raw[0] == '\'') && raw[len(raw)-1] == raw[0] {
		return raw[1 : len(raw)-1]
	}
	return raw
}

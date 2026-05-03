// Package scanner wraps tree-sitter Python parsing with a small, allocation-
// conscious surface that rule code consumes. Mirrors the role of Krit's
// internal/scanner package, scoped down for MVP — no flat AST, no
// cross-file index, no parse cache yet.
package scanner

import (
	"context"
	"fmt"
	"os"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// ParsedFile is the unit of input handed to rules. The tree is owned by
// the file and must be closed via Close when analysis is complete.
type ParsedFile struct {
	Path   string
	Source []byte
	Tree   *sitter.Tree
}

// Close releases tree-sitter resources.
func (p *ParsedFile) Close() {
	if p.Tree != nil {
		p.Tree.Close()
		p.Tree = nil
	}
}

var parserPool = sync.Pool{
	New: func() any {
		p := sitter.NewParser()
		p.SetLanguage(python.GetLanguage())
		return p
	},
}

// ParseFile reads path from disk and parses it as Python.
func ParseFile(path string) (*ParsedFile, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return ParseSource(path, src)
}

// ParseSource parses src as Python under the logical path.
func ParseSource(path string, src []byte) (*ParsedFile, error) {
	p := parserPool.Get().(*sitter.Parser)
	defer parserPool.Put(p)
	tree, err := p.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &ParsedFile{Path: path, Source: src, Tree: tree}, nil
}

// NodeText returns the source slice covered by node.
func NodeText(src []byte, n *sitter.Node) string {
	if n == nil {
		return ""
	}
	return string(src[n.StartByte():n.EndByte()])
}

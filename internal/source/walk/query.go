package walk

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source"
)

func (m *Model) Root() *parser.Node {
	if m == nil || m.File == nil {
		return nil
	}
	return m.File.Root
}

func (m *Model) Source() []byte {
	if m == nil || m.File == nil {
		return nil
	}
	return m.File.Source
}

func (m *Model) Parent(n *parser.Node) *parser.Node {
	if m == nil || n == nil {
		return nil
	}
	return m.parents[n]
}

func (m *Model) NextSibling(n *parser.Node) *parser.Node {
	p := m.Parent(n)
	if p == nil {
		return nil
	}
	for i, c := range p.Children {
		if c == n && i+1 < len(p.Children) {
			return p.Children[i+1]
		}
	}
	return nil
}

func (m *Model) PrevSibling(n *parser.Node) *parser.Node {
	p := m.Parent(n)
	if p == nil {
		return nil
	}
	for i, c := range p.Children {
		if c == n && i > 0 {
			return p.Children[i-1]
		}
	}
	return nil
}

func (m *Model) Ancestors(n *parser.Node) []*parser.Node {
	var out []*parser.Node
	for cur := m.Parent(n); cur != nil; cur = m.Parent(cur) {
		out = append(out, cur)
	}
	return out
}

func (m *Model) OfKind(k parser.Kind) []*parser.Node {
	if m == nil {
		return nil
	}
	return m.byKind[k]
}

func (m *Model) All() []*parser.Node {
	if m == nil || m.File == nil || m.File.Root == nil {
		return nil
	}
	var nodes []*parser.Node
	var collect func(*parser.Node)
	collect = func(node *parser.Node) {
		nodes = append(nodes, node)
		for _, child := range node.Children {
			collect(child)
		}
	}
	collect(m.File.Root)
	return nodes
}

func (m *Model) Range(n *parser.Node) source.Range {
	if m == nil || n == nil {
		return source.Range{}
	}
	return m.LineTable.Range(n.Start, n.End)
}

func (m *Model) Text(n *parser.Node) string {
	if m == nil || n == nil || m.File == nil {
		return ""
	}
	return n.Text(m.File.Source)
}

func (m *Model) TokenText(tok token.Token) string {
	if m == nil || m.File == nil {
		return ""
	}
	return tok.Text(m.File.Source)
}

func (m *Model) EnclosingFunction(n *parser.Node) *parser.Node {
	for _, a := range m.Ancestors(n) {
		if a.Kind == parser.KindFunctionDefinition {
			return a
		}
	}
	return nil
}

func (m *Model) Iter(visit func(*parser.Node)) {
	if m == nil || m.File == nil || m.File.Root == nil {
		return
	}
	var iter func(*parser.Node)
	iter = func(node *parser.Node) {
		visit(node)
		for _, child := range node.Children {
			iter(child)
		}
	}
	iter(m.File.Root)
}

func (m *Model) IterKind(k parser.Kind, visit func(*parser.Node)) {
	if m == nil {
		return
	}
	for _, n := range m.byKind[k] {
		visit(n)
	}
}

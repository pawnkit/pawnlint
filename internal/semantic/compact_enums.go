package semantic

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

func (m *CompactModel) evaluateCompactEnums() {
	for _, declaration := range m.Walk.OfKind(parser.KindEnumDeclaration) {
		if m.Walk.Tree.Field(declaration, "increment") != syntax.NoNode || m.Walk.Uncertain(declaration) {
			continue
		}
		body := m.Walk.Tree.Field(declaration, "body")
		if body == syntax.NoNode {
			continue
		}
		current := int64(0)
		known := true
		for _, entry := range m.compactEnumEntries(body) {
			symbol := m.declSymbols[entry]
			if symbol == nil || symbol.Ambiguous {
				known = false
				continue
			}
			if explicit := m.Walk.Tree.Field(entry, "value"); explicit != syntax.NoNode {
				current, known = m.Eval(explicit)
			}
			if known {
				m.constants[symbol] = current
			}
			step, stepKnown := m.compactEnumEntryWidth(entry)
			if !known || !stepKnown {
				known = false
				continue
			}
			current = cell(current + step)
		}
		if root := m.declSymbols[declaration]; root != nil && !root.Ambiguous && known {
			m.constants[root] = current
		}
	}
}

func (m *CompactModel) compactEnumEntries(body syntax.NodeID) []syntax.NodeID {
	var entries []syntax.NodeID
	var collect func(syntax.NodeID)
	collect = func(node syntax.NodeID) {
		switch m.Walk.Tree.Kind(node) {
		case parser.KindConditionalRegion, parser.KindConditionalBranch:
			for index := 0; index < m.Walk.Tree.ChildCount(node); index++ {
				collect(m.Walk.Tree.Child(node, index))
			}
		case parser.KindEnumEntry:
			entries = append(entries, node)
		}
	}
	for index := 0; index < m.Walk.Tree.ChildCount(body); index++ {
		collect(m.Walk.Tree.Child(body, index))
	}
	return entries
}

func (m *CompactModel) compactEnumEntryWidth(entry syntax.NodeID) (int64, bool) {
	width := int64(1)
	for index := 0; index < m.Walk.Tree.ChildCount(entry); index++ {
		child := m.Walk.Tree.Child(entry, index)
		if m.Walk.Tree.Kind(child) != parser.KindDimension {
			continue
		}
		size, ok := m.Eval(m.Walk.Tree.Field(child, "size"))
		if !ok || size <= 0 {
			return 0, false
		}
		width = cell(width * size)
		if width <= 0 {
			return 0, false
		}
	}
	return width, true
}

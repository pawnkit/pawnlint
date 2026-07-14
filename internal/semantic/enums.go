package semantic

import "github.com/pawnkit/pawn-parser"

func (m *Model) evaluateEnums() {
	for _, declaration := range m.Walk.OfKind(parser.KindEnumDeclaration) {
		if declaration.Field("increment") != nil || m.Walk.Uncertain(declaration) {
			continue
		}
		body := declaration.Field("body")
		if body == nil {
			continue
		}
		current := int64(0)
		known := true
		for _, entry := range body.Children {
			if entry.Kind != parser.KindEnumEntry {
				continue
			}
			symbol := m.declSymbols[entry]
			if symbol == nil || symbol.Ambiguous {
				known = false
				continue
			}
			if explicit := entry.Field("value"); explicit != nil {
				current, known = m.Eval(explicit)
			}
			if known {
				m.constantValues[symbol] = current
			}
			step, stepKnown := m.enumEntryWidth(entry)
			if !known || !stepKnown {
				known = false
				continue
			}
			current = cell(current + step)
		}
		if root := m.declSymbols[declaration]; root != nil && !root.Ambiguous && known {
			m.constantValues[root] = current
		}
	}
}

func (m *Model) enumEntryWidth(entry *parser.Node) (int64, bool) {
	width := int64(1)
	for _, child := range entry.Children {
		if child.Kind != parser.KindDimension {
			continue
		}
		size, ok := m.Eval(child.Field("size"))
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

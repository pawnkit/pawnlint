package semantic

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
)

func (m *Model) declare(decl, nameNode *parser.Node, kind SymbolKind, scope *Scope, function *parser.Node) {
	if nameNode == nil || nameNode.HasError || m.Walk.Uncertain(decl) {
		return
	}
	name := m.Walk.Text(nameNode)
	if name == "" {
		return
	}
	tags := m.symbolTags(decl, kind, name)
	states, stateRaw := m.symbolStates(decl, kind)
	symbol := &Symbol{Name: name, Kind: kind, Decl: decl, NameNode: nameNode, Scope: scope, Function: function, Tags: tags, States: states, StateRaw: stateRaw}
	if len(tags) == 1 {
		symbol.Tag = tags[0]
	}
	if kind == SymbolEnumRoot || kind == SymbolEnumEntry {
		symbol.Constant = true
		symbol.Value = decl.Field("value")
	}
	if kind == SymbolGlobal || kind == SymbolLocal {
		declaration := m.Walk.Parent(decl)
		if hasQualifier(declaration, token.KwConst) {
			symbol.Constant = true
			symbol.Value = decl.Field("initializer")
		}
	}
	for _, existing := range scope.symbols[name] {
		if stateVariantsCoexist(existing, symbol) {
			continue
		}
		existing.Ambiguous = true
		symbol.Ambiguous = true
	}
	m.Symbols = append(m.Symbols, symbol)
	if kind == SymbolEnumRoot || kind == SymbolEnumEntry {
		m.declSymbols[decl] = symbol
	}
	if scope.symbols == nil {
		scope.symbols = make(map[string][]*Symbol)
	}
	scope.symbols[name] = append(scope.symbols[name], symbol)
}

func (m *Model) symbolStates(decl *parser.Node, kind SymbolKind) ([]string, bool) {
	if kind != SymbolFunction {
		return nil, false
	}
	selector := decl.Field("state")
	if selector == nil {
		return nil, false
	}
	if selector.Kind != parser.KindTaggedType {
		return nil, true
	}
	var states []string
	for _, child := range selector.Children {
		if child.Kind != parser.KindIdentifier {
			return nil, true
		}
		states = append(states, m.Walk.Text(child))
	}
	if len(states) == 0 {
		return nil, true
	}
	return states, false
}

func (m *Model) enclosingEnum(entry *parser.Node) *parser.Node {
	node := entry
	for {
		parent := m.Walk.Parent(node)
		if parent == nil {
			return nil
		}
		switch parent.Kind {
		case parser.KindConditionalRegion, parser.KindConditionalBranch:
			node = parent
			continue
		case parser.KindBlock:
			enum := m.Walk.Parent(parent)
			if enum != nil && enum.Kind == parser.KindEnumDeclaration {
				return enum
			}
			return nil
		default:
			return nil
		}
	}
}

func stateVariantsCoexist(left, right *Symbol) bool {
	if left.Kind != SymbolFunction || right.Kind != SymbolFunction {
		return false
	}
	if left.Decl.Kind != right.Decl.Kind {
		return true
	}
	leftState := left.Decl.Field("state") != nil
	rightState := right.Decl.Field("state") != nil
	if !leftState && !rightState {
		return false
	}
	if leftState != rightState {
		return true
	}
	if left.StateRaw || right.StateRaw {
		return false
	}
	for _, a := range left.States {
		for _, b := range right.States {
			if a == b {
				return false
			}
		}
	}
	return true
}

func (m *Model) symbolTags(decl *parser.Node, kind SymbolKind, name string) []string {
	if kind == SymbolEnumEntry {
		enum := m.enclosingEnum(decl)
		if enum != nil {
			if tags := m.tagNames(enum.Field("tag")); len(tags) != 0 {
				return tags
			}
			if enumName := enum.Field("name"); enumName != nil {
				return []string{m.Walk.Text(enumName)}
			}
		}
		return nil
	}
	if kind == SymbolEnumRoot {
		if tags := m.tagNames(decl.Field("tag")); len(tags) != 0 {
			return tags
		}
		return []string{name}
	}
	return m.tagNames(decl.Field("tag"))
}

func (m *Model) tagNames(node *parser.Node) []string {
	if node == nil || node.Kind != parser.KindTaggedType {
		return nil
	}
	var tags []string
	for _, child := range node.Children {
		if child.Kind != parser.KindIdentifier {
			return nil
		}
		tags = append(tags, m.Walk.Text(child))
	}
	if len(tags) == 0 && node.Tok.Kind == token.Identifier {
		tags = append(tags, node.Tok.Text(m.File.Source))
	}
	return tags
}

func hasQualifier(node *parser.Node, kind token.Kind) bool {
	if node == nil {
		return false
	}
	for _, child := range node.Children {
		if child.Tok.Kind == kind {
			return true
		}
	}
	return false
}

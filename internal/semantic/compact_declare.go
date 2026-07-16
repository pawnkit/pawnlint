package semantic

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

func (m *CompactModel) declareCompact(decl, nameNode syntax.NodeID, kind SymbolKind, scope *CompactScope, function syntax.NodeID) {
	if !m.Walk.Tree.Valid(nameNode) || m.Walk.Tree.HasError(nameNode) || m.Walk.Uncertain(decl) {
		return
	}
	name := m.Walk.Text(nameNode)
	if name == "" {
		return
	}
	tags := m.compactSymbolTags(decl, kind, name)
	states, stateRaw := m.compactSymbolStates(decl, kind)
	symbol := &CompactSymbol{Name: name, Kind: kind, Decl: decl, NameNode: nameNode, Scope: scope, Function: function, Tags: tags, States: states, StateRaw: stateRaw, Value: syntax.NoNode}
	if len(tags) == 1 {
		symbol.Tag = tags[0]
	}
	if kind == SymbolEnumRoot || kind == SymbolEnumEntry {
		symbol.Constant = true
		symbol.Value = m.Walk.Tree.Field(decl, "value")
	}
	if kind == SymbolGlobal || kind == SymbolLocal {
		declaration := m.Walk.Parent(decl)
		if m.compactHasQualifier(declaration, token.KwConst) {
			symbol.Constant = true
			symbol.Value = m.Walk.Tree.Field(decl, "initializer")
		}
	}
	for _, existing := range scope.symbols[name] {
		if compactStateVariantsCoexist(m, existing, symbol) {
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
		scope.symbols = make(map[string][]*CompactSymbol)
	}
	scope.symbols[name] = append(scope.symbols[name], symbol)
}

func (m *CompactModel) compactSymbolStates(decl syntax.NodeID, kind SymbolKind) ([]string, bool) {
	if kind != SymbolFunction {
		return nil, false
	}
	selector := m.Walk.Tree.Field(decl, "state")
	if selector == syntax.NoNode {
		return nil, false
	}
	if m.Walk.Tree.Kind(selector) != parser.KindTaggedType {
		return nil, true
	}
	var states []string
	for index := 0; index < m.Walk.Tree.ChildCount(selector); index++ {
		child := m.Walk.Tree.Child(selector, index)
		if m.Walk.Tree.Kind(child) != parser.KindIdentifier {
			return nil, true
		}
		states = append(states, m.Walk.Text(child))
	}
	return states, len(states) == 0
}

func (m *CompactModel) compactEnclosingEnum(entry syntax.NodeID) syntax.NodeID {
	node := entry
	for {
		parent := m.Walk.Parent(node)
		if parent == syntax.NoNode {
			return syntax.NoNode
		}
		switch m.Walk.Tree.Kind(parent) {
		case parser.KindConditionalRegion, parser.KindConditionalBranch:
			node = parent
		case parser.KindBlock:
			enum := m.Walk.Parent(parent)
			if enum != syntax.NoNode && m.Walk.Tree.Kind(enum) == parser.KindEnumDeclaration {
				return enum
			}
			return syntax.NoNode
		default:
			return syntax.NoNode
		}
	}
}

func compactStateVariantsCoexist(model *CompactModel, left, right *CompactSymbol) bool {
	if left.Kind != SymbolFunction || right.Kind != SymbolFunction {
		return false
	}
	if model.Walk.Tree.Kind(left.Decl) != model.Walk.Tree.Kind(right.Decl) {
		return true
	}
	leftState := model.Walk.Tree.Field(left.Decl, "state") != syntax.NoNode
	rightState := model.Walk.Tree.Field(right.Decl, "state") != syntax.NoNode
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

func (m *CompactModel) compactSymbolTags(decl syntax.NodeID, kind SymbolKind, name string) []string {
	if kind == SymbolEnumEntry {
		enum := m.compactEnclosingEnum(decl)
		if enum != syntax.NoNode {
			if tags := m.compactTagNames(m.Walk.Tree.Field(enum, "tag")); len(tags) != 0 {
				return tags
			}
			if enumName := m.Walk.Tree.Field(enum, "name"); enumName != syntax.NoNode {
				return []string{m.Walk.Text(enumName)}
			}
		}
		return nil
	}
	if kind == SymbolEnumRoot {
		if tags := m.compactTagNames(m.Walk.Tree.Field(decl, "tag")); len(tags) != 0 {
			return tags
		}
		return []string{name}
	}
	return m.compactTagNames(m.Walk.Tree.Field(decl, "tag"))
}

func (m *CompactModel) compactTagNames(node syntax.NodeID) []string {
	if node == syntax.NoNode || m.Walk.Tree.Kind(node) != parser.KindTaggedType {
		return nil
	}
	var tags []string
	for index := 0; index < m.Walk.Tree.ChildCount(node); index++ {
		child := m.Walk.Tree.Child(node, index)
		if m.Walk.Tree.Kind(child) != parser.KindIdentifier {
			return nil
		}
		tags = append(tags, m.Walk.Text(child))
	}
	if len(tags) == 0 && m.Walk.Tree.TokenKind(node) == token.Identifier {
		tags = append(tags, m.Walk.Tree.TokenText(node))
	}
	return tags
}

func (m *CompactModel) compactHasQualifier(node syntax.NodeID, kind token.Kind) bool {
	if node == syntax.NoNode {
		return false
	}
	for index := 0; index < m.Walk.Tree.ChildCount(node); index++ {
		if m.Walk.Tree.TokenKind(m.Walk.Tree.Child(node, index)) == kind {
			return true
		}
	}
	return false
}

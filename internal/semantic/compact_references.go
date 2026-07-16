package semantic

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

func (m *CompactModel) Resolve(node syntax.NodeID) *CompactSymbol {
	if m == nil || !m.Walk.Tree.Valid(node) {
		return nil
	}
	return m.facts[node].resolved
}

func (m *CompactModel) References(symbol *CompactSymbol) []CompactReference {
	return m.references[symbol]
}

func (m *CompactModel) UnresolvedReferences() []CompactUnresolvedReference {
	return m.unresolved
}

func (m *CompactModel) compactReferenceTarget(node syntax.NodeID) (ReferenceTarget, bool) {
	allowed, reference := m.compactReferenceFilter(node)
	if !reference {
		return 0, false
	}
	if allowed != nil && allowed(SymbolFunction) && !allowed(SymbolGlobal) {
		return ReferenceFunction, true
	}
	return ReferenceValue, true
}

func (m *CompactModel) Shadowed(symbol *CompactSymbol) *CompactSymbol {
	if symbol == nil || symbol.Scope == nil || symbol.Ambiguous {
		return nil
	}
	for scope := symbol.Scope.Parent; scope != nil; scope = scope.Parent {
		candidates := scope.symbols[symbol.Name]
		if len(candidates) == 0 {
			continue
		}
		if len(candidates) != 1 || candidates[0].Ambiguous {
			return nil
		}
		outer := candidates[0]
		if outer.Kind == SymbolLocal && m.Walk.Tree.Start(outer.NameNode) > m.Walk.Tree.Start(symbol.NameNode) {
			return nil
		}
		return outer
	}
	return nil
}

func (m *CompactModel) resolveCompact(node syntax.NodeID) *CompactSymbol {
	allowed, reference := m.compactReferenceFilter(node)
	if !reference {
		return nil
	}
	return m.compactResolveInScope(node, allowed)
}

func (m *CompactModel) ResolveAsCallTarget(node syntax.NodeID) *CompactSymbol {
	return m.compactResolveInScope(node, func(kind SymbolKind) bool { return kind != SymbolLabel })
}

func (m *CompactModel) compactResolveInScope(node syntax.NodeID, allowed func(SymbolKind) bool) *CompactSymbol {
	name := m.Walk.Text(node)
	for scope := m.facts[node].scope; scope != nil; scope = scope.Parent {
		var candidates []*CompactSymbol
		for _, candidate := range scope.symbols[name] {
			if allowed == nil || allowed(candidate.Kind) {
				candidates = append(candidates, candidate)
			}
		}
		if len(candidates) == 0 {
			continue
		}
		if len(candidates) == 1 {
			symbol := candidates[0]
			if symbol.Kind == SymbolLocal && m.Walk.Tree.Start(symbol.NameNode) > m.Walk.Tree.Start(node) {
				return nil
			}
			return symbol
		}
		if symbol := m.uniqueCompactFunctionDefinition(candidates); symbol != nil {
			return symbol
		}
		return nil
	}
	return nil
}

func (m *CompactModel) compactReferenceFilter(node syntax.NodeID) (func(SymbolKind) bool, bool) {
	parent := m.Walk.Parent(node)
	if parent == syntax.NoNode {
		return nil, false
	}
	switch m.Walk.Tree.Kind(parent) {
	case parser.KindTaggedType, parser.KindStateStatement,
		parser.KindDirectiveDefine, parser.KindDirectiveUndef, parser.KindDefinedExpression,
		parser.KindVariableDeclaration, parser.KindEnumDeclaration, parser.KindParameterList,
		parser.KindFunctionDefinition, parser.KindFunctionDeclaration, parser.KindParameter:
		return nil, false
	case parser.KindDimension:
		if m.Walk.Tree.Field(parent, "packed") == node {
			return nil, false
		}
	case parser.KindTaggedExpression:
		if m.Walk.Tree.Field(parent, "tag") == node {
			return nil, false
		}
	case parser.KindBinaryExpression:
		if m.Walk.Tree.TokenKind(parent) == token.Dot && m.Walk.Tree.Field(parent, "right") == node {
			return nil, false
		}
	case parser.KindGotoStatement:
		return func(kind SymbolKind) bool { return kind == SymbolLabel }, true
	case parser.KindCallExpression:
		if m.Walk.Tree.Field(parent, "function") == node || m.Walk.Tree.Field(parent, "callee") == node {
			return func(kind SymbolKind) bool { return kind == SymbolFunction }, true
		}
	}
	return func(kind SymbolKind) bool { return kind != SymbolLabel && kind != SymbolFunction }, true
}

func (m *CompactModel) uniqueCompactFunctionDefinition(candidates []*CompactSymbol) *CompactSymbol {
	var definition *CompactSymbol
	for _, candidate := range candidates {
		if candidate.Kind != SymbolFunction {
			return nil
		}
		if m.Walk.Tree.Kind(candidate.Decl) != parser.KindFunctionDefinition {
			continue
		}
		if definition != nil {
			return nil
		}
		definition = candidate
	}
	return definition
}

func (m *CompactModel) compactReferenceKind(node syntax.NodeID) ReferenceKind {
	parent := m.Walk.Parent(node)
	if parent == syntax.NoNode {
		return ReferenceRead
	}
	switch m.Walk.Tree.Kind(parent) {
	case parser.KindAssignmentExpression:
		if m.Walk.Tree.Field(parent, "left") == node {
			if m.Walk.Tree.TokenKind(parent) == token.Assign {
				return ReferenceWrite
			}
			return ReferenceReadWrite
		}
	case parser.KindUpdateExpression:
		return ReferenceReadWrite
	case parser.KindCallExpression:
		if m.Walk.Tree.Field(parent, "function") == node || m.Walk.Tree.Field(parent, "callee") == node {
			return ReferenceCall
		}
	}
	return ReferenceRead
}

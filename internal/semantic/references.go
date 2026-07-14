package semantic

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
)

func (m *Model) Resolve(node *parser.Node) *Symbol {
	return m.resolved[node]
}

func (m *Model) References(symbol *Symbol) []Reference {
	return m.references[symbol]
}

func (m *Model) UnresolvedReferences() []UnresolvedReference {
	return m.unresolved
}

func (m *Model) referenceTarget(node *parser.Node) (ReferenceTarget, bool) {
	allowed, reference := m.referenceFilter(node)
	if !reference {
		return 0, false
	}
	if allowed != nil && allowed(SymbolFunction) && !allowed(SymbolGlobal) {
		return ReferenceFunction, true
	}
	return ReferenceValue, true
}

func (m *Model) Shadowed(symbol *Symbol) *Symbol {
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
		if outer.Kind == SymbolLocal && outer.NameNode.Start > symbol.NameNode.Start {
			return nil
		}
		return outer
	}
	return nil
}

func (m *Model) resolve(node *parser.Node) *Symbol {
	allowed, reference := m.referenceFilter(node)
	if !reference {
		return nil
	}
	name := m.Walk.Text(node)
	for scope := m.nodeScopes[node]; scope != nil; scope = scope.Parent {
		var candidates []*Symbol
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
			if symbol.Kind == SymbolLocal && symbol.NameNode.Start > node.Start {
				return nil
			}
			return symbol
		}
		if symbol := uniqueFunctionDefinition(candidates); symbol != nil {
			return symbol
		}
		return nil
	}
	return nil
}

func (m *Model) referenceFilter(node *parser.Node) (func(SymbolKind) bool, bool) {
	parent := m.Walk.Parent(node)
	if parent == nil {
		return nil, false
	}
	switch parent.Kind {
	case parser.KindTaggedType, parser.KindStateStatement,
		parser.KindDirectiveDefine, parser.KindDirectiveUndef, parser.KindDefinedExpression,
		parser.KindVariableDeclaration, parser.KindEnumDeclaration, parser.KindParameterList:
		return nil, false
	case parser.KindFunctionDefinition, parser.KindFunctionDeclaration, parser.KindParameter:
		return nil, false
	case parser.KindDimension:
		if parent.Field("packed") == node {
			return nil, false
		}
	case parser.KindTaggedExpression:
		if parent.Field("tag") == node {
			return nil, false
		}
	case parser.KindBinaryExpression:
		if parent.Tok.Kind == token.Dot && parent.Field("right") == node {
			return nil, false
		}
	case parser.KindGotoStatement:
		return func(kind SymbolKind) bool { return kind == SymbolLabel }, true
	case parser.KindCallExpression:
		if parent.Field("function") == node || parent.Field("callee") == node {
			return func(kind SymbolKind) bool { return kind == SymbolFunction }, true
		}
	}
	return func(kind SymbolKind) bool { return kind != SymbolLabel && kind != SymbolFunction }, true
}

func uniqueFunctionDefinition(candidates []*Symbol) *Symbol {
	var definition *Symbol
	for _, candidate := range candidates {
		if candidate.Kind != SymbolFunction {
			return nil
		}
		if candidate.Decl.Kind != parser.KindFunctionDefinition {
			continue
		}
		if definition != nil {
			return nil
		}
		definition = candidate
	}
	return definition
}

func (m *Model) referenceKind(node *parser.Node) ReferenceKind {
	parent := m.Walk.Parent(node)
	if parent == nil {
		return ReferenceRead
	}
	switch parent.Kind {
	case parser.KindAssignmentExpression:
		if parent.Field("left") == node {
			if parent.Tok.Kind == token.Assign {
				return ReferenceWrite
			}
			return ReferenceReadWrite
		}
	case parser.KindUpdateExpression:
		return ReferenceReadWrite
	case parser.KindCallExpression:
		if parent.Field("function") == node || parent.Field("callee") == node {
			return ReferenceCall
		}
	}
	return ReferenceRead
}

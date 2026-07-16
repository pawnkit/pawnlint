package semantic

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

type CompactSymbol struct {
	Name      string
	Kind      SymbolKind
	Decl      syntax.NodeID
	NameNode  syntax.NodeID
	Scope     *CompactScope
	Ambiguous bool
	Function  syntax.NodeID
	Tag       string
	Tags      []string
	States    []string
	StateRaw  bool
	Constant  bool
	Value     syntax.NodeID
}

type CompactReference struct {
	Node syntax.NodeID
	Kind ReferenceKind
}

type CompactUnresolvedReference struct {
	Node   syntax.NodeID
	Kind   ReferenceKind
	Target ReferenceTarget
}

type CompactScope struct {
	Parent  *CompactScope
	Node    syntax.NodeID
	symbols map[string][]*CompactSymbol
}

type compactNodeFacts struct {
	resolved *CompactSymbol
	scope    *CompactScope
}

type CompactModel struct {
	File        *parser.CompactFile
	Walk        *walk.CompactModel
	Root        *CompactScope
	Symbols     []*CompactSymbol
	facts       []compactNodeFacts
	references  map[*CompactSymbol][]CompactReference
	unresolved  []CompactUnresolvedReference
	declSymbols map[syntax.NodeID]*CompactSymbol
	constants   map[*CompactSymbol]int64
}

func BuildCompact(file *parser.CompactFile, tree *walk.CompactModel) *CompactModel {
	if tree == nil {
		tree = walk.NewCompact("", file)
	}
	model := &CompactModel{
		File:        file,
		Walk:        tree,
		facts:       make([]compactNodeFacts, tree.Tree.Len()),
		references:  make(map[*CompactSymbol][]CompactReference),
		declSymbols: make(map[syntax.NodeID]*CompactSymbol),
		constants:   make(map[*CompactSymbol]int64),
	}
	model.Root = &CompactScope{Node: tree.Root()}
	model.collectCompact(tree.Root(), model.Root, syntax.NoNode, nil)
	for _, node := range tree.OfKind(parser.KindIdentifier) {
		if tree.Inactive(node) || model.compactDeclarationName(node) {
			continue
		}
		symbol := model.resolveCompact(node)
		if symbol == nil {
			if target, ok := model.compactReferenceTarget(node); ok {
				model.unresolved = append(model.unresolved, CompactUnresolvedReference{Node: node, Kind: model.compactReferenceKind(node), Target: target})
			}
			continue
		}
		model.facts[node].resolved = symbol
		model.references[symbol] = append(model.references[symbol], CompactReference{Node: node, Kind: model.compactReferenceKind(node)})
	}
	model.evaluateCompactEnums()
	return model
}

func (m *CompactModel) collectCompact(node syntax.NodeID, scope *CompactScope, function syntax.NodeID, functionScope *CompactScope) {
	if !m.Walk.Tree.Valid(node) {
		return
	}
	if m.Walk.Tree.Kind(node) == parser.KindIdentifier && !m.compactDeclarationName(node) {
		if _, reference := m.compactReferenceFilter(node); reference {
			m.facts[node].scope = scope
		}
	}
	switch m.Walk.Tree.Kind(node) {
	case parser.KindFunctionDefinition, parser.KindFunctionDeclaration:
		m.declareCompact(node, m.Walk.Tree.Field(node, "name"), SymbolFunction, scope, node)
		function = node
		scope = newCompactScope(scope, node)
		functionScope = scope
	case parser.KindParameter:
		m.declareCompact(node, m.Walk.Tree.Field(node, "name"), SymbolParameter, scope, function)
	case parser.KindVariableDeclarator:
		kind := SymbolGlobal
		if function != syntax.NoNode {
			kind = SymbolLocal
		}
		m.declareCompact(node, m.Walk.Tree.Field(node, "name"), kind, scope, function)
	case parser.KindEnumDeclaration:
		m.declareCompact(node, m.Walk.Tree.Field(node, "name"), SymbolEnumRoot, scope, function)
	case parser.KindEnumEntry:
		m.declareCompact(node, m.Walk.Tree.Field(node, "name"), SymbolEnumEntry, scope, function)
	case parser.KindLabelStatement:
		if functionScope != nil {
			m.declareCompact(node, m.Walk.Tree.Field(node, "label"), SymbolLabel, functionScope, function)
		}
	case parser.KindBlock:
		parent := m.Walk.Parent(node)
		if parent == syntax.NoNode || m.Walk.Tree.Kind(parent) != parser.KindEnumDeclaration {
			scope = newCompactScope(scope, node)
		}
	case parser.KindForStatement, parser.KindCaseClause, parser.KindDefaultClause:
		scope = newCompactScope(scope, node)
	}
	for index := 0; index < m.Walk.Tree.ChildCount(node); index++ {
		m.collectCompact(m.Walk.Tree.Child(node, index), scope, function, functionScope)
	}
}

func (m *CompactModel) compactDeclarationName(node syntax.NodeID) bool {
	parent := m.Walk.Parent(node)
	if parent == syntax.NoNode {
		return false
	}
	field := "name"
	if m.Walk.Tree.Kind(parent) == parser.KindLabelStatement {
		field = "label"
	}
	switch m.Walk.Tree.Kind(parent) {
	case parser.KindFunctionDefinition, parser.KindFunctionDeclaration, parser.KindParameter,
		parser.KindVariableDeclarator, parser.KindEnumDeclaration, parser.KindEnumEntry,
		parser.KindLabelStatement:
		return m.Walk.Tree.Field(parent, field) == node
	default:
		return false
	}
}

func newCompactScope(parent *CompactScope, node syntax.NodeID) *CompactScope {
	return &CompactScope{Parent: parent, Node: node}
}

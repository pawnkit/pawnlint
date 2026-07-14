package semantic

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

type SymbolKind uint8

const (
	SymbolFunction SymbolKind = iota
	SymbolGlobal
	SymbolLocal
	SymbolParameter
	SymbolEnumRoot
	SymbolEnumEntry
	SymbolLabel
)

type Symbol struct {
	Name      string
	Kind      SymbolKind
	Decl      *parser.Node
	NameNode  *parser.Node
	Scope     *Scope
	Ambiguous bool
	Function  *parser.Node
	Tag       string
	Tags      []string
	States    []string
	StateRaw  bool
	Constant  bool
	Value     *parser.Node
}

type ReferenceKind uint8

const (
	ReferenceRead ReferenceKind = iota
	ReferenceWrite
	ReferenceReadWrite
	ReferenceCall
)

type Reference struct {
	Node *parser.Node
	Kind ReferenceKind
}

type ReferenceTarget uint8

const (
	ReferenceValue ReferenceTarget = iota
	ReferenceFunction
)

type UnresolvedReference struct {
	Node   *parser.Node
	Kind   ReferenceKind
	Target ReferenceTarget
}

type Scope struct {
	Parent  *Scope
	Node    *parser.Node
	symbols map[string][]*Symbol
}

type Model struct {
	File           *parser.File
	Walk           *walk.Model
	Root           *Scope
	Symbols        []*Symbol
	resolved       map[*parser.Node]*Symbol
	references     map[*Symbol][]Reference
	unresolved     []UnresolvedReference
	nodeScopes     map[*parser.Node]*Scope
	declNames      map[*parser.Node]struct{}
	functionScopes map[*parser.Node]*Scope
	declSymbols    map[*parser.Node]*Symbol
	constantValues map[*Symbol]int64
}

func Build(file *parser.File, tree *walk.Model) *Model {
	if tree == nil {
		tree = walk.New("", file)
	}
	m := &Model{
		File:           file,
		Walk:           tree,
		resolved:       make(map[*parser.Node]*Symbol),
		references:     make(map[*Symbol][]Reference),
		nodeScopes:     make(map[*parser.Node]*Scope),
		declNames:      make(map[*parser.Node]struct{}),
		functionScopes: make(map[*parser.Node]*Scope),
		declSymbols:    make(map[*parser.Node]*Symbol),
		constantValues: make(map[*Symbol]int64),
	}
	m.Root = &Scope{Node: tree.Root(), symbols: make(map[string][]*Symbol)}
	m.collect(tree.Root(), m.Root, nil)
	for _, node := range tree.OfKind(parser.KindIdentifier) {
		if m.Walk.Inactive(node) {
			continue
		}
		if _, declared := m.declNames[node]; declared {
			continue
		}
		symbol := m.resolve(node)
		if symbol == nil {
			if target, ok := m.referenceTarget(node); ok {
				m.unresolved = append(m.unresolved, UnresolvedReference{Node: node, Kind: m.referenceKind(node), Target: target})
			}
			continue
		}
		m.resolved[node] = symbol
		m.references[symbol] = append(m.references[symbol], Reference{Node: node, Kind: m.referenceKind(node)})
	}
	m.evaluateEnums()
	return m
}

func (m *Model) collect(node *parser.Node, scope *Scope, function *parser.Node) {
	if node == nil {
		return
	}
	m.nodeScopes[node] = scope
	switch node.Kind {
	case parser.KindFunctionDefinition, parser.KindFunctionDeclaration:
		m.declare(node, node.Field("name"), SymbolFunction, scope, node)
		function = node
		scope = newScope(scope, node)
		m.functionScopes[node] = scope
	case parser.KindParameter:
		m.declare(node, node.Field("name"), SymbolParameter, scope, function)
	case parser.KindVariableDeclarator:
		kind := SymbolGlobal
		if function != nil {
			kind = SymbolLocal
		}
		m.declare(node, node.Field("name"), kind, scope, function)
	case parser.KindEnumDeclaration:
		m.declare(node, node.Field("name"), SymbolEnumRoot, scope, function)
	case parser.KindEnumEntry:
		m.declare(node, node.Field("name"), SymbolEnumEntry, scope, function)
	case parser.KindLabelStatement:
		if functionScope := m.functionScopes[function]; functionScope != nil {
			m.declare(node, node.Field("label"), SymbolLabel, functionScope, function)
		}
	case parser.KindBlock:
		parent := m.Walk.Parent(node)
		if parent == nil || parent.Kind != parser.KindEnumDeclaration {
			scope = newScope(scope, node)
		}
	case parser.KindForStatement, parser.KindCaseClause, parser.KindDefaultClause:
		scope = newScope(scope, node)
	}
	for _, child := range node.Children {
		m.collect(child, scope, function)
	}
}

func newScope(parent *Scope, node *parser.Node) *Scope {
	return &Scope{Parent: parent, Node: node, symbols: make(map[string][]*Symbol)}
}

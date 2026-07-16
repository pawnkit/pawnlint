package walk

import (
	"sort"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

type CompactModel struct {
	Tree       *syntax.CompactTree
	Path       string
	LineTable  *source.LineTable
	branches   []branchState
	states     []uint8
	directives []syntax.NodeID
	defines    *DefineContext
	snapshots  []DefineSnapshot
	complete   bool
}

const (
	compactUncertain uint8 = 1 << iota
	compactInactive
)

func NewCompact(path string, file *parser.CompactFile) *CompactModel {
	return NewCompactWithContext(path, file, NewDefineContext(nil), nil, false, nil)
}

func NewCompactWithDefines(path string, file *parser.CompactFile, defines []string) *CompactModel {
	return NewCompactWithContext(path, file, NewDefineContext(defines), nil, false, nil)
}

func NewCompactWithDefineSnapshots(path string, file *parser.CompactFile, defines []string, snapshots []DefineSnapshot) *CompactModel {
	return NewCompactWithContext(path, file, NewDefineContext(defines), snapshots, false, nil)
}

func NewCompactWithDefineContext(path string, file *parser.CompactFile, defines []string, snapshots []DefineSnapshot, complete bool) *CompactModel {
	return NewCompactWithContext(path, file, NewDefineContext(defines), snapshots, complete, nil)
}

func NewCompactWithContext(path string, file *parser.CompactFile, defines *DefineContext, snapshots []DefineSnapshot, complete bool, lineTable *source.LineTable) *CompactModel {
	tree := syntax.NewCompactTree(file)
	if lineTable == nil {
		lineTable = source.NewLineTable(tree.Source())
	}
	if defines == nil {
		defines = NewDefineContext(nil)
	}
	model := &CompactModel{
		Tree:      tree,
		Path:      path,
		LineTable: lineTable,
		branches:  make([]branchState, tree.Len()),
		states:    make([]uint8, tree.Len()),
		defines:   defines,
		snapshots: cloneDefineSnapshots(snapshots),
		complete:  complete,
	}
	model.directives = append(model.directives, tree.OfKind(parser.KindDirectiveDefine)...)
	model.directives = append(model.directives, tree.OfKind(parser.KindDirectiveUndef)...)
	sort.SliceStable(model.directives, func(i, j int) bool {
		return tree.Start(model.directives[i]) < tree.Start(model.directives[j])
	})
	if tree.Valid(tree.Root()) {
		model.indexCompactConditionalStates()
		model.indexCompactNodeStates()
	}
	return model
}

func (m *CompactModel) Root() syntax.NodeID {
	if m == nil || m.Tree == nil {
		return syntax.NoNode
	}
	return m.Tree.Root()
}

func (m *CompactModel) Source() []byte {
	if m == nil || m.Tree == nil {
		return nil
	}
	return m.Tree.Source()
}

func (m *CompactModel) Parent(node syntax.NodeID) syntax.NodeID {
	if m == nil || m.Tree == nil {
		return syntax.NoNode
	}
	return m.Tree.Parent(node)
}

func (m *CompactModel) NextSibling(node syntax.NodeID) syntax.NodeID {
	parent := m.Parent(node)
	if parent == syntax.NoNode {
		return syntax.NoNode
	}
	for index := 0; index < m.Tree.ChildCount(parent); index++ {
		if m.Tree.Child(parent, index) == node && index+1 < m.Tree.ChildCount(parent) {
			return m.Tree.Child(parent, index+1)
		}
	}
	return syntax.NoNode
}

func (m *CompactModel) PrevSibling(node syntax.NodeID) syntax.NodeID {
	parent := m.Parent(node)
	if parent == syntax.NoNode {
		return syntax.NoNode
	}
	for index := 0; index < m.Tree.ChildCount(parent); index++ {
		if m.Tree.Child(parent, index) == node && index > 0 {
			return m.Tree.Child(parent, index-1)
		}
	}
	return syntax.NoNode
}

func (m *CompactModel) Ancestors(node syntax.NodeID) []syntax.NodeID {
	var ancestors []syntax.NodeID
	for current := m.Parent(node); current != syntax.NoNode; current = m.Parent(current) {
		ancestors = append(ancestors, current)
	}
	return ancestors
}

func (m *CompactModel) OfKind(kind parser.Kind) []syntax.NodeID {
	if m == nil || m.Tree == nil {
		return nil
	}
	return m.Tree.OfKind(kind)
}

func (m *CompactModel) All() []syntax.NodeID {
	if m == nil || m.Tree == nil || !m.Tree.Valid(m.Root()) {
		return nil
	}
	nodes := make([]syntax.NodeID, 0, m.Tree.Len())
	m.Iter(func(node syntax.NodeID) {
		nodes = append(nodes, node)
	})
	return nodes
}

func (m *CompactModel) Range(node syntax.NodeID) source.Range {
	if m == nil || m.Tree == nil || !m.Tree.Valid(node) {
		return source.Range{}
	}
	return m.LineTable.Range(m.Tree.Start(node), m.Tree.End(node))
}

func (m *CompactModel) Text(node syntax.NodeID) string {
	if m == nil || m.Tree == nil {
		return ""
	}
	return m.Tree.Text(node)
}

func (m *CompactModel) EnclosingFunction(node syntax.NodeID) syntax.NodeID {
	for _, ancestor := range m.Ancestors(node) {
		if m.Tree.Kind(ancestor) == parser.KindFunctionDefinition {
			return ancestor
		}
	}
	return syntax.NoNode
}

func (m *CompactModel) Iter(visit func(syntax.NodeID)) {
	if m == nil || m.Tree == nil || !m.Tree.Valid(m.Root()) {
		return
	}
	var iter func(syntax.NodeID)
	iter = func(node syntax.NodeID) {
		visit(node)
		for index := 0; index < m.Tree.ChildCount(node); index++ {
			iter(m.Tree.Child(node, index))
		}
	}
	iter(m.Root())
}

func (m *CompactModel) IterKind(kind parser.Kind, visit func(syntax.NodeID)) {
	for _, node := range m.OfKind(kind) {
		visit(node)
	}
}

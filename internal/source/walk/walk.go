// Package walk provides traversal and query helpers over the pawn-parser CST,
// including a cached index of nodes by Kind.
package walk

import (
	"sort"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/source"
)

type Model struct {
	File      *parser.File
	Path      string
	LineTable *source.LineTable
	index     *Index
	branches  map[*parser.Node]branchState
	states    map[*parser.Node]nodeState
	defines   *DefineContext
	snapshots []DefineSnapshot
	complete  bool
}

type Index struct {
	parents    map[*parser.Node]*parser.Node
	byKind     [parser.KindMacroInvocation + 1][]*parser.Node
	directives []*parser.Node
}

type DefineSnapshot struct {
	Offset  int
	Defines []string
}

type DefineContext struct {
	names []string
}

type branchState uint8

type nodeState uint8

var compilerDefines = []string{
	"true", "false", "EOS", "cellbits", "cellmax", "cellmin", "charbits",
	"charmin", "charmax", "ucharmax", "__Pawn", "__PawnBuild", "__line",
	"__compat", "debug",
}

const (
	branchActive branchState = iota
	branchInactive
	branchUncertain
)

const (
	nodeUncertain nodeState = 1 << iota
	nodeInactive
)

func New(path string, pf *parser.File) *Model {
	return NewWithDefines(path, pf, nil)
}

func NewWithDefines(path string, pf *parser.File, defines []string) *Model {
	return NewWithDefineSnapshots(path, pf, defines, nil)
}

func NewWithDefineSnapshots(path string, pf *parser.File, defines []string, snapshots []DefineSnapshot) *Model {
	return NewWithDefineContext(path, pf, defines, snapshots, false)
}

func NewWithDefineContext(path string, pf *parser.File, defines []string, snapshots []DefineSnapshot, complete bool) *Model {
	return NewWithContext(path, pf, NewDefineContext(defines), snapshots, complete, nil, nil)
}

func NewDefineContext(defines []string) *DefineContext {
	names := make([]string, 0, len(compilerDefines)+len(defines))
	names = append(names, compilerDefines...)
	names = append(names, defines...)
	sort.Strings(names)
	unique := names[:0]
	for _, name := range names {
		if name != "" && (len(unique) == 0 || unique[len(unique)-1] != name) {
			unique = append(unique, name)
		}
	}
	return &DefineContext{names: unique}
}

func NewWithContext(path string, pf *parser.File, defines *DefineContext, snapshots []DefineSnapshot, complete bool, lineTable *source.LineTable, index *Index) *Model {
	var src []byte
	if pf != nil {
		src = pf.Source
	}
	if lineTable == nil {
		lineTable = source.NewLineTable(src)
	}
	if defines == nil {
		defines = NewDefineContext(nil)
	}
	if index == nil {
		index = NewIndex(pf)
	}
	m := &Model{
		File:      pf,
		Path:      path,
		LineTable: lineTable,
		index:     index,
		branches:  make(map[*parser.Node]branchState),
		states:    make(map[*parser.Node]nodeState),
		defines:   defines,
		snapshots: cloneDefineSnapshots(snapshots),
		complete:  complete,
	}
	if pf != nil && pf.Root != nil {
		m.indexConditionalStates()
		m.indexNodeStates()
	}
	return m
}

func NewIndex(pf *parser.File) *Index {
	index := &Index{}
	if pf != nil && pf.Root != nil {
		var counts [parser.KindMacroInvocation + 1]int
		directives := countIndexNodes(pf.Root, &counts)
		nodes := 0
		for kind, count := range counts {
			nodes += count
			if count != 0 {
				index.byKind[kind] = make([]*parser.Node, 0, count)
			}
		}
		index.parents = make(map[*parser.Node]*parser.Node, nodes)
		index.directives = make([]*parser.Node, 0, directives)
		index.add(pf.Root, nil)
	}
	return index
}

func countIndexNodes(node *parser.Node, counts *[parser.KindMacroInvocation + 1]int) int {
	if node == nil {
		return 0
	}
	counts[node.Kind]++
	directives := 0
	if node.Kind == parser.KindDirectiveDefine || node.Kind == parser.KindDirectiveUndef {
		directives++
	}
	for _, child := range node.Children {
		directives += countIndexNodes(child, counts)
	}
	return directives
}

func cloneDefineSnapshots(snapshots []DefineSnapshot) []DefineSnapshot {
	cloned := make([]DefineSnapshot, len(snapshots))
	for i, snapshot := range snapshots {
		cloned[i] = DefineSnapshot{Offset: snapshot.Offset, Defines: append([]string(nil), snapshot.Defines...)}
	}
	sort.SliceStable(cloned, func(i, j int) bool { return cloned[i].Offset < cloned[j].Offset })
	return cloned
}

func (i *Index) add(n, parent *parser.Node) {
	if n == nil {
		return
	}
	i.parents[n] = parent
	i.byKind[n.Kind] = append(i.byKind[n.Kind], n)
	if n.Kind == parser.KindDirectiveDefine || n.Kind == parser.KindDirectiveUndef {
		i.directives = append(i.directives, n)
	}
	for _, c := range n.Children {
		i.add(c, n)
	}
}
